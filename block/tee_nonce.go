package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// TEENonceData represents the data that gets signed in the TEE attestation
type TEENonceData struct {
	ChainID           string `json:"chain_id"`
	ValidatedHeight   uint64 `json:"validated_height"`   // The height validated by the TEE full node
	LastBlockHash     string `json:"last_block_hash"`
}

// CreateTEENonce creates a deterministic nonce string from the given parameters.
// This is a stub implementation using JSON. It will be replaced with protobuf
// or another deterministic serialization format in production.
func CreateTEENonce(chainID string, validatedHeight uint64, blockHash string) string {
	nonce := TEENonceData{
		ChainID:         chainID,
		ValidatedHeight: validatedHeight,
		LastBlockHash:   blockHash,
	}
	
	// Use json.Marshal for consistent ordering
	nonceBytes, _ := json.Marshal(nonce)
	return string(nonceBytes)
}

// HashTEENonce creates a SHA256 hash of the nonce and returns it as a hex string
func HashTEENonce(nonce string) string {
	hash := sha256.Sum256([]byte(nonce))
	return hex.EncodeToString(hash[:])
}

// ParseTEENonce parses a nonce string and extracts the chainID, validated height, and blockHash
func ParseTEENonce(nonceStr string) (chainID string, validatedHeight uint64, blockHash string, err error) {
	var nonce TEENonceData
	
	if err := json.Unmarshal([]byte(nonceStr), &nonce); err != nil {
		return "", 0, "", fmt.Errorf("unmarshal nonce: %w", err)
	}
	
	return nonce.ChainID, nonce.ValidatedHeight, nonce.LastBlockHash, nil
}