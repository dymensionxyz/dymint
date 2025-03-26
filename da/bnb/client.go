package bnb

import (
	"context"
	"crypto/ecdsa"
	"errors"

	"github.com/datahop/go-ethereum"
	"github.com/datahop/go-ethereum/core/types"
	"github.com/datahop/go-ethereum/crypto"

	"github.com/datahop/go-ethereum/crypto/kzg4844"
	"github.com/datahop/go-ethereum/ethclient"
	"github.com/datahop/go-ethereum/params"
	"github.com/holiman/uint256"

	"github.com/datahop/go-ethereum/common"
	"github.com/datahop/go-ethereum/rpc"
)

type BNBClient interface {
	SubmitBlob(blob []byte) ([]byte, error)
	GetBlob(frameRef []byte) ([]byte, error)
	GetAccountAddress() string
}

type Client struct {
	ethclient *ethclient.Client
	ctx       context.Context
	cfg       *BNBConfig
	account   *Account
}

type Account struct {
	Key  *ecdsa.PrivateKey
	addr common.Address
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
		ctx:       ctx,
		cfg:       config,
		account:   account,
	}

	return client, nil
}

// SubmitData sends blob data to Avail DA
func (c Client) SubmitBlob(blob []byte) ([]byte, error) {

	nonce, err := c.ethclient.PendingNonceAt(context.Background(), c.account.addr)
	if err != nil {
		return nil, err
	}

	blobTx, err := createBlobTx(c.account.Key, blob, c.cfg.To, nonce)

	if err != nil {
		return nil, err
	}

	err = c.ethclient.SendTransaction(context.Background(), blobTx)
	if err != nil {
		return nil, err
	}
	txhash := blobTx.Hash()
	return txhash.Bytes(), nil
}

// GetBlock retrieves a block from Near chain by block hash
func (c Client) GetBlob(txhash []byte) ([]byte, error) {
	var txSidecars []*types.BlobTxSidecar

	hash := common.BytesToHash(txhash)
	txSidecars, err := c.ethclient.BlobSidecarByTxHash(c.ctx, hash)
	if err != nil {
		return nil, err
	}

	if txSidecars == nil {
		return nil, ethereum.NotFound
	}

	var data []byte
	for _, blob := range txSidecars.Blobs {
		data = blob
		continue
	}
	return data, err
}

// GetAccountAddress returns configured account address
func (c Client) GetAccountAddress() string {
	return ""
}

func createBlobTx(key *ecdsa.PrivateKey, blob []byte, toAddr common.Address, nonce uint64) (*types.Transaction, error) {

	var b Blob
	b.FromData(blob)
	rawBlob := b.KZGBlob()
	commitment, err := kzg4844.BlobToCommitment(rawBlob)
	if err != nil {
		return nil, err
	}
	proof, err := kzg4844.ComputeBlobProof(rawBlob, commitment)
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
