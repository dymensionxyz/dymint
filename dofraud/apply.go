package dofraud

import (
	"fmt"

	"github.com/dymensionxyz/dymint/types"
)

type (
	FraudVariant = int
	FraudType    = int
)

// Variant
const (
	NoneVariant = iota
	DA
	Gossip
	Produce
)

// Type
const (
	NoneType = iota
	HeaderVersionBlock
	HeaderVersionApp
	HeaderChainID
	HeaderHeight
	HeaderTime
	HeaderLastHeaderHash
	HeaderDataHash
	HeaderConsensusHash
	HeaderAppHash
	HeaderLastResultsHash
	HeaderProposerAddr
	HeaderLastCommitHash
	HeaderSequencerHash
	HeaderNextSequencerHash
	Data
	LastCommit
)

type Cmd struct {
	*types.Block
	ts []FraudType
}

// The possibilites are simple, just change
type Frauds struct {
	frauds map[string]Cmd
}

type key struct {
	height  uint64
	variant FraudVariant
}

func (k key) String() string {
	return fmt.Sprintf("%d:%d", k.height, k.variant)
}

func (f *Frauds) Has(height uint64, variant FraudVariant) bool {
	_, ok := f.frauds[key{height, variant}.String()]
	return ok
}

// apply any loaded frauds, no-op if none
func (f *Frauds) Apply(log types.Logger, height uint64, fraudVariant FraudVariant, b *types.Block) bool {
	cmd, ok := f.frauds[key{height, fraudVariant}.String()]
	if !ok {
		return false
	}

	for _, fraud := range cmd.ts {
		switch fraud {
		case HeaderVersionBlock:
			b.Header.Version.Block = cmd.Header.Version.Block
		case HeaderVersionApp:
			b.Header.Version.App = cmd.Header.Version.App
		case HeaderChainID:
			b.Header.ChainID = cmd.Header.ChainID
		case HeaderHeight:
			b.Header.Height = cmd.Header.Height
		case HeaderTime:
			b.Header.Time = cmd.Header.Time
		case HeaderLastHeaderHash:
			b.Header.LastHeaderHash = cmd.Header.LastHeaderHash
		case HeaderDataHash:
			b.Header.DataHash = cmd.Header.DataHash
		case HeaderConsensusHash:
			b.Header.ConsensusHash = cmd.Header.ConsensusHash
		case HeaderAppHash:
			b.Header.AppHash = cmd.Header.AppHash
		case HeaderLastResultsHash:
			b.Header.LastResultsHash = cmd.Header.LastResultsHash
		case HeaderProposerAddr:
			b.Header.ProposerAddress = cmd.Header.ProposerAddress
		case HeaderLastCommitHash:
			b.Header.LastCommitHash = cmd.Header.LastCommitHash
		case HeaderSequencerHash:
			b.Header.SequencerHash = cmd.Header.SequencerHash
		case HeaderNextSequencerHash:
			b.Header.NextSequencersHash = cmd.Header.NextSequencersHash
		case Data:
			b.Data = cmd.Data
		case LastCommit:
			b.LastCommit = cmd.LastCommit
		default:
		}
	}

	log.Info("Applied fraud.", "height", height, "variant", fraudVariant, "types", cmd.ts)
	return true
}
