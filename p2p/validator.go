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

type StateGetter interface {
	GetProposerPubKey() tmcrypto.PubKey
	GetRevision() uint64
}

type GossipValidator func(*GossipMessage) bool

type IValidator interface {
	TxValidator(mp mempool.Mempool, mpoolIDS *nodemempool.MempoolIDs) GossipValidator
}

type Validator struct {
	logger      types.Logger
	stateGetter StateGetter
}

var _ IValidator = (*Validator)(nil)

func NewValidator(logger types.Logger, blockmanager StateGetter) *Validator {
	return &Validator{
		logger:      logger,
		stateGetter: blockmanager,
	}
}

func (v *Validator) TxValidator(mp mempool.Mempool, mpoolIDS *nodemempool.MempoolIDs) GossipValidator {
	return func(txMessage *GossipMessage) bool {
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
			return true
		case errors.Is(err, mempool.ErrTxTooLarge{}):
			return false
		case errors.Is(err, mempool.ErrPreCheck{}):
			return false
		case err != nil:
			return false
		}

		return res.GetCheckTx().Code == abci.CodeTypeOK
	}
}

func (v *Validator) BlockValidator() GossipValidator {
	return func(blockMsg *GossipMessage) bool {
		var gossipedBlock BlockData
		if err := gossipedBlock.UnmarshalBinary(blockMsg.Data); err != nil { // safe to be first?
			return false
		}
		if v.stateGetter.GetRevision() != gossipedBlock.Block.GetRevision() { // unfairly punished reputation?
			return false
		}
		if err := gossipedBlock.Validate(v.stateGetter.GetProposerPubKey()); err != nil {
			return false
		}

		return true
	}
}
