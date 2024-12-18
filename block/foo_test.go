package block

import (
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/types"
	protoutils "github.com/dymensionxyz/dymint/utils/proto"
	cmttypes "github.com/tendermint/tendermint/types"
)

func getBlock() *types.Block {

	lastCommit := &types.Commit{}

	block := &types.Block{
		Header: types.Header{
			Version: types.Version{
				Block: 2,
				App:   3,
			},
			ChainID:         "foo",
			Height:          32,
			Time:            time.Now().UTC().UnixNano(),
			LastHeaderHash:  [32]byte{},
			DataHash:        [32]byte{},
			ConsensusHash:   [32]byte{},
			AppHash:         [32]byte{},
			LastResultsHash: [32]byte{},
			ProposerAddress: e.localAddress,
		},
		Data: types.Data{
			Txs:                    toDymintTxs(make(cmttypes.Txs, 0)),
			IntermediateStateRoots: types.IntermediateStateRoots{RawRootsList: nil},
			Evidence:               types.EvidenceData{Evidence: nil},
			ConsensusMessages:      protoutils.FromProtoMsgSliceToAnySlice(nil),
		},
		LastCommit: *lastCommit,
	}
	copy(block.Header.LastCommitHash[:], types.GetLastCommitHash(lastCommit, &block.Header))
	copy(block.Header.DataHash[:], types.GetDataHash(block))
	copy(block.Header.SequencerHash[:], []byte("sequencerHash"))
	copy(block.Header.NextSequencersHash[:], []byte("nextSequencersHash"))
	return block
}

func TestBlockConsensusMessages(t *testing.T) {
	// Create a block
	b := getBlock()

	// Create a commit
	commit := &types.Commit{
		Height:     1,
		Round:      0,
		BlockID:    types.BlockID{Hash: block.Hash()},
		Signatures: []types.CommitSig{{BlockIDFlag: types.BlockIDFlagCommit, ValidatorAddress: []byte("validator1"), Timestamp: time.Now(), Signature: []byte("signature1")}},
	}

	// Change the consensus messages of the block
	block.Header.ConsensusHash = []byte("newConsensusHash")

	// Verify the commit
	if !commit.ValidateBasic() {
		t.Fatalf("Commit failed basic validation")
	}

	// Confirm the block still verifies the commit
	if !block.ValidateBasic() {
		t.Fatalf("Block failed basic validation")
	}

	if !block.HashesTo(commit.BlockID.Hash) {
		t.Fatalf("Block hash does not match commit block ID hash")
	}
}
