package block

import (
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

type CachedBlock struct {
	Block  *types.Block
	Commit *types.Commit
}
