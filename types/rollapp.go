package types

import (
	rollapptypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
)

type Rollapp struct {
	RollappID string
	Revisions []Revision
}

func (r Rollapp) LatestRevision() Revision {
	if len(r.Revisions) == 0 {
		// Revision 0 if no revisions exist.
		return Revision{}
	}
	return r.Revisions[len(r.Revisions)-1]
}

func (r Rollapp) GetRevisionForHeight(height uint64) Revision {
	for i := len(r.Revisions) - 1; i >= 0; i-- {
		if height >= r.Revisions[i].StartHeight {
			return r.Revisions[i]
		}
	}
	return Revision{}
}

func RollappFromProto(pb rollapptypes.Rollapp) Rollapp {
	revisions := make([]Revision, 0, len(pb.Revisions))
	for _, pbRevision := range pb.Revisions {
		revision := tmstate.Version{}
		revision.Consensus.App = pbRevision.Number
		revisions = append(revisions, Revision{
			Revision:    revision,
			StartHeight: pbRevision.StartHeight,
		})
	}
	return Rollapp{
		RollappID: pb.RollappId,
		Revisions: revisions,
	}
}
