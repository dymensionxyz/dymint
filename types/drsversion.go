package types

import (
	"sync"

	"github.com/dymensionxyz/dymint/types/pb/dymint"
	"github.com/dymensionxyz/dymint/version"
)

// Executor creates and applies blocks and maintains state.
type DRSVersionHistory struct {
	// The last DRS versions including height upgrade
	History []*dymint.DRSVersion
	drsMux  sync.Mutex
}

// GetDRSVersion returns the DRS version stored in rollapp params updates for a specific height.
// If drs history is empty (because there is no version update for non-finalized heights) it will return current version.
func (d *DRSVersionHistory) GetDRSVersion(height uint64) string {
	defer d.drsMux.Unlock()
	d.drsMux.Lock()

	drsVersion := ""
	for _, drs := range d.History {
		if height >= drs.Height {
			drsVersion = drs.Version
		} else {
			break
		}
	}
	if drsVersion == "" {
		return version.Commit
	}
	return drsVersion
}

// AddDRSVersion adds a new record for the DRS version update heights.
// Returns true if its added (because its actually a new version)
func (d *DRSVersionHistory) AddDRSVersion(height uint64, version string) bool {
	if len(d.History) > 0 && version == d.History[len(d.History)-1].Version {
		return false
	}
	defer d.drsMux.Unlock()
	d.drsMux.Lock()
	d.History = append(d.History, &dymint.DRSVersion{Height: height, Version: version})
	return true
}

// ClearDrsVersionHeights clears drs version previous to the specified height,
// sequencers clear anything previous to the last submitted height
// and full-nodes clear up to last finalized height
func (d *DRSVersionHistory) ClearDRSVersionHeights(height uint64) {
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
