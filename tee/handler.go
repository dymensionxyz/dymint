package tee

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
	
	"github.com/dymensionxyz/dymint/node"
)

type TEEAttestation struct {
	Token string `json:"token"`
	Nonce string `json:"nonce"`
	Error string `json:"error,omitempty"`
}

type TEENonce struct {
	ChainID       string `json:"chain_id"`
	Height        uint64 `json:"height"`
	LastBlockHash string `json:"last_block_hash"`
}

func (n TEENonce) MarshalJSONSorted() ([]byte, error) {
	m := map[string]interface{}{
		"chain_id":        n.ChainID,
		"height":          n.Height,
		"last_block_hash": n.LastBlockHash,
	}
	
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	var parts []string
	for _, k := range keys {
		val, _ := json.Marshal(m[k])
		parts = append(parts, fmt.Sprintf(`"%s":%s`, k, val))
	}
	
	return []byte("{" + strings.Join(parts, ",") + "}"), nil
}

func HandleTEEAttestation(node *node.Node) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if node == nil || node.BlockManager == nil {
			json.NewEncoder(w).Encode(TEEAttestation{
				Error: "Block manager not available",
			})
			return
		}

		validator := node.BlockManager.SettlementValidator
		if validator == nil {
			json.NewEncoder(w).Encode(TEEAttestation{
				Error: "Settlement validator not available",
			})
			return
		}

		lastValidatedHeight := validator.GetLastValidatedHeight()
		if lastValidatedHeight == 0 {
			json.NewEncoder(w).Encode(TEEAttestation{
				Error: "No blocks validated yet",
			})
			return
		}

		lastBlockHash, err := validator.GetLastValidatedBlockHash()
		if err != nil {
			json.NewEncoder(w).Encode(TEEAttestation{
				Error: fmt.Sprintf("Get last block hash: %v", err),
			})
			return
		}

		chainID := node.BlockManager.State.ChainID
		
		nonce := TEENonce{
			ChainID:       chainID,
			Height:        lastValidatedHeight,
			LastBlockHash: hex.EncodeToString(lastBlockHash),
		}

		nonceBytes, err := nonce.MarshalJSONSorted()
		if err != nil {
			json.NewEncoder(w).Encode(TEEAttestation{
				Error: fmt.Sprintf("Marshal nonce: %v", err),
			})
			return
		}

		nonceHash := sha256.Sum256(nonceBytes)
		nonceHashHex := hex.EncodeToString(nonceHash[:])

		token, err := getGCPAttestationToken(nonceHashHex)
		if err != nil {
			json.NewEncoder(w).Encode(TEEAttestation{
				Error: fmt.Sprintf("Get attestation token: %v", err),
			})
			return
		}

		json.NewEncoder(w).Encode(TEEAttestation{
			Token: token,
			Nonce: string(nonceBytes),
		})
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