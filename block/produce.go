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
func (m *Manager) ProduceBlockLoop(ctx context.Context) (err error) {
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
			return
		case <-ticker.C:
			// if empty blocks are configured to be enabled, and one is scheduled...
			produceEmptyBlock := firstBlock || m.Conf.MaxIdleTime == 0 || nextEmptyBlock.Before(time.Now())
			firstBlock = false

			var (
				block  *types.Block
				commit *types.Commit
			)
			block, commit, err = m.ProduceAndGossipBlock(ctx, produceEmptyBlock)
			if errors.Is(err, context.Canceled) {
				m.logger.Error("Produce and gossip: context canceled.", "error", err)
				return
			}
			if errors.Is(err, types.ErrSkippedEmptyBlock) {
				continue
			}
			if errors.Is(err, ErrNonRecoverable) {
				m.logger.Error("Produce and gossip: non-recoverable.", "error", err) // TODO: flush? or don't log at all?
				uevent.MustPublish(ctx, m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
				return
			}
			if err != nil {
				m.logger.Error("Produce and gossip: uncategorized, assuming recoverable.", "error", err)
				continue
			}

			// If IBC transactions are present, set proof required to true
			// This will set a shorter timer for the next block
			// currently we set it for all txs as we don't have a way to determine if an IBC tx is present (https://github.com/dymensionxyz/dymint/issues/709)
			nextEmptyBlock = time.Now().Add(m.Conf.MaxIdleTime)
			if 0 < len(block.Data.Txs) {
				nextEmptyBlock = time.Now().Add(m.Conf.MaxProofTime)
			} else {
				m.logger.Info("Produced empty block.")
			}

			// Send the size to the accumulated size channel
			// This will block in case the submitter is too slow and it's buffer is full
			size := uint64(block.ToProto().Size()) + uint64(commit.ToProto().Size())
			select {
			case <-ctx.Done():
				return
			case m.producedSizeCh <- size:
			}
		}
	}
}

func (m *Manager) ProduceAndGossipBlock(ctx context.Context, allowEmpty bool) (*types.Block, *types.Commit, error) {
	block, commit, err := m.produceBlock(allowEmpty)
	if err != nil {
		return nil, nil, fmt.Errorf("produce block: %w", err)
	}

	if err := m.gossipBlock(ctx, *block, *commit); err != nil {
		return nil, nil, fmt.Errorf("gossip block: %w", err)
	}

	return block, commit, nil
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

func (m *Manager) produceBlock(allowEmpty bool) (*types.Block, *types.Commit, error) {
	newHeight := m.State.NextHeight()
	lastHeaderHash, lastCommit, err := loadPrevBlock(m.Store, newHeight-1)
	if err != nil {
		if !m.State.IsGenesis() { // allow prevBlock not to be found only on genesis
			return nil, nil, fmt.Errorf("load prev block: %w: %w", err, ErrNonRecoverable)
		}
		lastHeaderHash = [32]byte{}
		lastCommit = &types.Commit{}
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
	} else if !errors.Is(err, gerrc.ErrNotFound) {
		return nil, nil, fmt.Errorf("load block: height: %d: %w: %w", newHeight, err, ErrNonRecoverable)
	} else {
		// limit to the max block data, so we don't create a block that is too big to fit in a batch
		maxBlockDataSize := uint64(float64(m.Conf.BlockBatchMaxSizeBytes) * types.MaxBlockSizeAdjustment)
		block = m.Executor.CreateBlock(newHeight, lastCommit, lastHeaderHash, m.State, maxBlockDataSize)
		if !allowEmpty && len(block.Data.Txs) == 0 {
			return nil, nil, fmt.Errorf("%w: %w", types.ErrSkippedEmptyBlock, ErrRecoverable)
		}

		abciHeaderPb := types.ToABCIHeaderPB(&block.Header)
		abciHeaderBytes, err := abciHeaderPb.Marshal()
		if err != nil {
			return nil, nil, fmt.Errorf("marshal abci header: %w: %w", err, ErrNonRecoverable)
		}
		proposerAddress := block.Header.ProposerAddress
		sign, err := m.LocalKey.Sign(abciHeaderBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("sign abci header: %w: %w", err, ErrNonRecoverable)
		}
		voteTimestamp := tmtime.Now()
		tmSignature, err := m.createTMSignature(block, proposerAddress, voteTimestamp)
		if err != nil {
			return nil, nil, fmt.Errorf("create tm signature: %w: %w", err, ErrNonRecoverable)
		}
		commit = &types.Commit{
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
	}

	if err := m.applyBlock(block, commit, blockMetaData{source: producedBlock}); err != nil {
		return nil, nil, fmt.Errorf("apply block: %w: %w", err, ErrNonRecoverable)
	}

	m.logger.Info("Block created.", "height", newHeight, "num_tx", len(block.Data.Txs))
	types.RollappBlockSizeBytesGauge.Set(float64(len(block.Data.Txs)))
	types.RollappBlockSizeTxsGauge.Set(float64(len(block.Data.Txs)))
	types.RollappHeightGauge.Set(float64(newHeight))
	return block, commit, nil
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
	raw_key, _ := m.LocalKey.Raw()
	tmprivkey := tmed25519.PrivKey(raw_key)
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
