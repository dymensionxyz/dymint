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
	defaultRpcRetryDelay    = 1 * time.Second
	defaultRpcCheckAttempts = 10
	namespaceVersion        = 0
	defaultGasPrices        = 0.1
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// Config stores Celestia DALC configuration parameters.
type Config struct {
	BaseURL        string              `json:"base_url"`
	AppNodeURL     string              `json:"app_node_url"`
	Timeout        time.Duration       `json:"timeout"`
	GasPrices      float64             `json:"gas_prices"`
	NamespaceIDStr string              `json:"namespace_id"`
	AuthToken      string              `json:"auth_token"`
	NamespaceID    openrpcns.Namespace `json:"-"`
}

var CelestiaDefaultConfig = Config{
	BaseURL:        "http://127.0.0.1:26658",
	AppNodeURL:     "",
	Timeout:        5 * time.Second,
	GasPrices:      defaultGasPrices,
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
		return err
	}

	// Check if NamespaceID is of correct length (10 bytes)
	if len(namespaceBytes) != openrpcns.NamespaceVersionZeroIDSize {
		return fmt.Errorf("invalid namespace id length: %v must be %v", len(namespaceBytes), openrpcns.NamespaceVersionZeroIDSize)
	}

	ns, err := openrpcns.New(openrpcns.NamespaceVersionZero, append(openrpcns.NamespaceVersionZeroPrefix, namespaceBytes...))
	if err != nil {
		return err
	}

	c.NamespaceID = ns
	return nil
}
