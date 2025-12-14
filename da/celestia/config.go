package celestia

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dymensionxyz/dymint/da"
)

const (
	namespaceVersion        = 0
	DefaultGasPrices        = 0.1
	maxBlobSizeBytes        = 500000
)

// Config stores Celestia DALC configuration parameters.
type Config struct {
	da.BaseConfig  `json:",inline"`
	BaseURL        string    `json:"base_url,omitempty"`
	GasPrices      float64   `json:"gas_prices,omitempty"`
	NamespaceIDStr string    `json:"namespace_id,omitempty"`
	AuthToken      string    `json:"auth_token,omitempty"`
	NamespaceID    Namespace `json:"-"`
}

var TestConfig = Config{
	BaseConfig: da.BaseConfig{
		Timeout: 5 * time.Second,
	},
	BaseURL:        "http://127.0.0.1:26658",
	GasPrices:      DefaultGasPrices,
	NamespaceIDStr: "",
	NamespaceID:    Namespace{Version: namespaceVersion, ID: []byte{0, 0, 0, 0, 0, 0, 0, 0, 255, 255}},
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
	if len(namespaceBytes) != NamespaceVersionZeroIDSize {
		return fmt.Errorf("wrong length: got: %v: expect %v", len(namespaceBytes), NamespaceVersionZeroIDSize)
	}

	ns, err := New(NamespaceVersionZero, append(NamespaceVersionZeroPrefix, namespaceBytes...))
	if err != nil {
		return err
	}

	c.NamespaceID = ns
	return nil
}
