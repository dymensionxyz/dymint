package celestia

import (
	"encoding/hex"
	"encoding/json"
	"time"
)

const (
	defaultTxPollingRetryDelay = 20 * time.Second
	defaultSubmitRetryDelay    = 10 * time.Second
	defaultTxPollingAttempts   = 5
)

// Config stores Celestia DALC configuration parameters.
type Config struct {
	BaseURL     string        `json:"base_url"`
	AppNodeURL  string        `json:"app_node_url"`
	Timeout     time.Duration `json:"timeout"`
	Fee         int64         `json:"fee"`
	GasLimit    uint64        `json:"gas_limit"`
	NamespaceID [8]byte       //`json:"namespace_id"`
}

// Define an auxiliary type to prevent infinite recursion in UnmarshalJSON
type auxConfig struct {
	BaseURL     string        `json:"base_url"`
	AppNodeURL  string        `json:"app_node_url"`
	Timeout     time.Duration `json:"timeout"`
	Fee         int64         `json:"fee"`
	GasLimit    uint64        `json:"gas_limit"`
	NamespaceID string        `json:"namespace_id"` // In the auxiliary type, NamespaceID is a string
}

var CelestiaDefaultConfig = Config{
	BaseURL:     "http://127.0.0.1:26659",
	AppNodeURL:  "",
	Timeout:     30 * time.Second,
	Fee:         20000,
	GasLimit:    20000000,
	NamespaceID: [8]byte{0, 0, 0, 0, 0, 0, 255, 255},
}

// UnmarshalJSON on Config type
func (c *Config) UnmarshalJSON(data []byte) error {
	var aux auxConfig
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Decode NamespaceID from string to byte array
	namespaceBytes, err := hex.DecodeString(aux.NamespaceID)
	if err != nil {
		return err
	}

	// Copy the decoded bytes into NamespaceID
	copy(c.NamespaceID[:], namespaceBytes)

	// Copy other fields
	c.BaseURL = aux.BaseURL
	c.AppNodeURL = aux.AppNodeURL
	c.Timeout = aux.Timeout
	c.Fee = aux.Fee
	c.GasLimit = aux.GasLimit

	return nil
}
