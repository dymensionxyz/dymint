package local_test

import (
	"encoding/hex"
	"os"
	"testing"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/local"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
)

func TestGetSequencers(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(err)
	proposerPubKey := proposerKey.GetPublic()
	pubKeybytes, err := proposerPubKey.Raw()
	require.NoError(err)

	sllayer := local.Client{}
	cfg := settlement.Config{ProposerPubKey: hex.EncodeToString(pubKeybytes)}
	err = sllayer.Init(cfg, nil, log.TestingLogger())
	require.NoError(err)

	sequencers, err := sllayer.GetSequencers()
	require.NoError(err)
	assert.Equal(1, len(sequencers))
	assert.Equal(pubKeybytes, sequencers[0].PublicKey.Bytes())

	proposer := sllayer.GetProposer()
	require.NotNil(proposer)
	assert.Equal(pubKeybytes, proposer.PublicKey.Bytes())
}

func TestSubmitBatch(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	logger := log.TestingLogger()
	pubsubServer := pubsub.NewServer()
	err := pubsubServer.Start()
	require.NoError(err)

	sllayer := local.Client{}
	err = sllayer.Init(settlement.Config{}, pubsubServer, logger)
	require.NoError(err)
	_, err = sllayer.GetLatestBatch()
	require.Error(err) // no batch should be present

	// Create a batches which will be submitted
	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(err)
	batch1, err := testutil.GenerateBatch(1, 1, proposerKey)
	require.NoError(err)
	batch2, err := testutil.GenerateBatch(2, 2, proposerKey)
	require.NoError(err)
	resultSubmitBatch := &da.ResultSubmitBatch{}
	resultSubmitBatch.SubmitMetaData = &da.DASubmitMetaData{}

	// Submit the first batch and check if it was successful
	err = sllayer.SubmitBatch(batch1, da.Mock, resultSubmitBatch)
	assert.NoError(err)
	assert.True(resultSubmitBatch.Code == 0) // success code

	// Check if the batch was submitted
	queriedBatch, err := sllayer.GetLatestBatch()
	require.NoError(err)
	assert.Equal(batch1.EndHeight(), queriedBatch.Batch.EndHeight)

	state, err := sllayer.GetHeightState(1)
	require.NoError(err)
	assert.Equal(queriedBatch.StateIndex, state.State.StateIndex)

	queriedBatch, err = sllayer.GetBatchAtIndex(state.State.StateIndex)
	require.NoError(err)
	assert.Equal(batch1.EndHeight(), queriedBatch.Batch.EndHeight)

	// Submit the 2nd batch and check if it was successful
	err = sllayer.SubmitBatch(batch2, da.Mock, resultSubmitBatch)
	assert.NoError(err)
	assert.True(resultSubmitBatch.Code == 0) // success code

	// Check if the batch was submitted
	queriedBatch, err = sllayer.GetLatestBatch()
	require.NoError(err)
	assert.Equal(batch2.EndHeight(), queriedBatch.Batch.EndHeight)

	state, err = sllayer.GetHeightState(2)
	require.NoError(err)
	assert.Equal(queriedBatch.StateIndex, state.State.StateIndex)

	queriedBatch, err = sllayer.GetBatchAtIndex(state.State.StateIndex)
	require.NoError(err)
	assert.Equal(batch2.EndHeight(), queriedBatch.Batch.EndHeight)

	// TODO: test event emitted
}

func TestPersistency(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	logger := log.TestingLogger()
	pubsubServer := pubsub.NewServer()
	err := pubsubServer.Start()
	require.NoError(err)

	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	require.NoError(err)
	proposerPubKey := proposerKey.GetPublic()
	pubKeybytes, err := proposerPubKey.Raw()
	require.NoError(err)

	sllayer := local.Client{}
	tmpdir, err := os.MkdirTemp("/tmp", "")
	defer os.RemoveAll(tmpdir) // Clean up after the test
	require.NoError(err)

	cfg := settlement.Config{KeyringHomeDir: tmpdir, ProposerPubKey: hex.EncodeToString(pubKeybytes)}
	err = sllayer.Init(cfg, pubsubServer, logger)
	require.NoError(err)

	_, err = sllayer.GetLatestBatch()
	assert.Error(err) // no batch should be present

	// Create a batches which will be submitted
	batch1, err := testutil.GenerateBatch(1, 1, proposerKey)
	require.NoError(err)
	resultSubmitBatch := &da.ResultSubmitBatch{}
	resultSubmitBatch.SubmitMetaData = &da.DASubmitMetaData{}

	// Submit the first batch and check if it was successful
	err = sllayer.SubmitBatch(batch1, da.Mock, resultSubmitBatch)
	assert.NoError(err)
	assert.True(resultSubmitBatch.Code == 0) // success code

	queriedBatch, err := sllayer.GetLatestBatch()
	require.NoError(err)
	assert.Equal(batch1.EndHeight(), queriedBatch.Batch.EndHeight)

	// Restart the layer and check if the batch is still present
	err = sllayer.Stop()
	require.NoError(err)
	sllayer = local.Client{}
	_ = sllayer.Init(cfg, pubsubServer, logger)
	queriedBatch, err = sllayer.GetLatestBatch()
	require.NoError(err)
	assert.Equal(batch1.EndHeight(), queriedBatch.Batch.EndHeight)
}
