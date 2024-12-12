package v1

import (
	"sync"
	"time"

	"github.com/tendermint/tendermint/types"
)



type WrappedTx struct {
	tx        types.Tx    
	hash      types.TxKey 
	height    int64       
	timestamp time.Time   

	mtx       sync.Mutex
	gasWanted int64           
	priority  int64           
	sender    string          
	peers     map[uint16]bool 
}


func (w *WrappedTx) Size() int64 { return int64(len(w.tx)) }


func (w *WrappedTx) SetPeer(id uint16) {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	if w.peers == nil {
		w.peers = map[uint16]bool{id: true}
	} else {
		w.peers[id] = true
	}
}


func (w *WrappedTx) HasPeer(id uint16) bool {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	_, ok := w.peers[id]
	return ok
}


func (w *WrappedTx) SetGasWanted(gas int64) {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	w.gasWanted = gas
}


func (w *WrappedTx) GasWanted() int64 {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return w.gasWanted
}


func (w *WrappedTx) SetSender(sender string) {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	w.sender = sender
}


func (w *WrappedTx) Sender() string {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return w.sender
}


func (w *WrappedTx) SetPriority(p int64) {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	w.priority = p
}


func (w *WrappedTx) Priority() int64 {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return w.priority
}
