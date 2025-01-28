package types

import "cosmossdk.io/math"

type Balance struct {
	Amount math.Int
	Denom  string
}
