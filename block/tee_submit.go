package block

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// gcpConfidentialSpaceRootCert is the GCP Confidential Space root certificate
// This certificate is used to verify the attestation tokens from GCP
// Source: https://cloud.google.com/confidential-computing/confidential-space/docs/reference/attestation-tokens#token-verification
//
//go:embed gcp_confidential_space_root.pem
var gcpConfidentialSpaceRootCert []byte

// submitTEEAttestation submits a TEE attestation to the hub
func (m *Manager) submitTEEAttestation(attestation *TEEAttestationResponse) error {
	if attestation == nil {
		return fmt.Errorf("attestation is nil")
	}
	
	if attestation.Token == "" {
		return fmt.Errorf("attestation token is empty")
	}
	
	// Parse the nonce to get the height information
	var nonce struct {
		ChainID       string `json:"chain_id"`
		Height        uint64 `json:"height"`
		LastBlockHash string `json:"last_block_hash"`
	}
	
	if err := json.Unmarshal([]byte(attestation.Nonce), &nonce); err != nil {
		return fmt.Errorf("unmarshal nonce: %w", err)
	}
	
	// Calculate the nonce hash to verify it matches what was sent to GCP
	nonceHash := sha256.Sum256([]byte(attestation.Nonce))
	nonceHashHex := hex.EncodeToString(nonceHash[:])
	
	m.logger.Info("Submitting TEE attestation",
		"height", nonce.Height,
		"chain_id", nonce.ChainID,
		"block_hash", nonce.LastBlockHash,
		"token_length", len(attestation.Token),
		"nonce_hash", nonceHashHex,
	)
	
	// Submit the attestation to the hub
	// The hub will verify the token signature using the embedded GCP root certificate
	// and immediately finalize the state update if valid
	err := m.submitTEEAttestationToHub(
		nonce.Height,
		attestation.Token,
		gcpConfidentialSpaceRootCert,
		attestation.Nonce,
	)
	if err != nil {
		return fmt.Errorf("submit TEE attestation to hub: %w", err)
	}
	
	m.logger.Info("TEE attestation submitted successfully",
		"height", nonce.Height,
	)
	
	return nil
}

// submitTEEAttestationToHub sends the TEE attestation to the hub for verification and immediate finalization
// This method will be replaced with the actual settlement client call once the hub API is implemented
func (m *Manager) submitTEEAttestationToHub(height uint64, token string, pemCert []byte, nonce string) error {
	// When the hub API is ready, this will call:
	// return m.SLClient.SubmitTEEAttestation(m.State.ChainID, height, token, pemCert, nonce)
	
	// For now, we simulate the submission for testing purposes
	// The hub will verify:
	// 1. The token signature using the provided PEM certificate
	// 2. The nonce matches the expected format and includes the correct height
	// 3. The attestation is from a valid TEE environment
	// If all checks pass, the state update at the given height is immediately finalized
	
	m.logger.Debug("TEE attestation submission simulated (hub API not yet implemented)",
		"rollapp_id", m.State.ChainID,
		"height", height,
	)
	
	return nil
}