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
	ticker := time.NewTicker(ProposerMonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			nextProposer, err := m.SLClient.GetNextProposer()
			if err != nil {
				continue
			}

			if nextProposer == nil {
				continue
			}

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
			err := m.UpdateSequencerSetFromSL() // what is the point in this? what justification is there?
			if err != nil {
			}
		}
	}
}

func (m *Manager) AmIProposerOnSL() (bool, error) {
	localProposerKeyBytes, _ := m.LocalKey.GetPublic().Raw()

	SLProposer, err := m.SLClient.GetProposerAtHeight(-1)
	if err != nil {
		return false, fmt.Errorf("get proposer at height: %w", err)
	}
	return bytes.Equal(SLProposer.PubKey().Bytes(), localProposerKeyBytes), nil
}

func (m *Manager) AmIProposerOnRollapp() bool {
	if m.State.GetProposer() == nil { // when can it occur?
		return false
	}
	localProposerKeyBytes, _ := m.LocalKey.GetPublic().Raw() // should be method, dryer
	rollappProposer := m.State.GetProposerPubKey().Bytes()

	return bytes.Equal(rollappProposer, localProposerKeyBytes)
}

func (m *Manager) ShouldRotate() (bool, error) {
	nextProposer, err := m.SLClient.GetNextProposer()
	if err != nil {
		return false, err
	}
	if nextProposer == nil {
		return false, nil
	}

	amIProposerOnSL, err := m.AmIProposerOnSL()
	if err != nil {
		return false, fmt.Errorf("am i proposer on SL: %w", err)
	}
	return amIProposerOnSL, nil
}

func (m *Manager) rotate(ctx context.Context) {
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

	panic("rotate: sequencer is no longer the proposer. restarting as a full node")
}

func (m *Manager) CreateAndPostLastBatch(ctx context.Context, nextSeqHash [32]byte) error {
	h := m.State.Height()
	block, err := m.Store.LoadBlock(h)
	if err != nil {
		return fmt.Errorf("load block: height: %d: %w", h, err)
	}

	if bytes.Equal(block.Header.NextSequencersHash[:], nextSeqHash[:]) {
	} else {
		err := m.ProduceApplyGossipLastBlock(ctx, nextSeqHash)
		if err != nil {
			return fmt.Errorf("produce apply gossip last block: %w", err)
		}
	}

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

func (m *Manager) UpdateSequencerSetFromSL() error {
	seqs, err := m.SLClient.GetAllSequencers() // cache/debounce?
	if err != nil {
		return fmt.Errorf("get all sequencers from the hub: %w", err)
	}
	m.Sequencers.Set(seqs)
	return nil
}

func (m *Manager) UpdateProposerFromSL() error {
	SLProposer, err := m.SLClient.GetProposerAtHeight(int64(m.State.NextHeight()))
	if err != nil {
		return fmt.Errorf("get proposer at height: %w", err)
	}
	m.State.SetProposer(SLProposer)
	return nil
}
