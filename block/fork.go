package block

import (
	"context"
	"time"

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
			if err := m.checkForkUpdate(); err != nil {
				continue
			}
		}
	}
}

// checkForkUpdate checks if the hub has a fork update
func (m *Manager) checkForkUpdate() error {
	rollapp, err := m.SLClient.GetRollapp()
	if err != nil {
		return err
	}

	lastBlock, err := m.Store.LoadBlock(m.State.Height())

	if m.shouldStopNode(rollapp, lastBlock) {
		createInstruction(rollapp)
	}

	return nil
}

func createInstruction(rollapp *types.Rollapp) {
	instruction := types.Instruction{
		Revision:            rollapp.Revision,
		RevisionStartHeight: rollapp.RevisionStartHeight,
		Sequencer:
	}
}

// shouldStopNode determines if a rollapp node should be stopped based on revision criteria.
//
// This method checks two conditions to decide if a node should be stopped:
// 1. If the current state height is greater than or equal to the rollapp's revision start height
// 2. If the block's app version (equivalent to revision) is less than the rollapp's revision
func (m *Manager) shouldStopNode(rollapp *types.Rollapp, block *types.Block) bool {
	revision := block.Header.Version.App
	if m.State.Height() >= rollapp.RevisionStartHeight && revision < rollapp.Revision {
		return true
	}

	return false
}
