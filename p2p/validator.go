package p2p

import (
	"errors"

	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/mempool"
	nodemempool "github.com/dymensionxyz/dymint/node/mempool"
	abci "github.com/tendermint/tendermint/abci/types"
	corep2p "github.com/tendermint/tendermint/p2p"
)

// GossipValidator is a callback function type.
type GossipValidator func(*GossipMessage) bool

// IValidator is an interface for implementing validators of messages gossiped in the p2p network.
type IValidator interface {
	// Tx creates a pubsub validator that uses the node's mempool to check the
	// transaction. If the transaction is valid, then it is added to the mempool
	Tx(mp mempool.Mempool, mpoolIDS *nodemempool.MempoolIDs) GossipValidator
}

// Validator is a validator for messages gossiped in the p2p network.
type Validator struct {
	logger log.Logger
}

var _ IValidator = (*Validator)(nil)

// NewValidator creates a new Validator.
func NewValidator(logger log.Logger) *Validator {
	return &Validator{
		logger: logger,
	}
}

// Tx creates a pubsub validator that uses the node's mempool to check the
// transaction. If the transaction is valid, then it is added to the mempool.
func (v *Validator) Tx(mp mempool.Mempool, mpoolIDS *nodemempool.MempoolIDs) GossipValidator {
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
