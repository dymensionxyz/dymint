package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/test/loadtime/report"
)

const (
	blockStoreDBName = "dymint"
)

var mainPrefix = [1]byte{0}

// BlockStore is a thin wrapper around the DefaultStore which will be used for inspecting the blocks
type BlockStore struct {
	*store.DefaultStore
	base   uint64
	height uint64
}

// Height implements report.BlockStore.
func (b *BlockStore) Height() uint64 {
	return b.height
}

// Base will be used to get the block height of the first block we want to generate the report for
func (b *BlockStore) Base() uint64 {
	return b.base
}

func getStore(directory string) *store.PrefixKV {
	dataDirectory := directory[strings.LastIndex(directory, "/")+1:]
	baseDirectory := directory[:len(directory)-len(dataDirectory)]
	baseKV := store.NewDefaultKVStore(baseDirectory, dataDirectory, blockStoreDBName)
	mainKV := store.NewPrefixKV(baseKV, mainPrefix[:])
	return mainKV
}

func newBlockStore(kvstore store.KV, baseHeight uint64) *BlockStore {
	s := store.New(kvstore)
	state, err := s.LoadState()
	if err != nil {
		log.Fatalf("loading state %s", err)
	}
	return &BlockStore{
		DefaultStore: s,
		base:         baseHeight,
		height:       state.Height(),
	}
}

var (
	dir        = flag.String("data-dir", "", "path to the directory containing the dymint database")
	csvOut     = flag.String("csv", "", "dump the extracted latencies as raw csv for use in additional tooling")
	baseHeight = flag.Uint64("base-height", 1, "base height to start the report from")
)

func main() {
	flag.Parse()
	if *dir == "" {
		log.Fatalf("must specify a data-dir")
	}
	d := strings.TrimPrefix(*dir, "~/")
	if d != *dir {
		h, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		d = h + "/" + d
	}
	if *baseHeight == 0 {
		*baseHeight = uint64(1)
	}
	_, err := os.Stat(d)
	if err != nil {
		panic(err)
	}
	mainKV := getStore(d)
	s := newBlockStore(mainKV, *baseHeight)
	rs, err := report.GenerateFromBlockStore(s)
	if err != nil {
		panic(err)
	}
	if *csvOut != "" {
		cf, err := os.Create(*csvOut)
		if err != nil {
			panic(err)
		}
		w := csv.NewWriter(cf)
		err = w.WriteAll(toCSVRecords(rs.List()))
		if err != nil {
			panic(err)
		}
		return
	}
	for _, r := range rs.List() {
		fmt.Printf(""+
			"Experiment ID: %s\n\n"+
			"\tConnections: %d\n"+
			"\tRate: %d\n"+
			"\tSize: %d\n\n"+
			"\tTotal Valid Tx: %d\n"+
			"\tTPS: %d\n"+
			"\tTotal Negative Latencies: %d\n"+
			"\tMinimum Latency: %s\n"+
			"\tMaximum Latency: %s\n"+
			"\tAverage Latency: %s\n"+
			"\tStandard Deviation: %s\n\n", r.ID, r.Connections, r.Rate, r.Size, len(r.All), r.TPS, r.NegativeCount, r.Min, r.Max, r.Avg, r.StdDev)
	}
	fmt.Printf("Total Invalid Tx: %d\n", rs.ErrorCount())
}

func toCSVRecords(rs []report.Report) [][]string {
	total := 0
	for _, v := range rs {
		total += len(v.All)
	}
	res := make([][]string, total+1)

	res[0] = []string{"experiment_id", "block_time", "duration_ns", "tx_hash", "connections", "rate", "size"}
	offset := 1
	for _, r := range rs {
		idStr := r.ID.String()
		connStr := strconv.FormatUint(r.Connections, 10)
		rateStr := strconv.FormatUint(r.Rate, 10)
		sizeStr := strconv.FormatUint(r.Size, 10)
		for i, v := range r.All {
			res[offset+i] = []string{idStr, strconv.FormatInt(v.BlockTime.UnixNano(), 10), strconv.FormatInt(int64(v.Duration), 10), fmt.Sprintf("%X", v.Hash), connStr, rateStr, sizeStr}
		}
		offset += len(r.All)
	}
	return res
}
