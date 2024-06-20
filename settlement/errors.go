package settlement

import (
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

// ErrBatchNotAccepted is returned when a batch is not accepted by the settlement layer.
var ErrBatchNotAccepted = fmt.Errorf("batch not accepted: %w", gerrc.ErrUnknown)
