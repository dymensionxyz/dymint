package p2p

import (
	"context"
	"errors"

	"github.com/dymensionxyz/dymint/mempool"
	nodemempool "github.com/dymensionxyz/dymint/node/mempool"
	"github.com/dymensionxyz/dymint/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/pubsub"
	corep2p "github.com/tendermint/tendermint/p2p"
)

// GossipValidator is a callback function type.
type GossipValidator func(*GossipMessage) bool

// IValidator is an interface for implementing validators of messages gossiped in the p2p network.
type IValidator interface {
	// TxValidator creates a pubsub validator that uses the node's mempool to check the
	// transaction. If the transaction is valid, then it is added to the mempool
	TxValidator(mp mempool.Mempool, mpoolIDS *nodemempool.MempoolIDs) GossipValidator
}

// Validator is a validator for messages gossiped in the p2p network.
type Validator struct {
	logger            types.Logger
	localPubsubServer *pubsub.Server
}

var _ IValidator = (*Validator)(nil)

// NewValidator creates a new Validator.
func NewValidator(logger types.Logger, pusbsubServer *pubsub.Server) *Validator {
	return &Validator{
		logger:            logger,
		localPubsubServer: pusbsubServer,
	}
}

// TxValidator creates a pubsub validator that uses the node's mempool to check the
// transaction. If the transaction is valid, then it is added to the mempool.
func (v *Validator) TxValidator(mp mempool.Mempool, mpoolIDS *nodemempool.MempoolIDs) GossipValidator {
	return func(txMessage *GossipMessage) bool {
		v.logger.Debug("transaction received", "bytes", len(txMessage.Data))
		checkTxResCh := make(chan *abci.Response, 1)
		err := mp.CheckTx(txMessage.Data, func(resp *abci.Response) {
			checkTxResCh <- resp
		}, mempool.TxInfo{
			SenderID:    mpoolIDS.GetForPeer(txMessage.From),
			SenderP2PID: corep2p.ID(txMessage.From),
		})
		switch {
		case errors.Is(err, mempool.ErrTxInCache):
			return true
		case errors.Is(err, mempool.ErrMempoolIsFull{}):
			return true
		case errors.Is(err, mempool.ErrTxTooLarge{}):
			return false
		case errors.Is(err, mempool.ErrPreCheck{}):
			return false
		default:
		}
		res := <-checkTxResCh
		checkTxResp := res.GetCheckTx()

		return checkTxResp.Code == abci.CodeTypeOK
	}
}

// BlockValidator runs basic checks on the gossiped block
func (v *Validator) BlockValidator() GossipValidator {
	return func(blockMsg *GossipMessage) bool {
		v.logger.Debug("block event received", "from", blockMsg.From, "bytes", len(blockMsg.Data))
		var gossipedBlock GossipedBlock
		if err := gossipedBlock.UnmarshalBinary(blockMsg.Data); err != nil {
			v.logger.Error("deserialize gossiped block", "error", err)
			return false
		}
		if err := gossipedBlock.Validate(); err != nil {
			v.logger.Error("Invalid gossiped block", "error", err)
			return false
		}
		err := v.localPubsubServer.PublishWithEvents(context.Background(), gossipedBlock, map[string][]string{EventTypeKey: {EventNewGossipedBlock}})
		if err != nil {
			v.logger.Error("publishing event", "err", err)
			return false
		}
		return true
	}
}
