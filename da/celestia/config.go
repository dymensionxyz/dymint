package celestia

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"

	openrpcns "github.com/celestiaorg/celestia-openrpc/types/namespace"
)

const (
	defaultRpcRetryDelay    = 3 * time.Second
	namespaceVersion        = 0
	DefaultGasPrices        = 0.1
	defaultRpcRetryAttempts = 5
	maxBlobSizeBytes        = 500000
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// Config stores Celestia DALC configuration parameters.
type Config struct {
	BaseURL        string               `json:"base_url,omitempty"`
	Timeout        time.Duration        `json:"timeout,omitempty"`
	GasPrices      float64              `json:"gas_prices,omitempty"`
	NamespaceIDStr string               `json:"namespace_id,omitempty"`
	AuthToken      string               `json:"auth_token,omitempty"`
	Backoff        uretry.BackoffConfig `json:"backoff,omitempty"`
	RetryAttempts  *int                 `json:"retry_attempts,omitempty"`
	RetryDelay     time.Duration        `json:"retry_delay,omitempty"`
	NamespaceID    openrpcns.Namespace  `json:"-"`
}

var TestConfig = Config{
	BaseURL:        "http://127.0.0.1:26658",
	Timeout:        5 * time.Second,
	GasPrices:      DefaultGasPrices,
	NamespaceIDStr: "",
	NamespaceID:    openrpcns.Namespace{Version: namespaceVersion, ID: []byte{0, 0, 0, 0, 0, 0, 0, 0, 255, 255}},
}

func generateRandNamespaceID() string {
	nID := make([]byte, 10)
	_, err := rand.Read(nID)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(nID)
}

func (c *Config) InitNamespaceID() error {
	if c.NamespaceIDStr == "" {
		c.NamespaceIDStr = generateRandNamespaceID()
	}
	// Decode NamespaceID from string to byte array
	namespaceBytes, err := hex.DecodeString(c.NamespaceIDStr)
	if err != nil {
		return fmt.Errorf("decode string: %w", err)
	}

	// Check if NamespaceID is of correct length (10 bytes)
	if len(namespaceBytes) != openrpcns.NamespaceVersionZeroIDSize {
		return fmt.Errorf("wrong length: got: %v: expect %v", len(namespaceBytes), openrpcns.NamespaceVersionZeroIDSize)
	}

	ns, err := openrpcns.New(openrpcns.NamespaceVersionZero, append(openrpcns.NamespaceVersionZeroPrefix, namespaceBytes...))
	if err != nil {
		return err
	}

	c.NamespaceID = ns
	return nil
}
