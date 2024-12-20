package block_test

import (
	"context"
	"testing"

	proto "github.com/gogo/protobuf/types"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/proxy"
)

// produce a block, mutate it, confirm the sig check fails
func TestProduceNewBlockMutation(t *testing.T) {
	// Init app
	app := testutil.GetAppMock(testutil.Commit, testutil.EndBlock)
	commitHash := [32]byte{1}
	app.On("Commit", mock.Anything).Return(abci.ResponseCommit{Data: commitHash[:]})
	app.On("EndBlock", mock.Anything).Return(abci.ResponseEndBlock{
		RollappParamUpdates: &abci.RollappParams{
			Da:         "mock",
			DrsVersion: 0,
		},
		ConsensusParamUpdates: &abci.ConsensusParams{
			Block: &abci.BlockParams{
				MaxGas:   40000000,
				MaxBytes: 500000,
			},
		},
	})
	// Create proxy app
	clientCreator := proxy.NewLocalClientCreator(app)
	proxyApp := proxy.NewAppConns(clientCreator)
	err := proxyApp.Start()
	require.NoError(t, err)
	// Init manager
	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, proxyApp, nil)
	require.NoError(t, err)
	// Produce block
	b, c, err := manager.ProduceApplyGossipBlock(context.Background(), block.ProduceBlockOptions{AllowEmpty: true})
	require.NoError(t, err)
	// Validate state is updated with the commit hash
	assert.Equal(t, uint64(1), manager.State.Height())
	assert.Equal(t, commitHash, manager.State.AppHash)

	err = b.ValidateBasic()
	require.NoError(t, err)

	// TODO: better way to write test is to clone the block and commit and check a table of mutations
	bd := p2p.BlockData{
		Block:  *b,
		Commit: *c,
	}
	err = bd.Validate(manager.State.GetProposerPubKey())
	require.NoError(t, err)

	b.Data.ConsensusMessages = []*proto.Any{{}}
	bd = p2p.BlockData{
		Block:  *b,
		Commit: *c,
	}
	err = bd.Validate(manager.State.GetProposerPubKey())
	require.Error(t, err)
}
