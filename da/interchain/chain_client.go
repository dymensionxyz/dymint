package interchain

import (
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
)

type DAClient interface {
	Context() sdkclient.Context
	BroadcastTx(accountName string, msgs ...sdktypes.Msg) (cosmosclient.Response, error)
}

func getCosmosClientOptions(config DAConfig) []cosmosclient.Option {
	options := []cosmosclient.Option{
		cosmosclient.WithAddressPrefix(config.AddressPrefix),
		cosmosclient.WithBroadcastMode(flags.BroadcastSync),
		cosmosclient.WithNodeAddress(config.NodeAddress),
		cosmosclient.WithFees(config.GasFees),
		cosmosclient.WithGasLimit(config.GasLimit),
		cosmosclient.WithGasPrices(config.GasPrices),
		cosmosclient.WithKeyringBackend(cosmosaccount.KeyringBackend(config.KeyringBackend)),
		cosmosclient.WithHome(config.KeyringHomeDir),
	}
	return options
}
