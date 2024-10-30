package block

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/types"
)

func (m *Manager) MonitorProposerRotation(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Minute) // TODO: make this configurable
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			next, err := m.SLClient.GetNextProposer()
			if err != nil {
				m.logger.Error("Check rotation in progress", "err", err)
				continue
			}
			if next != nil {
				m.rotate(ctx)
			}
		}
	}
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
	return m.AmIProposerOnSL() || m.AmIProposerOnRollapp()
}

// AmIProposerOnSL checks if the current node is the proposer on the hub
// Proposer on the Hub is not necessarily the proposer on the L2 during rotation phase.
func (m *Manager) AmIProposerOnSL() bool {
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
	return m.AmIProposerOnSL(), nil
}

// rotate rotates current proposer by doing the following:
// 1. Creating last block with the new proposer, which will stop him from producing blocks.
// 2. Submitting the last batch
// 3. Panicing so the node restarts as full node
// Note: In case he already created his last block, he will only try to submit the last batch.
func (m *Manager) rotate(ctx context.Context) {
	// Get Next Proposer from SL. We assume such exists (even if empty proposer) otherwise function wouldn't be called.
	nextProposer, err := m.SLClient.GetNextProposer()
	if err != nil {
		panic(fmt.Sprintf("rotate: fetch next proposer set from Hub: %v", err))
	}

	err = m.CreateAndPostLastBatch(ctx, [32]byte(nextProposer.MustHash()))
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

	// check if the last block already produced with nextProposerHash set.
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
