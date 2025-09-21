package types

import (
	"github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/proto/tendermint/version"

	tmtypes "github.com/tendermint/tendermint/types"
)

// ToABCIHeaderPB converts Dymint header to Header format defined in ABCI.
// Caller should fill all the fields that are not available in Dymint header (like ChainID).
// WARNING: THIS IS A LOSSY CONVERSION
func ToABCIHeaderPB(header *Header) types.Header {
	tmheader := ToABCIHeader(header)
	return *tmheader.ToProto()
}

// ToABCIHeader converts Dymint header to Header format defined in ABCI.
// Caller should fill all the fields that are not available in Dymint header (like ChainID).
// WARNING: THIS IS A LOSSY CONVERSION
func ToABCIHeader(header *Header) tmtypes.Header {
	return tmtypes.Header{
		Version: version.Consensus{
			Block: header.Version.Block,
			App:   header.Version.App,
		},
		Height: int64(header.Height), //nolint:gosec // height is non-negative and falls in int64
		Time:   header.GetTimestamp(),
		LastBlockID: tmtypes.BlockID{
			Hash: header.LastHeaderHash[:],
			PartSetHeader: tmtypes.PartSetHeader{
				Total: 1,
				Hash:  header.LastHeaderHash[:],
			},
		},
		LastCommitHash:     header.LastCommitHash[:],
		DataHash:           header.DataHash[:],
		ValidatorsHash:     header.SequencerHash[:],
		NextValidatorsHash: header.NextSequencersHash[:],
		ConsensusHash:      header.ConsensusHash[:],
		AppHash:            header.AppHash[:],
		LastResultsHash:    header.LastResultsHash[:],
		EvidenceHash:       header.DymHash(), // Overloaded, we don't need the evidence field because we don't use comet.
		ProposerAddress:    header.ProposerAddress,
		ChainID:            header.ChainID,
	}
}

// ToABCIBlock converts Dymint block into block format defined by ABCI.
// Returned block should pass `ValidateBasic`.
func ToABCIBlock(block *Block) (*tmtypes.Block, error) {
	abciHeader := ToABCIHeader(&block.Header)
	abciLastCommit := ToABCICommit(&block.LastCommit)
	abciBlock := tmtypes.Block{
		Header: abciHeader,
		Evidence: tmtypes.EvidenceData{
			Evidence: nil,
		},
		LastCommit: abciLastCommit,
	}
	abciBlock.Txs = ToABCIBlockDataTxs(&block.Data)
	abciBlock.DataHash = block.Header.DataHash[:]

	return &abciBlock, nil
}

// ToABCIBlockDataTxs converts Dymint block-data into block-data format defined by ABCI.
func ToABCIBlockDataTxs(data *Data) []tmtypes.Tx {
	txs := make([]tmtypes.Tx, len(data.Txs))
	for i := range data.Txs {
		txs[i] = tmtypes.Tx(data.Txs[i])
	}
	return txs
}

// ToABCIBlockMeta converts Dymint block into BlockMeta format defined by ABCI
func ToABCIBlockMeta(block *Block) (*tmtypes.BlockMeta, error) {
	tmblock, err := ToABCIBlock(block)
	if err != nil {
		return nil, err
	}
	blockID := tmtypes.BlockID{Hash: tmblock.Hash()}

	return &tmtypes.BlockMeta{
		BlockID:   blockID,
		BlockSize: tmblock.Size(),
		Header:    tmblock.Header,
		NumTxs:    len(tmblock.Txs),
	}, nil
}

// ToABCICommit converts Dymint commit into commit format defined by ABCI.
// This function only converts fields that are available in Dymint commit.
// Other fields (especially ValidatorAddress and Timestamp of Signature) has to be filled by caller.
func ToABCICommit(commit *Commit) *tmtypes.Commit {
	tmCommit := tmtypes.Commit{
		Height: int64(commit.Height), //nolint:gosec // height is non-negative and falls in int64
		Round:  0,
		BlockID: tmtypes.BlockID{
			Hash: commit.HeaderHash[:],
			PartSetHeader: tmtypes.PartSetHeader{
				Total: 1,
				Hash:  commit.HeaderHash[:],
			},
		},
	}
	// Very old commits may not have a TM signature already. Should run a prior version of
	// dymint to handle them.
	tmCommit.Signatures = append(tmCommit.Signatures, commit.TMSignature)
	return &tmCommit
}
