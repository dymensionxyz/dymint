package p2p

import (
	"errors"

	"github.com/dymensionxyz/dymint/mempool"
	nodemempool "github.com/dymensionxyz/dymint/node/mempool"
	"github.com/dymensionxyz/dymint/types"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	corep2p "github.com/tendermint/tendermint/p2p"
)

type StateGetter interface {
	SafeProposerPubKey() (tmcrypto.PubKey, error)
	GetRevision() uint64
}

// GossipValidator is a callback function type.
type GossipValidator func(*GossipMessage) pubsub.ValidationResult

// IValidator is an interface for implementing validators of messages gossiped in the p2p network.
type IValidator interface {
	// TxValidator creates a pubsub validator that uses the node's mempool to check the
	// transaction. If the transaction is want, then it is added to the mempool
	TxValidator(mp mempool.Mempool, mpoolIDS *nodemempool.MempoolIDs) GossipValidator
}

// Validator is a validator for messages gossiped in the p2p network.
type Validator struct {
	logger      types.Logger
	stateGetter StateGetter
}

var _ IValidator = (*Validator)(nil)

// NewValidator creates a new Validator.
func NewValidator(logger types.Logger, blockmanager StateGetter) *Validator {
	return &Validator{
		logger:      logger,
		stateGetter: blockmanager,
	}
}

// TxValidator creates a pubsub validator that uses the node's mempool to check the
// transaction.
// False means the TX is considered invalid and should not be gossiped.
func (v *Validator) TxValidator(mp mempool.Mempool, mpoolIDS *nodemempool.MempoolIDs) GossipValidator {
	return func(txMessage *GossipMessage) pubsub.ValidationResult {
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
			return pubsub.ValidationAccept
		case errors.Is(err, mempool.ErrMempoolIsFull{}):
			return pubsub.ValidationAccept // we have no reason to believe that we should throw away the message
		case errors.Is(err, mempool.ErrTxTooLarge{}):
			return pubsub.ValidationReject
		case errors.Is(err, mempool.ErrPreCheck{}):
			return pubsub.ValidationReject
		case err != nil:
			v.logger.Error("Check tx.", "error", err)
			return pubsub.ValidationReject
		}

		if res.GetCheckTx().Code == abci.CodeTypeOK {
			return pubsub.ValidationAccept
		}
		return pubsub.ValidationReject
	}
}

// BlockValidator runs basic checks on the gossiped block
func (v *Validator) BlockValidator() GossipValidator {
	return func(blockMsg *GossipMessage) pubsub.ValidationResult {
		var gossipedBlock BlockData
		if err := gossipedBlock.UnmarshalBinary(blockMsg.Data); err != nil {
			v.logger.Error("Deserialize gossiped block.", "error", err)
			return pubsub.ValidationReject
		}
		if v.stateGetter.GetRevision() != gossipedBlock.Block.GetRevision() {
			return pubsub.ValidationReject
		}
		propKey, err := v.stateGetter.SafeProposerPubKey()
		if err != nil {
			v.logger.Error("Get proposer pubkey.", "error", err)
			return pubsub.ValidationIgnore
		}
		if err := gossipedBlock.Validate(propKey); err != nil {
			v.logger.Error("P2P block validation.", "height", gossipedBlock.Block.Header.Height, "err", err)
			return pubsub.ValidationReject
		}

		return pubsub.ValidationAccept
	}
}
