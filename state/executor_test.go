package state

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

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

	abciconv "github.com/dymensionxyz/dymint/conv/abci"
	"github.com/dymensionxyz/dymint/mempool"
	mempoolv1 "github.com/dymensionxyz/dymint/mempool/v1"
	"github.com/dymensionxyz/dymint/mocks"
	"github.com/dymensionxyz/dymint/types"
)

func TestCreateBlock(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	logger := log.TestingLogger()

	app := &mocks.Application{}
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})

	clientCreator := proxy.NewLocalClientCreator(app)
	abciClient, err := clientCreator.NewABCIClient()
	require.NoError(err)
	require.NotNil(clientCreator)
	require.NotNil(abciClient)

	nsID := "0102030405060708"

	mpool := mempoolv1.NewTxMempool(logger, cfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(abciClient), 0)
	executor := NewBlockExecutor([]byte("test address"), nsID, "test", mpool, proxy.NewAppConns(clientCreator), nil, logger)

	state := types.State{}
	state.ConsensusParams.Block.MaxBytes = 100
	state.ConsensusParams.Block.MaxGas = 100000
	state.Validators = tmtypes.NewValidatorSet(nil)

	// empty block
	block := executor.CreateBlock(1, &types.Commit{}, [32]byte{}, state)
	require.NotNil(block)
	assert.Empty(block.Data.Txs)
	assert.Equal(uint64(1), block.Header.Height)

	// one small Tx
	err = mpool.CheckTx([]byte{1, 2, 3, 4}, func(r *abci.Response) {}, mempool.TxInfo{})
	require.NoError(err)
	block = executor.CreateBlock(2, &types.Commit{}, [32]byte{}, state)
	require.NotNil(block)
	assert.Equal(uint64(2), block.Header.Height)
	assert.Len(block.Data.Txs, 1)

	// now there are 3 Txs, and only two can fit into single block
	err = mpool.CheckTx([]byte{4, 5, 6, 7}, func(r *abci.Response) {}, mempool.TxInfo{})
	require.NoError(err)
	err = mpool.CheckTx(make([]byte, 100), func(r *abci.Response) {}, mempool.TxInfo{})
	require.NoError(err)
	block = executor.CreateBlock(3, &types.Commit{}, [32]byte{}, state)
	require.NotNil(block)
	assert.Len(block.Data.Txs, 2)
}

func TestApplyBlock(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	logger := log.TestingLogger()

	// Mock ABCI app
	app := &mocks.Application{}
	app.On("CheckTx", mock.Anything).Return(abci.ResponseCheckTx{})
	app.On("BeginBlock", mock.Anything).Return(abci.ResponseBeginBlock{})
	app.On("DeliverTx", mock.Anything).Return(abci.ResponseDeliverTx{})
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{})
	var mockAppHash [32]byte
	_, err := rand.Read(mockAppHash[:])
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

	nsID := "0102030405060708"
	chainID := "test"

	// Init mempool
	mpool := mempoolv1.NewTxMempool(logger, cfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(abciClient), 0)
	eventBus := tmtypes.NewEventBus()
	require.NoError(eventBus.Start())

	// Mock app connections
	appConns := &mocks.AppConns{}
	appConns.On("Consensus").Return(abciClient)
	appConns.On("Query").Return(abciClient)
	executor := NewBlockExecutor([]byte("test address"), nsID, chainID, mpool, appConns, eventBus, logger)

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
	state := types.State{
		NextValidators: tmtypes.NewValidatorSet(nil),
		Validators:     tmtypes.NewValidatorSet(nil),
		LastValidators: tmtypes.NewValidatorSet(nil),
	}
	state.InitialHeight = 1
	state.LastBlockHeight = 0
	state.ConsensusParams.Block.MaxBytes = 100
	state.ConsensusParams.Block.MaxGas = 100000

	// Create first block with one Tx from mempool
	_ = mpool.CheckTx([]byte{1, 2, 3, 4}, func(r *abci.Response) {}, mempool.TxInfo{})
	require.NoError(err)
	block := executor.CreateBlock(1, &types.Commit{Height: 0}, [32]byte{}, state)
	require.NotNil(block)
	assert.Equal(uint64(1), block.Header.Height)
	assert.Len(block.Data.Txs, 1)

	// Create proposer for the block
	proposerKey := ed25519.GenPrivKey()
	proposer := &types.Sequencer{
		PublicKey: proposerKey.PubKey(),
	}
	// Create commit for the block
	abciHeaderPb := abciconv.ToABCIHeaderPB(&block.Header)
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
	err = executor.Validate(state, block, commit, proposer)
	require.NoError(err)
	resp, err := executor.Execute(context.Background(), state, block)
	require.NoError(err)
	require.NotNil(resp)
	newState, err := executor.UpdateStateFromResponses(resp, state, block)
	require.NoError(err)
	require.NotNil(newState)
	assert.Equal(int64(1), newState.LastBlockHeight)
	err = executor.Commit(context.Background(), &newState, block, resp)
	require.NoError(err)
	assert.Equal(mockAppHash, newState.AppHash)
	newState.LastStoreHeight = uint64(newState.LastBlockHeight)

	// Create another block with multiple Tx from mempool
	require.NoError(mpool.CheckTx([]byte{0, 1, 2, 3, 4}, func(r *abci.Response) {}, mempool.TxInfo{}))
	require.NoError(mpool.CheckTx([]byte{5, 6, 7, 8, 9}, func(r *abci.Response) {}, mempool.TxInfo{}))
	require.NoError(mpool.CheckTx([]byte{1, 2, 3, 4, 5}, func(r *abci.Response) {}, mempool.TxInfo{}))
	require.NoError(mpool.CheckTx(make([]byte, 90), func(r *abci.Response) {}, mempool.TxInfo{}))
	block = executor.CreateBlock(2, commit, [32]byte{}, newState)
	require.NotNil(block)
	assert.Equal(uint64(2), block.Header.Height)
	assert.Len(block.Data.Txs, 3)

	// Get the header bytes
	abciHeaderPb = abciconv.ToABCIHeaderPB(&block.Header)
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
	err = executor.Validate(state, block, invalidCommit, proposer)
	require.Error(types.ErrInvalidSignature)

	// Create a valid commit for the block
	signature, err = proposerKey.Sign(abciHeaderBytes)
	require.NoError(err)
	commit = &types.Commit{
		Height:     block.Header.Height,
		HeaderHash: block.Header.Hash(),
		Signatures: []types.Signature{signature},
	}

	// Apply the block
	err = executor.Validate(newState, block, commit, proposer)
	require.NoError(err)
	resp, err = executor.Execute(context.Background(), state, block)
	require.NoError(err)
	require.NotNil(resp)
	newState, err = executor.UpdateStateFromResponses(resp, state, block)
	require.NoError(err)
	require.NotNil(newState)
	assert.Equal(int64(2), newState.LastBlockHeight)
	err = executor.Commit(context.Background(), &newState, block, resp)
	require.NoError(err)

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
