package block

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

func (m *Manager) MonitorSequencerRotation(ctx context.Context, rotateC chan string) error {
	sequencerRotationEventClient := "sequencer_rotation"
	subscription, err := m.Pubsub.Subscribe(ctx, sequencerRotationEventClient, settlement.EventQueryRotationStarted)
	if err != nil {
		panic("Error subscribing to events")
	}
	defer m.Pubsub.UnsubscribeAll(ctx, sequencerRotationEventClient) //nolint:errcheck

	ticker := time.NewTicker(3 * time.Minute) // TODO: make this configurable
	defer ticker.Stop()

	var nextSeqAddr string
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			next, err := m.SLClient.CheckRotationInProgress()
			if err != nil {
				m.logger.Error("Check rotation in progress", "err", err)
				continue
			}
			if next == nil {
				continue
			}
			nextSeqAddr = next.SettlementAddress
		case event := <-subscription.Out():
			eventData, _ := event.Data().(*settlement.EventDataRotationStarted)
			nextSeqAddr = eventData.NextSeqAddr
		}
		break // break out of the loop after getting the next sequencer address
	}
	// we get here once a sequencer rotation signal is received
	m.logger.Info("Sequencer rotation started.", "next_seq", nextSeqAddr)
	rotateC <- nextSeqAddr
	return fmt.Errorf("sequencer rotation started. signal to stop production")
}

// IsProposer checks if the local node is the proposer
// In case of sequencer rotation, there's a phase where proposer rotated on L2 but hasn't yet rotated on hub.
// for this case, the old proposer counts as "sequencer" as well, so he'll be able to submit the last state update.
func (m *Manager) IsProposer() bool {
	localProposerKey, _ := m.LocalKey.GetPublic().Raw()
	l2Proposer := m.GetProposerPubKey().Bytes()

	var expectedHubProposer []byte
	hubProposer := m.SLClient.GetProposer()
	if hubProposer != nil {
		expectedHubProposer = hubProposer.PubKey().Bytes()
	}

	// check if recovering from halt
	if l2Proposer == nil && hubProposer != nil {
		m.State.Sequencers.SetProposer(hubProposer)
	}

	// we run sequencer flow if we're proposer on L2 or hub (can be different during rotation phase, before hub receives the last state update)
	return bytes.Equal(l2Proposer, localProposerKey) || bytes.Equal(expectedHubProposer, localProposerKey)
}

// CompleteRotationIfNeeded checks if the sequencer is in the middle of rotation (I'm the proposer, but needs to complete rotation)
// returns the next sequencer address and a flag if the sequencer is the old proposer and the hub waits for the last batch
func (m *Manager) CompleteRotationIfNeeded(ctx context.Context) error {
	localProposerKey, _ := m.LocalKey.GetPublic().Raw()
	next, err := m.SLClient.CheckRotationInProgress()
	if err != nil {
		return err
	}
	if next == nil {
		return nil
	}
	// rotation in not completed yet! check if we're the old proposer and needs to complete rotation
	if bytes.Equal(m.SLClient.GetProposer().PubKey().Bytes(), localProposerKey) {
		// complete rotation before running the main loops
		m.handleRotationReq(ctx, next.SettlementAddress)
		m.isProposer = false
		m.logger.Info("Sequencer is no longer the proposer")
	}
	return nil
}

// handleRotationReq completes the rotation flow once a signal is received from the SL
// this called after manager shuts down the block producer and submitter
func (m *Manager) handleRotationReq(ctx context.Context, nextSeqAddr string) {
	m.logger.Info("Sequencer rotation started. Production stopped on this sequencer", "nextSeqAddr", nextSeqAddr)
	err := m.CompleteRotation(ctx, nextSeqAddr)
	if err != nil {
		m.logger.Error("Complete rotation", "err", err)
		panic(err)
	}
}

// CompleteRotation completes the sequencer rotation flow
// the sequencer will create his last block, with the next sequencer's hash, to handover the proposer role
// then it will submit all the data accumulated thus far and mark the last state update
// if nextSeqAddr is empty, the nodes will halt after applying the block produced
func (m *Manager) CompleteRotation(ctx context.Context, nextSeqAddr string) error {
	// validate nextSeq is in the bonded set
	var nextSeqHash [32]byte
	if nextSeqAddr != "" {
		seq := m.State.Sequencers.GetByAddress(nextSeqAddr)
		if seq == nil {
			return types.ErrMissingProposerPubKey
		}
		copy(nextSeqHash[:], seq.MustHash())
	}

	err := m.CreateAndPostLastBatch(ctx, nextSeqHash)
	if err != nil {
		return fmt.Errorf("create and post last batch: %w", err)
	}

	m.logger.Info("Sequencer rotation completed. sequencer is no longer the proposer", "nextSeqAddr", nextSeqAddr)
	return nil
}

// CreateAndPostLastBatch creates and posts the last batch to the hub
// this called after manager shuts down the block producer and submitter
func (m *Manager) CreateAndPostLastBatch(ctx context.Context, nextSeqHash [32]byte) error {
	h := m.State.Height()
	block, err := m.Store.LoadBlock(h)
	if err != nil {
		return fmt.Errorf("load block: height: %d: %w", h, err)
	}

	// check if the last block already produced with nextProposerHash set
	if bytes.Equal(block.Header.NextSequencersHash[:], nextSeqHash[:]) {
		m.logger.Debug("Last block already produced and applied.")
	} else {
		err := m.ProduceApplyGossipLastBlock(ctx, nextSeqHash)
		if err != nil {
			return fmt.Errorf("produce apply gossip last block: %w", err)
		}
	}

	// Submit all data accumulated thus far and the last state update
	for {
		b, err := m.CreateAndSubmitBatch(m.Conf.BatchSubmitBytes, true)
		if err != nil {
			return fmt.Errorf("CreateAndSubmitBatch last batch: %w", err)
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
	m.State.Sequencers.SetSequencers(seqs)
	m.logger.Debug("Updated bonded sequencer set.", "newSet", m.State.Sequencers.String())
	return nil
}

// UpdateProposer updates the proposer from the hub
func (m *Manager) UpdateProposer() error {
	m.State.Sequencers.SetProposer(m.SLClient.GetProposer())
	return nil
}

// UpdateLastSubmittedHeight will update last height submitted height upon events.
// This may be necessary in case we crashed/restarted before getting response for our submission to the settlement layer.
func (m *Manager) UpdateSequencerSet(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBondedSequencer)
	if !ok {
		m.logger.Error("onReceivedBatch", "err", "wrong event data received")
		return
	}

	if m.State.Sequencers.GetByAddress(eventData.SeqAddr) != nil {
		m.logger.Debug("Sequencer not added from new bonded sequencer event because already in the list.")
		return
	}

	newSequencer, err := m.SLClient.GetSequencerByAddress(eventData.SeqAddr)
	if err != nil {
		m.logger.Error("Unable to add new sequencer from event. err:%w", err)
		return
	}
	sequencers := append(m.State.Sequencers.Sequencers, newSequencer)
	m.State.Sequencers.SetSequencers(sequencers)
}
