package block_test

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/block"
	"github.com/stretchr/testify/require"
)

type testArgs struct {
	nParallel                 int           // number of instances to run in parallel
	testDuration              time.Duration // how long to run one instance of the test (should be short)
	batchSkew                 uint64        // max number of batches to get ahead
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

	nProducedBytes := atomic.Uint64{} // tracking how many actual bytes have been produced but not submitted so far
	producedBytesC := make(chan int)  // producer sends on here, and can be blocked by not consuming from here

	// the time of the last block produced or the last batch submitted or the last starting of the node
	timeLastProgress := atomic.Int64{}

	go func() { // simulate block production
		go func() { // another thread to check system properties
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				// producer shall not get too far ahead
				absoluteMax := (args.batchSkew + 1) * args.batchBytes // +1 is because the producer is always blocked after the fact
				require.True(t, nProducedBytes.Load() < absoluteMax)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			time.Sleep(approx(args.produceTime))
			nBytes := rand.Intn(args.produceBytes) // simulate block production
			nProducedBytes.Add(uint64(nBytes))
			producedBytesC <- nBytes

			timeLastProgress.Store(time.Now().Unix())
		}
	}()

	submitBatch := func(maxSize uint64) (uint64, error) { // mock the batch submission
		time.Sleep(approx(args.submitTime))
		if rand.Float64() < args.submissionHaltProbability {
			time.Sleep(args.submissionHaltTime)
			timeLastProgress.Store(time.Now().Unix()) // we have now recovered
		}
		consumed := rand.Intn(int(maxSize))
		nProducedBytes.Add(^uint64(consumed - 1)) // subtract

		timeLastProgressT := time.Unix(timeLastProgress.Load(), 0)
		absoluteMax := int64(1.5 * float64(args.maxTime)) // allow some leeway for code execution
		require.True(t, time.Since(timeLastProgressT).Milliseconds() < absoluteMax)

		timeLastProgress.Store(time.Now().Unix()) // we have submitted  batch
		return uint64(consumed), nil
	}

	block.SubmitLoopInner(
		ctx,
		producedBytesC,
		args.batchSkew,
		args.maxTime,
		args.batchBytes,
		submitBatch,
	)
}

// Make sure the producer does not get too far ahead
func TestSubmitLoopFastProducerHaltingSubmitter(t *testing.T) {
	testSubmitLoop(
		t,
		testArgs{
			nParallel:    100,
			testDuration: 2 * time.Second,
			batchSkew:    10,
			batchBytes:   100,
			maxTime:      10 * time.Millisecond,
			submitTime:   2 * time.Millisecond,
			produceBytes: 20,
			produceTime:  2 * time.Millisecond,
			// a relatively long possibility of the submitter halting
			// tests the case where we need to stop the producer getting too far ahead
			submissionHaltTime:        50 * time.Millisecond,
			submissionHaltProbability: 0.01,
		},
	)
}

// Make sure the timer works even if the producer is slow
func TestSubmitLoopTimer(t *testing.T) {
	testSubmitLoop(
		t,
		testArgs{
			nParallel:    100,
			testDuration: 2 * time.Second,
			batchSkew:    10,
			batchBytes:   100,
			maxTime:      10 * time.Millisecond,
			submitTime:   2 * time.Millisecond,
			produceBytes: 20,
			produceTime:  50 * time.Millisecond,
		},
	)
}
