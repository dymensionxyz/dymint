package client

import "testing"

func TestMinMaxFilter(t *testing.T) {
	_, _, err := filterMinMax(900, 1000, 0, 0)
	if err != nil {
		t.FailNow()
	}
	_, _, err = filterMinMax(900, 1000, 950, 970)
	if err != nil {
		t.FailNow()
	}
	_, _, err = filterMinMax(900, 1000, 1000, 1000)
	if err != nil {
		t.FailNow()
	}
	_, _, err = filterMinMax(900, 1000, 999, 999)
	if err != nil {
		t.FailNow()
	}
	_, _, err = filterMinMax(1000, 1000, 999, 999)
	if err != nil {
		t.FailNow()
	}
	_, _, err = filterMinMax(1000, 1000, 1000, 1000)
	if err != nil {
		t.FailNow()
	}
	_, _, err = filterMinMax(1001, 1000, 0, 0)
	if err != nil {
		t.FailNow()
	}
}
