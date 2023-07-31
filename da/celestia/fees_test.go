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
		{
			// time="2023-07-30T18:23:38Z" level=debug msg="Submitting to da blob with size[size 506524]" module=celestia
			// time="2023-07-30T18:23:38Z" level=error msg="Failed to submit DA batch. Emitting health event and trying again[txResponse insufficient fees; got: 100000utia required: 668882utia: insufficient fee code 13]" module=celestia
			desc:        "logged while using fixed price",
			blobSize:    506524,
			measuredGas: 6688820,
			gasAdjust:   gasAdjust,
		},
		{
			// time="2023-07-31T09:09:38Z" level=debug msg="Submitting to da blob with size[size 1499572]" module=celestia
			// time="2023-07-31T09:09:40Z" level=error msg="Failed to submit DA batch. Emitting health event and trying again[txResponse insufficient fees; got: 1000000utia required: 1959844utia: insufficient fee code 13]" module=celestia
			desc:        "logged while using fixed price",
			blobSize:    1499572,
			measuredGas: 19598440,
			gasAdjust:   gasAdjust,
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
