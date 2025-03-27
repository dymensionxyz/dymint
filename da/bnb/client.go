package bnb

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/dymensionxyz/go-ethereum/core/types"
	"github.com/dymensionxyz/go-ethereum/crypto"

	"github.com/dymensionxyz/go-ethereum/crypto/kzg4844"
	"github.com/dymensionxyz/go-ethereum/ethclient"
	"github.com/dymensionxyz/go-ethereum/params"
	"github.com/holiman/uint256"

	"github.com/dymensionxyz/go-ethereum/common"
	"github.com/dymensionxyz/go-ethereum/rpc"
)

type BNBClient interface {
	SubmitBlob(blob []byte) (common.Hash, error)
	GetBlob(txHash string) ([]byte, error)
	GetAccountAddress() string
}

type Client struct {
	ethclient *ethclient.Client
	rpcClient *rpc.Client
	ctx       context.Context
	cfg       *BNBConfig
	account   *Account
}

type Account struct {
	Key  *ecdsa.PrivateKey
	addr common.Address
}

type BigInt struct {
	*big.Int
}

func (i *BigInt) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	// Remove quotes and "0x" prefix if present
	s = strings.Trim(s, "\"")
	s = strings.TrimPrefix(s, "0x")

	// Parse the hexadecimal string
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return fmt.Errorf("invalid hex number: %s", s)
	}

	i.Int = n
	return nil
}

type BlobSidecar struct {
	Blobs       []*kzg4844.Blob       `json:"blobs,omitempty"`
	Commitments []*kzg4844.Commitment `json:"commitments,omitempty"`
	Proofs      []*kzg4844.Proof      `json:"proofs,omitempty"`
}

type BlobSidecarTx struct {
	Sidecar     *BlobSidecar `json:"blobSidecar,omitempty"`
	BlockNumber *BigInt      `json:"blockNumber,omitempty"`
	BlockHash   *string      `json:"blockHash,omitempty"`
	TxIndex     string       `json:"txIndex,omitempty"`
	TxHash      *common.Hash `json:"txHash"`
}

// IndexedBlobHash represents a blob hash that commits to a single blob confirmed in a block.  The
// index helps us avoid unnecessary blob to blob hash conversions to find the right content in a
// sidecar.
type IndexedBlobHash struct {
	Index uint64      // absolute index in the block, a.k.a. position in sidecar blobs array
	Hash  common.Hash // hash of the blob, used for consistency checks
}

var _ BNBClient = &Client{}

func NewClient(ctx context.Context, config *BNBConfig) (BNBClient, error) {

	rpcClient, err := rpc.DialContext(ctx, config.Endpoint)
	if err != nil {
		return nil, err
	}

	account, err := fromHexKey(config.PrivateKey)
	if err != nil {
		return nil, err
	}

	client := Client{
		ethclient: ethclient.NewClient(rpcClient),
		rpcClient: rpcClient,
		ctx:       ctx,
		cfg:       config,
		account:   account,
	}

	return client, nil
}

// SubmitData sends blob data to Avail DA
func (c Client) SubmitBlob(blob []byte) (common.Hash, error) {

	nonce, err := c.ethclient.PendingNonceAt(context.Background(), c.account.addr)
	if err != nil {
		return common.Hash{}, err
	}

	blobTx, err := createBlobTx(c.account.Key, blob, common.HexToAddress(ArchivePoolAddress), nonce)

	if err != nil {
		return common.Hash{}, err
	}

	err = c.ethclient.SendTransaction(context.Background(), blobTx)
	if err != nil {
		return common.Hash{}, err
	}
	txhash := blobTx.Hash()
	return txhash, nil
}

// GetBlock retrieves a block from Near chain by block hash
func (c Client) GetBlob(txhash string) ([]byte, error) {

	blobSidecar, err := c.BlobSidecarByTxHash(c.ctx, txhash)
	if err != nil {
		return nil, err
	}

	if blobSidecar == nil {
		return nil, gerrc.ErrNotFound
	}

	var data []byte
	for _, blob := range blobSidecar.Sidecar.Blobs {
		b := (Blob)(*blob)
		data, err = b.ToData()
		if err == nil {
			break
		}
	}
	if data == nil {
		return nil, fmt.Errorf("Error recovering data from blob")
	}
	return data, err
}

// GetAccountAddress returns configured account address
func (c Client) GetAccountAddress() string {
	return ""
}

// BlobSidecarByTxHash return a sidecar of a given blob transaction
func (c Client) BlobSidecarByTxHash(ctx context.Context, hash string) (*BlobSidecarTx, error) {
	var sidecar *BlobSidecarTx
	err := c.rpcClient.CallContext(ctx, &sidecar, "eth_getBlobSidecarByTxHash", hash)
	if err != nil {
		return nil, err
	}
	if sidecar == nil {
		return nil, gerrc.ErrNotFound
	}

	return sidecar, nil

}

func createBlobTx(key *ecdsa.PrivateKey, blobData []byte, toAddr common.Address, nonce uint64) (*types.Transaction, error) {

	var b Blob
	b.FromData(blobData)
	rawBlob := b.KZGBlob()
	commitment, err := kzg4844.BlobToCommitment(*rawBlob)
	if err != nil {
		return nil, err
	}
	proof, err := kzg4844.ComputeBlobProof(*rawBlob, commitment)
	if err != nil {
		return nil, err
	}

	sidecar := &types.BlobTxSidecar{
		Blobs:       []kzg4844.Blob{*rawBlob},
		Commitments: []kzg4844.Commitment{commitment},
		Proofs:      []kzg4844.Proof{proof},
	}

	blobtx := &types.BlobTx{
		ChainID:    uint256.NewInt(97),
		Nonce:      nonce,
		GasTipCap:  uint256.NewInt(10 * params.GWei),
		GasFeeCap:  uint256.NewInt(10 * params.GWei),
		Gas:        25000,
		To:         toAddr,
		Value:      nil,
		Data:       nil,
		Sidecar:    sidecar,
		BlobFeeCap: uint256.NewInt(3 * params.GWei),
		BlobHashes: sidecar.BlobHashes(),
	}

	signer := types.NewCancunSigner(blobtx.ChainID.ToBig())
	return types.MustSignNewTx(key, signer, blobtx), nil
}

func fromHexKey(hexkey string) (*Account, error) {
	key, err := crypto.HexToECDSA(hexkey)
	if err != nil {
		return &Account{}, err
	}
	pubKey := key.Public()
	pubKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		err = errors.New("publicKey is not of type *ecdsa.PublicKey")
		return &Account{}, err
	}
	addr := crypto.PubkeyToAddress(*pubKeyECDSA)
	return &Account{key, addr}, nil
}
