package report

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"gonum.org/v1/gonum/stat"

	"github.com/dymensionxyz/dymint/test/loadtime/payload"
	"github.com/dymensionxyz/dymint/types"
)





type BlockStore interface {
	Height() uint64
	Base() uint64
	LoadBlock(uint64) (*types.Block, error)
}


type DataPoint struct {
	Duration  time.Duration
	BlockTime time.Time
	Hash      []byte
}



type Report struct {
	ID                      uuid.UUID
	Rate, Connections, Size uint64
	Max, Min, Avg, StdDev   time.Duration

	
	
	
	
	
	NegativeCount int

	
	TPS uint64

	
	
	
	All []DataPoint

	
	sum int64
}


type Reports struct {
	s map[uuid.UUID]Report
	l []Report

	
	
	
	errorCount int
}


func (rs *Reports) List() []Report {
	return rs.l
}


func (rs *Reports) ErrorCount() int {
	return rs.errorCount
}

func (rs *Reports) addDataPoint(id uuid.UUID, l time.Duration, bt time.Time, hash []byte, conns, rate, size uint64) {
	r, ok := rs.s[id]
	if !ok {
		r = Report{
			Max:         0,
			Min:         math.MaxInt64,
			ID:          id,
			Connections: conns,
			Rate:        rate,
			Size:        size,
		}
		rs.s[id] = r
	}
	r.All = append(r.All, DataPoint{Duration: l, BlockTime: bt, Hash: hash})
	if l > r.Max {
		r.Max = l
	}
	if l < r.Min {
		r.Min = l
	}
	if int64(l) < 0 {
		r.NegativeCount++
	}
	
	
	
	r.sum += int64(l)
	rs.s[id] = r
}

func (rs *Reports) calculateAll() {
	rs.l = make([]Report, 0, len(rs.s))
	for _, r := range rs.s {
		if len(r.All) == 0 {
			r.Min = 0
			rs.l = append(rs.l, r)
			continue
		}
		r.Avg = time.Duration(r.sum / int64(len(r.All)))
		r.StdDev = time.Duration(int64(stat.StdDev(toFloat(r.All), nil)))
		r.TPS = calculateTPS(r.All)
		rs.l = append(rs.l, r)
	}
}


func calculateTPS(in []DataPoint) uint64 {
	
	blocks := make(map[time.Time]int)
	for _, v := range in {
		blocks[v.BlockTime]++
	}
	
	var blockTimes []time.Time
	for k := range blocks {
		blockTimes = append(blockTimes, k)
	}
	sort.Slice(blockTimes, func(i, j int) bool {
		return blockTimes[i].Before(blockTimes[j])
	})
	
	TPS := uint64(0)
	for index, blockTime := range blockTimes {
		currentTx := blocks[blockTime]
		for _, nextBlockTime := range blockTimes[index+1:] {
			currentTx += blocks[nextBlockTime]
			blockTimeDifference := nextBlockTime.Sub(blockTime)
			if blockTimeDifference > time.Second {
				currentTPS := uint64(float64(currentTx) / blockTimeDifference.Seconds())
				if currentTPS > TPS {
					TPS = currentTPS
				}
			}
		}
	}

	return TPS
}

func (rs *Reports) addError() {
	rs.errorCount++
}



func GenerateFromBlockStore(s BlockStore) (*Reports, error) {
	type payloadData struct {
		id                      uuid.UUID
		l                       time.Duration
		bt                      time.Time
		hash                    []byte
		connections, rate, size uint64
		err                     error
	}
	type txData struct {
		tx types.Tx
		bt time.Time
	}
	reports := &Reports{
		s: make(map[uuid.UUID]Report),
	}

	
	
	
	
	
	const poolSize = 16

	txc := make(chan txData)
	pdc := make(chan payloadData, poolSize)

	wg := &sync.WaitGroup{}
	wg.Add(poolSize)
	for i := 0; i < poolSize; i++ {
		go func() {
			defer wg.Done()
			for b := range txc {
				p, err := payload.FromBytes(b.tx)
				if err != nil {
					pdc <- payloadData{err: err}
					continue
				}

				l := b.bt.Sub(p.Time)
				idb := (*[16]byte)(p.Id)
				pdc <- payloadData{
					l:           l,
					bt:          b.bt,
					hash:        b.tx.Hash(),
					id:          uuid.UUID(*idb),
					connections: p.Connections,
					rate:        p.Rate,
					size:        p.GetSize_(),
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(pdc)
	}()

	go func() {
		for i := s.Base(); i <= s.Height(); i++ {
			cur, err := s.LoadBlock(i)
			if err != nil {
				panic(err)
			}
			for _, tx := range cur.Data.Txs {
				utcFromUnixNano := cur.Header.GetTimestamp()
				txc <- txData{tx: tx, bt: utcFromUnixNano}
			}
		}
		close(txc)
	}()
	for pd := range pdc {
		if pd.err != nil {
			reports.addError()
			continue
		}
		reports.addDataPoint(pd.id, pd.l, pd.bt, pd.hash, pd.connections, pd.rate, pd.size)
	}
	reports.calculateAll()
	return reports, nil
}

func toFloat(in []DataPoint) []float64 {
	r := make([]float64, len(in))
	for i, v := range in {
		r[i] = float64(int64(v.Duration))
	}
	return r
}
