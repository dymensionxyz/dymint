package interchain

import "github.com/celestiaorg/celestia-openrpc/types/sdk"

type ChainConfig struct {
	ChainID     string      // The chain ID of the DA chain
	ClientID    string      // This is the IBC client ID on Dymension hub for the DA chain
	ChainParams ChainParams // The params of the DA chain
}

type ChainParams struct {
	CostPerByte sdk.Coin // Calculate the fees needed to pay to submit the blob
	MaxBlobSize uint32   // Max size of blobs accepted by the chain
}

// DefaultChainConfig returns the default config of the test chain implemented in the interchain-da repo:
// https://github.com/dymensionxyz/interchain-da
func DefaultChainConfig() ChainConfig {
	return ChainConfig{}
}
