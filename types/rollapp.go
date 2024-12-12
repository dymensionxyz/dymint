package types

import rollapptypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"

type Rollapp struct {
	RollappID string
	Revisions []Revision
}

type Revision struct {
	Number      uint64
	StartHeight uint64
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
		revisions = append(revisions, Revision{
			Number:      pbRevision.Number,
			StartHeight: pbRevision.StartHeight,
		})
	}
	return Rollapp{
		RollappID: pb.RollappId,
		Revisions: revisions,
	}
}
