package block

import (
	"bytes"
	"context"
	"fmt"
	"time"
)

const (
	ProposerMonitorInterval = 3 * time.Minute
)

var errRotationRequested = fmt.Errorf("sequencer rotation started. signal to stop production")

func (m *Manager) MonitorProposerRotation(ctx context.Context) error {
	ticker := time.NewTicker(ProposerMonitorInterval) // TODO: make this configurable
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			nextProposer, err := m.SLClient.GetNextProposer()
			if err != nil {
				m.logger.Error("Check rotation in progress", "err", err)
				continue
			}
			// no rotation in progress
			if nextProposer == nil {
				continue
			}

			// we get here once a sequencer rotation signal is received
			m.logger.Info("Sequencer rotation started.", "nextSeqAddr", nextProposer.SettlementAddress)
			return errRotationRequested
		}
	}
}

func (m *Manager) MonitorSequencerSetUpdates(ctx context.Context) error {
	ticker := time.NewTicker(m.Conf.SequencerSetUpdateInterval)
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

// AmIProposerOnSL checks if the current node is the proposer on the hub
// Proposer on the Hub is not necessarily the proposer on the Rollapp during rotation phase.
func (m *Manager) AmIProposerOnSL() (bool, error) {
	localProposerKeyBytes, _ := m.LocalKey.GetPublic().Raw()
	// get hub proposer key
	SLProposer, err := m.SLClient.GetProposerAtHeight(-1)
	if err != nil {
		return false, fmt.Errorf("get proposer at height: %w", err)
	}
	return bytes.Equal(SLProposer.PubKey().Bytes(), localProposerKeyBytes), nil
}

// AmIProposerOnRollapp checks if the current node is the proposer on the rollapp.
// Proposer on the rollapp is not necessarily the proposer on the hub during rotation phase.
func (m *Manager) AmIProposerOnRollapp() bool {
	if m.State.GetProposer() == nil {
		return false
	}
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
	amIProposerOnSL, err := m.AmIProposerOnSL()
	if err != nil {
		return false, fmt.Errorf("am i proposer on SL: %w", err)
	}
	return amIProposerOnSL, nil
}

// rotate rotates current proposer by doing the following:
// 1. Creating last block with the new proposer, which will stop him from producing blocks.
// 2. Submitting the last batch
// 3. Panicing so the node restarts as full node
// Note: In case he already created his last block, he will only try to submit the last batch.
func (m *Manager) rotate(ctx context.Context) {
	// Get Next Proposer from SL. We assume such exists (even if empty proposer) otherwise function wouldn't be called.
	nextProposer, err := m.SLClient.GetNextProposer()
	if err != nil || nextProposer == nil {
		panic(fmt.Sprintf("rotate: fetch next proposer set from Hub: %v", err))
	}

	var nextProposerHash [32]byte
	if !nextProposer.IsEmpty() {
		nextProposerHash = [32]byte(nextProposer.MustHash())
	}

	err = m.CreateAndPostLastBatch(ctx, nextProposerHash)
	if err != nil {
		panic(fmt.Sprintf("rotate: create and post last batch: %v", err))
	}

	m.logger.Info("Sequencer rotation completed. sequencer is no longer the proposer", "nextSeqAddr", nextProposer.SettlementAddress)

	panic("rotate: sequencer is no longer the proposer. restarting as a full node")
}

// CreateAndPostLastBatch creates and posts the last batch to the hub
// this called after manager shuts down the block producer and submitter
func (m *Manager) CreateAndPostLastBatch(ctx context.Context, nextSeqHash [32]byte) error {
	h := m.State.Height()
	block, err := m.Store.LoadBlock(h)
	if err != nil {
		return fmt.Errorf("load block: height: %d: %w", h, err)
	}

	// check if the last block already produced with NextProposerHash set.
	// After creating the last block, the sequencer will be restarted so it will not be able to produce blocks anymore.
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

// UpdateSequencerSetFromSL updates the sequencer set from the SL. The sequencer set is saved only in memory.
// It will be persisted to the store when the block is produced (only in the proposer mode).
// Proposer is not changed here.
func (m *Manager) UpdateSequencerSetFromSL() error {
	seqs, err := m.SLClient.GetAllSequencers()
	if err != nil {
		return fmt.Errorf("get all sequencers from the hub: %w", err)
	}
	m.Sequencers.Set(seqs)
	m.logger.Debug("Updated bonded sequencer set.", "newSet", m.Sequencers.String())
	return nil
}

// UpdateProposerFromSL queries the hub and updates the local dymint state proposer at the current height
func (m *Manager) UpdateProposerFromSL() error {
	SLProposer, err := m.SLClient.GetProposerAtHeight(int64(m.State.NextHeight())) //nolint:gosec // height is non-negative and falls in int64
	if err != nil {
		return fmt.Errorf("get proposer at height: %w", err)
	}
	m.logger.Debug("Updating proposer to ", SLProposer.SettlementAddress)
	m.State.SetProposer(SLProposer)
	return nil
}
