package celestia

import (
	"encoding/hex"
	"time"

	cnc "github.com/celestiaorg/go-cnc"
)

const (
	defaultTxPollingRetryDelay = 20 * time.Second
	defaultSubmitRetryDelay    = 10 * time.Second
	defaultTxPollingAttempts   = 5
	namespaceVersion           = 0
	defaultGasPrices           = 0.1
	gasAdjustment              = 12
)

// Config stores Celestia DALC configuration parameters.
type Config struct {
	BaseURL        string        `json:"base_url"`
	AppNodeURL     string        `json:"app_node_url"`
	Timeout        time.Duration `json:"timeout"`
	Fee            int64         `json:"fee"`
	GasPrices      float64       `json:"gas_prices"`
	GasLimit       uint64        `json:"gas_limit"`
	NamespaceIDStr string        `json:"namespace_id"`
	NamespaceID    cnc.Namespace `json:"-"`
}

var CelestiaDefaultConfig = Config{
	BaseURL:        "http://127.0.0.1:26659",
	AppNodeURL:     "",
	Timeout:        30 * time.Second,
	Fee:            0,
	GasLimit:       20000000,
	GasPrices:      defaultGasPrices,
	NamespaceIDStr: "000000000000ffff",
	NamespaceID:    cnc.Namespace{Version: namespaceVersion, ID: []byte{0, 0, 0, 0, 0, 0, 255, 255}},
}

func (c *Config) InitNamespaceID() error {
	// Decode NamespaceID from string to byte array
	namespaceBytes, err := hex.DecodeString(c.NamespaceIDStr)
	if err != nil {
		return err
	}
	// TODO(omritoptix): a hack. need to enforce in the config
	if len(namespaceBytes) != cnc.NamespaceIDSize {
		// pad namespaceBytes with 0s
		namespaceBytes = append(make([]byte, cnc.NamespaceIDSize-len(namespaceBytes)), namespaceBytes...)
	}
	c.NamespaceID, err = cnc.New(namespaceVersion, namespaceBytes)
	if err != nil {
		return err
	}
	return nil
}
