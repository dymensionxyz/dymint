package errors

import (
	"github.com/dymensionxyz/dymint/types"
	"golang.org/x/sync/errgroup"
)

// ErrgroupGoAndLogImmediate calls eg.Go on the errgroup but it will log the error immediately when it occurs
// instead of waiting for all goroutines in the group to finish first. This has the advantage of making sure all
// errors are logged, not just the first one, and it is more immediate. Also, it is guaranteed, in case that
// of the goroutines is not properly context aware.
func ErrgroupGoAndLogImmediate(logger types.Logger, eg *errgroup.Group, fn func() error) {
	eg.Go(func() error {
		err := fn()
		if err != nil {
			logger.Error("ErrGroup goroutine.", "err", err)
		}
		return err
	})
}
