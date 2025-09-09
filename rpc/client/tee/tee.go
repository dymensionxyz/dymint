package tee

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/dymensionxyz/dymint/node"
)

// Response represents the TEE token response
type Response struct {
	Token string `json:"token"`
	Nonce string `json:"nonce"`
}

// GetToken retrieves a TEE attestation token
func GetToken(node *node.Node) (Response, error) {
	if !node.BlockManager.Conf.TEE.Enabled {
		return Response{}, fmt.Errorf("TEE is not enabled")
	}

	// For now, return an error as this endpoint is not fully implemented
	// The actual TEE attestation is handled by the TEEFinalizer in the block package
	return Response{}, fmt.Errorf("TEE RPC endpoint not implemented - attestation handled by TEEFinalizer")
}

var (
	socketPath    = "/run/container_launcher/teeserver.sock"
	tokenEndpoint = "http://localhost/v1/token"
)

// getGCPAttestationToken requests an attestation token from GCP
// This is kept for reference but actual implementation is in tee/tee.go
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