package block

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	uerrors "github.com/dymensionxyz/dymint/utils/errors"
	uevent "github.com/dymensionxyz/dymint/utils/event"
	"golang.org/x/sync/errgroup"
)

// setFraudHandler sets the fraud handler for the block manager.
func (m *Manager) runAsFullNode(ctx context.Context, eg *errgroup.Group) error {
	// update latest finalized height
	err := m.updateLastFinalizedHeightFromSettlement()
	if err != nil {
		return fmt.Errorf("sync block manager from settlement: %w", err)
	}

	// Start the settlement validation loop in the background
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SettlementValidateLoop(ctx)
	})

	// Subscribe to new (or finalized) state updates events.
	go uevent.MustSubscribe(ctx, m.Pubsub, "syncLoop", settlement.EventQueryNewSettlementBatchAccepted, m.onNewStateUpdate, m.logger)
	go uevent.MustSubscribe(ctx, m.Pubsub, "validateLoop", settlement.EventQueryNewSettlementBatchFinalized, m.onNewStateUpdateFinalized, m.logger)

	// Subscribe to P2P received blocks events (used for P2P syncing).
	go uevent.MustSubscribe(ctx, m.Pubsub, "applyGossipedBlocksLoop", p2p.EventQueryNewGossipedBlock, m.OnReceivedBlock, m.logger)
	go uevent.MustSubscribe(ctx, m.Pubsub, "applyBlockSyncBlocksLoop", p2p.EventQueryNewBlockSyncBlock, m.OnReceivedBlock, m.logger)

	return nil
}

func (m *Manager) runAsProposer(ctx context.Context, eg *errgroup.Group) error {
	// Subscribe to batch events, to update last submitted height in case batch confirmation was lost. This could happen if the sequencer crash/restarted just after submitting a batch to the settlement and by the time we query the last batch, this batch wasn't accepted yet.
	go uevent.MustSubscribe(ctx, m.Pubsub, "updateSubmittedHeightLoop", settlement.EventQueryNewSettlementBatchAccepted, m.UpdateLastSubmittedHeight, m.logger)

	// Sequencer must wait till the DA light client is synced. Otherwise it will fail when submitting blocks.
	// Full-nodes does not need to wait, but if it tries to fetch blocks from DA heights previous to the DA light client height it will fail, and it will retry till it reaches the height.
	m.DAClient.WaitForSyncing()

	// Sequencer must wait till node is synced till last submittedHeight, in case it is not
	m.waitForSettlementSyncing()

	// check if sequencer in the middle of rotation
	nextSeqAddr, missing, err := m.MissingLastBatch()
	if err != nil {
		return fmt.Errorf("checking if missing last batch: %w", err)
	}
	// if sequencer is in the middle of rotation, complete rotation instead of running the main loop
	if missing {
		m.handleRotationReq(ctx, nextSeqAddr)
		return nil
	}

	// populate the bytes produced channel
	bytesProducedC := make(chan int)

	// channel to signal sequencer rotation started
	rotateSequencerC := make(chan string, 1)

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SubmitLoop(ctx, bytesProducedC)
	})
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		bytesProducedC <- m.GetUnsubmittedBytes() // load unsubmitted bytes from previous run
		return m.ProduceBlockLoop(ctx, bytesProducedC)
	})
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.MonitorSequencerRotation(ctx, rotateSequencerC)
	})

	go func() {
		_ = eg.Wait()
		// Check if exited due to sequencer rotation signal
		select {
		case nextSeqAddr := <-rotateSequencerC:
			m.handleRotationReq(ctx, nextSeqAddr)
		default:
			m.logger.Info("Block manager err group finished.")
		}
	}()

	return nil

}
