package near

import (
	"strconv"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
)

type NearClient interface {
	SubmitBlob(blob []byte) ([]byte, error)
	GetBlob(frameRef []byte) ([]byte, error)
	GetAccountAddress() string
}

type Client struct {
	rpcClient *rpc.Client
	cfg       BNBConfig
}

var _ NearClient = &Client{}

// NewClient returns a DA avail client
func NewClient(config BNBConfig) (NearClient, error) {

	near, err := near.NewConfig(config.Account, config.Contract, config.Key, config.Network, config.Namespace)
	if err != nil {
		return nil, err
	}

	client := Client{
		near: *near,
	}
	return client, nil
}

// SubmitData sends blob data to Avail DA
func (c Client) SubmitBlob(blob []byte) ([]byte, error) {
	frameRefBytes, err := c.near.ForceSubmit(data)
	if err != nil {
		return nil, err
	}

	return frameRefBytes, nil
}

// GetBlock retrieves a block from Near chain by block hash
func (c Client) GetBlob(frameRef []byte) ([]byte, error) {
	var txSidecars []*BSCBlobTxSidecar
	number := rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(blockID))
	err := c.rpcClient.CallContext(ctx, &txSidecars, "eth_getBlobSidecars", number.String())
	if err != nil {
		return nil, err
	}
	if txSidecars == nil {
		return nil, ethereum.NotFound
	}
	idx := 0
	for _, txSidecar := range txSidecars {
		txIndex, err := util.HexToUint64(txSidecar.TxIndex)
		if err != nil {
			return nil, err
		}
		for j := range txSidecar.BlobSidecar.Blobs {
			sidecars = append(sidecars,
				&types2.GeneralSideCar{
					Sidecar: structs.Sidecar{
						Index:         strconv.Itoa(idx),
						Blob:          txSidecar.BlobSidecar.Blobs[j],
						KzgCommitment: txSidecar.BlobSidecar.Commitments[j],
						KzgProof:      txSidecar.BlobSidecar.Proofs[j],
					},
					TxIndex: int64(txIndex),
					TxHash:  txSidecar.TxHash,
				},
			)
			idx++
		}
	}
	return sidecars, err
}

// GetAccountAddress returns configured account address
func (c Client) GetAccountAddress() string {
	return ""
}
