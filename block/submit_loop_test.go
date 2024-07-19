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

func TestSubmitLoopInner(t *testing.T) {
	/*
		producer will not stop if skew is not exceeded
		producer will stop if skew is exceeded
		submitter will submit when time elapses no matter what
		submitter will submit when enough bytes to submit a batch
	*/
	t.Run("", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
		defer cancel()

		c := make(chan int)
		batchSkew := uint64(10)
		batchBytes := uint64(100)
		maxTime := time.Millisecond * 500
		submitTime := time.Millisecond * 100
		produceTime := time.Millisecond * 100
		submissionHaltTime := time.Millisecond * 10000
		bz := atomic.Uint64{}

		approx := func(d time.Duration) time.Duration {
			base := int(float64(d) * 0.8)
			factor := int(float64(d) * 0.4)
			return time.Duration(base + rand.Intn(factor))
		}

		// TODO: can pick random params, but need to be careful

		produce := func() {
			for {
				time.Sleep(approx(produceTime))
				x := 10 + rand.Intn(10)
				after := bz.Add(uint64(x))
				t.Log(fmt.Sprintf("producer actual bytes: %d", after))
				c <- x
			}
		}

		submit := func(maxSize uint64) (uint64, error) {
			time.Sleep(approx(submitTime))
			if rand.Intn(100) < 3 {
				fmt.Println("submitter is going to halt")
				time.Sleep(submissionHaltTime)
			}
			y := rand.Intn(int(maxSize))
			bz.Add(^uint64(y - 1))
			return uint64(y), nil
		}

		go produce()

		block.SubmitLoopInner(
			ctx,
			c,
			batchSkew,
			maxTime,
			batchBytes,
			submit,
		)
	})
}
