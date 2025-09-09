package tee

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	rollapptypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
)

// TEEResponse represents the response from the full node's /tee endpoint
type TEEResponse struct {
	Token string                `json:"token"`
	Nonce rollapptypes.TEENonce `json:"nonce"`
}

// TEEFinalizer handles fast finalization using TEE attestations
// It runs on the sequencer and queries the full node for attestations
type TEEFinalizer struct {
	config        config.TEEConfig
	logger        types.Logger
	sidecarClient *http.Client
	hubClient     settlement.ClientI
	rollappID     string
}

func NewTEEFinalizer(config config.TEEConfig, logger types.Logger, slClient settlement.ClientI, rollappID string) *TEEFinalizer {
	return &TEEFinalizer{
		config: config,
		logger: logger,
		sidecarClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		hubClient: slClient,
		rollappID: rollappID,
	}
}

func (f *TEEFinalizer) Start(ctx context.Context) error {
	ticker := time.NewTicker(f.config.AttestationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			f.logger.Info("Stopping TEE attestation client")
			return nil
		case <-ticker.C:
			if err := f.fetchAndSubmitAttestation(); err != nil {
				f.logger.Error("Attestation fetch error", "error", err)
			}
		}
	}
}

// fetchAndSubmitAttestation fetches attestation from full node and submits to hub
func (f *TEEFinalizer) fetchAndSubmitAttestation() error {
	// Query the full node's /tee endpoint
	attestation, err := f.queryFullNodeTEE()
	if err != nil {
		return fmt.Errorf("query full node TEE: %w", err)
	}

	// TODO: check if attestation.Nonce.CurrHeight is greater than  settlement GetLatestFinalizedHeight

	var finalizedIx uint64
	var currIx uint64

	err = f.hubClient.SubmitTEEAttestation(
		attestation.Token,
		attestation.Nonce,
		finalizedIx,
		currIx,
	)
	if err != nil {
		return fmt.Errorf("submit attestation to hub: %w", err)
	}

	return nil
}

func (f *TEEFinalizer) queryFullNodeTEE() (*TEEResponse, error) {
	url := fmt.Sprintf("%s/tee", f.config.SidecarURL)

	resp, err := f.sidecarClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request attestation: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("full node returned status %d: %s", resp.StatusCode, string(body))
	}

	var attestation TEEResponse
	if err := json.Unmarshal(body, &attestation); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &attestation, nil
}
