package celestia

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/celestiaorg/go-square/merkle"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/celestia/client"
)

// SubmitMetaData contains submission meta data about a batch on the Data Availability Layer, included in da path.
type SubmitMetaData struct {
	// Height is the height of the block in the da layer
	Height uint64
	// Namespace ID
	Namespace []byte
	// Share commitment, for each blob, used to obtain blobs and proofs
	Commitment da.Commitment
	// Initial position for each blob in the NMT
	Index int
	// Number of shares of each blob
	Length int
	// any NMT root for the specific height, necessary for non-inclusion proof
	Root []byte
}

// ToPath converts a SubmitMetaData to a path.
func (d *SubmitMetaData) ToPath() string {

	path := []string{
		strconv.FormatUint(d.Height, 10),
		strconv.Itoa(d.Index),
		strconv.Itoa(d.Length),
		hex.EncodeToString(d.Commitment),
		hex.EncodeToString(d.Namespace),
		hex.EncodeToString(d.Root),
	}

	return strings.Join(path, da.PathSeparator)
}

// FromPath parses a path to a SubmitMetaData.
func (d *SubmitMetaData) FromPath(path string) (*SubmitMetaData, error) {
	pathParts := strings.FieldsFunc(path, func(r rune) bool { return r == rune(da.PathSeparator[0]) })
	if len(pathParts) != 6 {
		return nil, fmt.Errorf("invalid DA path")
	}

	height, err := strconv.ParseUint(pathParts[0], 10, 64)
	if err != nil {
		return nil, err
	}

	submitData := &SubmitMetaData{
		Height: height,
	}

	submitData.Index, err = strconv.Atoi(pathParts[1])
	if err != nil {
		return nil, err
	}
	submitData.Length, err = strconv.Atoi(pathParts[2])
	if err != nil {
		return nil, err
	}
	submitData.Commitment, err = hex.DecodeString(pathParts[3])
	if err != nil {
		return nil, err
	}
	submitData.Namespace, err = hex.DecodeString(pathParts[4])
	if err != nil {
		return nil, err
	}
	submitData.Root, err = hex.DecodeString(pathParts[5])
	if err != nil {
		return nil, err
	}

	return submitData, nil
}

// CheckMetaData contains meta data about a batch on the Data Availability Layer, necessary to check availability
type CheckMetaData struct {
	// Height is the height of the block in the da layer
	Height uint64
	// Client is the client to use to fetch data from the da layer
	Client da.Client
	// Submission index in the Hub
	SLIndex uint64
	// Namespace ID
	Namespace []byte
	// Share commitment, for each blob, used to obtain blobs and proofs
	Commitment da.Commitment
	// Initial position for each blob in the NMT
	Index int
	// Number of shares of each blob
	Length int
	// Proofs necessary to validate blob inclusion in the specific height
	Proofs []client.Proof
	// NMT roots for each NMT Proof
	NMTRoots []byte
	// Proofs necessary to validate blob inclusion in the specific height
	RowProofs []*merkle.Proof
	// any NMT root for the specific height, necessary for non-inclusion proof
	Root []byte
}
