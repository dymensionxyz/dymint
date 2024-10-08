package block_test

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateUpdateValidator_ValidateP2PBlocks(t *testing.T) {
	validator := &block.StateUpdateValidator{}

	proposerKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	batch, err := testutil.GenerateBatch(1, 10, proposerKey)
	require.NoError(t, err)

	doubleSignedBatch, err := testutil.GenerateBatch(1, 10, proposerKey)
	require.NoError(t, err)

	mixedBatch := make([]*types.Block, 10)
	copy(mixedBatch, batch.Blocks)
	mixedBatch[2] = doubleSignedBatch.Blocks[2]

	tests := []struct {
		name      string
		daBlocks  []*types.Block
		p2pBlocks []*types.Block
		wantErr   bool
	}{
		{
			name:      "Empty blocks",
			daBlocks:  []*types.Block{},
			p2pBlocks: []*types.Block{},
			wantErr:   false,
		},
		{
			name:      "Matching blocks",
			daBlocks:  batch.Blocks,
			p2pBlocks: batch.Blocks,
			wantErr:   false,
		},
		{
			name:      "double signing",
			daBlocks:  batch.Blocks,
			p2pBlocks: doubleSignedBatch.Blocks,
			wantErr:   true,
		},
		{
			name:      "mixed blocks",
			daBlocks:  batch.Blocks,
			p2pBlocks: mixedBatch,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateP2PBlocks(tt.daBlocks, tt.p2pBlocks)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStateUpdateValidator_ValidateDaBlocks(t *testing.T) {

	validator := &block.StateUpdateValidator{}

	proposerKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	batch, err := testutil.GenerateBatch(1, 2, proposerKey)
	require.NoError(t, err)

	tests := []struct {
		name          string
		slBatch       *settlement.ResultRetrieveBatch
		daBlocks      []*types.Block
		expectedError error
	}{
		{
			name: "Happy path - all validations pass",
			slBatch: &settlement.ResultRetrieveBatch{
				Batch: &settlement.Batch{
					BlockDescriptors: []settlement.BlockDescriptor{
						{Height: 1, StateRoot: batch.Blocks[0].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
						{Height: 2, StateRoot: batch.Blocks[1].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
					},
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			},
			daBlocks:      batch.Blocks,
			expectedError: nil,
		},
		{
			name: "Error - number of blocks mismatch",
			slBatch: &settlement.ResultRetrieveBatch{
				Batch: &settlement.Batch{
					BlockDescriptors: []settlement.BlockDescriptor{
						{Height: 1, StateRoot: batch.Blocks[0].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
						{Height: 2, StateRoot: batch.Blocks[1].Header.AppHash[:], Timestamp: batch.Blocks[1].Header.GetTimestamp()},
					},
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			},
			daBlocks:      []*types.Block{batch.Blocks[0]},
			expectedError: fmt.Errorf("num blocks mismatch between state update and DA batch. State index: 1 State update blocks: 2 DA batch blocks: 1"),
		},
		{
			name: "Error - height mismatch",
			slBatch: &settlement.ResultRetrieveBatch{
				Batch: &settlement.Batch{
					BlockDescriptors: []settlement.BlockDescriptor{
						{Height: 101, StateRoot: batch.Blocks[0].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
						{Height: 102, StateRoot: batch.Blocks[1].Header.AppHash[:], Timestamp: batch.Blocks[1].Header.GetTimestamp()},
					},
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			},
			daBlocks:      batch.Blocks,
			expectedError: fmt.Errorf("height mismatch between state update and DA batch. State index: 1 SL height: 101 DA height: 1"),
		},
		{
			name: "Error - state root mismatch",
			slBatch: &settlement.ResultRetrieveBatch{
				Batch: &settlement.Batch{
					BlockDescriptors: []settlement.BlockDescriptor{
						{Height: 1, StateRoot: batch.Blocks[1].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
						{Height: 2, StateRoot: batch.Blocks[1].Header.AppHash[:], Timestamp: batch.Blocks[1].Header.GetTimestamp()},
					},
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			},
			daBlocks:      batch.Blocks,
			expectedError: fmt.Errorf("state root mismatch between state update and DA batch. State index: 1: Height: 1 State root SL: %d State root DA: %d", batch.Blocks[1].Header.AppHash[:], batch.Blocks[0].Header.AppHash[:]),
		},
		{
			name: "Error - timestamp mismatch",
			slBatch: &settlement.ResultRetrieveBatch{
				Batch: &settlement.Batch{
					BlockDescriptors: []settlement.BlockDescriptor{
						{Height: 1, StateRoot: batch.Blocks[0].Header.AppHash[:], Timestamp: batch.Blocks[0].Header.GetTimestamp()},
						{Height: 2, StateRoot: batch.Blocks[1].Header.AppHash[:], Timestamp: batch.Blocks[1].Header.GetTimestamp().Add(1 * time.Second)},
					},
				},
				ResultBase: settlement.ResultBase{
					StateIndex: 1,
				},
			},
			daBlocks: batch.Blocks,
			expectedError: fmt.Errorf("timestamp mismatch between state update and DA batch. State index: 1: Height: 2 Timestamp SL: %s Timestamp DA: %s",
				batch.Blocks[1].Header.GetTimestamp().UTC().Add(1*time.Second), batch.Blocks[1].Header.GetTimestamp().UTC()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateDaBlocks(tt.slBatch, tt.daBlocks)

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
