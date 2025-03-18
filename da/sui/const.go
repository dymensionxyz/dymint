package sui

import "cosmossdk.io/math"

// SUI is the SUI token numeric representation.
// MIST is a base denom of SUI. 1 SUI = 10^9 MIST.
var SUI = math.NewIntWithDecimal(1, 9)
