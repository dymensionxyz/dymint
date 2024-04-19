package block

import (
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
)

// TODO: move to types package
type blockSource string

const (
	producedBlock blockSource = "produced"
	gossipedBlock blockSource = "gossip"
	daBlock       blockSource = "da"
)

type blockMetaData struct {
	source   blockSource
	daHeight uint64
}

type PendingBatch struct {
	daResult *da.ResultSubmitBatch
	batch    *types.Batch
}
