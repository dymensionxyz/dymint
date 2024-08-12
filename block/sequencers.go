package block

import (
	"bytes"
	"context"
	"fmt"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/google/uuid"
	tmtypes "github.com/tendermint/tendermint/types"
)

func (m *Manager) MonitorSequencerRotation(ctx context.Context, rotateC chan string) error {
	sequencerRotationEventClient := fmt.Sprintf("%s-%s", "sequencer_rotation", uuid.New().String())
	subscription, err := m.Pubsub.Subscribe(ctx, sequencerRotationEventClient, settlement.EventQueryRotationStarted)
	if err != nil {
		panic("Error subscribing to events")
	}
	defer m.Pubsub.UnsubscribeAll(ctx, sequencerRotationEventClient) //nolint:errcheck

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-subscription.Out():
			eventData, _ := event.Data().(*settlement.EventDataRotationStarted)
			nextSeqAddr := eventData.NextSeqAddr
			m.logger.Info("Sequencer rotation started.", "next_seq", nextSeqAddr)
			go func() {
				rotateC <- nextSeqAddr
			}()
			return fmt.Errorf("sequencer rotation started. signal to stop production")
		}
	}
}

// IsProposer checks if the local node is the proposer
// In case of sequencer rotation, there's a phase where proposer rotated on L2 but hasn't yet rotated on hub.
// for this case, the old proposer counts as "sequencer" as well, so he'll be able to submit the last state update.
func (m *Manager) IsProposer() bool {
	localProposerKey, _ := m.LocalKey.GetPublic().Raw()

	l2Proposer := m.GetProposerPubKey().Bytes()

	var expectedHubProposer []byte
	if m.SLClient.GetProposer() != nil {
		expectedHubProposer = m.SLClient.GetProposer().PublicKey.Bytes()
	}
	return bytes.Equal(l2Proposer, localProposerKey) || bytes.Equal(expectedHubProposer, localProposerKey)
}

// check rotation in progress (I'm the proposer, but needs to complete rotation)
func (m *Manager) MissingLastBatch() bool {
	localProposerKey, _ := m.LocalKey.GetPublic().Raw()
	expectedHubProposer := m.SLClient.GetProposer().PublicKey.Bytes()
	next, err := m.SLClient.IsRotationInProgress()
	if err != nil {
		panic(fmt.Errorf("get next proposer: %w", err))
	}
	return next != nil && bytes.Equal(expectedHubProposer, localProposerKey)
}

func (m *Manager) handleRotationReq(ctx context.Context, nextSeqAddr string) {
	m.logger.Info("Sequencer rotation started. Production stopped on this sequencer", "nextSeqAddr", nextSeqAddr)
	err := m.CompleteRotation(ctx, nextSeqAddr)
	if err != nil {
		panic(err)
	}

	// TODO: graceful fallback to full node
	m.logger.Info("Sequencer is no longer the proposer")
	// panic("sequencer is no longer the proposer")
}

// complete rotation
func (m *Manager) CompleteRotation(ctx context.Context, nextSeqAddr string) error {
	// validate nextSeq is in the bonded set
	var nextSeqHash [32]byte
	if nextSeqAddr != "" {
		val := m.State.Sequencers.GetByAddress([]byte(nextSeqAddr))
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
	_, _, err := m.ProduceApplyGossipLastBlock(ctx, nextSeqHash)
	if err != nil {
		return fmt.Errorf("produce apply gossip last block: %w", err)
	}
	// Submit all data accumulated thus far
	for {
		b, err := m.CreateBatch(m.Conf.BatchMaxSizeBytes, m.NextHeightToSubmit(), m.State.Height())
		if err != nil {
			return fmt.Errorf("create batch: %w", err)
		}
		if m.State.Height() == b.EndHeight() {
			b.LastBatch = true
		}

		if err := m.SubmitBatch(b); err != nil {
			return fmt.Errorf("submit batch: %w", err)
		}

		if b.LastBatch {
			break
		}
	}

	return nil
}

// UpdateSequencerSetFromSL updates the sequencer set from the SL
// proposer is not changed here
func (m *Manager) UpdateSequencerSetFromSL() error {
	seqs, err := m.SLClient.GetAllSequencers()
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
		newVals = append(newVals, val)
	}
	m.State.Sequencers.SetSequencers(newVals)
	m.logger.Debug("Updated bonded sequencer set.", "newSet", m.State.Sequencers.String())
	return nil
}

// updateProposer updates the proposer in the state
func (m *Manager) UpdateProposer() error {
	var p *tmtypes.Validator
	proposer := m.SLClient.GetProposer()
	if proposer != nil {
		tmPubKey, err := cryptocodec.ToTmPubKeyInterface(proposer.PublicKey)
		if err != nil {
			return err
		}
		p = tmtypes.NewValidator(tmPubKey, 1)
	}
	m.State.Sequencers.SetProposer(p)
	return nil
}
