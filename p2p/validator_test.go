package p2p_test

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/mempool"
	nodemempool "github.com/dymensionxyz/dymint/node/mempool"
	"github.com/dymensionxyz/dymint/p2p"
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

type mockMP struct {
	mempool.Mempool
	err error
}

func (m *mockMP) CheckTx(_ types.Tx, cb func(*abci.Response), _ mempool.TxInfo) error {
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
