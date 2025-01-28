package block_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
)

func TestHandleSequencerSetUpdate(t *testing.T) {
	seq1 := testutil.GenerateSequencer()
	seq2 := testutil.GenerateSequencer()
	seq3 := testutil.GenerateSequencer()
	seq4 := testutil.GenerateSequencer()
	testCases := []struct {
		name    string
		initial []types.Sequencer
		update  []types.Sequencer
		diff    []types.Sequencer
	}{
		{
			name:    "no diff",
			initial: []types.Sequencer{seq1, seq2, seq3},
			update:  []types.Sequencer{seq1, seq2, seq3},
			diff:    nil,
		},
		{
			name:    "diff",
			initial: []types.Sequencer{seq1, seq2},
			update:  []types.Sequencer{seq2, seq3},
			diff:    []types.Sequencer{seq3},
		},
		{
			name:    "update is larger",
			initial: []types.Sequencer{seq1, seq2},
			update:  []types.Sequencer{seq1, seq2, seq3},
			diff:    []types.Sequencer{seq3},
		},
		{
			name:    "initial is larger",
			initial: []types.Sequencer{seq1, seq2, seq3},
			update:  []types.Sequencer{seq2, seq4},
			diff:    []types.Sequencer{seq4},
		},
		{
			name:    "update is a subset of initial",
			initial: []types.Sequencer{seq1, seq2, seq3},
			update:  []types.Sequencer{seq1, seq2},
			diff:    nil,
		},
		{
			name:    "initial is empty",
			initial: []types.Sequencer{},
			update:  []types.Sequencer{seq1, seq2, seq3},
			diff:    []types.Sequencer{seq1, seq2, seq3},
		},
		{
			name:    "update is empty",
			initial: []types.Sequencer{seq1, seq2, seq3},
			update:  []types.Sequencer{},
			diff:    nil,
		},
		{
			name:    "both are empty",
			initial: []types.Sequencer{},
			update:  []types.Sequencer{},
			diff:    nil,
		},
		{
			name:    "initial is nil",
			initial: nil,
			update:  []types.Sequencer{seq1, seq2, seq3},
			diff:    []types.Sequencer{seq1, seq2, seq3},
		},
		{
			name:    "update is nil",
			initial: []types.Sequencer{seq1, seq2, seq3},
			update:  nil,
			diff:    nil,
		},
		{
			name:    "both are nil",
			initial: nil,
			update:  nil,
			diff:    nil,
		},
		{
			name:    "initial has duplicates",
			initial: []types.Sequencer{seq1, seq1, seq2, seq2},
			update:  []types.Sequencer{seq2, seq3},
			diff:    []types.Sequencer{seq3},
		},
		{
			name:    "update has duplicates",
			initial: []types.Sequencer{seq1, seq2},
			update:  []types.Sequencer{seq2, seq3, seq2, seq3},
			diff:    []types.Sequencer{seq3, seq3},
		},
		{
			name:    "initial and update have duplicates",
			initial: []types.Sequencer{seq1, seq2, seq2},
			update:  []types.Sequencer{seq2, seq3, seq2, seq3},
			diff:    []types.Sequencer{seq3, seq3},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a manger
			manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, manager)
			// Set the initial set in the store
			_, err = manager.Store.SaveLastBlockSequencerSet(tc.initial, nil)
			require.NoError(t, err)

			// Set the updated set in memory
			manager.Sequencers.Set(tc.update)

			// Update the set
			actualNewSet, err := manager.SnapshotSequencerSet()
			require.NoError(t, err)
			if len(tc.diff) == 0 {
				require.Empty(t, actualNewSet)
			} else {
				require.ElementsMatch(t, manager.Sequencers.GetAll(), actualNewSet)
			}

			// Get the consensus msgs queue and verify the result
			// Msgs are created only for the diff
			actualConsMsgs := manager.Executor.GetConsensusMsgs()
			require.Len(t, actualConsMsgs, len(tc.diff))

			expConsMsgs, err := block.ConsensusMsgsOnSequencerSetUpdate(tc.diff)
			require.NoError(t, err)

			require.ElementsMatch(t, expConsMsgs, actualConsMsgs)
		})
	}
}
