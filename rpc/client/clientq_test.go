package client

import "testing"

func TestFoo(t *testing.T) {

	a, b, err := filterMinMax(900, 1000, 0, 0)
	t.Log(a, b, err)
	a, b, err = filterMinMax(900, 1000, 950, 970)
	t.Log(a, b, err)
	a, b, err = filterMinMax(900, 1000, 1000, 1000)
	t.Log(a, b, err)
	a, b, err = filterMinMax(900, 1000, 999, 999)
	t.Log(a, b, err)
	a, b, err = filterMinMax(1001, 1000, 0, 0)
	t.Log(a, b, err)

}
