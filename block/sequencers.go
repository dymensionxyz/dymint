package block

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

func (m *Manager) MonitorProposerRotation(ctx context.Context, rotateC chan string) error {
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
			next, err := m.SLClient.GetNextProposer()
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

func (m *Manager) MonitorSequencerSetUpdates(ctx context.Context) error {
	ticker := time.NewTicker(3 * time.Minute) // TODO: make this configurable
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := m.UpdateSequencerSetFromSL()
			if err != nil {
				// this error is not critical
				m.logger.Error("Cannot fetch sequencer set from the Hub", "error", err)
			}
		}
	}
}

// AmIPropoesr checks if the the current node is the proposer either on L2 or on the hub.
// In case of sequencer rotation, there's a phase where proposer rotated on L2 but hasn't yet rotated on hub.
// for this case, 2 nodes will get `true` for `AmIProposer` so the l2 proposer can produce blocks and the hub proposer can submit his last batch.
func (m *Manager) AmIProposer() bool {
	return m.AmIProposerOnHub() || m.AmIProposerOnRollapp()
}

// AmIProposerOnHub checks if the current node is the proposer on the hub
// Proposer on the Hub is not necessarily the proposer on the L2 during rotation phase.
func (m *Manager) AmIProposerOnHub() bool {
	localProposerKeyBytes, _ := m.LocalKey.GetPublic().Raw()

	// get hub proposer key
	var hubProposerKeyBytes []byte
	hubProposer := m.SLClient.GetProposer()
	if hubProposer != nil {
		hubProposerKeyBytes = hubProposer.PubKey().Bytes()
	}
	return bytes.Equal(hubProposerKeyBytes, localProposerKeyBytes)
}

// AmIProposerOnRollapp checks if the current node is the proposer on the rollapp.
// Proposer on the rollapp is not necessarily the proposer on the hub during rotation phase.
func (m *Manager) AmIProposerOnRollapp() bool {
	localProposerKeyBytes, _ := m.LocalKey.GetPublic().Raw()
	rollappProposer := m.State.GetProposerPubKey().Bytes()

	return bytes.Equal(rollappProposer, localProposerKeyBytes)
}

// ShouldRotate checks if the we are in the middle of rotation and we are the rotating proposer (i.e current proposer on the hub).
// We check it by checking if there is a "next" proposer on the hub which is not us.
func (m *Manager) ShouldRotate() (bool, error) {
	nextProposer, err := m.SLClient.GetNextProposer()
	if err != nil {
		return false, err
	}
	if nextProposer == nil {
		return false, nil
	}
	// At this point we know that there is a next proposer,
	// so we should rotate only if we are the current proposer on the hub
	return m.AmIProposerOnHub(), nil
}

// rotate rotates current proposer by changing the next sequencer on the last block and submitting the last batch.
// The assumption is that at this point the sequencer has stopped producing blocks.
func (m *Manager) rotate(ctx context.Context, nextSeqAddr string) error {
	m.logger.Info("Sequencer rotation started. Production stopped on this sequencer", "nextSeqAddr", nextSeqAddr)

	// Update sequencers list from SL
	err := m.UpdateSequencerSetFromSL()
	if err != nil {
		// this error is not critical, try to complete the rotation anyway
		m.logger.Error("Cannot fetch sequencer set from the Hub", "error", err)
	}

	// validate nextSeq is in the bonded set
	var nextSeqHash [32]byte
	if nextSeqAddr != "" {
		seq, found := m.Sequencers.GetByAddress(nextSeqAddr)
		if !found {
			return types.ErrMissingProposerPubKey
		}
		copy(nextSeqHash[:], seq.MustHash())
	}

	err = m.CreateAndPostLastBatch(ctx, nextSeqHash)
	if err != nil {
		return fmt.Errorf("create and post last batch: %w", err)
	}

	m.logger.Info("Sequencer rotation completed. sequencer is no longer the proposer", "nextSeqAddr", nextSeqAddr)

	return fmt.Errorf("sequencer is no longer the proposer")
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

// UpdateSequencerSetFromSL updates the sequencer set from the SL.
// Proposer is not changed here.
func (m *Manager) UpdateSequencerSetFromSL() error {
	seqs, err := m.SLClient.GetAllSequencers()
	if err != nil {
		return fmt.Errorf("get all sequencers from the hub: %w", err)
	}
	err = m.HandleSequencerSetUpdate(seqs)
	if err != nil {
		return fmt.Errorf("handle sequencer set update: %w", err)
	}
	m.logger.Debug("Updated bonded sequencer set.", "newSet", m.Sequencers.String())
	return nil
}

// HandleSequencerSetUpdate calculates the diff between hub's and current sequencer sets and
// creates consensus messages for all new sequencers. The method updates the current state
// and is not thread-safe. Returns errors on serialization issues.
func (m *Manager) HandleSequencerSetUpdate(newSet []types.Sequencer) error {
	// find new (updated) sequencers
	newSequencers := types.SequencerListRightOuterJoin(m.Sequencers.GetAll(), newSet)
	// create consensus msgs for new sequencers
	msgs, err := ConsensusMsgsOnSequencerSetUpdate(newSequencers)
	if err != nil {
		return fmt.Errorf("consensus msgs on sequencers set update: %w", err)
	}
	// add consensus msgs to the stream
	m.Executor.AddConsensusMsgs(msgs...)
	// save the new sequencer set to the state
	m.Sequencers.Set(newSet)
	return nil
}

// UpdateProposerFromSL updates the proposer from the hub
func (m *Manager) UpdateProposerFromSL() error {
	m.State.SetProposer(m.SLClient.GetProposer())
	return nil
}
