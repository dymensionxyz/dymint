package p2p_test

import (
	"encoding/hex"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	mempoolv1 "github.com/dymensionxyz/dymint/mempool/v1"
	"github.com/dymensionxyz/dymint/types"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/proxy"

	cfg "github.com/tendermint/tendermint/config"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/block"
	"github.com/dymensionxyz/dymint/mempool"

	tmmocks "github.com/dymensionxyz/dymint/mocks/github.com/tendermint/tendermint/abci/types"

	nodemempool "github.com/dymensionxyz/dymint/node/mempool"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/registry"
)

func TestValidator_TxValidator(t *testing.T) {
	type args struct {
		mp      mempool.Mempool
		numMsgs int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid: tx already in cache",
			args: args{
				mp:      &mockMP{err: mempool.ErrTxInCache},
				numMsgs: 3,
			},
			want: true,
		}, {
			name: "valid: mempool is full",
			args: args{
				mp:      &mockMP{err: mempool.ErrMempoolIsFull{}},
				numMsgs: 3,
			},
			want: true,
		}, {
			name: "invalid: tx too large",
			args: args{
				mp:      &mockMP{err: mempool.ErrTxTooLarge{}},
				numMsgs: 3,
			},
			want: false,
		}, {
			name: "invalid: pre-check error",
			args: args{
				mp:      &mockMP{err: mempool.ErrPreCheck{}},
				numMsgs: 3,
			},
			want: false,
		}, {
			name: "valid: no error",
			args: args{
				mp:      &mockMP{},
				numMsgs: 3,
			},
			want: true,
		}, {
			name: "unknown error",
			args: args{
				mp:      &mockMP{err: assert.AnError},
				numMsgs: 3,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.TestingLogger()
			validateTx := p2p.NewValidator(logger, nil).TxValidator(tt.args.mp, nodemempool.NewMempoolIDs())
			valid := validateTx(txMsg)
			assert.Equalf(t, tt.want, valid, "validateTx() = %v, want %v", valid, tt.want)
		})
	}
}

func TestValidator_BlockValidator(t *testing.T) {
	// Create proposer for the block
	proposerKey := ed25519.GenPrivKey()
	// Create another key
	attackerKey := ed25519.GenPrivKey()

	tests := []struct {
		name        string
		proposerKey *ed25519.PrivKey
		valid       bool
	}{
		{
			name:        "valid: block signed by proposer",
			proposerKey: proposerKey,
			valid:       true,
		}, {
			name:        "invalid: bad signer",
			proposerKey: attackerKey,
			valid:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.TestingLogger()

			// Create Block executor
			app := &tmmocks.MockApplication{}

			clientCreator := proxy.NewLocalClientCreator(app)
			abciClient, err := clientCreator.NewABCIClient()
			require.NoError(t, err)
			require.NotNil(t, clientCreator)
			require.NotNil(t, abciClient)
			namespaceId := "0102030405060708"
			mpool := mempoolv1.NewTxMempool(logger, cfg.DefaultMempoolConfig(), proxy.NewAppConnMempool(abciClient), 0)
			executor, err := block.NewExecutor([]byte("test address"), namespaceId, "test", mpool, proxy.NewAppConns(clientCreator), nil, logger)
			assert.NoError(t, err)

			// Create state
			maxBytes := uint64(100)
			state := types.State{}
			state.ConsensusParams.Block.MaxBytes = int64(maxBytes)
			state.ConsensusParams.Block.MaxGas = 100000
			state.Validators = tmtypes.NewValidatorSet(nil)

			// Create empty block
			block := executor.CreateBlock(1, &types.Commit{}, [32]byte{}, &state, maxBytes)

			// Create slclient
			client := registry.GetClient(registry.Local)
			pubsubServer := pubsub.NewServer()
			err = pubsubServer.Start()
			require.NoError(t, err)
			err = client.Init(settlement.Config{ProposerPubKey: hex.EncodeToString(proposerKey.PubKey().Bytes())}, pubsubServer, log.TestingLogger())
			require.NoError(t, err)

			// Create commit for the block
			abciHeaderPb := types.ToABCIHeaderPB(&block.Header)
			abciHeaderBytes, err := abciHeaderPb.Marshal()
			require.NoError(t, err)
			var signature []byte
			if tt.valid {
				signature, err = proposerKey.Sign(abciHeaderBytes)
				require.NoError(t, err)
			} else {
				signature, err = attackerKey.Sign(abciHeaderBytes)
				require.NoError(t, err)
			}
			commit := &types.Commit{
				Height:     block.Header.Height,
				HeaderHash: block.Header.Hash(),
				Signatures: []types.Signature{signature},
			}

			// Create gossiped block
			gossipedBlock := p2p.GossipedBlock{Block: *block, Commit: *commit}
			gossipedBlockBytes, err := gossipedBlock.MarshalBinary()
			require.NoError(t, err)
			blockMsg := &p2p.GossipMessage{
				Data: gossipedBlockBytes,
				From: peer.ID("from"),
			}

			// Check block validity
			validateBlock := p2p.NewValidator(logger, client).BlockValidator()
			valid := validateBlock(blockMsg)
			require.Equal(t, tt.valid, valid)
		})
	}
}

type mockMP struct {
	mempool.Mempool
	err error
}

func (m *mockMP) CheckTx(_ tmtypes.Tx, cb func(*abci.Response), _ mempool.TxInfo) error {
	if cb != nil {
		code := abci.CodeTypeOK
		if m.err != nil {
			code = 1
		}
		cb(&abci.Response{
			Value: &abci.Response_CheckTx{CheckTx: &abci.ResponseCheckTx{Code: code}},
		})
	}
	return m.err
}

var txMsg = &p2p.GossipMessage{
	Data: []byte("data"),
	From: peer.ID("from"),
}
