package testutil

import (
	"crypto/rand"

	"github.com/celestiaorg/optimint/types"
)

func createRandomHashes() [][32]byte {
	h := [][32]byte{}
	for i := 0; i < 8; i++ {
		var h1 [32]byte
		_, err := rand.Read(h1[:])
		if err != nil {
			panic(err)
		}
		h = append(h, h1)
	}
	return h
}

// GenerateBlocks generates random blocks.
// TODO(omritoptix): Generate a chain and not just random blocks. An optional argument would be the previous block.
func GenerateBlocks(num int) []*types.Block {
	blocks := make([]*types.Block, num)
	for i := 0; i < num; i++ {
		h := createRandomHashes()
		block := &types.Block{
			Header: types.Header{
				Version: types.Version{
					Block: 1,
					App:   2,
				},
				NamespaceID:     [8]byte{0, 1, 2, 3, 4, 5, 6, 7},
				Height:          3,
				Time:            4567,
				LastHeaderHash:  h[0],
				LastCommitHash:  h[1],
				DataHash:        h[2],
				ConsensusHash:   h[3],
				AppHash:         h[4],
				LastResultsHash: h[5],
				ProposerAddress: []byte{4, 3, 2, 1},
				AggregatorsHash: h[6],
			},
			Data: types.Data{
				Txs:                    nil,
				IntermediateStateRoots: types.IntermediateStateRoots{RawRootsList: [][]byte{{0x1}}},
				Evidence:               types.EvidenceData{Evidence: nil},
			},
			LastCommit: types.Commit{
				Height:     8,
				HeaderHash: h[7],
				Signatures: []types.Signature{types.Signature([]byte{1, 1, 1}), types.Signature([]byte{2, 2, 2})},
			},
		}
		blocks[i] = block
	}
	return blocks
}

// GenerateBatch generates a batch out of random blocks
func GenerateBatch(startHeight uint64, endHeight uint64) *types.Batch {
	blocks := GenerateBlocks(int(endHeight - startHeight))
	batch := &types.Batch{
		StartHeight: startHeight,
		EndHeight:   endHeight,
		Blocks:      blocks,
	}
	return batch
}
