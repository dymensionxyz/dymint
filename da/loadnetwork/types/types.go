package types

import (
	"github.com/dymensionxyz/dymint/da"
)

const (
	ArchivePoolAddress            = "0x0000000000000000000000000000000000000000" // the data settling address, a unified standard across LoadNetwork archiving services
	LoadNetworkMaxTransactionSize = 8_388_608
)

// Config stores LoadNetwork client configuration parameters.
type Config struct {
	da.BaseConfig `json:",inline"`
	da.KeyConfig  `json:",inline"`
	ChainID       int64  `json:"chain_id,omitempty"`
	Endpoint      string `json:"endpoint,omitempty"`

	// Web3Signer config (alternative to KeyConfig)
	Web3SignerEndpoint      string `json:"web3_signer_endpoint,omitempty"`
	Web3SignerTLSCertFile   string `json:"web3_signer_tls_cert_file,omitempty"`
	Web3SignerTLSKeyFile    string `json:"web3_signer_tls_key_file,omitempty"`
	Web3SignerTLSCACertFile string `json:"web3_signer_tls_ca_cert_file,omitempty"`
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
