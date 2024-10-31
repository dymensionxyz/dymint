package types

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Instruction struct {
	Revision            uint64
	RevisionStartHeight uint64
	Sequencer           string
	FaultyDRS           *uint64
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
	return os.WriteFile(filePath, data, 0o644)
}

func LoadInstructionFromDisk(dir string) (Instruction, error) {
	var instruction Instruction

	filePath := filepath.Join(dir, instructionFileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Instruction{}, err
	}

	err = json.Unmarshal(data, &instruction)
	if err != nil {
		return Instruction{}, err
	}

	return instruction, nil
}
