package block

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/node/events"
	uevent "github.com/dymensionxyz/dymint/utils/event"

	"github.com/dymensionxyz/dymint/store"

	"github.com/dymensionxyz/dymint/types"
	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
	cmtproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
)

// ProduceBlockLoop is calling publishBlock in a loop as long as we're synced.
func (m *Manager) ProduceBlockLoop(ctx context.Context) {
	m.logger.Debug("Started produce loop")
	produceEmptyBlock := true // Allow the initial block to be empty

	// Main ticker for block production
	ticker := time.NewTicker(m.Conf.BlockTime)
	defer ticker.Stop()

	// Timer for empty blockss
	var emptyBlocksTimer <-chan time.Time
	resetEmptyBlocksTimer := func() {}
	// Setup ticker for empty blocks if enabled
	if 0 < m.Conf.EmptyBlocksMaxTime {
		t := time.NewTimer(m.Conf.EmptyBlocksMaxTime)
		emptyBlocksTimer = t.C
		resetEmptyBlocksTimer = func() {
			produceEmptyBlock = false
			t.Reset(m.Conf.EmptyBlocksMaxTime)
		}
		defer t.Stop()
	}

	// Timer for block progression to support IBC transfers
	forceCreationTimer := time.NewTimer(5 * time.Second) //TODO: change to own constant
	defer forceCreationTimer.Stop()
	forceCreationTimer.Stop() // Don't start it initially
	resetForceCreationTimer := func(lastBlockEmpty bool) {
		if lastBlockEmpty {
			forceCreationTimer.Stop()
		} else {
			forceCreationTimer.Reset(5 * time.Second)
		}
	}

	for {
		select {
		case <-ctx.Done(): // Context canceled
			return
		case <-forceCreationTimer.C: // Force block creation
			produceEmptyBlock = true
		case <-emptyBlocksTimer: // Empty blocks timeout
			produceEmptyBlock = true
			m.logger.Debug(fmt.Sprintf("no transactions, producing empty block: elapsed: %.2f", m.Conf.EmptyBlocksMaxTime.Seconds()))
		// Produce block
		case <-ticker.C:
			block, _, err := m.ProduceAndGossipBlock(ctx, produceEmptyBlock)
			if errors.Is(err, context.Canceled) {
				m.logger.Error("produce and gossip: context canceled", "error", err)
				return
			}
			if errors.Is(err, types.ErrSkippedEmptyBlock) {
				continue
			}
			if errors.Is(err, ErrNonRecoverable) {
				m.logger.Error("produce and gossip: non-recoverable", "error", err) // TODO: flush? or don't log at all?
				panic(fmt.Errorf("produce and gossip block: %w", err))
			}
			if err != nil {
				m.logger.Error("produce and gossip: uncategorized, assuming recoverable", "error", err)
				continue
			}
			resetEmptyBlocksTimer()
			isLastBlockEmpty := len(block.Data.Txs) == 0
			resetForceCreationTimer(isLastBlockEmpty)

			// Check if we should submit the accumulated data
			if m.shouldSubmitBatch() {
				select {
				case m.ShouldSubmitBatchCh <- true:
					m.logger.Info("new batch accumulated, signal sent to submit the batch")
				default:
					m.logger.Error("new batch accumulated, but channel is full, stopping block production until the signal is consumed")
					// emit unhealthy event for the node
					evt := &events.DataHealthStatus{Error: fmt.Errorf("submission channel is full")}
					uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
					// wait for the signal to be consumed
					m.ShouldSubmitBatchCh <- true
					m.logger.Info("resumed block production")
					// emit healthy event for the node
					evt = &events.DataHealthStatus{Error: nil}
					uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
				}
				m.AccumulatedProducedSize.Store(0)
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

func (m *Manager) updateAccumulatedSize(size uint64) {
	curr := m.AccumulatedProducedSize.Load()
	_ = m.AccumulatedProducedSize.CompareAndSwap(curr, curr+size)
}

// check if we should submit the accumulated data
func (m *Manager) shouldSubmitBatch() bool {
	// Check if accumulated size is greater than the max size
	// TODO: allow some tolerance for block size (aim for BlockBatchMaxSize +- 10%)
	return m.AccumulatedProducedSize.Load() > m.Conf.BlockBatchMaxSizeBytes
}

func (m *Manager) produceBlock(allowEmpty bool) (*types.Block, *types.Commit, error) {
	var (
		lastCommit     *types.Commit
		lastHeaderHash [32]byte
		newHeight      uint64
		err            error
	)

	if m.LastState.IsGenesis() {
		newHeight = uint64(m.LastState.InitialHeight)
		lastCommit = &types.Commit{}
		m.LastState.BaseHeight = newHeight
		if ok := m.Store.SetBase(newHeight); !ok {
			return nil, nil, fmt.Errorf("store set base: %d", newHeight)
		}
	} else {
		height := m.Store.Height()
		newHeight = height + 1
		lastCommit, err = m.Store.LoadCommit(height)
		if err != nil {
			return nil, nil, fmt.Errorf("load commit: height: %d: %w: %w", height, err, ErrNonRecoverable)
		}
		lastBlock, err := m.Store.LoadBlock(height)
		if err != nil {
			return nil, nil, fmt.Errorf("load block after load commit: height: %d: %w: %w", height, err, ErrNonRecoverable)
		}
		lastHeaderHash = lastBlock.Header.Hash()
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
		m.logger.Info("using pending block", "height", newHeight)
	} else if !errors.Is(err, store.ErrKeyNotFound) {
		return nil, nil, fmt.Errorf("load block: height: %d: %w: %w", newHeight, err, ErrNonRecoverable)
	} else {
		block = m.Executor.CreateBlock(newHeight, lastCommit, lastHeaderHash, m.LastState, m.Conf.BlockBatchMaxSizeBytes)
		if !allowEmpty && len(block.Data.Txs) == 0 {
			return nil, nil, fmt.Errorf("%w: %w", types.ErrSkippedEmptyBlock, ErrRecoverable)
		}

		abciHeaderPb := types.ToABCIHeaderPB(&block.Header)
		abciHeaderBytes, err := abciHeaderPb.Marshal()
		if err != nil {
			return nil, nil, fmt.Errorf("marshal abci header: %w: %w", err, ErrNonRecoverable)
		}
		proposerAddress := block.Header.ProposerAddress
		sign, err := m.ProposerKey.Sign(abciHeaderBytes)
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

	size := uint64(block.ToProto().Size() + commit.ToProto().Size())
	m.updateAccumulatedSize(size)

	m.logger.Info("block created", "height", newHeight, "num_tx", len(block.Data.Txs), "accumulated_size", m.AccumulatedProducedSize.Load())
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
	raw_key, _ := m.ProposerKey.Raw()
	tmprivkey := tmed25519.PrivKey(raw_key)
	tmprivkey.PubKey().Bytes()
	// Create a mock validator to sign the vote
	tmvalidator := tmtypes.NewMockPVWithParams(tmprivkey, false, false)
	err := tmvalidator.SignVote(m.LastState.ChainID, v)
	if err != nil {
		return nil, err
	}
	// Update the vote with the signature
	vote.Signature = v.Signature
	pubKey := tmprivkey.PubKey()
	voteSignBytes := tmtypes.VoteSignBytes(m.LastState.ChainID, v)
	if !pubKey.VerifySignature(voteSignBytes, vote.Signature) {
		return nil, fmt.Errorf("wrong signature")
	}
	return vote.Signature, nil
}
