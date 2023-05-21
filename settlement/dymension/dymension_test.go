package dymension

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"

	sequencertypes "github.com/dymensionxyz/dymension/x/sequencer/types"
	"github.com/dymensionxyz/dymint/log/test"
	mocks "github.com/dymensionxyz/dymint/mocks"
	settlementmocks "github.com/dymensionxyz/dymint/mocks/settlement"
	"github.com/dymensionxyz/dymint/settlement"

	sdkcodectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func TestGetSequencers(t *testing.T) {
	require := require.New(t)
	cosmosClientMock := mocks.NewCosmosClient(t)

	sequencerQueryClientMock := settlementmocks.NewSequencerQueryClient(t)
	count := 5
	sequencersRollappResponse, _ := generateSequencerByRollappResponse(t, count)
	sequencerQueryClientMock.On("SequencersByRollapp", mock.Anything, mock.Anything).Return(sequencersRollappResponse, nil)

	cosmosClientMock.On("GetRollappClient").Return(settlementmocks.NewRollAppQueryClient(t))
	cosmosClientMock.On("GetSequencerClient").Return(sequencerQueryClientMock)

	options := []Option{
		WithCosmosClient(cosmosClientMock),
	}

	pubsubServer := pubsub.NewServer()
	pubsubServer.Start()

	hubClient, err := newDymensionHubClient(settlement.Config{}, pubsubServer, test.NewLogger(t), options...)
	require.NoError(err)

	sequencers, err := hubClient.GetSequencers("mock-rollapp")
	require.NoError(err)
	require.Len(sequencers, count)
}

/* -------------------------------------------------------------------------- */
/*                                    Utils                                   */
/* -------------------------------------------------------------------------- */

func generateSequencerByRollappResponse(t *testing.T, count int) (*sequencertypes.QueryGetSequencersByRollappResponse, sequencertypes.SequencerInfo) {
	// Generate the proposer sequencer
	proposerPubKeyAny, err := sdkcodectypes.NewAnyWithValue(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	proposer := sequencertypes.SequencerInfo{
		Sequencer: sequencertypes.Sequencer{
			DymintPubKey: proposerPubKeyAny,
		},
		Status: sequencertypes.Proposer,
	}
	squencerInfoList := []sequencertypes.SequencerInfo{
		proposer,
	}
	// Generate the inactive sequencers
	for i := 0; i < count-1; i++ {
		nonProposerPubKeyAny, err := sdkcodectypes.NewAnyWithValue(secp256k1.GenPrivKey().PubKey())
		require.NoError(t, err)
		squencerInfoList = append(squencerInfoList, sequencertypes.SequencerInfo{
			Sequencer: sequencertypes.Sequencer{
				DymintPubKey: nonProposerPubKeyAny,
			},
			Status: sequencertypes.Inactive,
		})
	}
	response := &sequencertypes.QueryGetSequencersByRollappResponse{
		SequencerInfoList: squencerInfoList,
	}
	return response, proposer
}
