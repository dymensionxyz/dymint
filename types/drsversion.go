package types

import (
	"sync"

	"github.com/dymensionxyz/dymint/types/pb/dymint"
	"github.com/dymensionxyz/dymint/version"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

// Executor creates and applies blocks and maintains state.
type DRSVersionHistory struct {

	// The last DRS versions including height upgrade
	History []*dymint.DRSVersion
	drsMux  sync.Mutex
}

// GetDRSVersion returns the DRS version stored in rollapp params updates for a specific height.
// It only works for non-finalized heights.
// If drs history is empty (because there is no version update for non-finalized heights) it will return current version.
func (d *DRSVersionHistory) GetDRSVersion(height uint64) (string, error) {
	defer d.drsMux.Unlock()
	d.drsMux.Lock()

	if len(d.History) == 0 {
		return version.Commit, nil
	}
	drsVersion := ""
	for _, drs := range d.History {
		if height >= drs.Height {
			drsVersion = drs.Version
		} else {
			break
		}
	}
	if drsVersion == "" {
		return drsVersion, gerrc.ErrNotFound
	}
	return drsVersion, nil
}

// AddDRSVersion adds a new record for the DRS version update heights.
func (d *DRSVersionHistory) AddDRSVersion(height uint64, version string) {
	defer d.drsMux.Unlock()
	d.drsMux.Lock()
	d.History = append(d.History, &dymint.DRSVersion{Height: height, Version: version})
}

// ClearDrsVersionHeights clears drs version previous to the specified height,
// but keeping always the last drs version record.
// sequencers clear anything previous to the last submitted height
// and full-nodes clear up to last finalized height
func (d *DRSVersionHistory) ClearDRSVersionHeights(height uint64) {
	if len(d.History) == 1 {
		return
	}

	for i, drs := range d.History {
		if drs.Height < height {
			d.drsMux.Lock()
			d.History = d.History[i+1:]
			d.drsMux.Unlock()
		}
	}
}

// ToProto converts DRSVersion into protobuf representation and returns it.
func (s *DRSVersionHistory) ToProto() *dymint.DRS {
	return &dymint.DRS{
		DrsVersionHistory: s.History,
	}
}

// FromProto fills State with data from its protobuf representation.
func (s *DRSVersionHistory) FromProto(other *dymint.DRS) {
	s.History = other.DrsVersionHistory
}
