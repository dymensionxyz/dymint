package types

import (
	cmtbytes "github.com/tendermint/tendermint/libs/bytes"
	tmtypes "github.com/tendermint/tendermint/types"
)

const versionWithDefaultEvidence = 0

func evidenceHash(header *Header) cmtbytes.HexBytes {
	if header.Version.Block == versionWithDefaultEvidence {
		return new(tmtypes.EvidenceData).Hash()
	}

}
