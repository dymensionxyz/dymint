package errors

import (
	"github.com/dymensionxyz/dymint/types"
	"golang.org/x/sync/errgroup"
)

func ErrGroupGoLog(eg *errgroup.Group, logger types.Logger, fn func() error) {
	eg.Go(func() error {
		err := fn()
		if err != nil {
			logger.Error("ErrGroup goroutine.", "err", err)
		}
		return err
	})
}
