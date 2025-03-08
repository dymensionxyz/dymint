package kv

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	"github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/store"
)

func BenchmarkTxSearch(b *testing.B) {
	dbDir := b.TempDir()

	db := store.NewDefaultKVStore(dbDir, "db", "benchmark_tx_search_test")

	indexer := NewTxIndex(db)

	for i := 0; i < 35000; i++ {
		events := []abci.Event{
			{
				Type: "transfer",
				Attributes: []abci.EventAttribute{
					{Key: []byte("address"), Value: []byte(fmt.Sprintf("address_%d", i%100)), Index: true},
					{Key: []byte("amount"), Value: []byte("50"), Index: true},
				},
			},
		}

		txBz := make([]byte, 8)
		if _, err := rand.Read(txBz); err != nil {
			b.Errorf("failed produce random bytes: %s", err)
		}

		txResult := &abci.TxResult{
			Height: int64(i),
			Index:  0,
			Tx:     types.Tx(string(txBz)),
			Result: abci.ResponseDeliverTx{
				Data:   []byte{0},
				Code:   abci.CodeTypeOK,
				Log:    "",
				Events: events,
			},
		}

		if err := indexer.Index(txResult); err != nil {
			b.Errorf("index tx: %s", err)
		}
	}

	txQuery := query.MustParse("transfer.address = 'address_43' AND transfer.amount = 50")

	b.ResetTimer()

	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		if _, err := indexer.Search(ctx, txQuery); err != nil {
			b.Errorf("query for txs: %s", err)
		}
	}
}
