package eth

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/dymensionxyz/dymint/da/ethutils"
	"github.com/dymensionxyz/go-ethereum/common"
	"github.com/dymensionxyz/go-ethereum/core/types"
	"github.com/dymensionxyz/go-ethereum/crypto"
	"github.com/dymensionxyz/go-ethereum/crypto/kzg4844"
	"github.com/holiman/uint256"
)

type Account struct {
	Key  *ecdsa.PrivateKey
	addr common.Address
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

	if err := finishBlobTx(blobtx, gasTipCap, gasFeeCap, blobFeeCap); err != nil {
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

// calcGasFeeCap deterministically computes the recommended gas fee cap given
// the base fee and gasTipCap. The resulting gasFeeCap is equal to:
//
//	gasTipCap + 2*baseFee.
func calcGasFeeCap(baseFee, gasTipCap *big.Int) *big.Int {
	return new(big.Int).Add(
		gasTipCap,
		new(big.Int).Mul(baseFee, big.NewInt(2)),
	)
}

// finishBlobTx finishes creating a blob tx message by safely converting bigints to uint256
func finishBlobTx(message *types.BlobTx, tip, fee, blobFee *big.Int) error {
	var o bool
	if message.GasTipCap, o = uint256.FromBig(tip); o {
		return fmt.Errorf("GasTipCap overflow")
	}
	if message.GasFeeCap, o = uint256.FromBig(fee); o {
		return fmt.Errorf("GasFeeCap overflow")
	}
	if message.BlobFeeCap, o = uint256.FromBig(blobFee); o {
		return fmt.Errorf("BlobFeeCap overflow")
	}
	return nil
}
