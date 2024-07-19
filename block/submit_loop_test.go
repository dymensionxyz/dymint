package block_test

import (
	"context"
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
		ctx, cancel := context.WithCancel(context.Background())

		c := make(chan int)
		batchSkew := uint64(10)
		batchBytes := uint64(100)
		batchTime := time.Second
		submitTime := time.Second
		produceTime := time.Second
		bz := atomic.Uint64{}

		// TODO: can pick random params, but need to be careful

		produce := func() {
			_ = bz.Load()
			for {
				after := bz.Add(50)
				t.Log(after)
				time.Sleep(produceTime)
				c <- 50
			}
		}

		submit := func() (uint64, error) {
			time.Sleep(submitTime)
			x := bz.Load()
			y := rand.Intn(int(x))
			bz.Add(^uint64(y - 1))
			return uint64(y), nil
		}

		go produce()

		block.SubmitLoopInner(
			ctx,
			c,
			batchSkew,
			batchTime,
			batchBytes,
			submit,
		)

		time.Sleep(time.Second * 5)
		cancel()
	})
}
