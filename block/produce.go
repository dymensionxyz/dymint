package block

import (
	"context"
	"errors"
	"fmt"
	"time"

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

	ticker := time.NewTicker(m.Conf.BlockTime)
	defer ticker.Stop()

	// Allow the initial block to be empty
	produceEmptyBlock := true

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

	for {
		select {
		case <-ctx.Done(): // Context canceled
			return
		case <-m.produceEmptyBlockCh: // If we got a request for an empty block produce it and don't wait for the ticker
			produceEmptyBlock = true
		case <-emptyBlocksTimer: // Empty blocks timeout
			produceEmptyBlock = true
			m.logger.Debug(fmt.Sprintf("no transactions, producing empty block: elapsed: %.2f", m.Conf.EmptyBlocksMaxTime.Seconds()))
		// Produce block
		case <-ticker.C:
			err := m.ProduceAndGossipBlock(ctx, produceEmptyBlock)
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
		case shouldProduceBlocks := <-m.shouldProduceBlocksCh:
			for !shouldProduceBlocks {
				m.logger.Info("block production paused - awaiting positive continuation signal")
				shouldProduceBlocks = <-m.shouldProduceBlocksCh
			}
			m.logger.Info("resumed block production")
		}
	}
}

func (m *Manager) ProduceAndGossipBlock(ctx context.Context, allowEmpty bool) error {
	block, commit, err := m.produceBlock(allowEmpty)
	if err != nil {
		return fmt.Errorf("produce block: %w", err)
	}

	size := uint64(block.ToProto().Size() + commit.ToProto().Size())
	_ = m.updateAccumaltedSize(size)

	if err := m.gossipBlock(ctx, *block, *commit); err != nil {
		return fmt.Errorf("gossip block: %w", err)
	}

	return nil
}

func (m *Manager) updateAccumaltedSize(size uint64) bool {
	m.accumulatedProducedSize += size

	// Check if accumulated size is greater than the max size
	// TODO: allow some tolerance for block size (aim for BlockBatchMaxSize +- 10%)
	if m.accumulatedProducedSize > m.Conf.BlockBatchMaxSizeBytes {
		select {
		case m.shouldSubmitBatchCh <- true:
		default:
			m.logger.Debug("new batch accumualted, but channel is full, skipping submission signal")
		}
		m.accumulatedProducedSize = 0
		return true
	}
	return false
}

func (m *Manager) produceBlock(allowEmpty bool) (*types.Block, *types.Commit, error) {
	m.produceBlockMutex.Lock()
	defer m.produceBlockMutex.Unlock()
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

	m.logger.Info("block created", "height", newHeight, "num_tx", len(block.Data.Txs))
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
