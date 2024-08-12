package block

import (
	"bytes"
	"context"
	"fmt"

	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
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
func (m *Manager) MissingLastBatch() (string, error) {
	localProposerKey, _ := m.LocalKey.GetPublic().Raw()
	next, err := m.SLClient.CheckRotationInProgress()
	if err != nil {
		return "", err
	}
	if next == nil {
		return "", nil
	}
	// rotation in progress,
	// check if we're the old proposer and needs to complete rotation
	if !bytes.Equal(next.PublicKey.Bytes(), localProposerKey) {
		return next.Address, nil
	}

	return "", nil
}

// handleRotationReq completes the rotation flow once a signal is received from the SL
// this called after manager shuts down the block producer and submitter
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

// CompleteRotation completes the sequencer rotation flow
// the sequencer will create his last block, with the next sequencer's hash, to handover the proposer role
// then it will submit all the data accumulated thus far and mark the last state update
// if nextSeqAddr is empty, the nodes will halt after applying the block produced
func (m *Manager) CompleteRotation(ctx context.Context, nextSeqAddr string) error {
	// validate nextSeq is in the bonded set
	var nextSeqHash [32]byte
	if nextSeqAddr != "" {
		val := m.State.Sequencers.GetByAddress([]byte(nextSeqAddr))
		if val == nil {
			return types.ErrMissingProposerPubKey
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
	newSeqList := make([]*tmtypes.Validator, 0, len(seqs))
	for _, seq := range seqs {
		tmSeq, err := seq.TMValidator()
		if err != nil {
			return err
		}
		newSeqList = append(newSeqList, tmSeq)
	}
	m.State.Sequencers.SetSequencers(newSeqList)
	m.logger.Debug("Updated bonded sequencer set.", "newSet", m.State.Sequencers.String())
	return nil
}

// updateProposer updates the proposer in the state
func (m *Manager) UpdateProposer() error {
	var (
		err error
		p   *tmtypes.Validator
	)
	proposer := m.SLClient.GetProposer()
	if proposer != nil {
		p, err = proposer.TMValidator()
		if err != nil {
			return err
		}
	}
	m.State.Sequencers.SetProposer(p)
	return nil
}
