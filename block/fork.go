package block

import (
	"context"
	"fmt"
	"github.com/dymensionxyz/dymint/version"
	"strconv"
	"time"

	"github.com/gogo/protobuf/proto"

	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/types"
	sequencers "github.com/dymensionxyz/dymint/types/pb/rollapp/sequencers/types"
	uevent "github.com/dymensionxyz/dymint/utils/event"
)

const (
	LoopInterval = 15 * time.Second
)

// MonitorForkUpdateLoop monitors the hub for fork updates in a loop
func (m *Manager) MonitorForkUpdateLoop(ctx context.Context) error {
	err := m.checkForkUpdate(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(LoopInterval) // TODO make this configurable
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

// forkNeeded returns true if the fork file exists
func (m *Manager) forkNeeded() bool {
	if _, err := types.LoadInstructionFromDisk(m.RootDir); err == nil {
		return true
	}

	return false
}

// handleSequencerForkTransition handles the sequencer fork transition
func (m *Manager) handleSequencerForkTransition(instruction types.Instruction) {
	var consensusMsgs []proto.Message
	if instruction.FaultyDRS != nil {
		drsAsInt, err := strconv.Atoi(version.DrsVersion)
		if err != nil {
			panic(fmt.Sprintf("error converting DRS version to int: %v", err))
		}

		if *instruction.FaultyDRS == uint64(drsAsInt) {
			panic(fmt.Sprintf("running faulty DRS version %d", *instruction.FaultyDRS))
		}

		msgUpgradeDRS := &sequencers.MsgUpgradeDRS{
			DrsVersion: uint64(drsAsInt),
		}

		consensusMsgs = append(consensusMsgs, msgUpgradeDRS)
	}

	// Always bump the account sequences
	msgBumpSequences := &sequencers.MsgBumpAccountSequences{Authority: "gov"}

	consensusMsgs = append(consensusMsgs, msgBumpSequences)

	// Create a new block with the consensus messages
	m.Executor.AddConsensusMsgs(consensusMsgs...)

	block, commit, err := m.produceBlock(true, nil, true)
	if err != nil {
		panic(fmt.Sprintf("produce block: %v", err))
	}

	err = m.applyBlock(block, commit, types.BlockMetaData{Source: types.Produced})

	// Create another block emtpy
	block, commit, err = m.produceBlock(true, nil, true)
	if err != nil {
		panic(fmt.Sprintf("produce empty block: %v", err))
	}

	err = m.applyBlock(block, commit, types.BlockMetaData{Source: types.Produced})

	// Create Batch and send
	_, err = m.CreateAndSubmitBatch(m.Conf.BatchSubmitBytes, false)
	if err != nil {
		panic(fmt.Sprintf("create and submit batch: %v", err))
	}
}
