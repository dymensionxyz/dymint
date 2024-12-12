package types

import (
	"github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"
)

func ToABCIHeaderPB(header *Header) types.Header {
	tmheader := ToABCIHeader(header)
	return *tmheader.ToProto()
}

func ToABCIHeader(header *Header) tmtypes.Header {
	return tmtypes.Header{
		Version: version.Consensus{
			Block: header.Version.Block,
			App:   header.Version.App,
		},
		Height: int64(header.Height),
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
		EvidenceHash:       new(tmtypes.EvidenceData).Hash(),
		ProposerAddress:    header.ProposerAddress,
		ChainID:            header.ChainID,
	}
}

func ToABCIBlock(block *Block) (*tmtypes.Block, error) {
	abciHeader := ToABCIHeader(&block.Header)
	abciCommit := ToABCICommit(&block.LastCommit, &block.Header)

	if len(abciCommit.Signatures) == 1 {
		abciCommit.Signatures[0].ValidatorAddress = block.Header.ProposerAddress
	}
	abciBlock := tmtypes.Block{
		Header: abciHeader,
		Evidence: tmtypes.EvidenceData{
			Evidence: nil,
		},
		LastCommit: abciCommit,
	}
	abciBlock.Data.Txs = ToABCIBlockDataTxs(&block.Data)
	abciBlock.Header.DataHash = block.Header.DataHash[:]

	return &abciBlock, nil
}

func ToABCIBlockDataTxs(data *Data) []tmtypes.Tx {
	txs := make([]tmtypes.Tx, len(data.Txs))
	for i := range data.Txs {
		txs[i] = tmtypes.Tx(data.Txs[i])
	}
	return txs
}

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

func ToABCICommit(commit *Commit, header *Header) *tmtypes.Commit {
	headerHash := header.Hash()
	tmCommit := tmtypes.Commit{
		Height: int64(commit.Height),
		Round:  0,
		BlockID: tmtypes.BlockID{
			Hash: headerHash[:],
			PartSetHeader: tmtypes.PartSetHeader{
				Total: 1,
				Hash:  headerHash[:],
			},
		},
	}

	if len(commit.TMSignature.Signature) == 0 {
		for _, sig := range commit.Signatures {
			commitSig := tmtypes.CommitSig{
				BlockIDFlag: tmtypes.BlockIDFlagCommit,
				Signature:   sig,
			}
			tmCommit.Signatures = append(tmCommit.Signatures, commitSig)
		}

		if len(commit.Signatures) == 1 {
			tmCommit.Signatures[0].ValidatorAddress = header.ProposerAddress
			tmCommit.Signatures[0].Timestamp = header.GetTimestamp()
		}
	} else {
		tmCommit.Signatures = append(tmCommit.Signatures, commit.TMSignature)
	}

	return &tmCommit
}
