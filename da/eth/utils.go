package eth

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"

	"github.com/dymensionxyz/dymint/da/ethutils"
	"github.com/dymensionxyz/go-ethereum/common"
	"github.com/dymensionxyz/go-ethereum/core/types"
	"github.com/dymensionxyz/go-ethereum/crypto"
	"github.com/dymensionxyz/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

type Account struct {
	Key  *ecdsa.PrivateKey
	addr common.Address
}

type estimateGasParams struct {
	From         common.Address   `json:"from"`
	To           *common.Address  `json:"to"`
	Data         hexutil.Bytes    `json:"data"`
	MaxFeePerGas *hexutil.Big     `json:"maxFeePerGas"`
	AccessList   types.AccessList `json:"accessList"`
	BlobHashes   []common.Hash    `json:"blobVersionedHashes,omitempty"`
}

// BlobSidecarResponse represents the structure of the Beacon node blob response
type BeaconChainResponse struct {
	Data BeaconChainData `json:"data"`
}

// BlobSidecarResponse represents the structure of the Beacon node blob response
type BeaconChainData struct {
	Message BeaconChainMessage `json:"message"`
}

// BlobSidecarResponse represents the structure of the Beacon node blob response
type BeaconChainMessage struct {
	Slot string          `json:"slot"`
	Body BeaconChainBody `json:"body"`
}

type BeaconChainBody struct {
	ExecutionPayload ExecutionPayload `json:"execution_payload"`
}

type ExecutionPayload struct {
	BlockNumber string `json:"block_number"`
}

// BlobSidecarResponse represents the structure of the Beacon node blob response
type BlobSidecarResponse struct {
	Data []BlobSidecar `json:"data"`
}

type BlobSidecar struct {
	BlockRoot  string              `json:"block_root"`
	Index      string              `json:"index"`
	Blob       *kzg4844.Blob       `json:"blob"` // Base64-encoded blob data
	Commitment *kzg4844.Commitment `json:"kzg_commitment,omitempty"`
	Proof      *kzg4844.Proof      `json:"kzg_proof,omitempty"`
}

func createBlobTx(key *ecdsa.PrivateKey, chainId, gasLimit uint64, gasTipCap *big.Int, gasFeeCap *big.Int, blobFeeCap *big.Int, blobData []byte, toAddr common.Address, nonce uint64) (*types.Transaction, error) {
	var b ethutils.Blob
	err := b.FromData(blobData)
	if err != nil {
		return nil, err
	}

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
		ChainID:    uint256.NewInt(chainId),
		Nonce:      nonce,
		Gas:        gasLimit,
		To:         toAddr,
		Value:      nil,
		Data:       nil,
		Sidecar:    sidecar,
		BlobHashes: sidecar.BlobHashes(),
	}

	if err := ethutils.FinishBlobTx(blobtx, gasTipCap, gasFeeCap, blobFeeCap); err != nil {
		return nil, fmt.Errorf("failed to create blob transaction: %w", err)
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

// calculateNextBaseFee estimates base fee for EIP-1559
func calculateNextBaseFee(parent *types.Header) *big.Int {
	elasticityMultiplier := uint64(2)
	baseFeeMaxChangeDenominator := uint64(8)

	if parent.BaseFee == nil {
		return nil // Pre-EIP-1559 block
	}

	gasUsed := parent.GasUsed
	gasLimit := parent.GasLimit
	targetGas := gasLimit / elasticityMultiplier
	baseFee := new(big.Int).Set(parent.BaseFee)

	if gasUsed == targetGas {
		return baseFee
	}

	delta := int64(gasUsed - targetGas) //nolint:gosec // disable G115
	change := new(big.Int).Mul(baseFee, big.NewInt(delta))
	change.Div(change, big.NewInt(int64(targetGas)))                   //nolint:gosec // disable G115
	change.Div(change, big.NewInt(int64(baseFeeMaxChangeDenominator))) //nolint:gosec // disable G115

	nextBaseFee := new(big.Int)
	if gasUsed > targetGas {
		nextBaseFee.Add(baseFee, change)
	} else {
		nextBaseFee.Sub(baseFee, change)
	}

	if nextBaseFee.Cmp(big.NewInt(0)) < 0 {
		nextBaseFee = big.NewInt(0)
	}

	return nextBaseFee
}

// httpGet retrieves data from http client
func httpGet(url string, httpClient *http.Client) (io.ReadCloser, error) {

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("post failed: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %s, body: %s", resp.Status, body)
	}
	return resp.Body, nil
}
