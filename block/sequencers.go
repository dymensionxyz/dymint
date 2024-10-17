package block

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
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
	go func() {
		rotateC <- nextSeqAddr
	}()
	return fmt.Errorf("sequencer rotation started. signal to stop production")
}

func (m *Manager) MonitorSequencerSetUpdates(ctx context.Context, updatesC chan<- []types.Sequencer) error {
	ticker := time.NewTicker(3 * time.Minute) // TODO: make this configurable
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			currentSLSet, err := m.SLClient.GetAllSequencers()
			if err != nil {
				m.logger.Error("Get bonded sequencers", "err", err)
				continue
			}
			updatesC <- currentSLSet
		}
	}
}

// handleSequencerSetUpdate calculates the diff between hub's and current sequencer sets and
// creates consensus messages for all new sequencers. The method updates the current state
// and is not thread-safe. Returns errors on serialization issues.
func (m *Manager) handleSequencerSetUpdate(newSet []types.Sequencer) error {
	newSequencers := types.SequencerListRightOuterJoin(m.State.Sequencers.Sequencers, newSet)
	msgs, err := ConsensusMsgsOnSequencerSetUpdate(newSequencers)
	if err != nil {
		return fmt.Errorf("consensus msgs on sequencers set update: %w", err)
	}
	m.State.Sequencers.SetSequencers(newSequencers)
	m.Executor.consensusMessagesStream.Add(msgs...)
	return nil
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

// MissingLastBatch checks if the sequencer is in the middle of rotation (I'm the proposer, but needs to complete rotation)
// returns the next sequencer address and a flag if the sequencer is the old proposer and the hub waits for the last batch
func (m *Manager) MissingLastBatch() (string, bool, error) {
	localProposerKey, _ := m.LocalKey.GetPublic().Raw()
	next, err := m.SLClient.CheckRotationInProgress()
	if err != nil {
		return "", false, err
	}
	if next == nil {
		return "", false, nil
	}
	// rotation in progress,
	// check if we're the old proposer and needs to complete rotation
	curr := m.SLClient.GetProposer()
	isProposer := bytes.Equal(curr.PubKey().Bytes(), localProposerKey)
	return next.SettlementAddress, isProposer, nil
}

// handleRotationReq completes the rotation flow once a signal is received from the SL
// this called after manager shuts down the block producer and submitter. The method updates
// the state and is not thread-safe.
func (m *Manager) handleRotationReq(ctx context.Context, nextSeqAddr string) {
	err := m.UpdateSequencerSetFromSL()
	if err != nil {
		panic(fmt.Errorf("update sequencer set upon rotation: %w", err))
	}

	m.logger.Info("Sequencer rotation started. Production stopped on this sequencer", "nextSeqAddr", nextSeqAddr)
	err = m.CompleteRotation(ctx, nextSeqAddr)
	if err != nil {
		panic(fmt.Errorf("complete sequencer rotation: %w", err))
	}

	// TODO: graceful fallback to full node (https://github.com/dymensionxyz/dymint/issues/1008)
	m.logger.Info("Sequencer is no longer the proposer")
	panic("sequencer is no longer the proposer")
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
		copy(nextSeqHash[:], seq.Hash())
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

// UpdateSequencerSetFromSL updates the sequencer set from the SL proposer is not changed here.
// The method modifies the state and is not thread-safe.
func (m *Manager) UpdateSequencerSetFromSL() error {
	seqs, err := m.SLClient.GetAllSequencers()
	if err != nil {
		return err
	}
	err = m.handleSequencerSetUpdate(seqs)
	if err != nil {
		return err
	}
	m.logger.Debug("Updated bonded sequencer set.", "newSet", m.State.Sequencers.String())
	return nil
}

// UpdateProposer updates the proposer from the hub
func (m *Manager) UpdateProposer() error {
	m.State.Sequencers.SetProposer(m.SLClient.GetProposer())
	return nil
}
