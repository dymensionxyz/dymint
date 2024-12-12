package mempool

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/types"
)

const (
	MempoolChannel = byte(0x30)

	
	PeerCatchupSleepIntervalMS = 100

	
	
	UnknownPeerID uint16 = 0

	MaxActiveIDs = math.MaxUint16
)





type Mempool interface {
	
	
	CheckTx(tx types.Tx, callback func(*abci.Response), txInfo TxInfo) error

	
	
	RemoveTxByKey(txKey types.TxKey) error

	
	
	
	
	
	
	ReapMaxBytesMaxGas(maxBytes, maxGas int64) types.Txs

	
	
	
	ReapMaxTxs(max int) types.Txs

	
	
	Lock()

	
	Unlock()

	
	
	
	
	
	
	Update(
		blockHeight int64,
		blockTxs types.Txs,
		deliverTxResponses []*abci.ResponseDeliverTx,
	) error

	
	SetPreCheckFn(fn PreCheckFunc)

	
	SetPostCheckFn(fn PostCheckFunc)

	
	
	
	
	
	FlushAppConn() error

	
	Flush()

	
	
	
	
	
	TxsAvailable() <-chan struct{}

	
	
	EnableTxsAvailable()

	
	Size() int

	
	SizeBytes() int64
}




type PreCheckFunc func(types.Tx) error




type PostCheckFunc func(types.Tx, *abci.ResponseCheckTx) error



func PreCheckMaxBytes(maxBytes int64) PreCheckFunc {
	return func(tx types.Tx) error {
		txSize := types.ComputeProtoSizeForTxs([]types.Tx{tx})

		if txSize > maxBytes {
			return fmt.Errorf("tx size is too big: %d, max: %d", txSize, maxBytes)
		}

		return nil
	}
}



func PostCheckMaxGas(maxGas int64) PostCheckFunc {
	return func(tx types.Tx, res *abci.ResponseCheckTx) error {
		if maxGas == -1 {
			return nil
		}
		if res.GasWanted < 0 {
			return fmt.Errorf("gas wanted %d is negative",
				res.GasWanted)
		}
		if res.GasWanted > maxGas {
			return fmt.Errorf("gas wanted %d is greater than max gas %d",
				res.GasWanted, maxGas)
		}

		return nil
	}
}


var ErrTxInCache = errors.New("tx already exists in cache")


type TxKey [sha256.Size]byte



type ErrTxTooLarge struct {
	Max    int
	Actual int
}

func (e ErrTxTooLarge) Error() string {
	return fmt.Sprintf("Tx too large. Max size is %d, but got %d", e.Max, e.Actual)
}



type ErrMempoolIsFull struct {
	NumTxs      int
	MaxTxs      int
	TxsBytes    int64
	MaxTxsBytes int64
}

func (e ErrMempoolIsFull) Error() string {
	return fmt.Sprintf(
		"mempool is full: number of txs %d (max: %d), total txs bytes %d (max: %d)",
		e.NumTxs,
		e.MaxTxs,
		e.TxsBytes,
		e.MaxTxsBytes,
	)
}


type ErrPreCheck struct {
	Reason error
}

func (e ErrPreCheck) Error() string {
	return e.Reason.Error()
}


func IsPreCheckError(err error) bool {
	return errors.As(err, &ErrPreCheck{})
}
