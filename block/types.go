package block

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
