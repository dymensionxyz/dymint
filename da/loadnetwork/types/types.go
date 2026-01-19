package types

import (
	"time"

	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	ArchivePoolAddress            = "0x0000000000000000000000000000000000000000" // the data settling address, a unified standard across LoadNetwork archiving services
	LoadNetworkMaxTransactionSize = 8_388_608
)

// Config...LoadNetwork client configuration
type Config struct {
	Timeout  time.Duration `json:"timeout,omitempty"`
	ChainID  int64         `json:"chain_id,omitempty"`
	Endpoint string        `json:"endpoint,omitempty"`

	// Signer config (either private key or web3signer required)
	PrivateKeyEnv           string `json:"private_key_env,omitempty"`  // Environment variable name for private key (highest priority)
	PrivateKeyFile          string `json:"private_key_file,omitempty"` // Path to file containing private key (second priority)
	PrivateKeyHex           string `json:"private_key_hex,omitempty"`  // Private key directly in config (lowest priority, fallback only)
	Web3SignerEndpoint      string `json:"web3_signer_endpoint,omitempty"`
	Web3SignerTLSCertFile   string `json:"web3_signer_tls_cert_file,omitempty"`
	Web3SignerTLSKeyFile    string `json:"web3_signer_tls_key_file,omitempty"`
	Web3SignerTLSCACertFile string `json:"web3_signer_tls_ca_cert_file,omitempty"`

	// Retry config
	Backoff       uretry.BackoffConfig `json:"backoff,omitempty"`
	RetryAttempts *int                 `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration        `json:"retry_delay,omitempty"`
}

type RetrieverResponse struct {
	ArweaveBlockHash   string `json:"arweave_block_hash"`
	Calldata           string `json:"calldata"`
	WarDecodedCalldata string `json:"war_decoded_calldata"`
	LNBlockHash        string `json:"wvm_block_hash"`
}

type LNDymintBlob struct {
	ArweaveBlockHash string
	LNBlockHash      string
	LNTxHash         string
	LNBlockNumber    uint64
	Blob             []byte
}
