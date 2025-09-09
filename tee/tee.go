package tee

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

// TEEAttestationResponse represents the response from the TEE sidecar
type TEEAttestationResponse struct {
	Token string `json:"token"`
	Nonce string `json:"nonce"`
	Error string `json:"error,omitempty"`
}

// TEENonce represents the nonce data that gets signed by the TEE
type TEENonce struct {
	RollappID          string
	CurrHeight         uint64
	CurrStateRoot      []byte
	FinalizedHeight    uint64
	FinalizedStateRoot []byte
}

// Hash generates a hash of the nonce for attestation
func (n TEENonce) Hash() string {
	bz := []byte(n.RollappID)

	bzIx := make([]byte, 8)
	binary.BigEndian.PutUint64(bzIx, n.FinalizedHeight)
	bz = append(bz, bzIx...)
	bz = append(bz, n.FinalizedStateRoot...)

	bzIx = make([]byte, 8)
	binary.BigEndian.PutUint64(bzIx, n.CurrHeight)
	bz = append(bz, bzIx...)
	bz = append(bz, n.CurrStateRoot...)

	hash := sha256.Sum256(bz)
	return hex.EncodeToString(hash[:])
}

// TEEFinalizer handles fast finalization using TEE attestations
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

// fetchAndSubmitAttestation fetches attestation from sidecar and submits to hub
func (f *TEEFinalizer) fetchAndSubmitAttestation() error {
	// Get the latest validated height from the sidecar
	// For now, we'll get this from the settlement client
	latestHeight, err := f.slClient.GetLatestHeight()
	if err != nil {
		return fmt.Errorf("get latest height: %w", err)
	}

	if latestHeight == 0 {
		f.logger.Debug("No blocks to attest yet")
		return nil
	}

	// Get the latest batch to get state root
	latestBatch, err := f.slClient.GetLatestBatch()
	if err != nil {
		return fmt.Errorf("get latest batch: %w", err)
	}

	if latestBatch == nil || latestBatch.Batch == nil {
		f.logger.Debug("No batches found on hub")
		return nil
	}

	// Create nonce with current state
	// For now, leave finalized fields empty as requested
	nonce := TEENonce{
		RollappID:          f.rollappID,
		CurrHeight:         latestBatch.EndHeight,
		CurrStateRoot:      make([]byte, 32), // TODO: Get actual state root when available
		FinalizedHeight:    0,                 // Empty for now
		FinalizedStateRoot: make([]byte, 0),   // Empty for now
	}

	// Get attestation from sidecar
	attestation, err := f.getAttestationFromSidecar(nonce)
	if err != nil {
		return fmt.Errorf("get attestation from sidecar: %w", err)
	}

	if attestation.Error != "" {
		return fmt.Errorf("sidecar returned error: %s", attestation.Error)
	}

	if attestation.Token == "" {
		return fmt.Errorf("empty attestation token")
	}

	f.logger.Info("Got TEE attestation", 
		"nonce_hash", nonce.Hash(),
		"curr_height", nonce.CurrHeight,
		"token_length", len(attestation.Token))

	// Submit to hub
	err = f.slClient.SubmitTEEAttestation(
		attestation.Token,
		nonce,
		latestBatch.StateIndex,
	)
	if err != nil {
		return fmt.Errorf("submit attestation to hub: %w", err)
	}

	f.logger.Info("Successfully submitted TEE attestation",
		"state_index", latestBatch.StateIndex,
		"height", nonce.CurrHeight)

	return nil
}

// getAttestationFromSidecar fetches attestation from TEE sidecar
func (f *TEEFinalizer) getAttestationFromSidecar(nonce TEENonce) (*TEEAttestationResponse, error) {
	nonceHash := nonce.Hash()
	url := fmt.Sprintf("%s/getToken?nonce=%s", f.config.SidecarURL, nonceHash)

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
		return nil, fmt.Errorf("sidecar returned status %d: %s", resp.StatusCode, string(body))
	}

	var attestation TEEAttestationResponse
	if err := json.Unmarshal(body, &attestation); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Set the nonce in the response for tracking
	attestation.Nonce = nonceHash

	return &attestation, nil
}