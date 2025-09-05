package json

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
)

// TEEAttestation represents the attestation response structure
type TEEAttestation struct {
	Token string `json:"token"`
	Nonce string `json:"nonce"`
	Error string `json:"error,omitempty"`
}

// TEENonce represents the nonce data that gets signed
type TEENonce struct {
	ChainID       string `json:"chain_id"`
	Height        uint64 `json:"height"`
	LastBlockHash string `json:"last_block_hash"`
}

// MarshalJSONSorted returns deterministic sorted JSON
func (n TEENonce) MarshalJSONSorted() ([]byte, error) {
	// Create a map with sorted keys for deterministic ordering
	m := map[string]interface{}{
		"chain_id":        n.ChainID,
		"height":          n.Height,
		"last_block_hash": n.LastBlockHash,
	}
	
	// Get sorted keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	// Build JSON manually with sorted keys
	var parts []string
	for _, k := range keys {
		val, _ := json.Marshal(m[k])
		parts = append(parts, fmt.Sprintf(`"%s":%s`, k, val))
	}
	
	return []byte("{" + strings.Join(parts, ",") + "}"), nil
}

// handleTEEAttestation handles TEE attestation requests
func (h *handler) handleTEEAttestation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the client from the handler's service
	client := h.srv.client
	if client == nil {
		json.NewEncoder(w).Encode(TEEAttestation{
			Error: "Client not initialized",
		})
		return
	}

	// Access the block manager through the client's node
	node := client.GetNode()
	if node == nil || node.BlockManager == nil {
		json.NewEncoder(w).Encode(TEEAttestation{
			Error: "Block manager not available",
		})
		return
	}

	// Get the settlement validator
	validator := node.BlockManager.SettlementValidator
	if validator == nil {
		json.NewEncoder(w).Encode(TEEAttestation{
			Error: "Settlement validator not available",
		})
		return
	}

	// Get the last validated height
	lastValidatedHeight := validator.GetLastValidatedHeight()
	if lastValidatedHeight == 0 {
		json.NewEncoder(w).Encode(TEEAttestation{
			Error: "No blocks validated yet",
		})
		return
	}

	// Get the last validated block hash
	lastBlockHash, err := validator.GetLastValidatedBlockHash()
	if err != nil {
		json.NewEncoder(w).Encode(TEEAttestation{
			Error: fmt.Sprintf("Get last block hash: %v", err),
		})
		return
	}

	// Get chain ID from state
	chainID := node.BlockManager.State.ChainID
	
	// Create the nonce
	nonce := TEENonce{
		ChainID:       chainID,
		Height:        lastValidatedHeight,
		LastBlockHash: hex.EncodeToString(lastBlockHash),
	}

	// Serialize nonce to sorted JSON for deterministic output
	nonceBytes, err := nonce.MarshalJSONSorted()
	if err != nil {
		json.NewEncoder(w).Encode(TEEAttestation{
			Error: fmt.Sprintf("Marshal nonce: %v", err),
		})
		return
	}

	// Calculate hash of nonce - this will be the nonce sent to GCP
	// GCP requires nonce to be 10-74 bytes, our hash is 32 bytes which fits
	nonceHash := sha256.Sum256(nonceBytes)
	nonceHashHex := hex.EncodeToString(nonceHash[:])

	// Call GCP TEE attestation service to get attestation token
	token, err := getGCPAttestationToken(nonceHashHex)
	if err != nil {
		json.NewEncoder(w).Encode(TEEAttestation{
			Error: fmt.Sprintf("Get attestation token: %v", err),
		})
		return
	}

	// Return the attestation
	json.NewEncoder(w).Encode(TEEAttestation{
		Token: token,
		Nonce: string(nonceBytes),
	})
}

var (
	socketPath    = "/run/container_launcher/teeserver.sock"
	tokenEndpoint = "http://localhost/v1/token"
)

// getGCPAttestationToken retrieves an attestation token from GCP TEE attestation service
func getGCPAttestationToken(nonceHex string) (string, error) {
	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	// Create request body - using PKI token type for confidential space
	audience := "dymension"
	body := fmt.Sprintf(`{
		"audience": "%s",
		"nonces": ["%s"],
		"token_type": "PKI"
	}`, audience, nonceHex)

	resp, err := httpClient.Post(tokenEndpoint, "application/json", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("request token: %w", err)
	}
	defer resp.Body.Close()

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("attestation service returned status %d: %s", resp.StatusCode, string(tokenBytes))
	}

	return string(tokenBytes), nil
}