package types

import (
	"time"

	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"
)

// Hash returns ABCI-compatible hash of a header.
func (h *Header) Hash() [32]byte {
	abciHeader := tmtypes.Header{
		Version: tmversion.Consensus{
			Block: h.Version.Block,
			App:   h.Version.App,
		},
		Height: int64(h.Height),
		Time:   time.Unix(0, int64(h.Time)),
		LastBlockID: tmtypes.BlockID{
			Hash: h.LastHeaderHash[:],
			PartSetHeader: tmtypes.PartSetHeader{
				Total: 1,
				Hash:  h.LastHeaderHash[:],
			},
		},
		LastCommitHash:     h.LastCommitHash[:],
		DataHash:           h.DataHash[:],
		ValidatorsHash:     h.AggregatorsHash[:],
		NextValidatorsHash: h.AggregatorsHash[:],
		ConsensusHash:      h.ConsensusHash[:],
		AppHash:            h.AppHash[:],
		LastResultsHash:    h.LastResultsHash[:],
		EvidenceHash:       new(tmtypes.EvidenceData).Hash(),
		ProposerAddress:    h.ProposerAddress,
		ChainID:            h.ChainID,
	}
	var hash [32]byte
	copy(hash[:], abciHeader.Hash())
	return hash
}

// Hash returns ABCI-compatible hash of a block.
func (b *Block) Hash() [32]byte {
	return b.Header.Hash()
}
