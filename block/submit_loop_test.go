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
	for range 50 { // do multiple simulations in parallel
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
	go func() {                       // simulate block production
		go func() { // another thread to check system properties
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				// producer shall not get too far ahead
				require.True(t, nProducedBytes.Load() < (args.batchSkew+1)*args.batchBytes)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			time.Sleep(approx(args.produceTime))
			producedBytesC <- rand.Intn(args.produceBytes)
		}
	}()

	lastSubmitTime := time.Time{}
	submitBatch := func(maxSize uint64) (uint64, error) { // mock the batch submission
		time.Sleep(approx(args.submitTime))
		if rand.Float64() < args.submissionHaltProbability {
			time.Sleep(args.submissionHaltTime)
		}
		consumed := rand.Intn(int(maxSize))
		nProducedBytes.Add(^uint64(consumed - 1)) // subtract

		require.True(t, lastSubmitTime == time.Time{} || float64(time.Since(lastSubmitTime)) < 1.5*float64(args.maxTime))
		lastSubmitTime = time.Now()

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
			testDuration:              time.Second,
			batchSkew:                 10,
			batchBytes:                100,
			maxTime:                   50 * time.Millisecond,
			submitTime:                2 * time.Millisecond,
			produceBytes:              20,
			produceTime:               2 * time.Millisecond,
			submissionHaltTime:        50 * time.Millisecond,
			submissionHaltProbability: 0.01,
		},
	)
}

// Make sure the timer works even if the producer is slow
func TestSubmitLoopTimer(t *testing.T) {
	t.Skip()
	testSubmitLoop(
		t,
		testArgs{
			testDuration: 60 * time.Second,
			batchSkew:    10,
			batchBytes:   100,
			maxTime:      1000 * time.Millisecond,
			submitTime:   20 * time.Millisecond,
			produceBytes: 500,
			produceTime:  5000 * time.Millisecond,
		},
	)
}
