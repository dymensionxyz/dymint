package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
)

func TestSequencerListsDiff(t *testing.T) {
	seq1 := testutil.GenerateSequencer()
	seq2 := testutil.GenerateSequencer()
	seq3 := testutil.GenerateSequencer()
	seq4 := testutil.GenerateSequencer()

	testCases := []struct {
		name string
		A    []types.Sequencer
		B    []types.Sequencer
		exp  []types.Sequencer
	}{
		{
			name: "no diff",
			A:    []types.Sequencer{seq1, seq2, seq3},
			B:    []types.Sequencer{seq1, seq2, seq3},
			exp:  nil,
		},
		{
			name: "diff",
			A:    []types.Sequencer{seq1, seq2},
			B:    []types.Sequencer{seq2, seq3},
			exp:  []types.Sequencer{seq3},
		},
		{
			name: "B is larger",
			A:    []types.Sequencer{seq1, seq2},
			B:    []types.Sequencer{seq1, seq2, seq3},
			exp:  []types.Sequencer{seq3},
		},
		{
			name: "A is larger",
			A:    []types.Sequencer{seq1, seq2, seq3},
			B:    []types.Sequencer{seq2, seq4},
			exp:  []types.Sequencer{seq4},
		},
		{
			name: "B is a subset of A",
			A:    []types.Sequencer{seq1, seq2, seq3},
			B:    []types.Sequencer{seq1, seq2},
			exp:  nil,
		},
		{
			name: "A is empty",
			A:    []types.Sequencer{},
			B:    []types.Sequencer{seq1, seq2, seq3},
			exp:  []types.Sequencer{seq1, seq2, seq3},
		},
		{
			name: "B is empty",
			A:    []types.Sequencer{seq1, seq2, seq3},
			B:    []types.Sequencer{},
			exp:  nil,
		},
		{
			name: "both are empty",
			A:    []types.Sequencer{},
			B:    []types.Sequencer{},
			exp:  nil,
		},
		{
			name: "A is nil",
			A:    nil,
			B:    []types.Sequencer{seq1, seq2, seq3},
			exp:  []types.Sequencer{seq1, seq2, seq3},
		},
		{
			name: "B is nil",
			A:    []types.Sequencer{seq1, seq2, seq3},
			B:    nil,
			exp:  nil,
		},
		{
			name: "both are nil",
			A:    nil,
			B:    nil,
			exp:  nil,
		},
		{
			name: "A has duplicates",
			A:    []types.Sequencer{seq1, seq1, seq2, seq2},
			B:    []types.Sequencer{seq2, seq3},
			exp:  []types.Sequencer{seq3},
		},
		{
			name: "B has duplicates",
			A:    []types.Sequencer{seq1, seq2},
			B:    []types.Sequencer{seq2, seq3, seq2, seq3},
			exp:  []types.Sequencer{seq3, seq3},
		},
		{
			name: "A and B have duplicates",
			A:    []types.Sequencer{seq1, seq2, seq2},
			B:    []types.Sequencer{seq2, seq3, seq2, seq3},
			exp:  []types.Sequencer{seq3, seq3},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual1 := types.SequencerListRightOuterJoin(tc.A, tc.B)
			require.ElementsMatch(t, tc.exp, actual1)
		})
	}
}
