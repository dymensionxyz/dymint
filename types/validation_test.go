package types

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	"github.com/tendermint/tendermint/proto/tendermint/version"

	"github.com/dymensionxyz/dymint/fraud"
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
			Time:            uint64(currentTime.Unix()),
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
		name    string
		block   *Block
		state   *State
		wantErr bool
		errMsg  string
		isFraud bool
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
					Time:            uint64(currentTime.Unix()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:   validState,
			wantErr: true,
			errMsg:  "b version mismatch",
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
					Time:            uint64(currentTime.Unix()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:   validState,
			wantErr: true,
			errMsg:  "b version mismatch",
			isFraud: false,
		},
		{
			name: "Invalid height",
			block: &Block{
				Header: Header{
					Version:         validBlock.Header.Version,
					Height:          11,
					Time:            uint64(currentTime.Unix()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:   validState,
			wantErr: true,
			errMsg:  "height mismatch",
			isFraud: true,
		},
		{
			name: "Invalid AppHash",
			block: &Block{
				Header: Header{
					Version:         validBlock.Header.Version,
					Height:          10,
					Time:            uint64(currentTime.Unix()),
					AppHash:         [32]byte{9, 9, 9},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:   validState,
			wantErr: true,
			errMsg:  "AppHash mismatch",
			isFraud: true,
		},
		{
			name: "Invalid LastResultsHash",
			block: &Block{
				Header: Header{
					Version:         validBlock.Header.Version,
					Height:          10,
					Time:            uint64(currentTime.Unix()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{9, 9, 9},
					ProposerAddress: []byte("proposer"),
					DataHash:        [32]byte(GetDataHash(validBlock)),
				},
			},
			state:   validState,
			wantErr: true,
			errMsg:  "LastResultsHash mismatch",
			isFraud: true,
		},
		{
			name: "Future block time",
			block: &Block{
				Header: Header{
					Version:         validBlock.Header.Version,
					Height:          10,
					Time:            uint64(currentTime.Add(2 * MaxDrift).Unix()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte("proposer"),
				},
			},
			state:   validState,
			wantErr: true,
			errMsg:  "Sequencer",
			isFraud: true,
		},
		{
			name: "Invalid proposer address",
			block: &Block{
				Header: Header{
					Version:         validBlock.Header.Version,
					Height:          10,
					Time:            uint64(currentTime.Unix()),
					AppHash:         [32]byte{1, 2, 3},
					LastResultsHash: [32]byte{4, 5, 6},
					ProposerAddress: []byte{},
				},
			},
			state:   validState,
			wantErr: true,
			errMsg:  "no proposer address",
			isFraud: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.block.ValidateWithState(tt.state)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				if tt.isFraud {
					require.True(t, errors.Is(err, fraud.ErrFraud))
				} else {
					require.False(t, errors.Is(err, fraud.ErrFraud))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
