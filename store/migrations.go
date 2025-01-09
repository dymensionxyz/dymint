package store

import (
	"encoding/binary"
	"fmt"

	"github.com/dymensionxyz/dymint/version"
)

// Migrates 2D to 3D-ready store (nim + mande)
func Run3DMigration(s *DefaultStore, da string) error {
	state, err := s.LoadState()
	if err != nil {
		return err
	}
	// if DrsVersion is set it means no need to run migration
	if state.RollappParams.DrsVersion != 0 {
		return fmt.Errorf("3D migration is not needed")
	}
	// we set DA and DRS_version rollapp params
	state.RollappParams.Da = da
	ver, err := version.GetDRSVersion()
	if err != nil {
		return err
	}
	state.RollappParams.DrsVersion = ver

	// proposer is set to nil to force updating it from SL on startup
	state.SetProposer(nil)

	_, err = s.SaveState(state, nil)
	if err != nil {
		return err
	}

	// we set validation to next height to skip validation of any previous block/batch to migration
	_, err = s.SaveValidationHeight(state.NextHeight(), nil)
	if err != nil {
		return err
	}

	// baseHeight is using sourcePrefix in pre-3D store
	var baseHeight uint64
	b, err := s.db.Get(sourcePrefix[:])
	if err != nil {
		baseHeight = uint64(0)
	} else {
		baseHeight = binary.LittleEndian.Uint64(b)
	}
	err = s.SaveBaseHeight(baseHeight)
	if err != nil {
		return err
	}

	// blocksyncBaseHeight is using validatedHeightPrefix in pre-3D store
	var bsBaseHeight uint64
	b, err = s.db.Get(validatedHeightPrefix[:])
	if err != nil {
		bsBaseHeight = uint64(0)
	} else {
		bsBaseHeight = binary.LittleEndian.Uint64(b)
	}
	err = s.SaveBlockSyncBaseHeight(bsBaseHeight)
	if err != nil {
		return err
	}

	// indexerBaseHeight is using baseHeightPrefix in pre-3D store
	var indexerBaseHeight uint64
	b, err = s.db.Get(baseHeightPrefix[:])
	if err != nil {
		indexerBaseHeight = uint64(0)
	} else {
		indexerBaseHeight = binary.LittleEndian.Uint64(b)
	}
	err = s.SaveIndexerBaseHeight(indexerBaseHeight)
	if err != nil {
		return err
	}

	return nil
}
