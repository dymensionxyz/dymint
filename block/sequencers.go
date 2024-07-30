package block

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/google/uuid"
	tmtypes "github.com/tendermint/tendermint/types"
)

func (m *Manager) MonitorSequencerRotation(ctx context.Context) string {
	sequencerRotationEventClient := fmt.Sprintf("%s-%s", "sequencer_rotation", uuid.New().String())
	subscription, err := m.Pubsub.Subscribe(ctx, sequencerRotationEventClient, settlement.EventQueryRotationStarted)
	if err != nil {
		panic("Error subscribing to events")
	}
	defer m.Pubsub.UnsubscribeAll(ctx, sequencerRotationEventClient)

	for {
		select {
		case <-ctx.Done():
			return ""
		case event := <-subscription.Out():
			eventData, _ := event.Data().(*settlement.EventDataRotationStarted)
			nextSeqAddr := eventData.NextSeqAddr
			m.logger.Info("Sequencer rotation started.", "next_seq", nextSeqAddr)
			return nextSeqAddr
		}
	}
}

// In case of sequencer rotation, there's a phase where proposer rotated on L2 but hasn't yet rotated on hub.
// for this case, the old proposer counted as "sequencer" as well, so he'll be able to submit blocks.
func (m *Manager) IsSequencer() bool {
	localProposerKey, _ := m.LocalKey.GetPublic().Raw()
	l2Proposer := m.GetProposerPubKey().Bytes()
	expectedHubProposer := m.SLClient.GetProposer().PublicKey.Bytes()
	return bytes.Equal(l2Proposer, localProposerKey) || bytes.Equal(expectedHubProposer, localProposerKey)
}

// check rotation in progress (I'm the proposer, but needs to complete rotation)
func (m *Manager) MissingLastBatch() bool {
	localProposerKey, _ := m.LocalKey.GetPublic().Raw()
	expectedHubProposer := m.SLClient.GetProposer().PublicKey.Bytes()
	next, err := m.SLClient.GetNextProposer()
	if err != nil {
		panic(fmt.Errorf("get next proposer: %w", err))
	}
	return next != nil && bytes.Equal(expectedHubProposer, localProposerKey)
}

func (m *Manager) GetPreviousBlockHashes(forHeight uint64) (lastHeaderHash [32]byte, lastCommit *types.Commit, err error) {
	lastHeaderHash, lastCommit, err = loadPrevBlock(m.Store, forHeight-1)
	if err != nil {
		if !m.State.IsGenesis() { // allow prevBlock not to be found only on genesis
			return [32]byte{}, nil, fmt.Errorf("load prev block: %w: %w", err, ErrNonRecoverable)
		}
		lastHeaderHash = [32]byte{}
		lastCommit = &types.Commit{}
	}
	return lastHeaderHash, lastCommit, nil
}

func (m *Manager) produceLastBlock(nextProposerHash [32]byte) (*types.Block, *types.Commit, error) {
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
	} else if !errors.Is(err, gerrc.ErrNotFound) {
		return nil, nil, fmt.Errorf("load block: height: %d: %w: %w", newHeight, err, ErrNonRecoverable)
	} else {
		// Create a new block
		block, commit, err = m.createLastBlock(newHeight, lastCommit, lastHeaderHash, nextProposerHash)
		if err != nil {
			return nil, nil, fmt.Errorf("create new block: %w", err)
		}
	}

	m.logger.Info("Last block created.", "height", newHeight, "next_seq_hash", nextProposerHash)
	types.RollappBlockSizeBytesGauge.Set(float64(len(block.Data.Txs)))
	types.RollappBlockSizeTxsGauge.Set(float64(len(block.Data.Txs)))
	return block, commit, nil
}

// complete rotation
func (m *Manager) CompleteRotation(ctx context.Context, nextSeqAddr string) error {
	// validate nextSeq is in the bonded set
	var nextSeqHash [32]byte
	if nextSeqAddr != "" {
		val := m.State.ActiveSequencer.GetByAddress([]byte(nextSeqAddr))
		if val == nil {
			return fmt.Errorf("next sequencer not found in bonded set")
		}
		copy(nextSeqHash[:], val.PubKey.Address().Bytes())
	}

	err := m.CreateAndPostLastBatch(ctx, nextSeqHash)
	if err != nil {
		return fmt.Errorf("create and post last batch: %w", err)
	}

	m.logger.Info("Sequencer rotation completed. sequencer is no longer the proposer", "nextSeqAddr", nextSeqAddr)
	return nil
}

func (m *Manager) CreateAndPostLastBatch(ctx context.Context, nextSeqHash [32]byte) error {
	//check if the last block is already produced
	// if so, the active validator set is already updated
	if bytes.Equal(m.State.ActiveSequencer.ProposerHash, nextSeqHash[:]) {
		m.logger.Debug("Last block already produced. Skipping.")
	} else {
		_, _, err := m.ProduceApplyGossipLastBlock(ctx, nextSeqHash)
		if err != nil {
			return fmt.Errorf("produce apply gossip last block: %w", err)
		}
	}

	// Submit all data accumulated thus far
	for {
		b, err := m.CreateBatch(m.Conf.BatchMaxSizeBytes, m.NextHeightToSubmit(), m.State.Height())
		if err != nil {
			return fmt.Errorf("create batch: %w", err)
		}

		//FIXME: syb
		if err := m.SubmitBatch(b); err != nil {
			return fmt.Errorf("submit batch: %w", err)
		}

		if m.State.Height() == b.EndHeight() {
			break
		}
	}

	return nil
}

// add bonded sequencers to the seqSet without changing the proposer
func (m *Manager) UpdateBondedSequencerSetFromSL() error {
	seqs, err := m.SLClient.GetSequencers()
	if err != nil {
		return err
	}
	newVals := make([]*tmtypes.Validator, 0, len(seqs))
	for _, seq := range seqs {
		tmPubKey, err := cryptocodec.ToTmPubKeyInterface(seq.PublicKey)
		if err != nil {
			return err
		}
		val := tmtypes.NewValidator(tmPubKey, 1)

		// check if not exists already
		if m.State.ActiveSequencer.HasAddress(val.Address) {
			continue
		}

		newVals = append(newVals, val)
	}
	// update state on changes
	if len(newVals) > 0 {
		newVals = append(newVals, m.State.ActiveSequencer.Validators...)
		m.State.ActiveSequencer.SetBondedValidators(newVals)
	}

	m.logger.Debug("Updated bonded sequencer set", "newSet", m.State.ActiveSequencer.String())
	return nil
}
