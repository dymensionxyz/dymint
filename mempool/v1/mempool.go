package v1

import (
	"fmt"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/proxy"
	"github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/mempool/clist"
)

var _ mempool.Mempool = (*TxMempool)(nil)

type TxMempoolOption func(*TxMempool)

type TxMempool struct {
	logger       log.Logger
	config       *config.MempoolConfig
	proxyAppConn proxy.AppConnMempool
	metrics      *mempool.Metrics
	cache        mempool.TxCache

	txsBytes  int64
	txRecheck int64

	mtx                  *sync.RWMutex
	notifiedTxsAvailable bool
	txsAvailable         chan struct{}
	preCheck             mempool.PreCheckFunc
	postCheck            mempool.PostCheckFunc
	height               int64

	txs        *clist.CList
	txByKey    map[types.TxKey]*clist.CElement
	txBySender map[string]*clist.CElement
}

func NewTxMempool(
	logger log.Logger,
	cfg *config.MempoolConfig,
	proxyAppConn proxy.AppConnMempool,
	height int64,
	options ...TxMempoolOption,
) *TxMempool {
	txmp := &TxMempool{
		logger:       logger,
		config:       cfg,
		proxyAppConn: proxyAppConn,
		metrics:      mempool.NopMetrics(),
		cache:        mempool.NopTxCache{},
		txs:          clist.New(),
		mtx:          new(sync.RWMutex),
		height:       height,
		txByKey:      make(map[types.TxKey]*clist.CElement),
		txBySender:   make(map[string]*clist.CElement),
	}
	if cfg.CacheSize > 0 {
		txmp.cache = mempool.NewLRUTxCache(cfg.CacheSize)
	}

	proxyAppConn.SetResponseCallback(txmp.recheckTxCallback)

	for _, opt := range options {
		opt(txmp)
	}

	return txmp
}

func WithPreCheck(f mempool.PreCheckFunc) TxMempoolOption {
	return func(txmp *TxMempool) { txmp.preCheck = f }
}

func WithPostCheck(f mempool.PostCheckFunc) TxMempoolOption {
	return func(txmp *TxMempool) { txmp.postCheck = f }
}

func WithMetrics(metrics *mempool.Metrics) TxMempoolOption {
	return func(txmp *TxMempool) { txmp.metrics = metrics }
}

func (txmp *TxMempool) Lock() { txmp.mtx.Lock() }

func (txmp *TxMempool) Unlock() { txmp.mtx.Unlock() }

func (txmp *TxMempool) Size() int { return txmp.txs.Len() }

func (txmp *TxMempool) SizeBytes() int64 { return atomic.LoadInt64(&txmp.txsBytes) }

func (txmp *TxMempool) FlushAppConn() error {
	txmp.mtx.Unlock()
	defer txmp.mtx.Lock()

	return txmp.proxyAppConn.FlushSync()
}

func (txmp *TxMempool) EnableTxsAvailable() {
	txmp.mtx.Lock()
	defer txmp.mtx.Unlock()

	txmp.txsAvailable = make(chan struct{}, 1)
}

func (txmp *TxMempool) TxsAvailable() <-chan struct{} { return txmp.txsAvailable }

func (txmp *TxMempool) CheckTx(tx types.Tx, cb func(*abci.Response), txInfo mempool.TxInfo) error {
	height, err := func() (int64, error) {
		txmp.mtx.RLock()
		defer txmp.mtx.RUnlock()

		if len(tx) > txmp.config.MaxTxBytes {
			return 0, mempool.ErrTxTooLarge{Max: txmp.config.MaxTxBytes, Actual: len(tx)}
		}

		if txmp.preCheck != nil {
			if err := txmp.preCheck(tx); err != nil {
				return 0, mempool.ErrPreCheck{Reason: err}
			}
		}

		if err := txmp.proxyAppConn.Error(); err != nil {
			return 0, err
		}

		txKey := tx.Key()

		if !txmp.cache.Push(tx) {

			if elt, ok := txmp.txByKey[txKey]; ok {
				w, _ := elt.Value.(*WrappedTx)
				w.SetPeer(txInfo.SenderID)
			}
			return 0, mempool.ErrTxInCache
		}
		return txmp.height, nil
	}()
	if err != nil {
		return err
	}

	reqRes := txmp.proxyAppConn.CheckTxAsync(abci.RequestCheckTx{Tx: tx})
	if err := txmp.proxyAppConn.FlushSync(); err != nil {
		return err
	}
	reqRes.SetCallback(func(res *abci.Response) {
		wtx := &WrappedTx{
			tx:        tx,
			hash:      tx.Key(),
			timestamp: time.Now().UTC(),
			height:    height,
		}
		wtx.SetPeer(txInfo.SenderID)
		txmp.initialTxCallback(wtx, res)
		if cb != nil {
			cb(res)
		}
	})
	return nil
}

func (txmp *TxMempool) RemoveTxByKey(txKey types.TxKey) error {
	txmp.mtx.Lock()
	defer txmp.mtx.Unlock()
	return txmp.removeTxByKey(txKey)
}

func (txmp *TxMempool) removeTxByKey(key types.TxKey) error {
	if elt, ok := txmp.txByKey[key]; ok {
		w, _ := elt.Value.(*WrappedTx)
		delete(txmp.txByKey, key)
		delete(txmp.txBySender, w.sender)
		txmp.txs.Remove(elt)
		elt.DetachPrev()
		elt.DetachNext()
		atomic.AddInt64(&txmp.txsBytes, -w.Size())
		return nil
	}
	return fmt.Errorf("transaction %x not found", key)
}

func (txmp *TxMempool) removeTxByElement(elt *clist.CElement) {
	w, _ := elt.Value.(*WrappedTx)
	delete(txmp.txByKey, w.tx.Key())
	delete(txmp.txBySender, w.sender)
	txmp.txs.Remove(elt)
	elt.DetachPrev()
	elt.DetachNext()
	atomic.AddInt64(&txmp.txsBytes, -w.Size())
}

func (txmp *TxMempool) Flush() {
	txmp.mtx.Lock()
	defer txmp.mtx.Unlock()

	cur := txmp.txs.Front()
	for cur != nil {
		next := cur.Next()
		txmp.removeTxByElement(cur)
		cur = next
	}
	txmp.cache.Reset()

	atomic.StoreInt64(&txmp.txRecheck, 0)
}

func (txmp *TxMempool) allEntriesSorted() []*WrappedTx {
	txmp.mtx.RLock()
	defer txmp.mtx.RUnlock()

	all := make([]*WrappedTx, 0, len(txmp.txByKey))
	for _, tx := range txmp.txByKey {
		all = append(all, tx.Value.(*WrappedTx))
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].priority == all[j].priority {
			return all[i].timestamp.Before(all[j].timestamp)
		}
		return all[i].priority > all[j].priority
	})
	return all
}

func (txmp *TxMempool) ReapMaxBytesMaxGas(maxBytes, maxGas int64) types.Txs {
	var totalGas, totalBytes int64

	var keep []types.Tx
	for _, w := range txmp.allEntriesSorted() {

		totalGas += w.gasWanted
		totalBytes += types.ComputeProtoSizeForTxs([]types.Tx{w.tx})
		if (maxGas >= 0 && totalGas > maxGas) || (maxBytes >= 0 && totalBytes > maxBytes) {
			break
		}
		keep = append(keep, w.tx)
	}
	return keep
}

func (txmp *TxMempool) TxsWaitChan() <-chan struct{} { return txmp.txs.WaitChan() }

func (txmp *TxMempool) TxsFront() *clist.CElement { return txmp.txs.Front() }

func (txmp *TxMempool) ReapMaxTxs(max int) types.Txs {
	var keep []types.Tx

	for _, w := range txmp.allEntriesSorted() {
		if max >= 0 && len(keep) >= max {
			break
		}
		keep = append(keep, w.tx)
	}
	return keep
}

func (txmp *TxMempool) Update(
	blockHeight int64,
	blockTxs types.Txs,
	deliverTxResponses []*abci.ResponseDeliverTx,
) error {
	if txmp.mtx.TryLock() {
		txmp.mtx.Unlock()
		panic("mempool: Update caller does not hold the lock")
	}

	if len(blockTxs) != len(deliverTxResponses) {
		panic(fmt.Sprintf("mempool: got %d transactions but %d DeliverTx responses",
			len(blockTxs), len(deliverTxResponses)))
	}

	txmp.height = blockHeight
	txmp.notifiedTxsAvailable = false

	for i, tx := range blockTxs {

		if deliverTxResponses[i].Code == abci.CodeTypeOK {
			_ = txmp.cache.Push(tx)
		} else if !txmp.config.KeepInvalidTxsInCache {
			txmp.cache.Remove(tx)
		}

		_ = txmp.removeTxByKey(tx.Key())
	}

	txmp.purgeExpiredTxs(blockHeight)

	size := txmp.Size()
	txmp.metrics.Size.Set(float64(size))
	if size > 0 {
		if txmp.config.Recheck {
			txmp.recheckTransactions()
		} else {
			txmp.notifyTxsAvailable()
		}
	}
	return nil
}

func (txmp *TxMempool) SetPreCheckFn(fn mempool.PreCheckFunc) {
	txmp.preCheck = fn
}

func (txmp *TxMempool) SetPostCheckFn(fn mempool.PostCheckFunc) {
	txmp.postCheck = fn
}

func (txmp *TxMempool) initialTxCallback(wtx *WrappedTx, res *abci.Response) {
	checkTxRes, ok := res.Value.(*abci.Response_CheckTx)
	if !ok {
		txmp.logger.Error("mempool: received incorrect result type in CheckTx callback",
			"expected", reflect.TypeOf(&abci.Response_CheckTx{}).Name(),
			"got", reflect.TypeOf(res.Value).Name(),
		)
		return
	}

	txmp.mtx.Lock()
	defer txmp.mtx.Unlock()

	var err error
	if txmp.postCheck != nil {
		err = txmp.postCheck(wtx.tx, checkTxRes.CheckTx)
	}

	if err != nil || checkTxRes.CheckTx.Code != abci.CodeTypeOK {
		txmp.logger.Info(
			"rejected bad transaction",
			"priority", wtx.Priority(),
			"tx", fmt.Sprintf("%X", wtx.tx.Hash()),
			"peer_id", wtx.peers,
			"code", checkTxRes.CheckTx.Code,
			"post_check_err", err,
			"log", checkTxRes.CheckTx.Log,
		)

		txmp.metrics.FailedTxs.Add(1)

		if !txmp.config.KeepInvalidTxsInCache {
			txmp.cache.Remove(wtx.tx)
		}

		if err != nil {
			checkTxRes.CheckTx.MempoolError = err.Error()
		}
		return
	}

	priority := checkTxRes.CheckTx.Priority
	sender := checkTxRes.CheckTx.Sender

	if sender != "" {
		elt, ok := txmp.txBySender[sender]
		if ok {
			w := elt.Value.(*WrappedTx)
			txmp.logger.Debug(
				"rejected valid incoming transaction; tx already exists for sender",
				"tx", fmt.Sprintf("%X", w.tx.Hash()),
				"sender", sender,
			)
			checkTxRes.CheckTx.MempoolError = fmt.Sprintf("rejected valid incoming transaction; tx already exists for sender %q (%X)",
				sender, w.tx.Hash())
			txmp.metrics.RejectedTxs.Add(1)
			return
		}
	}

	if err := txmp.canAddTx(wtx); err != nil {
		var victims []*clist.CElement
		var victimBytes int64
		for cur := txmp.txs.Front(); cur != nil; cur = cur.Next() {
			cw := cur.Value.(*WrappedTx)
			if cw.priority < priority {
				victims = append(victims, cur)
				victimBytes += cw.Size()
			}
		}

		if len(victims) == 0 || victimBytes < wtx.Size() {
			txmp.cache.Remove(wtx.tx)
			txmp.logger.Error(
				"rejected valid incoming transaction; mempool is full",
				"tx", fmt.Sprintf("%X", wtx.tx.Hash()),
				"err", err.Error(),
			)
			checkTxRes.CheckTx.MempoolError = fmt.Sprintf("rejected valid incoming transaction; mempool is full (%X)",
				wtx.tx.Hash())
			txmp.metrics.RejectedTxs.Add(1)
			return
		}

		txmp.logger.Debug("evicting lower-priority transactions",
			"new_tx", fmt.Sprintf("%X", wtx.tx.Hash()),
			"new_priority", priority,
		)

		sort.Slice(victims, func(i, j int) bool {
			iw := victims[i].Value.(*WrappedTx)
			jw := victims[j].Value.(*WrappedTx)
			if iw.Priority() == jw.Priority() {
				return iw.timestamp.After(jw.timestamp)
			}
			return iw.Priority() < jw.Priority()
		})

		var evictedBytes int64
		for _, vic := range victims {
			w := vic.Value.(*WrappedTx)

			txmp.logger.Debug(
				"evicted valid existing transaction; mempool full",
				"old_tx", fmt.Sprintf("%X", w.tx.Hash()),
				"old_priority", w.priority,
			)
			txmp.removeTxByElement(vic)
			txmp.cache.Remove(w.tx)
			txmp.metrics.EvictedTxs.Add(1)

			evictedBytes += w.Size()
			if evictedBytes >= wtx.Size() {
				break
			}
		}
	}

	wtx.SetGasWanted(checkTxRes.CheckTx.GasWanted)
	wtx.SetPriority(priority)
	wtx.SetSender(sender)
	txmp.insertTx(wtx)

	txmp.metrics.TxSizeBytes.Observe(float64(wtx.Size()))
	txmp.metrics.Size.Set(float64(txmp.Size()))
	txmp.logger.Debug(
		"inserted new valid transaction",
		"priority", wtx.Priority(),
		"tx", fmt.Sprintf("%X", wtx.tx.Hash()),
		"height", txmp.height,
		"num_txs", txmp.Size(),
	)
	txmp.notifyTxsAvailable()
}

func (txmp *TxMempool) insertTx(wtx *WrappedTx) {
	elt := txmp.txs.PushBack(wtx)
	txmp.txByKey[wtx.tx.Key()] = elt
	if s := wtx.Sender(); s != "" {
		txmp.txBySender[s] = elt
	}

	atomic.AddInt64(&txmp.txsBytes, wtx.Size())
}

func (txmp *TxMempool) recheckTxCallback(req *abci.Request, res *abci.Response) {
	checkTxRes, ok := res.Value.(*abci.Response_CheckTx)
	if !ok {
		return
	}

	numLeft := atomic.AddInt64(&txmp.txRecheck, -1)
	if numLeft == 0 {
		defer txmp.notifyTxsAvailable()
	} else if numLeft < 0 {
		return
	}

	txmp.metrics.RecheckTimes.Add(1)
	tx := types.Tx(req.GetCheckTx().Tx)

	txmp.mtx.Lock()
	defer txmp.mtx.Unlock()

	elt, ok := txmp.txByKey[tx.Key()]
	if !ok {
		return
	}
	wtx := elt.Value.(*WrappedTx)

	var err error
	if txmp.postCheck != nil {
		err = txmp.postCheck(tx, checkTxRes.CheckTx)
	}

	if checkTxRes.CheckTx.Code == abci.CodeTypeOK && err == nil {
		wtx.SetPriority(checkTxRes.CheckTx.Priority)
		return
	}

	txmp.logger.Debug(
		"existing transaction no longer valid; failed re-CheckTx callback",
		"priority", wtx.Priority(),
		"tx", fmt.Sprintf("%X", wtx.tx.Hash()),
		"err", err,
		"code", checkTxRes.CheckTx.Code,
	)
	txmp.removeTxByElement(elt)
	txmp.metrics.FailedTxs.Add(1)
	if !txmp.config.KeepInvalidTxsInCache {
		txmp.cache.Remove(wtx.tx)
	}
	txmp.metrics.Size.Set(float64(txmp.Size()))
}

func (txmp *TxMempool) recheckTransactions() {
	if txmp.Size() == 0 {
		panic("mempool: cannot run recheck on an empty mempool")
	}
	txmp.logger.Debug(
		"executing re-CheckTx for all remaining transactions",
		"num_txs", txmp.Size(),
		"height", txmp.height,
	)

	txmp.mtx.Unlock()
	defer txmp.mtx.Lock()

	atomic.StoreInt64(&txmp.txRecheck, int64(txmp.txs.Len()))
	for e := txmp.txs.Front(); e != nil; e = e.Next() {
		wtx := e.Value.(*WrappedTx)

		_ = txmp.proxyAppConn.CheckTxAsync(abci.RequestCheckTx{
			Tx:   wtx.tx,
			Type: abci.CheckTxType_Recheck,
		})
		if err := txmp.proxyAppConn.FlushSync(); err != nil {
			atomic.AddInt64(&txmp.txRecheck, -1)
			txmp.logger.Error("mempool: error flushing re-CheckTx", "key", wtx.tx.Key(), "err", err)
		}
	}

	txmp.proxyAppConn.FlushAsync()
}

func (txmp *TxMempool) canAddTx(wtx *WrappedTx) error {
	numTxs := txmp.Size()
	txBytes := txmp.SizeBytes()

	if numTxs >= txmp.config.Size || wtx.Size()+txBytes > txmp.config.MaxTxsBytes {
		return mempool.ErrMempoolIsFull{
			NumTxs:      numTxs,
			MaxTxs:      txmp.config.Size,
			TxsBytes:    txBytes,
			MaxTxsBytes: txmp.config.MaxTxsBytes,
		}
	}

	return nil
}

func (txmp *TxMempool) purgeExpiredTxs(blockHeight int64) {
	if txmp.config.TTLNumBlocks == 0 && txmp.config.TTLDuration == 0 {
		return
	}

	now := time.Now()
	cur := txmp.txs.Front()
	for cur != nil {

		next := cur.Next()

		w := cur.Value.(*WrappedTx)
		if txmp.config.TTLNumBlocks > 0 && (blockHeight-w.height) > txmp.config.TTLNumBlocks {
			txmp.removeTxByElement(cur)
			txmp.cache.Remove(w.tx)
			txmp.metrics.EvictedTxs.Add(1)
		} else if txmp.config.TTLDuration > 0 && now.Sub(w.timestamp) > txmp.config.TTLDuration {
			txmp.removeTxByElement(cur)
			txmp.cache.Remove(w.tx)
			txmp.metrics.EvictedTxs.Add(1)
		}
		cur = next
	}
}

func (txmp *TxMempool) notifyTxsAvailable() {
	if txmp.Size() == 0 {
		return
	}

	if txmp.txsAvailable != nil && !txmp.notifiedTxsAvailable {

		txmp.notifiedTxsAvailable = true

		select {
		case txmp.txsAvailable <- struct{}{}:
		default:
		}
	}
}
