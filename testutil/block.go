package testutil

import (
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/types"
)

func GetRandomBlock(height uint64, nTxs int) *types.Block {
	block := &types.Block{
		Header: types.Header{
			Height: height,
		},
		Data: types.Data{
			Txs: make(types.Txs, nTxs),
			IntermediateStateRoots: types.IntermediateStateRoots{
				RawRootsList: make([][]byte, nTxs),
			},
		},
	}

	for i := 0; i < nTxs; i++ {
		block.Data.Txs[i] = GetRandomTx()
		block.Data.IntermediateStateRoots.RawRootsList[i] = GetRandomBytes(32)
	}

	return block
}

// TODO(tzdybal): extract to some common place
func GetRandomValidatorSet() *tmtypes.ValidatorSet {
	pubKey := ed25519.GenPrivKey().PubKey()
	return &tmtypes.ValidatorSet{
		Proposer: &tmtypes.Validator{PubKey: pubKey, Address: pubKey.Address()},
		Validators: []*tmtypes.Validator{
			{PubKey: pubKey, Address: pubKey.Address()},
		},
	}
}
