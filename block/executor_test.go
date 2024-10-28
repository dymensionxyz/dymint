package block_test

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	prototypes "github.com/gogo/protobuf/types"
	"github.com/golang/groupcache/testpb"

	"github.com/dymensionxyz/dymint/block"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	"github.com/tendermint/tendermint/proxy"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/mempool"
	mempoolv1 "github.com/dymensionxyz/dymint/mempool/v1"
	tmmocks "github.com/dymensionxyz/dymint/mocks/github.com/tendermint/tendermint/abci/types"
	tmmocksproxy "github.com/dymensionxyz/dymint/mocks/github.com/tendermint/tendermint/proxy"

	"github.com/dymensionxyz/dymint/types"
)

// TODO: test UpdateProposerFromBlock
func TestCreateBlock(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	logger := log.TestingLogger()

	app := &tmmocks.MockApplication{}
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})

	clientCreator := proxy.NewLocalClientCreator(app)
	abciClient, err := clientCreator.NewABCIClient()
	require.NoError(err)
	require.NotNil(clientCreator)
	require.NotNil(abciClient)

	mpool := mempoolv1.NewTxMempool(logger, cfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(abciClient), 0)
	executor, err := block.NewExecutor([]byte("test address"), "test", mpool, proxy.NewAppConns(clientCreator), nil, nil, logger)
	assert.NoError(err)

	maxBytes := uint64(100)

	// Create a valid proposer for the block
	proposerKey := ed25519.GenPrivKey()
	tmPubKey, err := cryptocodec.ToTmPubKeyInterface(proposerKey.PubKey())
	require.NoError(err)

	// Init state
	state := &types.State{}
	state.SetProposer(types.NewSequencerFromValidator(*tmtypes.NewValidator(tmPubKey, 1)))
	state.ConsensusParams.Block.MaxBytes = int64(maxBytes)
	state.ConsensusParams.Block.MaxGas = 100000
	// empty block
	block := executor.CreateBlock(1, &types.Commit{}, [32]byte{}, [32]byte(state.GetProposerHash()), state, maxBytes)
	require.NotNil(block)
	assert.Empty(block.Data.Txs)
	assert.Equal(uint64(1), block.Header.Height)

	// one small Tx
	err = mpool.CheckTx([]byte{1, 2, 3, 4}, func(r *abci.Response) {}, mempool.TxInfo{})
	require.NoError(err)
	block = executor.CreateBlock(2, &types.Commit{}, [32]byte{}, [32]byte(state.GetProposerHash()), state, maxBytes)
	require.NotNil(block)
	assert.Equal(uint64(2), block.Header.Height)
	assert.Len(block.Data.Txs, 1)

	// now there are 3 Txs, and only two can fit into single block
	err = mpool.CheckTx([]byte{4, 5, 6, 7}, func(r *abci.Response) {}, mempool.TxInfo{})
	require.NoError(err)
	err = mpool.CheckTx(make([]byte, 100), func(r *abci.Response) {}, mempool.TxInfo{})
	require.NoError(err)
	block = executor.CreateBlock(3, &types.Commit{}, [32]byte{}, [32]byte(state.GetProposerHash()), state, maxBytes)
	require.NotNil(block)
	assert.Len(block.Data.Txs, 2)
}

func TestCreateBlockWithConsensusMessages(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	logger := log.TestingLogger()
	app := &tmmocks.MockApplication{}
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	clientCreator := proxy.NewLocalClientCreator(app)
	abciClient, err := clientCreator.NewABCIClient()
	require.NoError(err)
	require.NotNil(clientCreator)
	require.NotNil(abciClient)
	mpool := mempoolv1.NewTxMempool(logger, cfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(abciClient), 0)

	name, city := "test1", ""
	theMsg1 := &testpb.TestMessage{
		Name: &name,
		City: &city,
	}

	name, city = "test2", ""
	theMsg2 := &testpb.TestMessage{
		Name: &name,
		City: &city,
	}

	// Create a mock ConsensusMessagesStream
	mockStream := &MockConsensusMessagesStream{}
	mockStream.On("GetConsensusMessages").Return([]proto.Message{
		theMsg1,
		theMsg2,
	}, nil)

	executor, err := block.NewExecutor([]byte("test address"), "test", mpool, proxy.NewAppConns(clientCreator), nil, mockStream, logger)
	assert.NoError(err)

	maxBytes := uint64(1000)
	proposerKey := ed25519.GenPrivKey()
	tmPubKey, err := cryptocodec.ToTmPubKeyInterface(proposerKey.PubKey())
	require.NoError(err)

	state := &types.State{}
	state.SetProposer(types.NewSequencerFromValidator(*tmtypes.NewValidator(tmPubKey, 1)))
	state.ConsensusParams.Block.MaxBytes = int64(maxBytes)
	state.ConsensusParams.Block.MaxGas = 100000

	block := executor.CreateBlock(1, &types.Commit{}, [32]byte{}, [32]byte(state.GetProposerHash()[:]), state, maxBytes)

	require.NotNil(block)
	assert.Empty(block.Data.Txs)
	assert.Equal(uint64(1), block.Header.Height)
	assert.Len(block.Data.ConsensusMessages, 2)

	// Verify the content of ConsensusMessages
	theType, err := proto.Marshal(theMsg1)
	require.NoError(err)

	anyMsg1 := &prototypes.Any{
		TypeUrl: proto.MessageName(theMsg1),
		Value:   theType,
	}
	require.NoError(err)

	theType, err = proto.Marshal(theMsg2)
	require.NoError(err)

	anyMsg2 := &prototypes.Any{
		TypeUrl: proto.MessageName(theMsg2),
		Value:   theType,
	}
	require.NoError(err)

	assert.True(proto.Equal(anyMsg1, block.Data.ConsensusMessages[0]))
	assert.True(proto.Equal(anyMsg2, block.Data.ConsensusMessages[1]))

	mockStream.AssertExpectations(t)
}

// MockConsensusMessagesStream is a mock implementation of ConsensusMessagesStream
type MockConsensusMessagesStream struct {
	mock.Mock
}

func (m *MockConsensusMessagesStream) GetConsensusMessages() ([]proto.Message, error) {
	args := m.Called()
	return args.Get(0).([]proto.Message), args.Error(1)
}

func TestApplyBlock(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	logger := log.TestingLogger()

	// Create a valid proposer for the block
	proposerKey := ed25519.GenPrivKey()
	tmPubKey, err := cryptocodec.ToTmPubKeyInterface(proposerKey.PubKey())
	require.NoError(err)

	// Mock ABCI app
	app := &tmmocks.MockApplication{}
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	app.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	app.On("DeliverTx", mock.Anything).Return(abci.ResponseDeliverTx{})
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:      "celestia",
			Version: "abcde",
		},
		ConsensusParamUpdates: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxGas:   100,
				MaxBytes: 100,
			},
		},
	})
	var mockAppHash [32]byte
	_, err = rand.Read(mockAppHash[:])
	require.NoError(err)
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{
		Data: mockAppHash[:],
	})
	app.On("Info", mock.Anything).Return(abci.ResponseInfo{
		LastBlockHeight:  0,
		LastBlockAppHash: []byte{0},
	})

	// Mock ABCI client
	clientCreator := proxy.NewLocalClientCreator(app)
	abciClient, err := clientCreator.NewABCIClient()
	require.NoError(err)
	require.NotNil(clientCreator)
	require.NotNil(abciClient)

	chainID := "test"

	// Init mempool
	mpool := mempoolv1.NewTxMempool(logger, cfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(abciClient), 0)
	eventBus := tmtypes.NewEventBus()
	require.NoError(eventBus.Start())

	// Mock app connections
	appConns := &tmmocksproxy.MockAppConns{}
	appConns.On("Consensus").Return(abciClient)
	appConns.On("Query").Return(abciClient)
	executor, err := block.NewExecutor(proposerKey.PubKey().Address(), chainID, mpool, appConns, eventBus, nil, logger)
	assert.NoError(err)

	// Subscribe to tx events
	txQuery, err := query.New("tm.event='Tx'")
	require.NoError(err)
	txSub, err := eventBus.Subscribe(context.Background(), "test", txQuery, 1000)
	require.NoError(err)
	require.NotNil(txSub)

	// Subscribe to block header events
	headerQuery, err := query.New("tm.event='NewBlockHeader'")
	require.NoError(err)
	headerSub, err := eventBus.Subscribe(context.Background(), "test", headerQuery, 100)
	require.NoError(err)
	require.NotNil(headerSub)

	// Init state
	state := &types.State{}
	state.SetProposer(types.NewSequencerFromValidator(*tmtypes.NewValidator(tmPubKey, 1)))
	state.InitialHeight = 1
	state.ChainID = chainID
	state.SetHeight(0)
	maxBytes := uint64(10000)
	state.ConsensusParams.Block.MaxBytes = int64(maxBytes)
	state.ConsensusParams.Block.MaxGas = 100000
	state.RollappParams.Da = "mock"
	state.LastHeaderHash = [32]byte{0x01}

	// Create first block with one Tx from mempool
	_ = mpool.CheckTx([]byte{1, 2, 3, 4}, func(r *abci.Response) {}, mempool.TxInfo{})
	require.NoError(err)
	block := executor.CreateBlock(1, &types.Commit{Height: 0}, [32]byte{0x01}, [32]byte(state.GetProposerHash()), state, maxBytes)
	require.NotNil(block)
	assert.Equal(uint64(1), block.Header.Height)
	assert.Len(block.Data.Txs, 1)

	// Create commit for the block
	abciHeaderPb := types.ToABCIHeaderPB(&block.Header)
	abciHeaderBytes, err := abciHeaderPb.Marshal()
	require.NoError(err)
	signature, err := proposerKey.Sign(abciHeaderBytes)
	require.NoError(err)
	commit := &types.Commit{
		Height:     block.Header.Height,
		HeaderHash: block.Header.Hash(),
		Signatures: []types.Signature{signature},
	}

	// Apply the block
	err = types.ValidateProposedTransition(state, block, commit, state.GetProposerPubKey())
	require.NoError(err)

	resp, err := executor.ExecuteBlock(block)
	require.NoError(err)
	require.NotNil(resp)
	appHash, _, err := executor.Commit(state, block, resp)
	require.NoError(err)
	executor.UpdateStateAfterCommit(state, resp, appHash, block.Header.Height, block.Header.Hash())
	assert.Equal(uint64(1), state.Height())
	assert.Equal(mockAppHash, state.AppHash)

	// Create another block with multiple Tx from mempool
	require.NoError(mpool.CheckTx([]byte{0, 1, 2, 3, 4}, func(r *abci.Response) {}, mempool.TxInfo{}))
	require.NoError(mpool.CheckTx([]byte{5, 6, 7, 8, 9}, func(r *abci.Response) {}, mempool.TxInfo{}))
	require.NoError(mpool.CheckTx([]byte{1, 2, 3, 4, 5}, func(r *abci.Response) {}, mempool.TxInfo{}))
	require.NoError(mpool.CheckTx(make([]byte, 9990), func(r *abci.Response) {}, mempool.TxInfo{}))
	block = executor.CreateBlock(2, commit, block.Header.Hash(), [32]byte(state.GetProposerHash()), state, maxBytes)
	require.NotNil(block)
	assert.Equal(uint64(2), block.Header.Height)
	assert.Len(block.Data.Txs, 3)

	// Get the header bytes
	abciHeaderPb = types.ToABCIHeaderPB(&block.Header)
	abciHeaderBytes, err = abciHeaderPb.Marshal()
	require.NoError(err)

	// Create invalid commit for the block with wrong signature
	invalidProposerKey := ed25519.GenPrivKey()
	invalidSignature, err := invalidProposerKey.Sign(abciHeaderBytes)
	require.NoError(err)
	invalidCommit := &types.Commit{
		Height:     block.Header.Height,
		HeaderHash: block.Header.Hash(),
		Signatures: []types.Signature{invalidSignature},
	}

	// Apply the block with an invalid commit
	err = types.ValidateProposedTransition(state, block, invalidCommit, state.GetProposerPubKey())
	require.ErrorContains(err, types.ErrInvalidSignature.Error())

	// Create a valid commit for the block
	signature, err = proposerKey.Sign(abciHeaderBytes)
	require.NoError(err)
	commit = &types.Commit{
		Height:     block.Header.Height,
		HeaderHash: block.Header.Hash(),
		Signatures: []types.Signature{signature},
	}

	// Apply the block
	err = types.ValidateProposedTransition(state, block, commit, state.GetProposerPubKey())
	require.NoError(err)
	resp, err = executor.ExecuteBlock(block)
	require.NoError(err)
	require.NotNil(resp)
	_, _, err = executor.Commit(state, block, resp)
	require.NoError(err)
	executor.UpdateStateAfterCommit(state, resp, appHash, block.Header.Height, block.Header.Hash())
	assert.Equal(uint64(2), state.Height())

	// check rollapp params update
	assert.Equal(state.RollappParams.Da, "celestia")
	assert.Equal(state.RollappParams.Version, "abcde")
	assert.Equal(state.ConsensusParams.Block.MaxBytes, int64(100))
	assert.Equal(state.ConsensusParams.Block.MaxGas, int64(100))

	// wait for at least 4 Tx events, for up to 3 second.
	// 3 seconds is a fail-scenario only
	timer := time.NewTimer(3 * time.Second)
	txs := make(map[int64]int)
	cnt := 0
	for cnt != 4 {
		select {
		case evt := <-txSub.Out():
			cnt++
			data, ok := evt.Data().(tmtypes.EventDataTx)
			assert.True(ok)
			assert.NotEmpty(data.Tx)
			txs[data.Height]++
		case <-timer.C:
			t.FailNow()
		}
	}
	assert.Zero(len(txSub.Out())) // expected exactly 4 Txs - channel should be empty
	assert.EqualValues(1, txs[1])
	assert.EqualValues(3, txs[2])

	require.EqualValues(2, len(headerSub.Out()))
	for h := 1; h <= 2; h++ {
		evt := <-headerSub.Out()
		data, ok := evt.Data().(tmtypes.EventDataNewBlockHeader)
		assert.True(ok)
		if data.Header.Height == 2 {
			assert.EqualValues(3, data.NumTxs)
		}
	}
}
