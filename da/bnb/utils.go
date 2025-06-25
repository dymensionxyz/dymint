package bnb

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

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
