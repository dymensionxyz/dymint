package block

import (
	"context"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/node/events"
	uevent "github.com/dymensionxyz/dymint/utils/event"

	"github.com/dymensionxyz/dymint/types"
)

// MonitorForkUpdate listens to the hub
func (m *Manager) MonitorForkUpdate(ctx context.Context) error {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := m.checkForkUpdate(ctx); err != nil {
				continue
			}
		}
	}
}

// checkForkUpdate checks if the hub has a fork update
func (m *Manager) checkForkUpdate(ctx context.Context) error {
	rollapp, err := m.SLClient.GetRollapp()
	if err != nil {
		return err
	}

	lastBlock, err := m.Store.LoadBlock(m.State.Height())
	if err != nil {
		return err
	}

	if m.shouldStopNode(rollapp, lastBlock) {
		err = m.createInstruction(rollapp, lastBlock)
		if err != nil {
			return err
		}

		m.freezeNode(ctx)
	}

	return nil
}

func (m *Manager) createInstruction(rollapp *types.Rollapp, block *types.Block) error {
	info, err := m.SLClient.GetStateInfo(block.Header.Height)
	if err != nil {
		return err
	}

	instruction := types.Instruction{
		Revision:            rollapp.Revision,
		RevisionStartHeight: rollapp.RevisionStartHeight,
		Sequencer:           info.NextProposer,
	}

	err = types.PersistInstructionToDisk(m.RootDir, instruction)
	if err != nil {
		return err
	}

	return nil
}

// shouldStopNode determines if a rollapp node should be stopped based on revision criteria.
//
// This method checks two conditions to decide if a node should be stopped:
// 1. If the current state height is greater than or equal to the rollapp's revision start height
// 2. If the block's app version (equivalent to revision) is less than the rollapp's revision
func (m *Manager) shouldStopNode(rollapp *types.Rollapp, block *types.Block) bool {
	revision := block.Header.Version.App
	if m.State.Height() >= rollapp.RevisionStartHeight && revision < rollapp.Revision {
		m.logger.Info(
			"Freezing node due to fork update",
			"local_block_height",
			m.State.Height(),
			"rollapp_revision_start_height",
			rollapp.RevisionStartHeight,
			"local_revision",
			revision,
			"rollapp_revision",
			rollapp.Revision,
		)
		return true
	}

	return false
}

// freezeNode stops the rollapp node
func (m *Manager) freezeNode(ctx context.Context) {
	m.logger.Info("Freezing node due to fork update")

	err := fmt.Errorf("node frozen due to fork update")
	uevent.MustPublish(ctx, m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
}
