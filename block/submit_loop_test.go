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

// TODO: improve the tests to actually add some requirements

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

// Make sure the producer does not get too far ahead
func TestSubmitLoopFastProducerHaltingSubmitter(t *testing.T) {
	t.Skip()
	testSubmitLoopInner(
		t,
		testArgs{
			testDuration:              60 * time.Minute,
			batchSkew:                 10,
			batchBytes:                100,
			maxTime:                   500 * time.Millisecond,
			submitTime:                100 * time.Millisecond,
			produceBytes:              20,
			produceTime:               100 * time.Millisecond,
			submissionHaltTime:        30000 * time.Millisecond,
			submissionHaltProbability: 0.01,
		},
	)
}

// Make sure the timer works even if the producer is slow
func TestSubmitLoopTimer(t *testing.T) {
	t.Skip()
	testSubmitLoopInner(
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
