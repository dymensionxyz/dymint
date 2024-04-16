package block

import (
	"context"
	"errors"
	"fmt"
	"time"

	abciconv "github.com/dymensionxyz/dymint/conv/abci"
	"github.com/dymensionxyz/dymint/types"
	tmed25519 "github.com/tendermint/tendermint/crypto/ed25519"
	cmtproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
)

// ProduceBlockLoop is calling publishBlock in a loop as long as we're synced.
func (m *Manager) ProduceBlockLoop(ctx context.Context) {
	m.logger.Debug("Started produce loop")

	ticker := time.NewTicker(m.conf.BlockTime)
	defer ticker.Stop()

	// Allow the initial block to be empty
	produceEmptyBlock := true

	var emptyBlocksTimer <-chan time.Time
	resetEmptyBlocksTimer := func() {}
	// Setup ticker for empty blocks if enabled
	if 0 < m.conf.EmptyBlocksMaxTime {
		t := time.NewTicker(m.conf.EmptyBlocksMaxTime)
		emptyBlocksTimer = t.C
		resetEmptyBlocksTimer = func() {
			produceEmptyBlock = true
			t.Reset(m.conf.EmptyBlocksMaxTime)
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
			m.logger.Debug(fmt.Sprintf("no transactions, producing empty block: elapsed: %.2f", m.conf.EmptyBlocksMaxTime.Seconds()))
		// Produce block
		case <-ticker.C:
			err := m.produceAndGossipBlock(ctx, produceEmptyBlock)
			if errors.Is(err, ErrRecoverable) {
				m.logger.Info("produce and gossip: recoverable", "error", err)
				continue
			}
			if errors.Is(err, ErrNonRecoverable) {
				m.logger.Error("produce and gossip: non-recoverable", "error", err) // TODO: flush? or don't log at all?
				panic(fmt.Errorf("produce and gossip block: %w", err))
			}
			if err != nil {
				m.logger.Error("produce and gossip: uncategorized", "error", err)
				continue
			}
			resetEmptyBlocksTimer()
		case shouldProduceBlocks := <-m.shouldProduceBlocksCh:
			for !shouldProduceBlocks {
				m.logger.Info("block production paused - awaiting positive continuation signal")
				shouldProduceBlocks = <-m.shouldProduceBlocksCh
			}
			m.logger.Info("resumed block resumed")
		}
	}
}

func (m *Manager) produceAndGossipBlock(ctx context.Context, allowEmpty bool) error {
	block, commit, err := m.produceBlock(ctx, allowEmpty)
	if err != nil {
		return fmt.Errorf("produce block: %w", err)
	}

	if err := m.gossipBlock(ctx, *block, *commit); err != nil {
		return fmt.Errorf("gossip block: %w: %w", err, ErrNonRecoverable) // from inspecting the gossip impl, it's clear there is no point retrying
	}

	return nil
}

func (m *Manager) produceBlock(ctx context.Context, allowEmpty bool) (*types.Block, *types.Commit, error) {
	m.produceBlockMutex.Lock()
	defer m.produceBlockMutex.Unlock()
	var (
		lastCommit     *types.Commit
		lastHeaderHash [32]byte
		newHeight      uint64
		err            error
	)

	if m.lastState.IsGenesis() {
		newHeight = uint64(m.lastState.InitialHeight)
		lastCommit = &types.Commit{}
		m.lastState.BaseHeight = newHeight
		m.store.SetBase(newHeight)
	} else {
		height := m.store.Height()
		newHeight = height + 1
		lastCommit, err = m.store.LoadCommit(height)
		if err != nil {
			return nil, nil, fmt.Errorf("load commit: height: %d: %w: %w", height, err, ErrNonRecoverable)
		}
		lastBlock, err := m.store.LoadBlock(height)
		if err != nil {
			return nil, nil, fmt.Errorf("load block after load commit: height: %d: %w: %w", height, err, ErrNonRecoverable)
		}
		lastHeaderHash = lastBlock.Header.Hash()
	}

	var block *types.Block
	var commit *types.Commit
	// Check if there's an already stored block and commit at a newer height
	// If there is use that instead of creating a new block
	pendingBlock, err := m.store.LoadBlock(newHeight)
	if err == nil {
		block = pendingBlock
		commit, err = m.store.LoadCommit(newHeight)
		if err != nil {
			return nil, nil, fmt.Errorf("load commit after load block: height: %d: %w: %w", newHeight, err, ErrNonRecoverable)
		}
		m.logger.Info("using pending block", "height", newHeight)
	} else {
		block = m.executor.CreateBlock(newHeight, lastCommit, lastHeaderHash, m.lastState)
		if !allowEmpty && len(block.Data.Txs) == 0 {
			return nil, nil, fmt.Errorf("%w: %w", types.ErrSkippedEmptyBlock, ErrRecoverable)
		}
		abciHeaderPb := abciconv.ToABCIHeaderPB(&block.Header)
		abciHeaderBytes, err := abciHeaderPb.Marshal()
		if err != nil {
			return nil, nil, fmt.Errorf("marshal abci header: %w: %w", err, ErrNonRecoverable)
		}
		proposerAddress := block.Header.ProposerAddress
		sign, err := m.proposerKey.Sign(abciHeaderBytes)
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

	if err := m.applyBlock(ctx, block, commit, blockMetaData{source: producedBlock}); err != nil {
		// TODO: ask michael what to do about invalid height, can it happen in the happy path?
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
	raw_key, _ := m.proposerKey.Raw()
	tmprivkey := tmed25519.PrivKey(raw_key)
	tmprivkey.PubKey().Bytes()
	// Create a mock validator to sign the vote
	tmvalidator := tmtypes.NewMockPVWithParams(tmprivkey, false, false)
	err := tmvalidator.SignVote(m.lastState.ChainID, v)
	if err != nil {
		return nil, err
	}
	// Update the vote with the signature
	vote.Signature = v.Signature
	pubKey := tmprivkey.PubKey()
	voteSignBytes := tmtypes.VoteSignBytes(m.lastState.ChainID, v)
	if !pubKey.VerifySignature(voteSignBytes, vote.Signature) {
		return nil, fmt.Errorf("wrong signature")
	}
	return vote.Signature, nil
}
