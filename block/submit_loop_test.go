package block_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/block"
)

type testArgs struct {
	testDuration              time.Duration
	batchSkew                 uint64
	batchBytes                uint64
	maxTime                   time.Duration
	submitTime                time.Duration
	produceTime               time.Duration
	produceBytes              int
	submissionHaltTime        time.Duration
	submissionHaltProbability float64
}

func testSubmitLoopInner(
	t *testing.T,
	args testArgs,
) {
	ctx, cancel := context.WithTimeout(context.Background(), args.testDuration)
	defer cancel()

	c := make(chan int)

	bz := atomic.Uint64{}

	approx := func(d time.Duration) time.Duration {
		base := int(float64(d) * 0.8)
		factor := int(float64(d) * 0.4)
		return time.Duration(base + rand.Intn(factor))
	}

	// TODO: can pick random params, but need to be careful

	produce := func() {
		for {
			time.Sleep(approx(args.produceTime))
			x := rand.Intn(args.produceBytes)
			after := bz.Add(uint64(x))
			t.Log(fmt.Sprintf("actual pending bytes: %d", after))
			c <- x
		}
	}

	submit := func(maxSize uint64) (uint64, error) {
		time.Sleep(approx(args.submitTime))
		if rand.Float64() < args.submissionHaltProbability {
			fmt.Println("submitter transient halt")
			time.Sleep(args.submissionHaltTime)
		}
		y := rand.Intn(int(maxSize))
		bz.Add(^uint64(y - 1))
		return uint64(y), nil
	}

	go produce()

	block.SubmitLoopInner(
		ctx,
		c,
		args.batchSkew,
		args.maxTime,
		args.batchBytes,
		submit,
	)
}

func TestSubmitLoopInner(t *testing.T) {
	/*
		producer will not stop if skew is not exceeded
		producer will stop if skew is exceeded
		submitter will submit when time elapses no matter what
		submitter will submit when enough bytes to submit a batch
	*/
	t.Run("fast producer, halting submitter", func(t *testing.T) { // make sure the producer cannot get too far ahead
		testSubmitLoopInner(
			t,
			testArgs{
				testDuration:              60 * time.Second,
				batchSkew:                 10,
				batchBytes:                100,
				maxTime:                   500 * time.Millisecond,
				submitTime:                100 * time.Millisecond,
				produceBytes:              20,
				produceTime:               100 * time.Millisecond,
				submissionHaltTime:        1000 * time.Millisecond,
				submissionHaltProbability: 0.01,
			},
		)
	})
	t.Run("slow producer", func(t *testing.T) { // make sure the timer works
		testSubmitLoopInner(
			t,
			testArgs{
				testDuration:              60 * time.Second,
				batchSkew:                 10,
				batchBytes:                100,
				maxTime:                   100 * time.Millisecond,
				submitTime:                20 * time.Millisecond,
				produceBytes:              100,
				produceTime:               500 * time.Millisecond,
				submissionHaltTime:        1000 * time.Millisecond,
				submissionHaltProbability: 0,
			},
		)
	})
}
