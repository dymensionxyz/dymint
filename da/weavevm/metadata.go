package weavevm

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/dymensionxyz/dymint/da"
)

// SubmitMetaData contains meta data about a batch on the Data Availability Layer.
type SubmitMetaData struct {
	// Height is the height of the block in the da layer
	Height uint64
	// Share commitment, for each blob, used to obtain blobs and proofs
	Commitment da.Commitment
	// WeaveVM tx hash
	WvmTxHash string
}

// ToPath converts a SubmitMetaData to a path.
func (d *SubmitMetaData) ToPath() string {
	commitment := hex.EncodeToString(d.Commitment)
	path := []string{
		strconv.FormatUint(d.Height, 10),
		commitment,
		d.WvmTxHash,
	}
	for i, part := range path {
		path[i] = strings.Trim(part, da.PathSeparator)
	}
	return strings.Join(path, da.PathSeparator)
}

// FromPath parses a path to a SubmitMetaData.
func (d *SubmitMetaData) FromPath(path string) (*SubmitMetaData, error) {
	pathParts := strings.FieldsFunc(path, func(r rune) bool { return r == rune(da.PathSeparator[0]) })
	if len(pathParts) != 3 {
		return nil, fmt.Errorf("invalid DA path")
	}

	height, err := strconv.ParseUint(pathParts[0], 10, 64)
	if err != nil {
		return nil, err
	}

	submitData := &SubmitMetaData{
		Height: height,
	}

	submitData.Commitment, err = hex.DecodeString(pathParts[1])
	if err != nil {
		return nil, err
	}
	submitData.WvmTxHash = pathParts[2]

	return submitData, nil
}
