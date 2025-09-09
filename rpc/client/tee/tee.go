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

type Response struct {
	Token string `json:"token"`
	Nonce string `json:"nonce"`
}

func GetToken(node *node.Node) (Response, error) {
	if !node.BlockManager.Conf.TEE.Enabled {
		return Response{}, fmt.Errorf("TEE is not enabled")
	}

	validator := node.BlockManager.SettlementValidator
	if validator == nil {
		return Response{}, fmt.Errorf("Settlement validator not available")
	}

	lastValidatedHeight := validator.GetLastValidatedHeight()
	if lastValidatedHeight == 0 {
		return Response{}, fmt.Errorf("No blocks validated yet")
	}

	chainID := node.BlockManager.State.ChainID

	// TODO!! build correct nonce, pass to GCP to get token

	token, err := getGCPAttestationToken(nonceHash)
	if err != nil {
		return Response{}, fmt.Errorf("Get attestation token: %v", err)
	}

	return Response{Token: token, Nonce: nonce}, nil
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
