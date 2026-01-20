package types_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/proto/tendermint/version"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	protoutils "github.com/dymensionxyz/dymint/utils/proto"
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

	sequencers := []types.Sequencer{testutil.GenerateSequencer()}
	consensusMsgs, err := block.ConsensusMsgsOnSequencerSetUpdate(sequencers)
	require.NoError(err)

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
				Height:                3,
				Time:                  4567,
				LastHeaderHash:        h[0],
				LastCommitHash:        h[1],
				DataHash:              h[2],
				ConsensusHash:         h[3],
				AppHash:               h[4],
				LastResultsHash:       h[5],
				ProposerAddress:       []byte{4, 3, 2, 1},
				NextSequencersHash:    h[6],
				ConsensusMessagesHash: types.ConsMessagesHash(nil),
			},
			Data: types.Data{
				Txs:               nil,
				ConsensusMessages: protoutils.FromProtoMsgSliceToAnySlice(consensusMsgs...),
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

	cases := []struct {
		name  string
		state types.State
	}{
		{
			name: "with max bytes",
			state: types.State{
				Revisions: []types.Revision{{Revision: tmstate.Version{Consensus: version.Consensus{App: 0, Block: 1}}, StartHeight: 0}},
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
				Revisions: []types.Revision{
					{
						StartHeight: 0,
						Revision: tmstate.Version{
							Consensus: tmversion.Consensus{
								Block: 123,
								App:   456,
							},
							Software: "dymint",
						},
					},
				},
				ChainID:       "testchain",
				InitialHeight: 987,
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
				RollappParams: pb.RollappParams{
					Da:         "mock",
					DrsVersion: 0,
				},
				LastResultsHash: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2},
				AppHash:         [32]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			if c.state.InitialHeight != 0 {
				c.state.SetHeight(986321)
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

func TestStateWithProposer(t *testing.T) {
	t.Parallel()

	proposer := testutil.GenerateSequencer()

	cases := []struct {
		name     string
		proposer *types.Sequencer
	}{
		{
			name:     "nil proposer",
			proposer: nil,
		},
		{
			name:     "non-nil proposer",
			proposer: &proposer,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state := new(types.State)

			state.SetProposer(c.proposer)

			pState, err := state.ToProto()
			require.NoError(t, err)
			require.NotNil(t, pState)

			bytes, err := pState.Marshal()
			require.NoError(t, err)
			require.NotEmpty(t, bytes)

			newProtoState := new(pb.State)
			err = newProtoState.Unmarshal(bytes)
			require.NoError(t, err)

			newState := new(types.State)
			err = newState.FromProto(newProtoState)
			require.NoError(t, err)

			assert.Equal(t, state.GetProposer(), newState.GetProposer())
		})
	}
}

func TestSequencersProtoSerialization(t *testing.T) {
	t.Parallel()

	// Create a sample Sequencer
	pubKey := ed25519.GenPrivKey().PubKey()
	sequencer := types.NewSequencer(pubKey, "settlementAddress", "rewardAddr", []string{"relayer1", "relayer2"})

	// Create a Sequencers slice
	sequencers := types.Sequencers{*sequencer}

	// Convert Sequencers to protobuf
	protoSet, err := sequencers.ToProto()
	require.NoError(t, err)
	require.NotNil(t, protoSet)

	// Marshal the protobuf to bytes
	bytes, err := protoSet.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, bytes)

	// Unmarshal the bytes back to protobuf
	var newProtoSet pb.SequencerSet
	err = newProtoSet.Unmarshal(bytes)
	require.NoError(t, err)

	// Convert protobuf back to Sequencers
	newSequencers, err := types.SequencersFromProto(&newProtoSet)
	require.NoError(t, err)

	// Assert that the original and new Sequencers are equal
	assert.Equal(t, sequencers, newSequencers)
}
