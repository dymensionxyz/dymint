package settlement

import (
	"fmt"

	"github.com/dymensionxyz/dymint/gerr"
)

// ErrBatchNotAccepted is returned when a batch is not accepted by the settlement layer.
var ErrBatchNotAccepted = fmt.Errorf("batch not accepted: %w", gerr.ErrUnknown)
