package block

import (
	"context"
	"errors"
	"fmt"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sequencers "github.com/dymensionxyz/dymension-rdk/x/sequencers/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/gogo/protobuf/proto"

	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/version"
)

const (
	ForkMonitorInterval = 15 * time.Second
	ForkMessage         = "rollapp fork detected. please rollback to height previous to rollapp_revision_start_height."
)

func (m *Manager) MonitorForkUpdateLoop(ctx context.Context) error {
	ticker := time.NewTicker(ForkMonitorInterval)
	defer ticker.Stop()

	for {
		if err := m.checkForkUpdate(ForkMessage); err != nil {
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (m *Manager) checkForkUpdate(msg string) error {
	defer m.forkMu.Unlock()
	m.forkMu.Lock()

	rollapp, err := m.SLClient.GetRollapp()
	if err != nil {
		return err
	}

	var (
		nextHeight       = m.State.NextHeight()
		actualRevision   = m.State.GetRevision()
		expectedRevision = rollapp.GetRevisionForHeight(nextHeight)
	)

	if shouldStopNode(expectedRevision, nextHeight, actualRevision) {
		instruction, err := m.createInstruction(expectedRevision)
		if err != nil {
			return err
		}

		err = types.PersistInstructionToDisk(m.RootDir, instruction)
		if err != nil {
			return err
		}
		m.freezeNode(fmt.Errorf("%s  local_block_height: %d rollapp_revision_start_height: %d local_revision: %d rollapp_revision: %d", msg, m.State.Height(), expectedRevision.StartHeight, actualRevision, expectedRevision.Number))
	}

	return nil
}

func (m *Manager) createInstruction(expectedRevision types.Revision) (types.Instruction, error) {
	obsoleteDrs, err := m.SLClient.GetObsoleteDrs()
	if err != nil {
		return types.Instruction{}, err
	}

	instruction := types.Instruction{
		Revision:            expectedRevision.Number,
		RevisionStartHeight: expectedRevision.StartHeight,
		FaultyDRS:           obsoleteDrs,
	}

	return instruction, nil
}

func shouldStopNode(
	expectedRevision types.Revision,
	nextHeight uint64,
	actualRevisionNumber uint64,
) bool {
	return nextHeight >= expectedRevision.StartHeight && actualRevisionNumber < expectedRevision.Number
}

func (m *Manager) getRevisionFromSL(height uint64) (types.Revision, error) {
	rollapp, err := m.SLClient.GetRollapp()
	if err != nil {
		return types.Revision{}, err
	}
	return rollapp.GetRevisionForHeight(height), nil
}

func (m *Manager) doFork(instruction types.Instruction) error {
	if m.State.Height() < instruction.RevisionStartHeight+1 {

		consensusMsgs, err := m.prepareDRSUpgradeMessages(instruction.FaultyDRS)
		if err != nil {
			return fmt.Errorf("prepare DRS upgrade messages: %v", err)
		}

		consensusMsgs = append(consensusMsgs, &sequencers.MsgBumpAccountSequences{Authority: authtypes.NewModuleAddress("sequencers").String()})

		err = m.createForkBlocks(instruction, consensusMsgs)
		if err != nil {
			return fmt.Errorf("validate fork blocks: %v", err)
		}
	}

	if err := m.submitForkBatch(instruction.RevisionStartHeight); err != nil {
		return fmt.Errorf("submit fork batch: %v", err)
	}

	return nil
}

func (m *Manager) prepareDRSUpgradeMessages(obsoleteDRS []uint32) ([]proto.Message, error) {
	drsVersion, err := version.GetDRSVersion()
	if err != nil {
		return nil, err
	}

	for _, drs := range obsoleteDRS {
		if drs == drsVersion {
			return nil, gerrc.ErrCancelled.Wrapf("obsolete DRS version: %d", drs)
		}
	}

	if m.State.RollappParams.DrsVersion == drsVersion {
		return []proto.Message{}, nil
	}

	return []proto.Message{
		&sequencers.MsgUpgradeDRS{
			Authority:  authtypes.NewModuleAddress("sequencers").String(),
			DrsVersion: uint64(drsVersion),
		},
	}, nil
}

func (m *Manager) createForkBlocks(instruction types.Instruction, consensusMsgs []proto.Message) error {
	nextHeight := m.State.NextHeight()

	for h := instruction.RevisionStartHeight; h < nextHeight; h++ {
		b, err := m.Store.LoadBlock(h)
		if err != nil {
			return gerrc.ErrInternal.Wrapf("load stored blocks: %d", h)
		}

		if 0 < len(b.Data.Txs) {
			return gerrc.ErrInternal.Wrapf("fork block has tx: %d", h)
		}

		if (h == instruction.RevisionStartHeight) != (0 < len(b.Data.ConsensusMessages)) {
			return gerrc.ErrInternal.Wrapf("fork block has wrong num cons messages: %d", h)
		}
	}

	for h := nextHeight; h < instruction.RevisionStartHeight+2; h++ {
		if h == instruction.RevisionStartHeight {
			m.Executor.AddConsensusMsgs(consensusMsgs...)
		}
		zero := uint64(0)
		if _, _, err := m.ProduceApplyGossipBlock(context.Background(), ProduceBlockOptions{
			AllowEmpty:       true,
			MaxData:          &zero,
			NextProposerHash: nil,
		}); err != nil {
			return fmt.Errorf("produce apply gossip: h: %d : %w", h, err)
		}
	}

	return nil
}

func (m *Manager) submitForkBatch(height uint64) error {
	resp, err := m.SLClient.GetBatchAtHeight(height)
	if err != nil && !errors.Is(err, gerrc.ErrNotFound) {
		return fmt.Errorf("getting batch at height: %v", err)
	}

	if resp != nil {
		return nil
	}

	if _, err = m.CreateAndSubmitBatch(m.Conf.BatchSubmitBytes, false); err != nil {
		return fmt.Errorf("creating and submitting batch: %v", err)
	}

	return nil
}

func (m *Manager) updateStateForNextRevision() error {
	nextRevision, err := m.getRevisionFromSL(m.State.NextHeight())
	if err != nil {
		return err
	}

	if nextRevision.StartHeight == m.State.NextHeight() {

		m.State.SetProposer(nil)

		m.State.RevisionStartHeight = nextRevision.StartHeight
		m.State.SetRevision(nextRevision.Number)

		_, err = m.Store.SaveState(m.State, nil)
		return err
	}
	return nil
}

func (m *Manager) doForkWhenNewRevision() error {
	defer m.forkMu.Unlock()
	m.forkMu.Lock()

	expectedRevision, err := m.getRevisionFromSL(m.State.NextHeight())
	if err != nil {
		return err
	}

	if m.LastSettlementHeight.Load() < expectedRevision.StartHeight {
		instruction, err := m.createInstruction(expectedRevision)
		if err != nil {
			return err
		}

		m.State.SetRevision(instruction.Revision)

		err = m.doFork(instruction)
		if err != nil {
			return err
		}
	}

	if expectedRevision.Number != m.State.GetRevision() {
		panic("Inconsistent expected revision number from Hub. Unable to fork")
	}

	return types.DeleteInstructionFromDisk(m.RootDir)
}
