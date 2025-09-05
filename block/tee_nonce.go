package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// CreateTEENonce creates a deterministic nonce string from the given parameters.
// This is a stub implementation using JSON. It will be replaced with protobuf
// or another deterministic serialization format in production.
func CreateTEENonce(chainID string, height uint64, blockHash string) string {
	// Stub implementation - will be replaced with sorted JSON or protobuf
	nonce := fmt.Sprintf(`{"chain_id":"%s","height":%d,"last_block_hash":"%s"}`, chainID, height, blockHash)
	return nonce
}

// HashTEENonce creates a SHA256 hash of the nonce and returns it as a hex string
func HashTEENonce(nonce string) string {
	hash := sha256.Sum256([]byte(nonce))
	return hex.EncodeToString(hash[:])
}

// ParseTEENonce parses a nonce string and extracts the chainID, height, and blockHash
func ParseTEENonce(nonceStr string) (chainID string, height uint64, blockHash string, err error) {
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