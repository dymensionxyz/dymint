package json

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TEEAttestation represents the attestation response structure
type TEEAttestation struct {
	Token string `json:"token"`
	Nonce string `json:"nonce"`
	Error string `json:"error,omitempty"`
}

// TEENonce represents the nonce data that gets signed
type TEENonce struct {
	Height        uint64 `json:"height"`
	StateIndex    uint64 `json:"state_index"` // For now using height as placeholder
	ChainID       string `json:"chain_id"`
	LastBlockHash string `json:"last_block_hash"`
}

// handleTEEAttestation handles TEE attestation requests
func (h *handler) handleTEEAttestation(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the client from the handler's service
	client := h.srv.client
	if client == nil {
		http.Error(w, "Client not initialized", http.StatusInternalServerError)
		return
	}

	// Access the block manager through the client's node
	node := client.GetNode()
	if node == nil || node.BlockManager == nil {
		http.Error(w, "Block manager not available", http.StatusServiceUnavailable)
		return
	}

	// Get the settlement validator
	validator := node.BlockManager.SettlementValidator
	if validator == nil {
		http.Error(w, "Settlement validator not available", http.StatusServiceUnavailable)
		return
	}

	// Get the last validated height
	lastValidatedHeight := validator.GetLastValidatedHeight()
	if lastValidatedHeight == 0 {
		response := TEEAttestation{
			Error: "No blocks validated yet",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get the last validated block hash
	lastBlockHash, err := validator.GetLastValidatedBlockHash()
	if err != nil {
		response := TEEAttestation{
			Error: fmt.Sprintf("Failed to get last block hash: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get chain ID from state
	chainID := node.BlockManager.State.ChainID
	
	// For now, use height as state index (will need proper mapping later)
	stateIndex := lastValidatedHeight

	// Create the nonce
	nonce := TEENonce{
		Height:        lastValidatedHeight,
		StateIndex:    stateIndex,
		ChainID:       chainID,
		LastBlockHash: hex.EncodeToString(lastBlockHash),
	}

	// Serialize nonce to JSON for hashing
	nonceBytes, err := json.Marshal(nonce)
	if err != nil {
		response := TEEAttestation{
			Error: fmt.Sprintf("Failed to marshal nonce: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Calculate hash of nonce
	nonceHash := sha256.Sum256(nonceBytes)
	nonceHashHex := hex.EncodeToString(nonceHash[:])

	// Check if custom nonce was provided in query params
	customNonce := r.URL.Query().Get("nonce")
	if customNonce != "" {
		nonceHashHex = customNonce
	}

	// Call GCP metadata service to get attestation token
	token, err := getGCPAttestationToken(nonceHashHex)
	if err != nil {
		response := TEEAttestation{
			Error: fmt.Sprintf("Failed to get attestation token: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return the attestation
	response := TEEAttestation{
		Token: token,
		Nonce: string(nonceBytes),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getGCPAttestationToken retrieves an attestation token from GCP metadata service
func getGCPAttestationToken(nonce string) (string, error) {
	// GCP metadata service URL
	url := fmt.Sprintf("http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token?audience=https://sts.googleapis.com&format=full&licenses=FALSE")
	
	// Add nonce if provided
	if nonce != "" {
		url = fmt.Sprintf("%s&nonce=%s", url, nonce)
	}

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Required header for GCP metadata service
	req.Header.Set("Metadata-Flavor", "Google")

	// Make request with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get attestation token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GCP metadata service returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read the token
	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read attestation token: %w", err)
	}

	return string(tokenBytes), nil
}