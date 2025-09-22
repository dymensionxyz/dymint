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

const (
	// GPC defined
	socketPath    = "/run/container_launcher/teeserver.sock"
	tokenEndpoint = "http://localhost/v1/token"
)

func GetToken(node *node.Node, dry bool) (tee.TEEResponse, error) {
	if !node.BlockManager.Conf.TeeEnabled {
		return tee.TEEResponse{}, fmt.Errorf("TEE is not enabled")
	}

	validator := node.BlockManager.SettlementValidator
	if validator == nil {
		return tee.TEEResponse{}, fmt.Errorf("settlement validator not available")
	}

	lastValidatedHeight := validator.GetLastValidatedHeight()
	if lastValidatedHeight == 0 {
		return tee.TEEResponse{}, fmt.Errorf("no blocks validated yet")
	}

	nonce := rollapptypes.TEENonce{
		RollappId:       node.BlockManager.State.ChainID,
		CurrHeight:      lastValidatedHeight,
		FinalizedHeight: validator.GetTrustedHeight(),
	}

	var token string
	if !dry {
		// enables checking things work without GCP
		var err error
		token, err = getGCPAttestationToken(nonce.Hash())
		if err != nil {
			return tee.TEEResponse{}, fmt.Errorf("get attestation token: %w", err)
		}
	}

	return tee.TEEResponse{
		Token: token,
		Nonce: nonce,
	}, nil
}

const audience = "dymension"

// getGCPAttestationToken requests an attestation token from GCP Confidential Space
func getGCPAttestationToken(nonceHex string) (string, error) {
	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	body := fmt.Sprintf(`{
		"audience": "%s",
		"nonces": ["%s"],
		"token_type": "PKI"
	}`, audience, nonceHex)

	resp, err := httpClient.Post(tokenEndpoint, "application/json", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("request token: %w", err)
	}
	//nolint:errcheck
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
