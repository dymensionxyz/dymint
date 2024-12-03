package block

import (
	"context"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	uerrors "github.com/dymensionxyz/dymint/utils/errors"
	uevent "github.com/dymensionxyz/dymint/utils/event"
	"golang.org/x/sync/errgroup"
)

const (
	syncLoop         = "syncLoop"
	validateLoop     = "validateLoop"
	p2pGossipLoop    = "applyGossipedBlocksLoop"
	p2pBlocksyncLoop = "applyBlockSyncBlocksLoop"
)

// setFraudHandler sets the fraud handler for the block manager.
func (m *Manager) runAsFullNode(ctx context.Context, eg *errgroup.Group) error {
	m.logger.Info("starting block manager", "mode", "full node")
	// update latest finalized height
	err := m.updateLastFinalizedHeightFromSettlement()
	if err != nil {
		return fmt.Errorf("sync block manager from settlement: %w", err)
	}

	// Start the settlement validation loop in the background
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SettlementValidateLoop(ctx)
	})

	m.subscribeFullNodeEvents(ctx)

	if _, instructionExists := m.forkNeeded(); instructionExists {
		// remove instruction file after fork to avoid enter fork loop again
		err := types.DeleteInstructionFromDisk(m.RootDir)
		if err != nil {
			return fmt.Errorf("deleting instruction file: %w", err)
		}
	}

	return nil
}

func (m *Manager) runAsProposer(ctx context.Context, eg *errgroup.Group) error {
	m.logger.Info("starting block manager", "mode", "proposer")
	// Subscribe to batch events, to update last submitted height in case batch confirmation was lost. This could happen if the sequencer crash/restarted just after submitting a batch to the settlement and by the time we query the last batch, this batch wasn't accepted yet.
	go uevent.MustSubscribe(ctx, m.Pubsub, "updateSubmittedHeightLoop", settlement.EventQueryNewSettlementBatchAccepted, m.UpdateLastSubmittedHeight, m.logger)
	// Subscribe to P2P received blocks events (used for P2P syncing).
	go uevent.MustSubscribe(ctx, m.Pubsub, p2pBlocksyncLoop, p2p.EventQueryNewBlockSyncBlock, m.OnReceivedBlock, m.logger)

	// Sequencer must wait till the DA light client is synced. Otherwise it will fail when submitting blocks.
	// Full-nodes does not need to wait, but if it tries to fetch blocks from DA heights previous to the DA light client height it will fail, and it will retry till it reaches the height.
	m.DAClient.WaitForSyncing()

	// Sequencer must wait till node is synced till last submittedHeight, in case it is not
	m.waitForSettlementSyncing()

	// checkRevisionAndFork executes fork if necessary
	err := m.checkRevisionAndFork()
	if err != nil {
		return err
	}

	// check if we should rotate
	shouldRotate, err := m.ShouldRotate()
	if err != nil {
		return fmt.Errorf("checking should rotate: %w", err)
	}
	if shouldRotate {
		m.rotate(ctx) // panics afterwards
	}

	// populate the bytes produced channel
	bytesProducedC := make(chan int)

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SubmitLoop(ctx, bytesProducedC)
	})

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		bytesProducedC <- m.GetUnsubmittedBytes() // load unsubmitted bytes from previous run
		return m.ProduceBlockLoop(ctx, bytesProducedC)
	})

	// Monitor and handling of the rotation
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.MonitorProposerRotation(ctx)
	})

	go func() {
		err = eg.Wait()
		// Check if loops exited due to sequencer rotation signal
		if errors.Is(err, errRotationRequested) {
			m.rotate(ctx)
		} else if err != nil {
			m.logger.Error("block manager exited with error", "error", err)
			m.freezeNode(err)
		}
	}()

	return nil
}

func (m *Manager) subscribeFullNodeEvents(ctx context.Context) {
	// Subscribe to new (or finalized) state updates events.
	go uevent.MustSubscribe(ctx, m.Pubsub, syncLoop, settlement.EventQueryNewSettlementBatchAccepted, m.onNewStateUpdate, m.logger)
	go uevent.MustSubscribe(ctx, m.Pubsub, validateLoop, settlement.EventQueryNewSettlementBatchFinalized, m.onNewStateUpdateFinalized, m.logger)

	// Subscribe to P2P received blocks events (used for P2P syncing).
	go uevent.MustSubscribe(ctx, m.Pubsub, p2pGossipLoop, p2p.EventQueryNewGossipedBlock, m.OnReceivedBlock, m.logger)
	go uevent.MustSubscribe(ctx, m.Pubsub, p2pBlocksyncLoop, p2p.EventQueryNewBlockSyncBlock, m.OnReceivedBlock, m.logger)
}
