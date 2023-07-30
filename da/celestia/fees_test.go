package celestia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	gasAdjust = 2
)

func Test(t *testing.T) {
	testCases := []struct {
		desc        string
		blobSize    uint32
		measuredGas uint64
		gasAdjust   float64
	}{
		{
			desc:        "",
			blobSize:    3576,
			measuredGas: 106230,
		},
		{
			desc:        "",
			blobSize:    6282,
			measuredGas: 120850,
		},
		{
			desc:        "",
			blobSize:    12074,
			measuredGas: 170002,
			gasAdjust:   gasAdjust,
		},
		{
			desc:        "",
			blobSize:    16908,
			measuredGas: 323804,
			gasAdjust:   gasAdjust,
		},
		{
			desc:        "",
			blobSize:    31343,
			measuredGas: 333852,
		},
		{
			desc:        "",
			blobSize:    43248,
			measuredGas: 432156,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			expectedGas := DefaultEstimateGas(tC.blobSize)
			if tC.gasAdjust != 0 {
				expectedGas = uint64(float64(expectedGas) * tC.gasAdjust)
			}
			assert.GreaterOrEqual(t, expectedGas, tC.measuredGas, "calculated gas is less than actually measured")
		})
	}
}
