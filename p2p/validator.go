package p2p

import (
	"errors"

	"github.com/dymensionxyz/dymint/mempool"
	nodemempool "github.com/dymensionxyz/dymint/node/mempool"
	"github.com/dymensionxyz/dymint/types"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	corep2p "github.com/tendermint/tendermint/p2p"
)

type GetProposerI interface {
	GetExpectedProposerPubKey() (tmcrypto.PubKey, error)
}

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
	logger         types.Logger
	expectedPubKey GetProposerI
}

var _ IValidator = (*Validator)(nil)

// NewValidator creates a new Validator.
func NewValidator(logger types.Logger, blockmanager GetProposerI) *Validator {
	return &Validator{
		logger:         logger,
		expectedPubKey: blockmanager,
	}
}

// TxValidator creates a pubsub validator that uses the node's mempool to check the
// transaction.
// False means the TX is considered invalid and should not be gossiped.
func (v *Validator) TxValidator(mp mempool.Mempool, mpoolIDS *nodemempool.MempoolIDs) GossipValidator {
	return func(txMessage *GossipMessage) bool {
		v.logger.Debug("Transaction received.", "bytes", len(txMessage.Data))
		var res *abci.Response
		err := mp.CheckTx(txMessage.Data, func(resp *abci.Response) {
			res = resp
		}, mempool.TxInfo{
			SenderID:    mpoolIDS.GetForPeer(txMessage.From),
			SenderP2PID: corep2p.ID(txMessage.From),
		})
		switch {
		case errors.Is(err, mempool.ErrTxInCache):
			return true
		case errors.Is(err, mempool.ErrMempoolIsFull{}):
			return true // we have no reason to believe that we should throw away the message
		case errors.Is(err, mempool.ErrTxTooLarge{}):
			return false
		case errors.Is(err, mempool.ErrPreCheck{}):
			return false
		case err != nil:
			v.logger.Error("Check tx.", "error", err)
			return false
		}

		return res.GetCheckTx().Code == abci.CodeTypeOK
	}
}

// BlockValidator runs basic checks on the gossiped block
func (v *Validator) BlockValidator() GossipValidator {
	return func(blockMsg *GossipMessage) bool {
		var gossipedBlock GossipedBlock
		if err := gossipedBlock.UnmarshalBinary(blockMsg.Data); err != nil {
			v.logger.Error("Deserialize gossiped block.", "error", err)
			return false
		}

		if err := gossipedBlock.Validate(); err != nil {
			v.logger.Error("Failed to validate gossiped block.", "height", gossipedBlock.Block.Header.Height, "error", err)
			return false
		}

		return true
	}
}
