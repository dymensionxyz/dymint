package celestia

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"

	openrpcns "github.com/rollkit/celestia-openrpc/types/namespace"
)

const (
	defaultRpcRetryDelay            = 30 * time.Second
	defaultRpcCheckAttempts         = 10
	namespaceVersion                = 0
	defaultGasPrices                = 0.1
	defaultGasAdjustment    float64 = 1.3
)

var defaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*4),
	uretry.WithMaxDelay(time.Second*30),
	uretry.WithGrowthFactor(1.6),
)

// Config stores Celestia DALC configuration parameters.
type Config struct {
	BaseURL        string              `json:"base_url,omitempty"`
	AppNodeURL     string              `json:"app_node_url,omitempty"`
	Timeout        time.Duration       `json:"timeout,omitempty"`
	Fee            int64               `json:"fee,omitempty"`
	GasPrices      float64             `json:"gas_prices,omitempty"`
	GasAdjustment  float64             `json:"gas_adjustment,omitempty"`
	GasLimit       uint64              `json:"gas_limit,omitempty"`
	NamespaceIDStr string              `json:"namespace_id,omitempty"`
	AuthToken      string              `json:"auth_token,omitempty"`
	NamespaceID    openrpcns.Namespace `json:"-"`
}

var DefaultConfig = Config{
	BaseURL:        "http://127.0.0.1:26658",
	AppNodeURL:     "",
	Timeout:        30 * time.Second,
	Fee:            0,
	GasLimit:       20000000,
	GasPrices:      defaultGasPrices,
	GasAdjustment:  defaultGasAdjustment,
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
