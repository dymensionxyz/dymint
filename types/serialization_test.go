package types_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/crypto/ed25519"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
)

func TestBlockSerializationRoundTrip(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	// create random hashes
	h := [][32]byte{}
	for i := 0; i < 8; i++ {
		var h1 [32]byte
		n, err := rand.Read(h1[:])
		require.Equal(32, n)
		require.NoError(err)
		h = append(h, h1)
	}

	cases := []struct {
		name  string
		input *types.Block
	}{
		{"empty block", &types.Block{}},
		{"full", &types.Block{
			Header: types.Header{
				Version: types.Version{
					Block: 1,
					App:   2,
				},
				NamespaceID:     [8]byte{},
				Height:          3,
				Time:            4567,
				LastHeaderHash:  h[0],
				LastCommitHash:  h[1],
				DataHash:        h[2],
				ConsensusHash:   h[3],
				AppHash:         h[4],
				LastResultsHash: h[5],
				ProposerAddress: []byte{4, 3, 2, 1},
				SequencersHash:  h[6],
			},
			Data: types.Data{
				Txs:                    nil,
				IntermediateStateRoots: types.IntermediateStateRoots{RawRootsList: [][]byte{{0x1}}},
				// TODO(tzdybal): update when we have actual evidence types
				Evidence: types.EvidenceData{Evidence: nil},
			},
			LastCommit: types.Commit{
				Height:     8,
				HeaderHash: h[7],
				Signatures: []types.Signature{types.Signature([]byte{1, 1, 1}), types.Signature([]byte{2, 2, 2})},
			},
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert := assert.New(t)
			blob, err := c.input.MarshalBinary()
			assert.NoError(err)
			assert.NotEmpty(blob)

			deserialized := &types.Block{}
			err = deserialized.UnmarshalBinary(blob)
			assert.NoError(err)

			assert.Equal(c.input, deserialized)
		})
	}
}

func TestStateRoundTrip(t *testing.T) {
	t.Parallel()

	valSet := getRandomValidatorSet()

	cases := []struct {
		name  string
		state types.State
	}{
		{
			"with max bytes",
			types.State{
				Validators:     valSet,
				NextValidators: valSet,
				ConsensusParams: tmproto.ConsensusParams{
					Block: tmproto.BlockParams{
						MaxBytes:   123,
						MaxGas:     456,
						TimeIotaMs: 789,
					},
				},
			},
		},
		{
			name: "with all fields set",
			state: types.State{
				Version: tmstate.Version{
					Consensus: tmversion.Consensus{
						Block: 123,
						App:   456,
					},
					Software: "dymint",
				},
				ChainID:                     "testchain",
				InitialHeight:               987,
				NextValidators:              valSet,
				Validators:                  valSet,
				LastHeightValidatorsChanged: 8272,
				ConsensusParams: tmproto.ConsensusParams{
					Block: tmproto.BlockParams{
						MaxBytes:   12345,
						MaxGas:     6543234,
						TimeIotaMs: 235,
					},
					Evidence: tmproto.EvidenceParams{
						MaxAgeNumBlocks: 100,
						MaxAgeDuration:  200,
						MaxBytes:        300,
					},
					Validator: tmproto.ValidatorParams{
						PubKeyTypes: []string{"secure", "more secure"},
					},
					Version: tmproto.VersionParams{
						AppVersion: 42,
					},
				},
				LastHeightConsensusParamsChanged: 12345,
				LastResultsHash:                  [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2},
				AppHash:                          [32]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			if c.state.InitialHeight != 0 {
				c.state.LastBlockHeight.Store(986321)
			}

			pState, err := c.state.ToProto()
			require.NoError(err)
			require.NotNil(pState)

			bytes, err := pState.Marshal()
			require.NoError(err)
			require.NotEmpty(bytes)

			var newProtoState pb.State
			var newState types.State
			err = newProtoState.Unmarshal(bytes)
			require.NoError(err)

			err = newState.FromProto(&newProtoState)
			require.NoError(err)

			assert.Equal(c.state, newState)
		})
	}
}

// copied from store_test.go
func getRandomValidatorSet() *tmtypes.ValidatorSet {
	pubKey := ed25519.GenPrivKey().PubKey()
	return &tmtypes.ValidatorSet{
		Proposer: &tmtypes.Validator{PubKey: pubKey, Address: pubKey.Address()},
		Validators: []*tmtypes.Validator{
			{PubKey: pubKey, Address: pubKey.Address()},
		},
	}
}
