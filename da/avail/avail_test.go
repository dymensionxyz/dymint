package avail_test

import (
	"encoding/json"
	"testing"

	availtypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/avail"
	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/avail"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"
)

const (
	seed = "copper mother insect grunt blue cute tell side welcome domain border oxygen"
)

// FIXME(omritoptix): This test is currently not working as I couldn't find a way to mock the SubmitAndWatchExtrinsic function.
// func TestSubmitBatch(t *testing.T) {
// 	assert := assert.New(t)
// 	require := require.New(t)
// 	configBytes, err := json.Marshal(avail.Config{
// 		Seed: seed,
// 	})
// 	require.NoError(err)
// 	// Create mock clients
// 	mockSubstrateApiClient := mocks.NewSubstrateApiI(t)
// 	// Configure DALC options
// 	options := []da.Option{
// 		avail.WithClient(mockSubstrateApiClient),
// 		avail.WithBatchRetryAttempts(1),
// 		avail.WithBatchRetryDelay(1 * time.Second),
// 		avail.WithTxInclusionTimeout(1 * time.Second),
// 	}
// 	// Subscribe to the health status event
// 	pubsubServer := pubsub.NewServer()
// 	pubsubServer.Start()
// 	// HealthSubscription, err := pubsubServer.Subscribe(context.Background(), "testSubmitBatch", da.EventQueryDAHealthStatus)
// 	assert.NoError(err)
// 	// Start the DALC
// 	dalc := avail.DataAvailabilityLayerClient{}
// 	err = dalc.Init(configBytes, pubsubServer, nil, test.NewLogger(t), options...)
// 	require.NoError(err)
// 	err = dalc.Start()
// 	require.NoError(err)
// 	// Set the mock functions
// 	metadata := availtypes.NewMetadataV14()
// 	metadata.AsMetadataV14 = availtypes.MetadataV14{
// 		Pallets: []availtypes.PalletMetadataV14{
// 			{
// 				Name:     "DataAvailability",
// 				HasCalls: true,
// 			},
// 			{
// 				Name:       "System",
// 				HasStorage: true,
// 				Storage: availtypes.StorageMetadataV14{
// 					Prefix: "System",
// 					Items: []availtypes.StorageEntryMetadataV14{
// 						{
// 							Name: "Account",
// 							Type: availtypes.StorageEntryTypeV14{
// 								IsPlainType: true,
// 								IsMap:       true,
// 								AsMap: availtypes.MapTypeV14{
// 									Hashers: []availtypes.StorageHasherV10{
// 										{
// 											IsIdentity: true,
// 										},
// 									},
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 		EfficientLookup: map[int64]*availtypes.Si1Type{
// 			0: {
// 				Def: availtypes.Si1TypeDef{
// 					Variant: availtypes.Si1TypeDefVariant{
// 						Variants: []availtypes.Si1Variant{
// 							{
// 								Name: "submit_data",
// 							},
// 						},
// 					},
// 				},
// 			},
// 		},
// 	}

// 	mockSubstrateApiClient.On("GetMetadataLatest").Return(metadata, nil)
// 	mockSubstrateApiClient.On("GetBlockHash", mock.Anything).Return(availtypes.NewHash([]byte("123")), nil)
// 	mockSubstrateApiClient.On("GetRuntimeVersionLatest").Return(availtypes.NewRuntimeVersion(), nil)
// 	mockSubstrateApiClient.On("GetStorageLatest", mock.Anything, mock.Anything).Return(true, nil)
// 	mockSubstrateApiClient.On("SubmitAndWatchExtrinsic", mock.Anything).Return(nil, nil)
// batch := &types.Batch{
// 	StartHeight: 0,
// 	EndHeight:   1,
// }
// res := dalc.SubmitBatch(batch)
// assert.Equal(res.Code, da.StatusSuccess)

// }

// TestRetriveBatches tests the RetrieveBatches function manages
// to decode the batches from the block extrinsics and only returns
// the batches relevant for our app id and method index.
func TestRetriveBatches(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	const appId = 123
	// Setup the config
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
	// Start the DALC
	dalc := avail.DataAvailabilityLayerClient{}
	err = dalc.Init(configBytes, pubsubServer, nil, testutil.NewLogger(t), options...)
	require.NoError(err)
	err = dalc.Start()
	require.NoError(err)
	// Set the mock functions
	mockSubstrateApiClient.On("GetBlockHash", mock.Anything).Return(availtypes.NewHash([]byte("123")), nil)
	// Build batches for the block extrinsics
	batch1 := types.Batch{StartHeight: 0, EndHeight: 1}
	batch2 := types.Batch{StartHeight: 2, EndHeight: 3}
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
	mockSubstrateApiClient.On("GetBlock", mock.Anything).Return(signedBlock, nil)
	// Retrieve the batches and make sure we only get the batches relevant for our app id
	daMetaData := &da.DASubmitMetaData{
		Height: 1,
	}
	batchResult := dalc.RetrieveBatches(daMetaData)
	assert.Equal(1, len(batchResult.Batches))
	assert.Equal(batch1.StartHeight, batchResult.Batches[0].StartHeight)
}
