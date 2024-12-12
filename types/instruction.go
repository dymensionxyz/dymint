package types

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Instruction struct {
	Revision            uint64
	RevisionStartHeight uint64
	FaultyDRS           []uint32
}

const instructionFileName = "instruction.json"

func PersistInstructionToDisk(dir string, instruction Instruction) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	data, err := json.Marshal(instruction)
	if err != nil {
		return err
	}

	filePath := filepath.Join(dir, instructionFileName)
	return os.WriteFile(filePath, data, 0o600)
}

func LoadInstructionFromDisk(dir string) (Instruction, error) {
	var instruction Instruction

	filePath := filepath.Join(dir, instructionFileName)
	data, err := os.ReadFile(filePath) // nolint:gosec
	if err != nil {
		return Instruction{}, err
	}

	err = json.Unmarshal(data, &instruction)
	if err != nil {
		return Instruction{}, err
	}

	return instruction, nil
}

func InstructionExists(dir string) bool {
	filePath := filepath.Join(dir, instructionFileName)
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func DeleteInstructionFromDisk(dir string) error {
	if !InstructionExists(dir) {
		return nil
	}
	filePath := filepath.Join(dir, instructionFileName)
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("deleting instruction file: %w", err)
	}
	return nil
}
