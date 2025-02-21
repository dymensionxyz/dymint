package client

import "testing"

func TestFoo(t *testing.T) {

	a, b, err := filterMinMax(1001, 1000, 0, 0)
	t.Log(a, b, err)

}
