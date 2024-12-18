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

func (m *Manager) runAsFullNode(ctx context.Context, eg *errgroup.Group) error {
	m.RunMode = RunModeFullNode

	err := m.updateLastFinalizedHeightFromSettlement()
	if err != nil {
		return fmt.Errorf("sync block manager from settlement: %w", err)
	}

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SettlementValidateLoop(ctx)
	})

	m.subscribeFullNodeEvents(ctx)

	return types.DeleteInstructionFromDisk(m.RootDir)
}

func (m *Manager) runAsProposer(ctx context.Context, eg *errgroup.Group) error {
	m.RunMode = RunModeProposer

	go uevent.MustSubscribe(ctx, m.Pubsub, "updateSubmittedHeightLoop", settlement.EventQueryNewSettlementBatchAccepted, m.UpdateLastSubmittedHeight, m.logger)

	go uevent.MustSubscribe(ctx, m.Pubsub, p2pBlocksyncLoop, p2p.EventQueryNewBlockSyncBlock, m.OnReceivedBlock, m.logger) // why proposer subscribed?

	m.DAClient.WaitForSyncing()

	m.waitForSettlementSyncing()

	amIProposerOnSL, err := m.AmIProposerOnSL()
	if err != nil {
		return fmt.Errorf("am i proposer on SL: %w", err)
	}
	if !amIProposerOnSL {
		return fmt.Errorf("the node is no longer the proposer. please restart.") // no punctuation allowed
	}

	err = m.UpdateProposerFromSL() // ???????? we know it from previous lines
	if err != nil {
		return err
	}

	err = m.doForkWhenNewRevision()
	if err != nil {
		return err
	}

	shouldRotate, err := m.ShouldRotate() // third time checking am I proposer here
	if err != nil {
		return fmt.Errorf("checking should rotate: %w", err)
	}
	if shouldRotate {
		m.rotate(ctx)
	}

	bytesProducedC := make(chan int)

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.SubmitLoop(ctx, bytesProducedC)
	})

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		bytesProducedC <- m.GetUnsubmittedBytes()
		return m.ProduceBlockLoop(ctx, bytesProducedC)
	})

	uerrors.ErrGroupGoLog(eg, m.logger, func() error {
		return m.MonitorProposerRotation(ctx)
	})

	go func() {
		err = eg.Wait()

		if errors.Is(err, errRotationRequested) {
			m.rotate(ctx)
		} else if err != nil {
			m.freezeNode(err)
		}
	}()

	return nil
}

func (m *Manager) subscribeFullNodeEvents(ctx context.Context) {
	go uevent.MustSubscribe(ctx, m.Pubsub, syncLoop, settlement.EventQueryNewSettlementBatchAccepted, m.onNewStateUpdate, m.logger)
	go uevent.MustSubscribe(ctx, m.Pubsub, validateLoop, settlement.EventQueryNewSettlementBatchFinalized, m.onNewStateUpdateFinalized, m.logger)

	go uevent.MustSubscribe(ctx, m.Pubsub, p2pGossipLoop, p2p.EventQueryNewGossipedBlock, m.OnReceivedBlock, m.logger)
	go uevent.MustSubscribe(ctx, m.Pubsub, p2pBlocksyncLoop, p2p.EventQueryNewBlockSyncBlock, m.OnReceivedBlock, m.logger)
}
