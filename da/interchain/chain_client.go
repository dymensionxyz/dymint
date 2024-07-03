package interchain

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/tendermint/tendermint/libs/bytes"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	interchainda "github.com/dymensionxyz/dymint/types/pb/interchain_da"
)

type daClient struct {
	cosmosclient.Client
	queryClient interchainda.QueryClient
	txService   tx.ServiceClient
}

func newDAClient(ctx context.Context, config DAConfig) (*daClient, error) {
	c, err := cosmosclient.New(ctx, []cosmosclient.Option{
		cosmosclient.WithAddressPrefix(config.AddressPrefix),
		cosmosclient.WithBroadcastMode(flags.BroadcastSync),
		cosmosclient.WithNodeAddress(config.NodeAddress),
		cosmosclient.WithFees(config.GasFees),
		cosmosclient.WithGasLimit(config.GasLimit),
		cosmosclient.WithGasPrices(config.GasPrices),
		cosmosclient.WithGasAdjustment(config.GasAdjustment),
		cosmosclient.WithKeyringBackend(cosmosaccount.KeyringBackend(config.KeyringBackend)),
		cosmosclient.WithHome(config.KeyringHomeDir),
	}...)
	if err != nil {
		return nil, fmt.Errorf("can't create DA layer cosmos client: %w", err)
	}
	return &daClient{
		Client:      c,
		queryClient: interchainda.NewQueryClient(c.Context()),
		txService:   tx.NewServiceClient(c.Context()),
	}, nil
}

func (c *daClient) Params(ctx context.Context) (interchainda.Params, error) {
	resp, err := c.queryClient.Params(ctx, &interchainda.QueryParamsRequest{})
	if err != nil {
		return interchainda.Params{}, fmt.Errorf("can't query DA layer params: %w", err)
	}
	return resp.GetParams(), nil
}

func (c *daClient) GetTx(ctx context.Context, txHash string) (*tx.GetTxResponse, error) {
	return c.txService.GetTx(ctx, &tx.GetTxRequest{Hash: txHash})
}

func (c *daClient) ABCIQueryWithProof(
	ctx context.Context,
	path string,
	data bytes.HexBytes,
	height int64,
) (*ctypes.ResultABCIQuery, error) {
	opts := rpcclient.ABCIQueryOptions{
		Height: height,
		Prove:  true,
	}
	return c.RPC.ABCIQueryWithOptions(ctx, path, data, opts)
}
