package block_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/block"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

type testArgs struct {
	nParallel                 int           // number of instances to run in parallel
	testDuration              time.Duration // how long to run one instance of the test (should be short)
	batchSkew                 time.Duration // time between last block produced and submitted
	skewMargin                time.Duration // skew margin allowed
	batchBytes                uint64        // max number of bytes in a batch
	maxTime                   time.Duration // maximum time to wait before submitting submissions
	submitTime                time.Duration // how long it takes to submit a batch
	produceTime               time.Duration // time between producing block
	produceBytes              int           // range of how many bytes each block (+ commit) is
	submissionHaltTime        time.Duration // how long to simulate batch submission failing/halting
	submissionHaltProbability float64       // probability of submission failing and causing a temporary halt
}

func testSubmitLoop(t *testing.T,
	args testArgs,
) {
	var wg sync.WaitGroup
	for range args.nParallel {
		wg.Add(1)
		go func() {
			testSubmitLoopInner(t, args)
			wg.Done()
		}()
	}
	wg.Wait()
}

func testSubmitLoopInner(
	t *testing.T,
	args testArgs,
) {
	ctx, cancel := context.WithTimeout(context.Background(), args.testDuration)
	defer cancel()

	// returns a duration in [0.8,1.2] * d
	approx := func(d time.Duration) time.Duration {
		base := int(float64(d) * 0.8)
		factor := int(float64(d) * 0.4)
		return time.Duration(base + rand.Intn(factor))
	}

	pendingBlocks := atomic.Uint64{}  // pending blocks to be submitted. gap between produced and submitted.
	nProducedBytes := atomic.Uint64{} // tracking how many actual bytes have been produced but not submitted so far
	producedBytesC := make(chan int)  // producer sends on here, and can be blocked by not consuming from here

	lastSettlementBlockTime := atomic.Int64{}
	lastBlockTime := atomic.Int64{}
	lastSettlementBlockTime.Store(time.Now().UTC().UnixNano())
	lastBlockTime.Store(time.Now().UTC().UnixNano())

	skewTime := func() time.Duration {
		blockTime := time.Unix(0, lastBlockTime.Load())
		settlementTime := time.Unix(0, lastSettlementBlockTime.Load())
		return blockTime.Sub(settlementTime)
	}
	skewNow := func() time.Duration {
		settlementTime := time.Unix(0, lastSettlementBlockTime.Load())
		return time.Since(settlementTime)
	}
	go func() { // simulate block production
		go func() { // another thread to check system properties
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				// producer shall not get too far ahead
				require.True(t, skewTime() < args.batchSkew+args.skewMargin, fmt.Sprintf("last produced blocks time not less than maximum skew time. produced block skew time: %s max skew: %s", skewTime(), args.batchSkew+args.skewMargin))
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			time.Sleep(approx(args.produceTime))

			if args.batchSkew <= skewNow() {
				continue
			}

			nBytes := rand.Intn(args.produceBytes) // simulate block production
			nProducedBytes.Add(uint64(nBytes))
			producedBytesC <- nBytes
			lastBlockTime.Store(time.Now().UTC().UnixNano())

			pendingBlocks.Add(1) // increase pending blocks to be submitted counter
		}
	}()

	submitBatch := func(maxSize uint64) (uint64, error) { // mock the batch submission
		time.Sleep(approx(args.submitTime))
		if rand.Float64() < args.submissionHaltProbability {
			time.Sleep(args.submissionHaltTime)
		}
		consumed := rand.Intn(int(maxSize))
		nProducedBytes.Add(^uint64(consumed - 1)) // subtract
		pendingBlocks.Store(0)                    // no pending blocks to be submitted
		lastSettlementBlockTime.Store(lastBlockTime.Load())
		return uint64(consumed), nil
	}
	accumulatedBlocks := func() uint64 {
		return pendingBlocks.Load()
	}
	pendingBytes := func() int {
		return int(nProducedBytes.Load())
	}
	isLastBatchRecent := func(time.Duration) bool {
		return true
	}

	block.SubmitLoopInner(ctx, log.NewNopLogger(), producedBytesC, args.batchSkew, accumulatedBlocks, pendingBytes, skewTime, isLastBatchRecent, args.maxTime, args.batchBytes, submitBatch)
}

// Make sure the producer does not get too far ahead
func TestSubmitLoopFastProducerHaltingSubmitter(t *testing.T) {
	testSubmitLoop(
		t,
		testArgs{
			nParallel:    50,
			testDuration: 4 * time.Second,
			batchSkew:    100 * time.Millisecond,
			skewMargin:   10 * time.Millisecond,
			batchBytes:   100,
			maxTime:      10 * time.Millisecond,
			submitTime:   2 * time.Millisecond,
			produceBytes: 20,
			produceTime:  2 * time.Millisecond,
			// a relatively long possibility of the submitter halting
			// tests the case where we need to stop the producer getting too far ahead
			submissionHaltTime:        150 * time.Millisecond,
			submissionHaltProbability: 0.05,
		},
	)
}

// Make sure the timer works even if the producer is slow
func TestSubmitLoopTimer(t *testing.T) {
	testSubmitLoop(
		t,
		testArgs{
			nParallel:    50,
			testDuration: 4 * time.Second,
			batchSkew:    100 * time.Millisecond,
			skewMargin:   10 * time.Millisecond,
			batchBytes:   100,
			maxTime:      10 * time.Millisecond,
			submitTime:   2 * time.Millisecond,
			produceBytes: 20,
			// a relatively long production time ensures we test the
			// case where the producer is slow but we want to submit anyway due to time
			produceTime: 50 * time.Millisecond,
		},
	)
}
