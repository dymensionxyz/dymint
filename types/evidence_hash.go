package types

import (
	"github.com/tendermint/tendermint/crypto/merkle"
	cmtbytes "github.com/tendermint/tendermint/libs/bytes"
	tmtypes "github.com/tendermint/tendermint/types"
)

const versionWithDefaultEvidence = 0

// we overload tendermint header evidence hash with our own stuff
// (we don't need evidence, because we don't use comet)
func evidenceHash(header *Header) cmtbytes.HexBytes {
	if header.Version.Block == versionWithDefaultEvidence {
		return new(tmtypes.EvidenceData).Hash()
	}
	return header.Extra.hash()
}

func (e *ExtraSignedData) hash() []byte {
	return merkle.HashFromByteSlices([][]byte{e.ConsensusMessagesHash[:]})
}
