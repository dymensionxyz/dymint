package sui

import "cosmossdk.io/math"

var (
	// SUI is the SUI token numeric representation.
	// MIST is a base denom of SUI. 1 SUI = 10^9 MIST.
	SUI = math.NewIntWithDecimal(1, 9)
)
