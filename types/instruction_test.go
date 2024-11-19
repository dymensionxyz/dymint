package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPersistInstruction(t *testing.T) {
	dir := t.TempDir()

	instructionWithNilFaultyDrs := Instruction{
		Revision:            1,
		RevisionStartHeight: 1,
		FaultyDRS:           nil,
	}

	err := PersistInstructionToDisk(dir, instructionWithNilFaultyDrs)
	require.NoError(t, err)

	instruction, err := LoadInstructionFromDisk(dir)
	require.NoError(t, err)
	require.Equal(t, instructionWithNilFaultyDrs, instruction)

	faultyDrs := []uint32{1}
	instructionWithFaultyDrs := Instruction{
		Revision:            1,
		RevisionStartHeight: 1,
		FaultyDRS:           faultyDrs,
	}

	err = PersistInstructionToDisk(dir, instructionWithFaultyDrs)
	require.NoError(t, err)

	instruction, err = LoadInstructionFromDisk(dir)
	require.NoError(t, err)
	require.Equal(t, instructionWithFaultyDrs, instruction)

	err = DeleteInstructionFromDisk(dir)
	require.NoError(t, err)

	_, err = LoadInstructionFromDisk(dir)
	require.Error(t, err)
}
