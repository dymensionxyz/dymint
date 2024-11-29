package types

import (
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	ArchivePoolAddress        = "0x0000000000000000000000000000000000000000" // the data settling address, a unified standard across WeaveVM archiving services
	WeaveVMMaxTransactionSize = 8_388_608
)

// Config...WeaveVM client configuration
type Config struct {
	Enabled bool
	// RPC endpoint of WeaveVM chain
	Endpoint string
	// WeaveVM chain id
	ChainID int64
	// Timeout on WeaveVM calls in seconds
	Timeout time.Duration

	// WeaveVm Private Key
	PrivateKeyHex string
	// Web3Signer configuration
	Web3SignerEndpoint      string
	Web3SignerTLSCertFile   string
	Web3SignerTLSKeyFile    string
	Web3SignerTLSCACertFile string

	Backoff       uretry.BackoffConfig `json:"backoff,omitempty"`
	RetryAttempts *int                 `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration        `json:"retry_delay,omitempty"`
}

type RetrieverResponse struct {
	ArweaveBlockHash   string `json:"arweave_block_hash"`
	Calldata           string `json:"calldata"`
	WarDecodedCalldata string `json:"war_decoded_calldata"`
	WvmBlockHash       string `json:"wvm_block_hash"`
}
