package block

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/store"
	uevent "github.com/dymensionxyz/dymint/utils/event"

	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
	cmtproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"

	"github.com/dymensionxyz/dymint/types"
)

// ProduceBlockLoop is calling publishBlock in a loop as long as we're synced.
// A signal will be sent to the bytesProduced channel for each block produced
// In this way it's possible to pause block production by not consuming the channel
func (m *Manager) ProduceBlockLoop(ctx context.Context, bytesProducedC chan int) error {
	m.logger.Info("Started block producer loop.")

	ticker := time.NewTicker(m.Conf.BlockTime)
	defer func() {
		ticker.Stop()
		m.logger.Info("Stopped block producer loop.")
	}()

	var nextEmptyBlock time.Time
	firstBlock := true

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// if empty blocks are configured to be enabled, and one is scheduled...
			produceEmptyBlock := firstBlock || m.Conf.MaxIdleTime == 0 || nextEmptyBlock.Before(time.Now())
			firstBlock = false

			block, commit, err := m.ProduceApplyGossipBlock(ctx, produceEmptyBlock)
			if errors.Is(err, context.Canceled) {
				m.logger.Error("Produce and gossip: context canceled.", "error", err)
				return nil
			}
			if errors.Is(err, types.ErrEmptyBlock) { // occurs if the block was empty but we don't want to produce one
				continue
			}
			if errors.Is(err, ErrNonRecoverable) {
				uevent.MustPublish(ctx, m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
				return err
			}
			if err != nil {
				m.logger.Error("Produce and gossip: uncategorized, assuming recoverable.", "error", err)
				continue
			}

			nextEmptyBlock = time.Now().Add(m.Conf.MaxIdleTime)
			if 0 < len(block.Data.Txs) {
				// the block wasn't empty so we want to make sure we don't wait too long before producing another one, in order to facilitate proofs for ibc
				// TODO: optimize to only do this if IBC transactions are present (https://github.com/dymensionxyz/dymint/issues/709)
				nextEmptyBlock = time.Now().Add(m.Conf.MaxProofTime)
			} else {
				m.logger.Info("Produced empty block.")
			}

			bytesProducedN := block.SizeBytes() + commit.SizeBytes()
			m.logger.Info("New block.", "size", uint64(block.ToProto().Size()))
			select {
			case <-ctx.Done():
				return nil
			case bytesProducedC <- bytesProducedN:
			default:
				evt := &events.DataHealthStatus{Error: fmt.Errorf("bytes produced channel is full: %w", gerrc.ErrResourceExhausted)}
				uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
				m.logger.Error("Enough bytes to build a batch have been accumulated, but too many batches are pending submission. " +
					"Pausing block production until a signal is consumed.")
				select {
				case <-ctx.Done():
					return nil
				case bytesProducedC <- bytesProducedN:
					evt := &events.DataHealthStatus{Error: nil}
					uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
					m.logger.Info("Resumed block production.")
				}
			}

		}
	}
}

// ProduceApplyGossipLastBlock produces and applies a block with the given nextProposerHash.
func (m *Manager) ProduceApplyGossipLastBlock(ctx context.Context, nextProposerHash [32]byte) (err error) {
	_, _, err = m.produceApplyGossip(ctx, true, &nextProposerHash)
	return err
}

func (m *Manager) ProduceApplyGossipBlock(ctx context.Context, allowEmpty bool) (block *types.Block, commit *types.Commit, err error) {
	return m.produceApplyGossip(ctx, allowEmpty, nil)
}

func (m *Manager) produceApplyGossip(ctx context.Context, allowEmpty bool, nextProposerHash *[32]byte) (block *types.Block, commit *types.Commit, err error) {
	block, commit, err = m.produceBlock(allowEmpty, nextProposerHash)
	if err != nil {
		return nil, nil, fmt.Errorf("produce block: %w", err)
	}

	if err := m.applyBlock(block, commit, types.BlockMetaData{Source: types.Produced}); err != nil {
		return nil, nil, fmt.Errorf("apply block: %w: %w", err, ErrNonRecoverable)
	}

	if err := m.gossipBlock(ctx, *block, *commit); err != nil {
		return nil, nil, fmt.Errorf("gossip block: %w", err)
	}

	return block, commit, nil
}

func (m *Manager) produceBlock(allowEmpty bool, nextProposerHash *[32]byte) (*types.Block, *types.Commit, error) {
	newHeight := m.State.NextHeight()
	lastHeaderHash, lastCommit, err := m.GetPreviousBlockHashes(newHeight)
	if err != nil {
		return nil, nil, fmt.Errorf("load prev block: %w", err)
	}

	var block *types.Block
	var commit *types.Commit
	// Check if there's an already stored block and commit at a newer height
	// If there is use that instead of creating a new block
	pendingBlock, err := m.Store.LoadBlock(newHeight)
	if err == nil {
		// Using an existing block
		block = pendingBlock
		commit, err = m.Store.LoadCommit(newHeight)
		if err != nil {
			return nil, nil, fmt.Errorf("load commit after load block: height: %d: %w: %w", newHeight, err, ErrNonRecoverable)
		}
		m.logger.Info("Using pending block.", "height", newHeight)
		return block, commit, nil
	} else if !errors.Is(err, gerrc.ErrNotFound) {
		return nil, nil, fmt.Errorf("load block: height: %d: %w: %w", newHeight, err, ErrNonRecoverable)
	}

	maxBlockDataSize := uint64(float64(m.Conf.BatchMaxSizeBytes) * types.MaxBlockSizeAdjustment)
	proposerHashForBlock := [32]byte(m.State.Sequencers.ProposerHash())
	// if nextProposerHash is set, we create a last block
	if nextProposerHash != nil {
		maxBlockDataSize = 0
		proposerHashForBlock = *nextProposerHash
	}
	block = m.Executor.CreateBlock(newHeight, lastCommit, lastHeaderHash, proposerHashForBlock, m.State, maxBlockDataSize)
	if !allowEmpty && len(block.Data.Txs) == 0 {
		return nil, nil, fmt.Errorf("%w: %w", types.ErrEmptyBlock, ErrRecoverable)
	}

	commit, err = m.createCommit(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create commit: %w: %w", err, ErrNonRecoverable)
	}

	m.logger.Info("Block created.", "height", newHeight, "num_tx", len(block.Data.Txs))
	types.RollappBlockSizeBytesGauge.Set(float64(len(block.Data.Txs)))
	types.RollappBlockSizeTxsGauge.Set(float64(len(block.Data.Txs)))
	return block, commit, nil
}

// create commit for block
func (m *Manager) createCommit(block *types.Block) (*types.Commit, error) {
	abciHeaderPb := types.ToABCIHeaderPB(&block.Header)
	abciHeaderBytes, err := abciHeaderPb.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal abci header: %w", err)
	}
	proposerAddress := block.Header.ProposerAddress
	sign, err := m.LocalKey.Sign(abciHeaderBytes)
	if err != nil {
		return nil, fmt.Errorf("sign abci header: %w", err)
	}
	voteTimestamp := tmtime.Now()
	tmSignature, err := m.createTMSignature(block, proposerAddress, voteTimestamp)
	if err != nil {
		return nil, fmt.Errorf("create tm signature: %w", err)
	}
	commit := &types.Commit{
		Height:     block.Header.Height,
		HeaderHash: block.Header.Hash(),
		Signatures: []types.Signature{sign},
		TMSignature: tmtypes.CommitSig{
			BlockIDFlag:      2,
			ValidatorAddress: proposerAddress,
			Timestamp:        voteTimestamp,
			Signature:        tmSignature,
		},
	}
	return commit, nil
}

func (m *Manager) createTMSignature(block *types.Block, proposerAddress []byte, voteTimestamp time.Time) ([]byte, error) {
	headerHash := block.Header.Hash()
	vote := tmtypes.Vote{
		Type:      cmtproto.PrecommitType,
		Height:    int64(block.Header.Height),
		Round:     0,
		Timestamp: voteTimestamp,
		BlockID: tmtypes.BlockID{Hash: headerHash[:], PartSetHeader: tmtypes.PartSetHeader{
			Total: 1,
			Hash:  headerHash[:],
		}},
		ValidatorAddress: proposerAddress,
		ValidatorIndex:   0,
	}
	v := vote.ToProto()
	// convert libp2p key to tm key
	// TODO: move to types
	rawKey, _ := m.LocalKey.Raw()
	tmprivkey := tmed25519.PrivKey(rawKey)
	tmprivkey.PubKey().Bytes()
	// Create a mock validator to sign the vote
	tmvalidator := tmtypes.NewMockPVWithParams(tmprivkey, false, false)
	err := tmvalidator.SignVote(m.State.ChainID, v)
	if err != nil {
		return nil, err
	}
	// Update the vote with the signature
	vote.Signature = v.Signature
	pubKey := tmprivkey.PubKey()
	voteSignBytes := tmtypes.VoteSignBytes(m.State.ChainID, v)
	if !pubKey.VerifySignature(voteSignBytes, vote.Signature) {
		return nil, fmt.Errorf("wrong signature")
	}
	return vote.Signature, nil
}

// GetPreviousBlockHashes returns the hash of the last block and the commit for the last block
// to be used as the previous block hash and commit for the next block
func (m *Manager) GetPreviousBlockHashes(forHeight uint64) (lastHeaderHash [32]byte, lastCommit *types.Commit, err error) {
	lastHeaderHash, lastCommit, err = loadPrevBlock(m.Store, forHeight-1) // prev height = forHeight - 1
	if err != nil {
		if !m.State.IsGenesis() { // allow prevBlock not to be found only on genesis
			return [32]byte{}, nil, fmt.Errorf("load prev block: %w: %w", err, ErrNonRecoverable)
		}
		lastHeaderHash = [32]byte{}
		lastCommit = &types.Commit{}
	}
	return lastHeaderHash, lastCommit, nil
}

func loadPrevBlock(store store.Store, height uint64) ([32]byte, *types.Commit, error) {
	lastCommit, err := store.LoadCommit(height)
	if err != nil {
		return [32]byte{}, nil, fmt.Errorf("load commit: height: %d: %w", height, err)
	}
	lastBlock, err := store.LoadBlock(height)
	if err != nil {
		return [32]byte{}, nil, fmt.Errorf("load block after load commit: height: %d: %w", height, err)
	}
	return lastBlock.Header.Hash(), lastCommit, nil
}
