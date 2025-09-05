package tee

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	
	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/node"
)

type TEEAttestation struct {
	Token string `json:"token"`
	Nonce string `json:"nonce"`
	Error string `json:"error,omitempty"`
}

func HandleTEEAttestation(node *node.Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if node == nil || node.BlockManager == nil {
			writeErrorResponse(w, "Block manager not available")
			return
		}

		validator := node.BlockManager.SettlementValidator
		if validator == nil {
			writeErrorResponse(w, "Settlement validator not available")
			return
		}

		lastValidatedHeight := validator.GetLastValidatedHeight()
		if lastValidatedHeight == 0 {
			writeErrorResponse(w, "No blocks validated yet")
			return
		}

		lastBlockHash, err := validator.GetLastValidatedBlockHash()
		if err != nil {
			writeErrorResponse(w, fmt.Sprintf("Get last block hash: %v", err))
			return
		}

		chainID := node.BlockManager.State.ChainID
		blockHashHex := hex.EncodeToString(lastBlockHash)
		
		nonce := block.CreateTEENonce(chainID, lastValidatedHeight, blockHashHex)
		nonceHash := block.HashTEENonce(nonce)

		token, err := getGCPAttestationToken(nonceHash)
		if err != nil {
			writeErrorResponse(w, fmt.Sprintf("Get attestation token: %v", err))
			return
		}

		response := TEEAttestation{
			Token: token,
			Nonce: nonce,
		}
		
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func writeErrorResponse(w http.ResponseWriter, errorMsg string) {
	response := TEEAttestation{Error: errorMsg}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
	}
}

var (
	socketPath    = "/run/container_launcher/teeserver.sock"
	tokenEndpoint = "http://localhost/v1/token"
)

func getGCPAttestationToken(nonceHex string) (string, error) {
	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

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