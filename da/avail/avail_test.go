package avail_test

import (
	"encoding/json"
	"testing"

	availtypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/avail"
	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/avail"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"
)

const (
	seed = "copper mother insect grunt blue cute tell side welcome domain border oxygen"
)

// TODO: there was another test here, it should be brought back https://github.com/dymensionxyz/dymint/issues/970

// TestRetrieveBatches tests the RetrieveBatches function manages
// to decode the batches from the block extrinsics and only returns
// the batches relevant for our app id and method index.
func TestRetrieveBatches(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	const appId = 123
	// Set up the config
	configBytes, err := json.Marshal(avail.Config{
		Seed:  seed,
		AppID: int64(appId),
	})
	require.NoError(err)
	// Create mock client
	mockSubstrateApiClient := mocks.NewMockSubstrateApiI(t)
	// Configure DALC options
	options := []da.Option{
		avail.WithClient(mockSubstrateApiClient),
	}
	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	assert.NoError(err)

	// set mocks for sync flow
	// Set the mock functions
	mockSubstrateApiClient.On("GetFinalizedHead", mock.Anything).Return(availtypes.NewHash([]byte("123")), nil)
	mockSubstrateApiClient.On("GetHeader", mock.Anything).Return(&availtypes.Header{Number: 1}, nil)
	mockSubstrateApiClient.On("GetBlockLatest", mock.Anything).Return(&availtypes.SignedBlock{Block: availtypes.Block{Header: availtypes.Header{Number: 1}}}, nil)

	// Start the DALC
	dalc := avail.DataAvailabilityLayerClient{}
	err = dalc.Init(configBytes, pubsubServer, nil, testutil.NewLogger(t), options...)
	require.NoError(err)
	err = dalc.Start()
	require.NoError(err)

	// Build batches for the block extrinsics
	batch1 := testutil.MustGenerateBatchAndKey(0, 1)
	batch2 := testutil.MustGenerateBatchAndKey(2, 3)
	batch1bytes, err := batch1.MarshalBinary()
	require.NoError(err)
	batch2bytes, err := batch2.MarshalBinary()
	require.NoError(err)
	// Build the signed block
	signedBlock := &availtypes.SignedBlock{
		Block: availtypes.Block{
			Extrinsics: []availtypes.Extrinsic{
				{
					Method: availtypes.Call{
						Args: availtypes.Args(batch1bytes),
						CallIndex: availtypes.CallIndex{
							MethodIndex:  avail.DataCallMethodIndex,
							SectionIndex: avail.DataCallSectionIndex,
						},
					},
					Signature: availtypes.ExtrinsicSignatureV4{
						AppID: availtypes.NewUCompactFromUInt(uint64(appId)),
					},
				},
				{
					Method: availtypes.Call{
						Args: availtypes.Args(batch2bytes),
						CallIndex: availtypes.CallIndex{
							MethodIndex:  avail.DataCallMethodIndex,
							SectionIndex: avail.DataCallMethodIndex,
						},
					},
				},
			},
		},
	}

	// Set the mock functions
	mockSubstrateApiClient.On("GetBlockHash", mock.Anything).Return(availtypes.NewHash([]byte("123")), nil)
	mockSubstrateApiClient.On("GetBlock", mock.Anything).Return(signedBlock, nil)

	// Retrieve the batches and make sure we only get the batches relevant for our app id
	daMetaData := &da.DASubmitMetaData{
		Height: 1,
	}
	batchResult := dalc.RetrieveBatches(daMetaData)
	assert.Equal(1, len(batchResult.Batches))
	assert.Equal(batch1.StartHeight(), batchResult.Batches[0].StartHeight())
}
