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
	Token string                  `json:"token"`
	Nonce rollapptypes.TEENonce   `json:"nonce"`
	Error string                  `json:"error,omitempty"`
}

// TEEFinalizer handles fast finalization using TEE attestations
// It runs on the sequencer and queries the full node for attestations
type TEEFinalizer struct {
	config    config.TEEConfig
	logger    types.Logger
	client    *http.Client
	slClient  settlement.ClientI
	rollappID string
}

// NewTEEFinalizer creates a new TEE finalizer
func NewTEEFinalizer(config config.TEEConfig, logger types.Logger, slClient settlement.ClientI, rollappID string) *TEEFinalizer {
	return &TEEFinalizer{
		config: config,
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		slClient:  slClient,
		rollappID: rollappID,
	}
}

// Start begins the TEE attestation loop
func (f *TEEFinalizer) Start(ctx context.Context) error {
	if !f.config.Enabled {
		f.logger.Info("TEE attestation disabled")
		return nil
	}

	ticker := time.NewTicker(f.config.AttestationInterval)
	defer ticker.Stop()

	f.logger.Info("Starting TEE attestation client", 
		"sidecar_url", f.config.SidecarURL, 
		"interval", f.config.AttestationInterval,
		"rollapp_id", f.rollappID)

	// Initial attempt
	if err := f.fetchAndSubmitAttestation(); err != nil {
		f.logger.Error("Initial attestation fetch error", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			f.logger.Info("Stopping TEE attestation client")
			return nil
		case <-ticker.C:
			if err := f.fetchAndSubmitAttestation(); err != nil {
				f.logger.Error("Attestation fetch error", "error", err)
				// Continue on error - wait for next interval
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

	if attestation.Error != "" {
		return fmt.Errorf("full node returned error: %s", attestation.Error)
	}

	if attestation.Token == "" {
		return fmt.Errorf("empty attestation token")
	}

	// Validate the nonce
	if err := attestation.Nonce.Validate(); err != nil {
		// For MVP, allow empty finalized fields
		f.logger.Debug("Nonce validation failed, using as-is for MVP", "error", err)
	}

	f.logger.Info("Got TEE attestation from full node", 
		"nonce_hash", attestation.Nonce.Hash(),
		"curr_height", attestation.Nonce.CurrHeight,
		"token_length", len(attestation.Token))

	// Get the latest batch to determine state index
	latestBatch, err := f.slClient.GetLatestBatch()
	if err != nil {
		return fmt.Errorf("get latest batch: %w", err)
	}

	if latestBatch == nil || latestBatch.Batch == nil {
		f.logger.Debug("No batches found on hub")
		return nil
	}

	// Check if the validated height is beyond what's already on the hub
	if attestation.Nonce.CurrHeight <= latestBatch.EndHeight {
		f.logger.Debug("TEE validated height not beyond latest batch",
			"validated", attestation.Nonce.CurrHeight,
			"latest_batch_end", latestBatch.EndHeight)
		return nil
	}

	// Submit to hub
	err = f.slClient.SubmitTEEAttestation(
		attestation.Token,
		attestation.Nonce,
		latestBatch.StateIndex, // Use current state index for curr_state_index
	)
	if err != nil {
		return fmt.Errorf("submit attestation to hub: %w", err)
	}

	f.logger.Info("Successfully submitted TEE attestation",
		"state_index", latestBatch.StateIndex,
		"height", attestation.Nonce.CurrHeight)

	return nil
}

// queryFullNodeTEE queries the full node's /tee endpoint
func (f *TEEFinalizer) queryFullNodeTEE() (*TEEResponse, error) {
	url := fmt.Sprintf("%s/tee", f.config.SidecarURL)

	resp, err := f.client.Get(url)
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