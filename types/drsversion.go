package types

import (
	"sync"

	"github.com/dymensionxyz/dymint/types/pb/dymint"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

// Executor creates and applies blocks and maintains state.
type DRSVersionHistory struct {
	// The last DRS versions including height upgrade
	History    []*dymint.DRSVersion
	drsMux     sync.Mutex
	lastHeight uint64
}

// GetDRSVersion returns the DRS version stored from rollapp params updates for a specific height.
// If height is already cleared it returns not found error.
func (d *DRSVersionHistory) GetDRSVersion(height uint64) (string, error) {
	drsVersion := ""

	if height < d.lastHeight {
		return drsVersion, gerrc.ErrNotFound
	}
	defer d.drsMux.Unlock()
	d.drsMux.Lock()

	for _, drs := range d.History {
		if height >= drs.Height {
			drsVersion = drs.Version
		} else {
			break
		}
	}

	return drsVersion, nil
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
// but keeping always the previous record.
// sequencers clear anything previous to the last submitted height
// and full-nodes clear up to last finalized height
func (d *DRSVersionHistory) ClearDRSVersionHeights(height uint64) {
	d.lastHeight = height
	for i, drs := range d.History {
		if drs.Height < height {
			d.drsMux.Lock()
			d.History = d.History[i:]
			d.drsMux.Unlock()
		}
	}
}

// ToProto converts DRSVersion into protobuf representation and returns it.
func (s *DRSVersionHistory) ToProto() *dymint.DRS {
	return &dymint.DRS{
		DrsVersionHistory: s.History,
		LastHeight:        s.lastHeight,
	}
}

// FromProto fills State with data from its protobuf representation.
func (s *DRSVersionHistory) FromProto(other *dymint.DRS) {
	s.History = other.DrsVersionHistory
	s.lastHeight = other.LastHeight
}
