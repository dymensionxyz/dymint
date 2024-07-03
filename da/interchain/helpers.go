package interchain

import (
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/da/interchain/ioutils"
	"github.com/dymensionxyz/dymint/types"
)

// EncodeBatch encodes the batch to the interchain DA layer format.
// The batch is represented in binary and gzipped.
func EncodeBatch(b *types.Batch) ([]byte, error) {
	// Prepare the blob data
	blob, err := b.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("can't marshal batch: %w", err)
	}

	// Gzip the blob
	gzipped, err := ioutils.Gzip(blob)
	if err != nil {
		return nil, fmt.Errorf("can't gzip batch: %w", err)
	}

	return gzipped, nil
}

// DecodeBatch decodes the batch from the interchain DA layer format.
// The incoming batch must be a gzipped binary.
func DecodeBatch(b []byte) (*types.Batch, error) {
	if !ioutils.IsGzip(b) {
		return nil, errors.New("batch is not gzip-compressed")
	}

	// Gunzip the blob
	binary, err := ioutils.Gunzip(b)
	if err != nil {
		return nil, fmt.Errorf("can't gunzip batch: %w", err)
	}

	// Prepare the blob data
	batch := new(types.Batch)
	err = batch.UnmarshalBinary(binary)
	if err != nil {
		return nil, fmt.Errorf("can't unmarshal batch: %w", err)
	}

	return batch, nil
}
