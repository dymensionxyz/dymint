package txindex

import (
	"context"
	"errors"

	indexer "github.com/dymensionxyz/dymint/indexers/blockindexer"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/tendermint/tendermint/libs/service"
	"github.com/tendermint/tendermint/types"
)

const (
	subscriber = "IndexerService"
)

type IndexerService struct {
	service.BaseService

	txIdxr    TxIndexer
	blockIdxr indexer.BlockIndexer
	eventBus  *types.EventBus
}

func NewIndexerService(
	txIdxr TxIndexer,
	blockIdxr indexer.BlockIndexer,
	eventBus *types.EventBus,
) *IndexerService {
	is := &IndexerService{txIdxr: txIdxr, blockIdxr: blockIdxr, eventBus: eventBus}
	is.BaseService = *service.NewBaseService(nil, "IndexerService", is)
	return is
}

func (is *IndexerService) OnStart() error {
	blockHeadersSub, err := is.eventBus.Subscribe(
		context.Background(),
		subscriber,
		types.EventQueryNewBlockHeader, 10)
	if err != nil {
		return err
	}

	txsSub, err := is.eventBus.Subscribe(context.Background(), subscriber, types.EventQueryTx, 10000)
	if err != nil {
		return err
	}

	go func() {
		for {
			msg := <-blockHeadersSub.Out()
			eventDataHeader, _ := msg.Data().(types.EventDataNewBlockHeader)
			height := eventDataHeader.Header.Height
			batch := NewBatch(eventDataHeader.NumTxs, height)

			for i := int64(0); i < eventDataHeader.NumTxs; i++ {
				msg2 := <-txsSub.Out()
				txResult := msg2.Data().(types.EventDataTx).TxResult

				if err = batch.Add(&txResult); err != nil {
					is.Logger.Error(
						"failed to add tx to batch",
						"height", height,
						"index", txResult.Index,
						"err", err,
					)
				}
			}

			if err := is.blockIdxr.Index(eventDataHeader); err != nil {
				is.Logger.Error("index block", "height", height, "err", err)
			} else {
				is.Logger.Debug("indexed block", "height", height)
			}

			if err = is.txIdxr.AddBatch(batch); err != nil {
				is.Logger.Error("index block txs", "height", height, "err", err)
			} else {
				is.Logger.Debug("indexed block txs", "height", height, "num_txs", eventDataHeader.NumTxs)
			}
		}
	}()
	return nil
}

func (is *IndexerService) OnStop() {
	if is.eventBus.IsRunning() {
		_ = is.eventBus.UnsubscribeAll(context.Background(), subscriber)
	}
}

func (is *IndexerService) Prune(to uint64, s store.Store) (uint64, error) {
	indexerBaseHeight, err := s.LoadIndexerBaseHeight()

	if errors.Is(err, gerrc.ErrNotFound) {
		is.Logger.Error("load indexer base height", "err", err)
	} else if err != nil {
		return 0, err
	}

	blockPruned, err := is.blockIdxr.Prune(indexerBaseHeight, to, is.Logger)
	if err != nil {
		return blockPruned, err
	}

	txPruned, err := is.txIdxr.Prune(indexerBaseHeight, to, is.Logger)
	if err != nil {
		return txPruned, err
	}

	err = s.SaveIndexerBaseHeight(to)
	if err != nil {
		is.Logger.Error("saving indexer base height", "err", err)
	}

	return blockPruned + txPruned, nil
}
