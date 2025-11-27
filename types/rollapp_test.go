package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/types"

	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	version "github.com/tendermint/tendermint/proto/tendermint/version"
)

func TestGetRevisionForHeight(t *testing.T) {
	tests := []struct {
		name     string
		rollapp  types.Rollapp
		height   uint64
		expected types.Revision
	}{
		{
			name: "no revisions",
			rollapp: types.Rollapp{
				RollappID: "test",
				Revisions: []types.Revision{},
			},
			height:   100,
			expected: types.Revision{},
		},
		{
			name: "single revision, height matches",
			rollapp: types.Rollapp{
				RollappID: "test",
				Revisions: []types.Revision{
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 1}},
						StartHeight: 50,
					},
				},
			},
			height:   50,
			expected: types.Revision{Revision: tmstate.Version{Consensus: version.Consensus{App: 1}}, StartHeight: 50},
		},
		{
			name: "single revision, height does not match",
			rollapp: types.Rollapp{
				RollappID: "test",
				Revisions: []types.Revision{
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 1}}, StartHeight: 50},
				},
			},
			height:   49,
			expected: types.Revision{},
		},
		{
			name: "multiple revisions, height matches first",
			rollapp: types.Rollapp{
				RollappID: "test",
				Revisions: []types.Revision{
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 1}}, StartHeight: 50},
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 2}}, StartHeight: 100},
				},
			},
			height:   50,
			expected: types.Revision{Revision: tmstate.Version{Consensus: version.Consensus{App: 1}}, StartHeight: 50},
		},
		{
			name: "multiple revisions, height matches second",
			rollapp: types.Rollapp{
				RollappID: "test",
				Revisions: []types.Revision{
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 1}}, StartHeight: 50},
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 2}}, StartHeight: 100},
				},
			},
			height:   100,
			expected: types.Revision{Revision: tmstate.Version{Consensus: version.Consensus{App: 2}}, StartHeight: 100},
		},
		{
			name: "multiple revisions, height between revisions",
			rollapp: types.Rollapp{
				RollappID: "test",
				Revisions: []types.Revision{
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 1}}, StartHeight: 50},
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 2}}, StartHeight: 100},
				},
			},
			height:   75,
			expected: types.Revision{Revision: tmstate.Version{Consensus: version.Consensus{App: 1}}, StartHeight: 50},
		},
		{
			name: "multiple revisions, height after last revision",
			rollapp: types.Rollapp{
				RollappID: "test",
				Revisions: []types.Revision{
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 1}}, StartHeight: 50},
					{Revision: tmstate.Version{Consensus: version.Consensus{App: 2}}, StartHeight: 100},
				},
			},
			height:   150,
			expected: types.Revision{Revision: tmstate.Version{Consensus: version.Consensus{App: 2}}, StartHeight: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.rollapp.GetRevisionForHeight(tt.height)
			require.Equal(t, tt.expected, result)
		})
	}
}
