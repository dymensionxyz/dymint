package types

import (
	"errors"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	"github.com/tendermint/tendermint/proto/tendermint/version"
)

func TestBlock_ValidateWithState(t *testing.T) {
	currentTime := time.Now().UTC()
	validState := &State{
		Version: tmstate.Version{
			Consensus: version.Consensus{
				Block: 1,
				App:   1,
			},
		},
		LastBlockHeight: atomic.Uint64{},
		AppHash:         [32]byte{1, 2, 3},
		LastResultsHash: [32]byte{4, 5, 6},
	}
	validState.LastBlockHeight.Store(9)

	validBlock := &Block{
		Header: Header{
			Version: Version{
				Block: 1,
				App:   1,
			},
			Height:          10,
			Time:            uint64(currentTime.UnixNano()),
			AppHash:         [32]byte{1, 2, 3},
			LastResultsHash: [32]byte{4, 5, 6},
			ProposerAddress: []byte("proposer"),
			DataHash:        [32]byte{},
		},
		Data:       Data{},
		LastCommit: Commit{},
	}
	validBlock.Header.DataHash = [32]byte(GetDataHash(validBlock))

	tests := []struct {
		name            string
		block           *Block
		state           *State
		wantErr         bool
		theErr          error
		expectedErrType interface{}
		isFraud         bool
	}{
		{
			name:    "Valid block",
			block:   validBlock,
			state:   validState,
			wantErr: false,
			isFraud: false,
		},
		{
			name: "Invalid block version",
			block: &Block{
				Header: Header{
					Version: Version{
						Block: 2,
						App:   1,
					},
					Height:          10,
					Time:            uint64(currentTime.UnixNano()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:   validState,
			theErr:  ErrVersionMismatch,
			wantErr: true,
			isFraud: false,
		},
		{
			name: "Invalid app version",
			block: &Block{
				Header: Header{
					Version: Version{
						Block: 1,
						App:   2,
					},
					Height:          10,
					Time:            uint64(currentTime.UnixNano()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:   validState,
			wantErr: true,
			theErr:  ErrVersionMismatch,
			isFraud: false,
		},
		{
			name: "Invalid height",
			block: &Block{
				Header: Header{
					Version:         validBlock.Header.Version,
					Height:          11,
					Time:            uint64(currentTime.UnixNano()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: &ErrFraudHeightMismatch{},
			isFraud:         true,
		},
		{
			name: "Invalid AppHash",
			block: &Block{
				Header: Header{
					Version:         validBlock.Header.Version,
					Height:          10,
					Time:            uint64(currentTime.UnixNano()),
					AppHash:         [32]byte{9, 9, 9},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:           validState,
			expectedErrType: &ErrFraudAppHashMismatch{},
			wantErr:         true,
			isFraud:         true,
		},
		{
			name: "Invalid LastResultsHash",
			block: &Block{
				Header: Header{
					Version:         validBlock.Header.Version,
					Height:          10,
					Time:            uint64(currentTime.UnixNano()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{9, 9, 9},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: &ErrLastResultsHashMismatch{},
			isFraud:         true,
		},
		{
			name: "Future block time",
			block: &Block{
				Header: Header{
					Version:         validBlock.Header.Version,
					Height:          10,
					Time:            uint64(currentTime.Add(2 * TimeFraudMaxDrift).UnixNano()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: &ErrTimeFraud{},
			isFraud:         true,
		},
		{
			name: "Invalid proposer address",
			block: &Block{
				Header: Header{
					Version:         validBlock.Header.Version,
					Height:          10,
					Time:            uint64(currentTime.UnixNano()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte{},
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: ErrEmptyProposerAddress,
			isFraud:         false,
		},
		{
			name: "Invalid height",
			block: &Block{
				Header: Header{
					Version:         validBlock.Header.Version,
					Height:          200,
					Time:            uint64(currentTime.UnixNano()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: &ErrInvalidBlockHeightFraud{},
			isFraud:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.block.ValidateWithState(tt.state)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.isFraud {
					require.True(t, errors.Is(err, gerrc.ErrFault))
					if tt.expectedErrType != nil {
						assert.True(t, errors.As(err, &tt.expectedErrType),
							"expected error of type %T, got %T", tt.expectedErrType, err)
					}
				} else {
					require.False(t, errors.Is(err, gerrc.ErrFault))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
