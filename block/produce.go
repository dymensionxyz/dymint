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

func (m *Manager) ProduceBlockLoop(ctx context.Context, bytesProducedC chan int) error {
	ticker := time.NewTicker(m.Conf.BlockTime)
	defer func() {
		ticker.Stop()
	}()

	var nextEmptyBlock time.Time
	firstBlock := true

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:

			if !m.AmIProposerOnRollapp() {
				continue
			}

			produceEmptyBlock := firstBlock || m.Conf.MaxIdleTime == 0 || nextEmptyBlock.Before(time.Now())
			firstBlock = false

			block, commit, err := m.ProduceApplyGossipBlock(ctx, ProduceBlockOptions{AllowEmpty: produceEmptyBlock})
			if errors.Is(err, context.Canceled) {
				return nil
			}
			if errors.Is(err, types.ErrEmptyBlock) {
				continue
			}
			if errors.Is(err, ErrNonRecoverable) {
				uevent.MustPublish(ctx, m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
				return err
			}

			if err != nil {
				continue
			}
			nextEmptyBlock = time.Now().Add(m.Conf.MaxIdleTime)
			if 0 < len(block.Data.Txs) {
				nextEmptyBlock = time.Now().Add(m.Conf.MaxProofTime)
			} else {
			}

			bytesProducedN := block.SizeBytes() + commit.SizeBytes()

			select {
			case <-ctx.Done():
				return nil
			case bytesProducedC <- bytesProducedN:
			default:
				evt := &events.DataHealthStatus{Error: fmt.Errorf("Block production paused. Time between last block produced and last block submitted higher than max skew time: %s last block in settlement time: %s %w", m.Conf.MaxSkewTime, time.Unix(0, m.LastBlockTimeInSettlement.Load()), gerrc.ErrResourceExhausted)}
				uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
				select {
				case <-ctx.Done():
					return nil
				case bytesProducedC <- bytesProducedN:
					evt := &events.DataHealthStatus{Error: nil}
					uevent.MustPublish(ctx, m.Pubsub, evt, events.HealthStatusList)
				}
			}

		}
	}
}

type ProduceBlockOptions struct {
	AllowEmpty       bool
	MaxData          *uint64
	NextProposerHash *[32]byte
}

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
	newSequencerSet, err := m.SnapshotSequencerSet()
	if err != nil {
		return nil, nil, fmt.Errorf("snapshot sequencer set: %w", err)
	}

	opts.AllowEmpty = opts.AllowEmpty || len(newSequencerSet) > 0

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

func (m *Manager) SnapshotSequencerSet() (sequencersAfterUpdate types.Sequencers, err error) {
	sequencersAfterUpdate = m.Sequencers.GetAll()

	lastSequencers, err := m.Store.LoadLastBlockSequencerSet()

	if err != nil && !errors.Is(err, gerrc.ErrNotFound) {
		return nil, fmt.Errorf("load last block sequencer set: %w: %w", err, ErrNonRecoverable)
	}

	newSequencers := types.SequencerListRightOuterJoin(lastSequencers, sequencersAfterUpdate)

	if len(newSequencers) == 0 {
		return nil, nil
	}

	msgs, err := ConsensusMsgsOnSequencerSetUpdate(newSequencers)
	if err != nil {
		return nil, fmt.Errorf("consensus msgs on sequencers set update: %w: %w", err, ErrNonRecoverable)
	}
	m.Executor.AddConsensusMsgs(msgs...)

	return sequencersAfterUpdate, nil
}

func (m *Manager) produceBlock(opts ProduceBlockOptions) (*types.Block, *types.Commit, error) {
	newHeight := m.State.NextHeight()
	lastHeaderHash, lastCommit, err := m.GetPreviousBlockHashes(newHeight)
	if err != nil {
		return nil, nil, fmt.Errorf("load prev block: %w", err)
	}

	var block *types.Block
	var commit *types.Commit

	pendingBlock, err := m.Store.LoadBlock(newHeight)
	if err == nil {

		block = pendingBlock
		commit, err = m.Store.LoadCommit(newHeight)
		if err != nil {
			return nil, nil, fmt.Errorf("load commit after load block: height: %d: %w: %w", newHeight, err, ErrNonRecoverable)
		}
		return block, commit, nil
	} else if !errors.Is(err, gerrc.ErrNotFound) {
		return nil, nil, fmt.Errorf("load block: height: %d: %w: %w", newHeight, err, ErrNonRecoverable)
	}

	maxBlockDataSize := uint64(float64(m.Conf.BatchSubmitBytes) * types.MaxBlockSizeAdjustment)
	if opts.MaxData != nil {
		maxBlockDataSize = *opts.MaxData
	}
	proposerHashForBlock := [32]byte(m.State.GetProposerHash())

	if opts.NextProposerHash != nil {
		maxBlockDataSize = 0
		proposerHashForBlock = *opts.NextProposerHash
	}

	block = m.Executor.CreateBlock(newHeight, lastCommit, lastHeaderHash, proposerHashForBlock, m.State, maxBlockDataSize)

	if !opts.AllowEmpty && len(block.Data.Txs) == 0 {
		return nil, nil, fmt.Errorf("%w: %w", types.ErrEmptyBlock, ErrRecoverable)
	}

	commit, err = m.createCommit(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create commit: %w: %w", err, ErrNonRecoverable)
	}

	types.RollappBlockSizeBytesGauge.Set(float64(len(block.Data.Txs)))
	types.RollappBlockSizeTxsGauge.Set(float64(len(block.Data.Txs)))
	return block, commit, nil
}

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

	rawKey, _ := m.LocalKey.Raw()
	tmprivkey := tmed25519.PrivKey(rawKey)
	tmprivkey.PubKey().Bytes()

	tmvalidator := tmtypes.NewMockPVWithParams(tmprivkey, false, false)
	err := tmvalidator.SignVote(m.State.ChainID, v)
	if err != nil {
		return nil, err
	}

	vote.Signature = v.Signature
	pubKey := tmprivkey.PubKey()
	voteSignBytes := tmtypes.VoteSignBytes(m.State.ChainID, v)
	if !pubKey.VerifySignature(voteSignBytes, vote.Signature) {
		return nil, fmt.Errorf("wrong signature")
	}
	return vote.Signature, nil
}

func (m *Manager) GetPreviousBlockHashes(forHeight uint64) (lastHeaderHash [32]byte, lastCommit *types.Commit, err error) {
	lastHeaderHash, lastCommit, err = getHeaderHashAndCommit(m.Store, forHeight-1)
	if err != nil {
		if !m.State.IsGenesis() {
			return [32]byte{}, nil, fmt.Errorf("load prev block: %w: %w", err, ErrNonRecoverable)
		}
		lastHeaderHash = [32]byte{}
		lastCommit = &types.Commit{}
	}
	return lastHeaderHash, lastCommit, nil
}

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
