package block

import (
	"github.com/dymensionxyz/dymint/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPersistInstruction(t *testing.T) {
	dir := t.TempDir()

	instructionWithNilFaultyDrs := types.Instruction{
		Revision:            1,
		RevisionStartHeight: 1,
		Sequencer:           "sequencer",
		FaultyDRS:           nil,
	}

	err := types.PersistInstructionToDisk(dir, instructionWithNilFaultyDrs)
	require.NoError(t, err)

	instruction, err := types.LoadInstructionFromDisk(dir)
	require.NoError(t, err)
	require.Equal(t, instructionWithNilFaultyDrs, instruction)

	faultyDrs := new(uint64)
	*faultyDrs = 1
	instructionWithFaultyDrs := types.Instruction{
		Revision:            1,
		RevisionStartHeight: 1,
		Sequencer:           "sequencer",
		FaultyDRS:           faultyDrs,
	}

	err = types.PersistInstructionToDisk(dir, instructionWithFaultyDrs)
	require.NoError(t, err)

	instruction, err = types.LoadInstructionFromDisk(dir)
	require.NoError(t, err)
	require.Equal(t, instructionWithFaultyDrs, instruction)
}
