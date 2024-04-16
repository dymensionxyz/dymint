package types

import (
	"time"

	"github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/proto/tendermint/version"
	types2 "github.com/tendermint/tendermint/types"
)

// ToABCIHeaderPB converts Dymint header to Header format defined in ABCI.
// Caller should fill all the fields that are not available in Dymint header (like ChainID).
func ToABCIHeaderPB(header *Header) types.Header {
	return types.Header{
		Version: version.Consensus{
			Block: header.Version.Block,
			App:   header.Version.App,
		},
		Height: int64(header.Height),
		Time:   time.Unix(0, int64(header.Time)),
		LastBlockId: types.BlockID{
			Hash: header.LastHeaderHash[:],
			PartSetHeader: types.PartSetHeader{
				Total: 1,
				Hash:  header.LastHeaderHash[:],
			},
		},
		LastCommitHash:     header.LastCommitHash[:],
		DataHash:           header.DataHash[:],
		ValidatorsHash:     header.AggregatorsHash[:],
		NextValidatorsHash: header.AggregatorsHash[:],
		ConsensusHash:      header.ConsensusHash[:],
		AppHash:            header.AppHash[:],
		LastResultsHash:    header.LastResultsHash[:],
		EvidenceHash:       new(types2.EvidenceData).Hash(),
		ProposerAddress:    header.ProposerAddress,
		ChainID:            header.ChainID,
	}
}

// ToABCIHeader converts Dymint header to Header format defined in ABCI.
// Caller should fill all the fields that are not available in Dymint header (like ChainID).
func ToABCIHeader(header *Header) types2.Header {
	return types2.Header{
		Version: version.Consensus{
			Block: header.Version.Block,
			App:   header.Version.App,
		},
		Height: int64(header.Height),
		Time:   time.Unix(0, int64(header.Time)),
		LastBlockID: types2.BlockID{
			Hash: header.LastHeaderHash[:],
			PartSetHeader: types2.PartSetHeader{
				Total: 1,
				Hash:  header.LastHeaderHash[:],
			},
		},
		LastCommitHash:     header.LastCommitHash[:],
		DataHash:           header.DataHash[:],
		ValidatorsHash:     header.AggregatorsHash[:],
		NextValidatorsHash: header.AggregatorsHash[:],
		ConsensusHash:      header.ConsensusHash[:],
		AppHash:            header.AppHash[:],
		LastResultsHash:    header.LastResultsHash[:],
		EvidenceHash:       new(types2.EvidenceData).Hash(),
		ProposerAddress:    header.ProposerAddress,
		ChainID:            header.ChainID,
	}
}

// ToABCIBlock converts Dymint block into block format defined by ABCI.
// Returned block should pass `ValidateBasic`.
func ToABCIBlock(block *Block) (*types2.Block, error) {
	abciHeader := ToABCIHeader(&block.Header)
	abciCommit := ToABCICommit(&block.LastCommit, &block.Header)
	// This assumes that we have only one signature
	if len(abciCommit.Signatures) == 1 {
		abciCommit.Signatures[0].ValidatorAddress = block.Header.ProposerAddress
	}
	abciBlock := types2.Block{
		Header: abciHeader,
		Evidence: types2.EvidenceData{
			Evidence: nil,
		},
		LastCommit: abciCommit,
	}
	abciBlock.Data.Txs = ToABCIBlockDataTxs(&block.Data)
	abciBlock.Header.DataHash = block.Header.DataHash[:]

	return &abciBlock, nil
}

// ToABCIBlockDataTxs converts Dymint block-data into block-data format defined by ABCI.
func ToABCIBlockDataTxs(data *Data) []types2.Tx {
	txs := make([]types2.Tx, len(data.Txs))
	for i := range data.Txs {
		txs[i] = types2.Tx(data.Txs[i])
	}
	return txs
}

// ToABCIBlockMeta converts Dymint block into BlockMeta format defined by ABCI
func ToABCIBlockMeta(block *Block) (*types2.BlockMeta, error) {
	tmblock, err := ToABCIBlock(block)
	if err != nil {
		return nil, err
	}
	blockID := types2.BlockID{Hash: tmblock.Hash()}

	return &types2.BlockMeta{
		BlockID:   blockID,
		BlockSize: tmblock.Size(),
		Header:    tmblock.Header,
		NumTxs:    len(tmblock.Txs),
	}, nil
}

// ToABCICommit converts Dymint commit into commit format defined by ABCI.
// This function only converts fields that are available in Dymint commit.
// Other fields (especially ValidatorAddress and Timestamp of Signature) has to be filled by caller.
func ToABCICommit(commit *Commit, header *Header) *types2.Commit {
	headerHash := header.Hash()
	tmCommit := types2.Commit{
		Height: int64(commit.Height),
		Round:  0,
		BlockID: types2.BlockID{
			Hash: headerHash[:],
			PartSetHeader: types2.PartSetHeader{
				Total: 1,
				Hash:  headerHash[:],
			},
		},
	}
	// Check if TMSignature exists. if not use the previous dymint signature for backwards compatibility.
	if len(commit.TMSignature.Signature) == 0 {
		for _, sig := range commit.Signatures {
			commitSig := types2.CommitSig{
				BlockIDFlag: types2.BlockIDFlagCommit,
				Signature:   sig,
			}
			tmCommit.Signatures = append(tmCommit.Signatures, commitSig)
		}
		// This assumes that we have only one signature
		if len(commit.Signatures) == 1 {
			tmCommit.Signatures[0].ValidatorAddress = header.ProposerAddress
			tmCommit.Signatures[0].Timestamp = time.Unix(0, int64(header.Time))
		}
	} else {
		tmCommit.Signatures = append(tmCommit.Signatures, commit.TMSignature)
	}

	return &tmCommit
}
