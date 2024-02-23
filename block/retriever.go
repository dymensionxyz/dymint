package block

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"

	"code.cloudfoundry.org/go-diodes"
	"github.com/celestiaorg/nmt"
	"github.com/dymensionxyz/dymint/da"
	"github.com/tendermint/tendermint/abci/types"
)

// RetriveLoop listens for new sync messages written to a ring buffer and in turn
// runs syncUntilTarget on the latest message in the ring buffer.
func (m *Manager) RetrieveLoop(ctx context.Context) {
	m.logger.Info("Started retrieve loop")
	syncTargetpoller := diodes.NewPoller(m.syncTargetDiode, diodes.WithPollingContext(ctx))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Get only the latest sync target
			syncTarget := syncTargetpoller.Next()
			err := m.syncUntilTarget(ctx, *(*uint64)(syncTarget))
			if err != nil {
				panic(err)
			}
			// Check if after we sync we are synced or a new syncTarget was already set.
			// If we are synced then signal all goroutines waiting on isSyncedCond.
			if m.store.Height() >= atomic.LoadUint64(&m.syncTarget) {
				m.logger.Info("Synced at height", "height", m.store.Height())
				m.isSyncedCond.L.Lock()
				m.isSyncedCond.Signal()
				m.isSyncedCond.L.Unlock()
			}
		}
	}
}

// syncUntilTarget syncs the block until the syncTarget is reached.
// It fetches the batches from the settlement, gets the DA height and gets
// the actual blocks from the DA.
func (m *Manager) syncUntilTarget(ctx context.Context, syncTarget uint64) error {
	currentHeight := m.store.Height()
	for currentHeight < syncTarget {
		currStateIdx := atomic.LoadUint64(&m.lastState.SLStateIndex) + 1
		m.logger.Info("Syncing until target", "height", currentHeight, "state_index", currStateIdx, "syncTarget", syncTarget)
		settlementBatch, err := m.settlementClient.RetrieveBatch(currStateIdx)
		if err != nil {
			return err
		}

		err = m.processNextDABatch(ctx, settlementBatch.MetaData.DA)
		if err != nil {
			return err
		}

		currentHeight = m.store.Height()

		err = m.updateStateIndex(settlementBatch.StateIndex)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) updateStateIndex(stateIndex uint64) error {
	atomic.StoreUint64(&m.lastState.SLStateIndex, stateIndex)
	_, err := m.store.UpdateState(m.lastState, nil)
	if err != nil {
		m.logger.Error("Failed to update state", "error", err)
		return err
	}
	return nil
}

func (m *Manager) processNextDABatch(ctx context.Context, daMetaData *da.DASubmitMetaData) error {
	m.logger.Debug("trying to retrieve batch from DA", "daHeight", daMetaData.Height)
	batchResp := m.fetchBatch(daMetaData)
	if batchResp.Code != da.StatusSuccess {
		return batchResp.Error
	}

	m.logger.Debug("retrieved batches", "n", len(batchResp.Batches), "daHeight", daMetaData.Height)

	for _, batch := range batchResp.Batches {
		for i, block := range batch.Blocks {
			err := m.applyBlock(ctx, block, batch.Commits[i], blockMetaData{source: daBlock, daHeight: daMetaData.Height})
			if err != nil {
				return err
			}
		}
	}
	err := m.attemptApplyCachedBlocks(ctx)
	if err != nil {
		m.logger.Debug("Error applying previous cached blocks", "err", err)
	}
	return nil
}

func (m *Manager) fetchBatch(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	// Check batch availability
	availabilityRes := m.retriever.CheckBatchAvailability(daMetaData)
	if availabilityRes.Code != da.StatusSuccess {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("Error fetching batch: %s", availabilityRes.Message),
				Error:   availabilityRes.Error,
			},
		}
	}
	m.lastBatch = &availabilityRes
	//batchRes.MetaData includes proofs necessary to open disputes with the Hub
	batchRes := m.retriever.RetrieveBatches(daMetaData)
	//TODO(srene) : for invalid transactions there is no specific error code since it will need to be validated somewhere else for fraud proving.
	//NMT proofs (availRes.MetaData.Proofs) are included in the result batchRes, necessary to be included in the dispute
	return batchRes
}

func (m *Manager) fraudProofPublishLoop(ctx context.Context) {
	for {
		select {
		case fraudProof := <-m.GetFraudProofOutChan():
			// Open a new file for writing only
			if m.dalc.GetClientType() == da.Celestia {
				m.addInclusionProofToFraudProof(fraudProof)
			}
			file, err := os.Create("fraudProof_rollapp_with_tx_and_inclusionproof.json")
			if err != nil {
				return
			}
			defer file.Close()

			// Serialize the struct to JSON and write it to the file
			jsonEncoder := json.NewEncoder(file)
			err = jsonEncoder.Encode(fraudProof)
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *Manager) addInclusionProofToFraudProof(fraudProof *types.FraudProof) error {

	prover := m.dalc.(da.ProofsRetriever)
	b, nmtRoots, rowProofs, err := prover.GetInclusionProofsCommitment(m.lastBatch.CheckMetaData.Height, m.lastBatch.CheckMetaData.Proofs[0], m.lastBatch.CheckMetaData.Commitment)
	if err != nil {
		return err
	}
	blobJSON, err := b.MarshalJSON()
	if err != nil {
		return err
	}
	fraudProof.InclusionProof.Blob = blobJSON

	nmtProofs := []*nmt.Proof(*m.lastBatch.CheckMetaData.Proofs[0])

	var proofs [][]byte

	for _, proof := range nmtProofs {
		proofJSON, err := proof.MarshalJSON()
		if err != nil {
			return err
		}
		proofs = append(proofs, proofJSON)
	}
	fraudProof.InclusionProof.Nmtproofs = proofs
	fraudProof.InclusionProof.Nmtroots = nmtRoots
	var rproofs [][]byte

	for _, rproof := range rowProofs {
		pbProof, err := rproof.ToProto().Marshal()
		if err != nil {
			return err
		}
		rproofs = append(rproofs, pbProof)

	}
	fraudProof.InclusionProof.Rproofs = rproofs
	fraudProof.InclusionProof.Dataroot = m.lastBatch.CheckMetaData.Root

	return nil
}
