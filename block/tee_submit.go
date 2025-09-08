package block

import (
	_ "embed"
	"fmt"
	"sync"
	"time"
)

//go:embed assets/gcp_confidential_space_root.pem
var gcpConfidentialSpaceRootCert []byte

// TEESubmissionState tracks the state needed for TEE attestation submission
type TEESubmissionState struct {
	mu sync.RWMutex
	
	// Last time we successfully submitted a TEE attestation
	lastSubmissionTime time.Time
	
	// Last height we attempted to fast-finalize
	lastAttemptedHeight uint64
	
	// Cache of the last finalized height from hub
	lastFinalizedHeight uint64
	lastFinalizedHeightTime time.Time
}

// shouldSubmitTEEAttestation determines if we should submit a TEE attestation
// based on the validated height from the TEE full node
func (m *Manager) shouldSubmitTEEAttestation(validatedHeight uint64) (bool, uint64, error) {
	if m.teeState == nil {
		m.teeState = &TEESubmissionState{}
	}
	
	m.teeState.mu.Lock()
	defer m.teeState.mu.Unlock()
	
	// Get the latest finalized height from hub (with 30 second cache)
	if time.Since(m.teeState.lastFinalizedHeightTime) > 30*time.Second {
		finalizedHeight, err := m.SLClient.GetLatestFinalizedHeight()
		if err != nil {
			return false, 0, fmt.Errorf("get latest finalized height: %w", err)
		}
		m.teeState.lastFinalizedHeight = finalizedHeight
		m.teeState.lastFinalizedHeightTime = time.Now()
	}
	
	// If the validated height is not beyond what's already finalized, nothing to do
	if validatedHeight <= m.teeState.lastFinalizedHeight {
		m.logger.Debug("TEE validated height not beyond finalized",
			"validated", validatedHeight,
			"finalized", m.teeState.lastFinalizedHeight)
		return false, 0, nil
	}
	
	// Get the latest batch to understand current state
	latestBatch, err := m.SLClient.GetLatestBatch()
	if err != nil {
		return false, 0, fmt.Errorf("get latest batch: %w", err)
	}
	
	if latestBatch == nil || latestBatch.Batch == nil {
		m.logger.Debug("No batches found on hub")
		return false, 0, nil
	}
	
	// Find which state updates can be finalized
	// We need to find the highest state index whose EndHeight <= validatedHeight
	var targetStateIndex uint64
	var targetEndHeight uint64
	
	// Start from the latest batch and work backwards to find pending batches
	for stateIdx := latestBatch.StateIndex; stateIdx > 0; stateIdx-- {
		batch, err := m.SLClient.GetBatchAtIndex(stateIdx)
		if err != nil {
			// If we can't get a batch, skip it
			continue
		}
		
		if batch == nil || batch.Batch == nil {
			continue
		}
		
		// If this batch is already finalized, we can stop looking
		if batch.EndHeight <= m.teeState.lastFinalizedHeight {
			break
		}
		
		// If the validated height covers this entire batch, it can be finalized
		if batch.EndHeight <= validatedHeight {
			if batch.EndHeight > targetEndHeight {
				targetStateIndex = stateIdx
				targetEndHeight = batch.EndHeight
			}
		}
	}
	
	// No complete state update can be finalized
	if targetStateIndex == 0 {
		m.logger.Debug("No complete state updates can be finalized",
			"validated_height", validatedHeight,
			"latest_batch_end", latestBatch.EndHeight)
		return false, 0, nil
	}
	
	// Don't resubmit if we recently tried this height
	if targetEndHeight == m.teeState.lastAttemptedHeight && 
	   time.Since(m.teeState.lastSubmissionTime) < 1*time.Minute {
		m.logger.Debug("Skipping TEE submission, recently attempted",
			"height", targetEndHeight)
		return false, 0, nil
	}
	
	m.logger.Info("TEE attestation should be submitted",
		"validated_height", validatedHeight,
		"target_state_index", targetStateIndex,
		"target_end_height", targetEndHeight,
		"last_finalized", m.teeState.lastFinalizedHeight)
	
	return true, targetStateIndex, nil
}

func (m *Manager) submitTEEAttestation(attestation *TEEAttestationResponse) error {
	if attestation == nil {
		return fmt.Errorf("attestation is nil")
	}
	
	if attestation.Token == "" {
		return fmt.Errorf("attestation token is empty")
	}
	
	chainID, validatedHeight, blockHash, err := ParseTEENonce(attestation.Nonce)
	if err != nil {
		return fmt.Errorf("parse nonce: %w", err)
	}
	
	// Check if we should submit based on the validated height
	shouldSubmit, targetStateIndex, err := m.shouldSubmitTEEAttestation(validatedHeight)
	if err != nil {
		return fmt.Errorf("check should submit: %w", err)
	}
	
	if !shouldSubmit {
		m.logger.Debug("Skipping TEE submission, no actionable state updates",
			"validated_height", validatedHeight)
		return nil
	}
	
	m.logger.Info("Submitting TEE attestation",
		"validated_height", validatedHeight,
		"target_state_index", targetStateIndex,
		"chain_id", chainID,
		"block_hash", blockHash,
		"token_length", len(attestation.Token),
	)
	
	// Update state before submission
	if m.teeState != nil {
		m.teeState.mu.Lock()
		m.teeState.lastSubmissionTime = time.Now()
		m.teeState.lastAttemptedHeight = validatedHeight
		m.teeState.mu.Unlock()
	}
	
	err = m.submitTEEAttestationToHub(
		targetStateIndex,
		validatedHeight,
		attestation.Token,
		gcpConfidentialSpaceRootCert,
		attestation.Nonce,
	)
	if err != nil {
		return fmt.Errorf("submit TEE attestation to hub: %w", err)
	}
	
	m.logger.Info("TEE attestation submitted successfully",
		"state_index", targetStateIndex,
		"validated_height", validatedHeight,
	)
	
	return nil
}

func (m *Manager) submitTEEAttestationToHub(stateIndex uint64, validatedHeight uint64, token string, pemCert []byte, nonce string) error {
	// When the hub API is ready, this will call:
	// MsgFastFinalizeWithTEE with:
	//   - creator: sequencer address
	//   - rollapp_id: m.State.ChainID
	//   - state_index: stateIndex (the highest state update that can be finalized)
	//   - attestation_token: token (JWT from GCP)
	//   - pem_cert: pemCert (GCP root cert for validation)
	//   - nonce: nonce (contains validated_height and other data that was signed)
	//
	// The hub will:
	//   1. Verify the JWT signature using the PEM cert
	//   2. Extract and verify the nonce matches what was signed
	//   3. Check validated_height covers the state update at state_index
	//   4. Fast-finalize all state updates up to state_index
	
	m.logger.Debug("TEE attestation submission simulated (hub API not yet implemented)",
		"rollapp_id", m.State.ChainID,
		"state_index", stateIndex,
		"validated_height", validatedHeight,
		"token_length", len(token),
		"pem_cert_length", len(pemCert),
		"nonce_length", len(nonce),
	)
	
	// TODO: Actual implementation will call hub TX submission
	// For now, simulate success
	return nil
}