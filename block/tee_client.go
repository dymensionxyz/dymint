package block

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dymensionxyz/dymint/types"
)

// TEEClient handles communication with the TEE sidecar
type TEEClient struct {
	endpoint string        // URL of the TEE sidecar attestation endpoint
	interval time.Duration // How often to poll for attestations
	logger   types.Logger
	client   *http.Client
}

// TEEConfig holds configuration for the TEE client
type TEEConfig struct {
	Enabled  bool          `json:"enabled"`
	Endpoint string        `json:"endpoint"`
	Interval time.Duration `json:"interval"`
}

// TEEAttestationResponse represents the response from the TEE attestation endpoint
type TEEAttestationResponse struct {
	Token string `json:"token"`
	Nonce string `json:"nonce"`
	Error string `json:"error,omitempty"`
}

func NewTEEClient(config TEEConfig, logger types.Logger) *TEEClient {
	return &TEEClient{
		endpoint: config.Endpoint,
		interval: config.Interval,
		logger:   logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *TEEClient) Start(ctx context.Context, submitFunc func(attestation *TEEAttestationResponse) error) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	
	c.logger.Info("Starting TEE attestation client", "endpoint", c.endpoint, "interval", c.interval)
	
	if err := c.fetchAndSubmitAttestation(submitFunc); err != nil {
		c.logger.Error("Initial attestation fetch error", "error", err)
	}
	
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping TEE attestation client")
			return
		case <-ticker.C:
			if err := c.fetchAndSubmitAttestation(submitFunc); err != nil {
				c.logger.Error("Attestation fetch error", "error", err)
			}
		}
	}
}

func (c *TEEClient) fetchAndSubmitAttestation(submitFunc func(*TEEAttestationResponse) error) error {
	attestation, err := c.GetAttestation()
	if err != nil {
		return fmt.Errorf("get attestation: %w", err)
	}
	
	if attestation.Error != "" {
		return fmt.Errorf("attestation error: %s", attestation.Error)
	}
	
	if attestation.Token == "" {
		return fmt.Errorf("empty attestation token")
	}
	
	c.logger.Info("Got TEE attestation", "nonce_length", len(attestation.Nonce))
	
	if err := submitFunc(attestation); err != nil {
		return fmt.Errorf("submit attestation: %w", err)
	}
	
	c.logger.Info("Successfully submitted TEE attestation")
	return nil
}

func (c *TEEClient) GetAttestation() (*TEEAttestationResponse, error) {
	url := fmt.Sprintf("%s/tee/attestation", c.endpoint)
	
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request attestation: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TEE sidecar returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var attestation TEEAttestationResponse
	if err := json.Unmarshal(body, &attestation); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	
	return &attestation, nil
}