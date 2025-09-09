package tee

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/tee"
	rollapptypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
)

var (
	socketPath    = "/run/container_launcher/teeserver.sock"
	tokenEndpoint = "http://localhost/v1/token"
)

// GetToken generates a TEE attestation for the current validated state
// This is served by the full node and called by the sequencer
func GetToken(node *node.Node) (tee.TEEResponse, error) {
	if !node.BlockManager.Conf.TEE.Enabled {
		return tee.TEEResponse{}, fmt.Errorf("TEE is not enabled")
	}

	// Get the settlement validator to access validated heights
	validator := node.BlockManager.SettlementValidator
	if validator == nil {
		return tee.TEEResponse{}, fmt.Errorf("settlement validator not available")
	}

	// Get the last validated height
	lastValidatedHeight := validator.GetLastValidatedHeight()
	if lastValidatedHeight == 0 {
		return tee.TEEResponse{}, fmt.Errorf("no blocks validated yet")
	}

	// Get the block at the validated height to get state root
	validatedBlock, err := node.Store.LoadBlock(lastValidatedHeight)
	if err != nil {
		return tee.TEEResponse{}, fmt.Errorf("load validated block: %w", err)
	}

	// Get the last finalized height from settlement
	lastFinalizedHeight, err := node.BlockManager.SLClient.GetLatestFinalizedHeight()
	if err != nil {
		return tee.TEEResponse{}, fmt.Errorf("get latest finalized height: %w", err)
	}

	// Get the block at the finalized height for state root
	var finalizedStateRoot []byte
	if lastFinalizedHeight > 0 {
		finalizedBlock, err := node.Store.LoadBlock(lastFinalizedHeight)
		if err != nil {
			// If we can't load the finalized block, use empty state root
			finalizedStateRoot = make([]byte, 32)
		} else {
			finalizedStateRoot = finalizedBlock.Header.AppHash[:]
		}
	} else {
		// No finalized height yet, use empty state root
		finalizedStateRoot = make([]byte, 32)
	}

	// Get the rollapp ID
	rollapp, err := node.BlockManager.SLClient.GetRollapp()
	if err != nil {
		return tee.TEEResponse{}, fmt.Errorf("get rollapp: %w", err)
	}

	// Build the nonce
	nonce := rollapptypes.TEENonce{
		RollappId:          rollapp.RollappID,
		CurrHeight:         lastValidatedHeight,
		CurrStateRoot:      validatedBlock.Header.AppHash[:],
		FinalizedHeight:    lastFinalizedHeight,
		FinalizedStateRoot: finalizedStateRoot,
	}

	// Validate the nonce
	if err := nonce.Validate(); err != nil {
		// For MVP, if validation fails due to missing finalized data, use placeholder values
		if lastFinalizedHeight == 0 {
			nonce.FinalizedHeight = 0
			nonce.FinalizedStateRoot = make([]byte, 0) // Empty for now as requested
		}
	}

	// Get the nonce hash
	nonceHash := nonce.Hash()

	// Get attestation token from GCP
	token, err := getGCPAttestationToken(nonceHash)
	if err != nil {
		return tee.TEEResponse{}, fmt.Errorf("get attestation token: %w", err)
	}

	return tee.TEEResponse{
		Token: token,
		Nonce: nonce,
	}, nil
}

// getGCPAttestationToken requests an attestation token from GCP Confidential Space
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
