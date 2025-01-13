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
			// Only produce if I'm the current rollapp proposer.
			if !m.AmIProposerOnRollapp() {
				continue
			}

			// if empty blocks are configured to be enabled, and one is scheduled...
			produceEmptyBlock := firstBlock || m.Conf.MaxIdleTime == 0 || nextEmptyBlock.Before(time.Now())
			firstBlock = false

			block, commit, err := m.ProduceApplyGossipBlock(ctx, ProduceBlockOptions{AllowEmpty: produceEmptyBlock})
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

			select {
			case <-ctx.Done():
				return nil
			case bytesProducedC <- bytesProducedN:
			default:
				evt := &events.DataHealthStatus{Error: fmt.Errorf("Block production paused. Time between last block produced and last block submitted higher than max skew time: %s last block in settlement time: %s %w", m.Conf.MaxSkewTime, m.GetLastBlockTimeInSettlement(), gerrc.ErrResourceExhausted)}
				uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
				m.logger.Error("Pausing block production until new batch is submitted.", "Batch skew time", m.GetBatchSkewTime(), "Max batch skew time", m.Conf.MaxSkewTime, "Last block in settlement time", m.GetLastBlockTimeInSettlement())
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

type ProduceBlockOptions struct {
	AllowEmpty       bool
	MaxData          *uint64
	NextProposerHash *[32]byte // optional, used for last block
}

// ProduceApplyGossipLastBlock produces and applies a block with the given NextProposerHash.
func (m *Manager) ProduceApplyGossipLastBlock(ctx context.Context, nextProposerHash [32]byte) (err error) {
	_, _, err = m.produceApplyGossip(ctx, ProduceBlockOptions{
		AllowEmpty:       true,
		NextProposerHash: &nextProposerHash,
	})
	return err
}

func (m *Manager) ProduceApplyGossipBlock(ctx context.Context, opts ProduceBlockOptions) (block *types.Block, commit *types.Commit, err error) {
	return m.produceApplyGossip(ctx, opts)
}

func (m *Manager) produceApplyGossip(ctx context.Context, opts ProduceBlockOptions) (block *types.Block, commit *types.Commit, err error) {
	// Snapshot sequencer set to check if there are sequencer set updates.
	// It fills the consensus messages queue for all the new sequencers.
	//
	// Note that there cannot be any recoverable errors between when the queue is filled and dequeued;
	// otherwise, the queue may grow uncontrollably if there is a recoverable error loop in the middle.
	//
	// All errors in this method are non-recoverable.
	newSequencerSet, err := m.SnapshotSequencerSet()
	if err != nil {
		return nil, nil, fmt.Errorf("snapshot sequencer set: %w", err)
	}
	// We do not want to wait for a new block created to propagate a new sequencer set.
	// Therefore, we force an empty block if there are any sequencer set updates.
	opts.AllowEmpty = opts.AllowEmpty || len(newSequencerSet) > 0

	// If I'm not the current rollapp proposer, I should not produce a blocks.
	block, commit, err = m.produceBlock(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("produce block: %w", err)
	}

	if err := m.applyBlock(block, commit, types.BlockMetaData{Source: types.Produced, SequencerSet: newSequencerSet}); err != nil {
		return nil, nil, fmt.Errorf("apply block: %w: %w", err, ErrNonRecoverable)
	}

	if err := m.gossipBlock(ctx, *block, *commit); err != nil {
		return nil, nil, fmt.Errorf("gossip block: %w", err)
	}

	return block, commit, nil
}

// SnapshotSequencerSet loads two versions of the sequencer set:
//   - the one that was used for the last block (from the store)
//   - and the most recent one (from the manager memory)
//
// It then calculates the diff between the two and creates consensus messages for the new sequencers,
// i.e., only for the diff between two sets. If there is any diff (i.e., the sequencer set is updated),
// the method returns the entire new set. The new set will be used for next block and will be stored
// in the state instead of the old set after the block production.
//
// The set from the state is dumped to memory on reboots. It helps to avoid sending unnecessary
// UspertSequencer consensus messages on reboots. This is not a 100% solution, because the sequencer set
// is not persisted in the store in full node mode. It's only used in the proposer mode. Therefore,
// on rotation from the full node to the proposer, the sequencer set is duplicated as consensus msgs.
// Though single-time duplication it's not a big deal.
func (m *Manager) SnapshotSequencerSet() (sequencersAfterUpdate types.Sequencers, err error) {
	// the most recent sequencer set
	sequencersAfterUpdate = m.Sequencers.GetAll()

	// the sequencer set that was used for the last block
	lastSequencers, err := m.Store.LoadLastBlockSequencerSet()
	// it's okay if the last sequencer set is not found, it can happen on genesis or after
	// rotation from the full node to the proposer
	if err != nil && !errors.Is(err, gerrc.ErrNotFound) {
		// unexpected error from the store is non-recoverable
		return nil, fmt.Errorf("load last block sequencer set: %w: %w", err, ErrNonRecoverable)
	}

	// diff between the two sequencer sets
	newSequencers := types.SequencerListRightOuterJoin(lastSequencers, sequencersAfterUpdate)

	if len(newSequencers) == 0 {
		// nothing to upsert, nothing to persist
		return nil, nil
	}

	// Create consensus msgs for new sequencers.
	// It can fail only on decoding or internal errors this is non-recoverable.
	msgs, err := ConsensusMsgsOnSequencerSetUpdate(newSequencers)
	if err != nil {
		return nil, fmt.Errorf("consensus msgs on sequencers set update: %w: %w", err, ErrNonRecoverable)
	}
	m.Executor.AddConsensusMsgs(msgs...)

	// return the entire new set if there is any update
	return sequencersAfterUpdate, nil
}

func (m *Manager) produceBlock(opts ProduceBlockOptions) (*types.Block, *types.Commit, error) {
	newHeight := m.State.NextHeight()
	lastHeaderHash, lastCommit, err := m.GetPreviousBlockHashes(newHeight)
	if err != nil {
		// the error here is always non-recoverable, see GetPreviousBlockHashes() for details
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

	maxBlockDataSize := uint64(float64(m.Conf.BatchSubmitBytes) * types.MaxBlockSizeAdjustment)
	if opts.MaxData != nil {
		maxBlockDataSize = *opts.MaxData
	}
	proposerHashForBlock := [32]byte(m.State.GetProposerHash())
	// if NextProposerHash is set, we create a last block
	if opts.NextProposerHash != nil {
		maxBlockDataSize = 0
		proposerHashForBlock = *opts.NextProposerHash
	}

	// dequeue consensus messages for the new sequencers while creating a new block
	block = m.Executor.CreateBlock(newHeight, lastCommit, lastHeaderHash, proposerHashForBlock, m.State, maxBlockDataSize)
	// this cannot happen if there are any sequencer set updates
	// AllowEmpty should be always true in this case
	if !opts.AllowEmpty && len(block.Data.Txs) == 0 {
		return nil, nil, fmt.Errorf("%w: %w", types.ErrEmptyBlock, ErrRecoverable)
	}

	commit, err = m.createCommit(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create commit: %w: %w", err, ErrNonRecoverable)
	}

	m.logger.Info("Block created.", "height", newHeight, "num_tx", len(block.Data.Txs), "size", block.SizeBytes()+commit.SizeBytes())
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
		Height:    int64(block.Header.Height), //nolint:gosec // height is non-negative and falls in int64
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
	lastHeaderHash, lastCommit, err = getHeaderHashAndCommit(m.Store, forHeight-1) // prev height = forHeight - 1
	if err != nil {
		if !m.State.IsGenesis() { // allow prevBlock not to be found only on genesis
			return [32]byte{}, nil, fmt.Errorf("load prev block: %w: %w", err, ErrNonRecoverable)
		}
		lastHeaderHash = [32]byte{}
		lastCommit = &types.Commit{}
	}
	return lastHeaderHash, lastCommit, nil
}

// getHeaderHashAndCommit returns the Header Hash and Commit for a given height
func getHeaderHashAndCommit(store store.Store, height uint64) ([32]byte, *types.Commit, error) {
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
