package celestia

import (
	"encoding/hex"
	"fmt"
	"time"

	openrpcns "github.com/rollkit/celestia-openrpc/types/namespace"
)

const (
	defaultTxPollingRetryDelay         = 20 * time.Second
	defaultSubmitRetryDelay            = 10 * time.Second
	defaultTxPollingAttempts           = 5
	namespaceVersion                   = 0
	defaultGasPrices                   = 0.1
	defaultGasAdjustment       float64 = 1.3
)

// Config stores Celestia DALC configuration parameters.
type Config struct {
	BaseURL        string              `json:"base_url"`
	AppNodeURL     string              `json:"app_node_url"`
	Timeout        time.Duration       `json:"timeout"`
	Fee            int64               `json:"fee"`
	GasPrices      float64             `json:"gas_prices"`
	GasAdjustment  float64             `json:"gas_adjustment"`
	GasLimit       uint64              `json:"gas_limit"`
	NamespaceIDStr string              `json:"namespace_id"`
	NamespaceID    openrpcns.Namespace `json:"-"`
}

var CelestiaDefaultConfig = Config{
	BaseURL:        "http://127.0.0.1:26659",
	AppNodeURL:     "",
	Timeout:        30 * time.Second,
	Fee:            0,
	GasLimit:       20000000,
	GasPrices:      defaultGasPrices,
	GasAdjustment:  defaultGasAdjustment,
	NamespaceIDStr: "00000000000000ffff",
	NamespaceID:    openrpcns.Namespace{Version: namespaceVersion, ID: []byte{0, 0, 0, 0, 0, 0, 0, 0, 255, 255}},
}

func (c *Config) InitNamespaceID() error {
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
