// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package kzg4844 implements the KZG crypto for EIP-4844.
package bnb

import (
	"embed"
	"encoding/json"
	"hash"
	"reflect"
	"sync"

	gokzg4844 "github.com/crate-crypto/go-kzg-4844"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

//go:embed trusted_setup.json
var content embed.FS

// Blob represents a 4844 data blob.
type KZGBlob [131072]byte

// UnmarshalJSON parses a blob in hex syntax.
func (b *KZGBlob) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(blobT, input, b[:])
}

// MarshalText returns the hex representation of b.
func (b KZGBlob) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

// context is the crypto primitive pre-seeded with the trusted setup parameters.
var kzgcontext *gokzg4844.Context

var (
	blobT       = reflect.TypeOf(Blob{})
	commitmentT = reflect.TypeOf(Commitment{})
	proofT      = reflect.TypeOf(Proof{})
)

// Commitment is a serialized commitment to a polynomial.
type Commitment [48]byte

// UnmarshalJSON parses a commitment in hex syntax.
func (c *Commitment) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(commitmentT, input, c[:])
}

// MarshalText returns the hex representation of c.
func (c Commitment) MarshalText() ([]byte, error) {
	return hexutil.Bytes(c[:]).MarshalText()
}

// Proof is a serialized commitment to the quotient polynomial.
type Proof [48]byte

// UnmarshalJSON parses a proof in hex syntax.
func (p *Proof) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(proofT, input, p[:])
}

// MarshalText returns the hex representation of p.
func (p Proof) MarshalText() ([]byte, error) {
	return hexutil.Bytes(p[:]).MarshalText()
}

// Point is a BLS field element.
type Point [32]byte

// Claim is a claimed evaluation value in a specific point.
type Claim [32]byte

// BlobToCommitment creates a small commitment out of a data blob.
func BlobToCommitment(blob KZGBlob) (Commitment, error) {

	return gokzgBlobToCommitment(blob)
}

// ComputeProof computes the KZG proof at the given point for the polynomial
// represented by the blob.
func ComputeProof(blob KZGBlob, point Point) (Proof, Claim, error) {

	return gokzgComputeProof(blob, point)
}

// VerifyProof verifies the KZG proof that the polynomial represented by the blob
// evaluated at the given point is the claimed value.
/*func VerifyProof(commitment Commitment, point Point, claim Claim, proof Proof) error {

	return gokzgVerifyProof(commitment, point, claim, proof)
}*/

// ComputeBlobProof returns the KZG proof that is used to verify the blob against
// the commitment.
//
// This method does not verify that the commitment is correct with respect to blob.
func ComputeBlobProof(blob KZGBlob, commitment Commitment) (Proof, error) {

	return gokzgComputeBlobProof(blob, commitment)
}

// VerifyBlobProof verifies that the blob data corresponds to the provided commitment.
func VerifyBlobProofkzg(blob KZGBlob, commitment Commitment, proof Proof) error {

	return gokzgVerifyBlobProof(blob, commitment, proof)
}

// CalcBlobHashV1 calculates the 'versioned blob hash' of a commitment.
// The given hasher must be a sha256 hash instance, otherwise the result will be invalid!
func CalcBlobHashV1(hasher hash.Hash, commit *Commitment) (vh [32]byte) {
	if hasher.Size() != 32 {
		panic("wrong hash size")
	}
	hasher.Reset()
	hasher.Write(commit[:])
	hasher.Sum(vh[:0])
	vh[0] = 0x01 // version
	return vh
}

// IsValidVersionedHash checks that h is a structurally-valid versioned blob hash.
func IsValidVersionedHash(h []byte) bool {
	return len(h) == 32 && h[0] == 0x01
}

// gokzgIniter ensures that we initialize the KZG library once before using it.
var gokzgIniter sync.Once

// gokzgInit initializes the KZG library with the provided trusted setup.
func gokzgInit() {
	config, err := content.ReadFile("trusted_setup.json")
	if err != nil {
		panic(err)
	}
	params := new(gokzg4844.JSONTrustedSetup)
	if err = json.Unmarshal(config, params); err != nil {
		panic(err)
	}
	kzgcontext, err = gokzg4844.NewContext4096(params)
	if err != nil {
		panic(err)
	}
}

// gokzgBlobToCommitment creates a small commitment out of a data blob.
func gokzgBlobToCommitment(blob KZGBlob) (Commitment, error) {
	gokzgIniter.Do(gokzgInit)

	commitment, err := kzgcontext.BlobToKZGCommitment((gokzg4844.Blob)(blob), 0)
	if err != nil {
		return Commitment{}, err
	}
	return (Commitment)(commitment), nil
}

// gokzgComputeProof computes the KZG proof at the given point for the polynomial
// represented by the blob.
func gokzgComputeProof(blob KZGBlob, point Point) (Proof, Claim, error) {
	gokzgIniter.Do(gokzgInit)

	proof, claim, err := kzgcontext.ComputeKZGProof((gokzg4844.Blob)(blob), (gokzg4844.Scalar)(point), 0)
	if err != nil {
		return Proof{}, Claim{}, err
	}
	return (Proof)(proof), (Claim)(claim), nil
}

// gokzgVerifyProof verifies the KZG proof that the polynomial represented by the blob
// evaluated at the given point is the claimed value.
func gokzgVerifyProof(commitment Commitment, point Point, claim Claim, proof Proof) error {
	gokzgIniter.Do(gokzgInit)

	return kzgcontext.VerifyKZGProof((gokzg4844.KZGCommitment)(commitment), (gokzg4844.Scalar)(point), (gokzg4844.Scalar)(claim), (gokzg4844.KZGProof)(proof))
}

// gokzgComputeBlobProof returns the KZG proof that is used to verify the blob against
// the commitment.
//
// This method does not verify that the commitment is correct with respect to blob.
func gokzgComputeBlobProof(blob KZGBlob, commitment Commitment) (Proof, error) {
	gokzgIniter.Do(gokzgInit)

	proof, err := kzgcontext.ComputeBlobKZGProof((gokzg4844.Blob)(blob), (gokzg4844.KZGCommitment)(commitment), 0)
	if err != nil {
		return Proof{}, err
	}
	return (Proof)(proof), nil
}

// gokzgVerifyBlobProof verifies that the blob data corresponds to the provided commitment.
func gokzgVerifyBlobProof(blob KZGBlob, commitment Commitment, proof Proof) error {
	gokzgIniter.Do(gokzgInit)

	return kzgcontext.VerifyBlobKZGProof((gokzg4844.Blob)(blob), (gokzg4844.KZGCommitment)(commitment), (gokzg4844.KZGProof)(proof))
}
