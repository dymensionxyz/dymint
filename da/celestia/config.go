package celestia

import (
	"encoding/hex"
	"time"
)

const (
	defaultTxPollingRetryDelay = 20 * time.Second
	defaultSubmitRetryDelay    = 10 * time.Second
	defaultTxPollingAttempts   = 5
)

// Config stores Celestia DALC configuration parameters.
type Config struct {
	BaseURL        string        `json:"base_url"`
	AppNodeURL     string        `json:"app_node_url"`
	Timeout        time.Duration `json:"timeout"`
	Fee            int64         `json:"fee"`
	GasLimit       uint64        `json:"gas_limit"`
	NamespaceIDStr string        `json:"namespace_id"`
	NamespaceID    [8]byte       `json:"-"`
}

var CelestiaDefaultConfig = Config{
	BaseURL:        "http://127.0.0.1:26659",
	AppNodeURL:     "",
	Timeout:        30 * time.Second,
	Fee:            20000,
	GasLimit:       20000000,
	NamespaceIDStr: "000000000000ffff",
	NamespaceID:    [8]byte{0, 0, 0, 0, 0, 0, 255, 255},
}

func (c *Config) InitNamespaceID() error {
	// Decode NamespaceID from string to byte array
	namespaceBytes, err := hex.DecodeString(c.NamespaceIDStr)
	if err != nil {
		return err
	}
	copy(c.NamespaceID[:], namespaceBytes)
	return nil
}
