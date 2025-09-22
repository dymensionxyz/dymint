package block

import (
	"context"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/tee"
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

	// Start the settlement validation loop in the background
	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SettlementValidateLoop(ctx)
	})

	m.subscribeFullNodeEvents(ctx)

	// remove instruction file after fork to avoid enter fork loop again
	return types.DeleteInstructionFromDisk(m.RootDir)
}

func (m *Manager) runAsProposer(ctx context.Context, eg *errgroup.Group) error {
	m.logger.Info("starting block manager", "mode", "proposer")
	// Subscribe to batch events, to update last submitted height in case batch confirmation was lost. This could happen if the sequencer crash/restarted just after submitting a batch to the settlement and by the time we query the last batch, this batch wasn't accepted yet.
	go uevent.MustSubscribe(ctx, m.Pubsub, "updateSubmittedHeightLoop", settlement.EventQueryNewSettlementBatchAccepted, m.UpdateLastSubmittedHeight, m.logger)
	// Subscribe to P2P received blocks events (used for P2P syncing).
	go uevent.MustSubscribe(ctx, m.Pubsub, p2pBlocksyncLoop, p2p.EventQueryNewBlockSyncBlock, m.OnReceivedBlock, m.logger)

	// Sequencer must wait till node is synced till last submittedHeight, in case it is not
	m.waitForSettlementSyncing()

	// it is checked again whether the node is the active proposer, since this could have changed after syncing.
	amIProposerOnSL, err := m.AmIProposerOnSL()
	if errors.Is(err, settlement.ErrProposerIsSentinel) {
		amIProposerOnSL = false
	} else if err != nil {
		return fmt.Errorf("am i proposer on SL: %w", err)
	}
	if !amIProposerOnSL {
		return fmt.Errorf("the node is no longer the proposer. please restart.")
	}

	// update l2 proposer from SL in case it changed after syncing
	err = m.UpdateProposerFromSL()
	if err != nil {
		return err
	}

	// doForkWhenNewRevision executes fork if necessary
	err = m.doForkWhenNewRevision()
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

	// Start TEE attestation client if enabled
	if m.Conf.TeeEnabled {
		teeClient := tee.NewTEEFinalizer(m.Conf, m.logger, m.SLClient)

		uerrors.ErrGroupGoLog(eg, m.logger, func() error {
			return teeClient.Start(ctx)
		})
	}

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
