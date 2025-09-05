package block

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

//go:embed assets/gcp_confidential_space_root.pem
var gcpConfidentialSpaceRootCert []byte

func (m *Manager) submitTEEAttestation(attestation *TEEAttestationResponse) error {
	if attestation == nil {
		return fmt.Errorf("attestation is nil")
	}
	
	if attestation.Token == "" {
		return fmt.Errorf("attestation token is empty")
	}
	
	chainID, height, blockHash, err := parseNonce(attestation.Nonce)
	if err != nil {
		return fmt.Errorf("parse nonce: %w", err)
	}
	
	m.logger.Info("Submitting TEE attestation",
		"height", height,
		"chain_id", chainID,
		"block_hash", blockHash,
		"token_length", len(attestation.Token),
	)
	
	err = m.submitTEEAttestationToHub(
		height,
		attestation.Token,
		gcpConfidentialSpaceRootCert,
		attestation.Nonce,
	)
	if err != nil {
		return fmt.Errorf("submit TEE attestation to hub: %w", err)
	}
	
	m.logger.Info("TEE attestation submitted successfully",
		"height", height,
	)
	
	return nil
}

func parseNonce(nonceStr string) (chainID string, height uint64, blockHash string, err error) {
	var nonce struct {
		ChainID       string `json:"chain_id"`
		Height        uint64 `json:"height"`
		LastBlockHash string `json:"last_block_hash"`
	}
	
	if err := json.Unmarshal([]byte(nonceStr), &nonce); err != nil {
		return "", 0, "", fmt.Errorf("unmarshal nonce: %w", err)
	}
	
	return nonce.ChainID, nonce.Height, nonce.LastBlockHash, nil
}

func createNonceHash(chainID string, height uint64, blockHash string) string {
	// Stub implementation - will be replaced with sorted JSON or protobuf
	nonce := fmt.Sprintf(`{"chain_id":"%s","height":%d,"last_block_hash":"%s"}`, chainID, height, blockHash)
	hash := sha256.Sum256([]byte(nonce))
	return hex.EncodeToString(hash[:])
}

func (m *Manager) submitTEEAttestationToHub(height uint64, token string, pemCert []byte, nonce string) error {
	// When the hub API is ready, this will call:
	// return m.SLClient.SubmitTEEAttestation(m.State.ChainID, height, token, pemCert, nonce)
	
	m.logger.Debug("TEE attestation submission simulated (hub API not yet implemented)",
		"rollapp_id", m.State.ChainID,
		"height", height,
	)
	
	return nil
}