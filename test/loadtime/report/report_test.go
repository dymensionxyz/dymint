package report_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/test/loadtime/payload"
	"github.com/dymensionxyz/dymint/test/loadtime/report"
	"github.com/dymensionxyz/dymint/test/pb/loadtime"
	"github.com/dymensionxyz/dymint/types"
	"github.com/google/uuid"
)

type mockBlockStore struct {
	base   uint64
	blocks []*types.Block
}

func (m *mockBlockStore) Height() uint64 {
	return uint64(len(m.blocks) - 1)
}

func (m *mockBlockStore) Base() uint64 {
	return m.base
}

func (m *mockBlockStore) LoadBlock(i uint64) (*types.Block, error) {
	return (m.blocks[i-m.base]), nil
}

func TestGenerateReport(t *testing.T) {
	t1 := time.Now()
	u := [16]byte(uuid.New())
	b1, err := payload.NewBytes(&loadtime.Payload{
		Id:    u[:],
		Time:  t1.Add(-10 * time.Second),
		Size_: 1024,
	})
	if err != nil {
		t.Fatalf("generating payload %s", err)
	}
	b2, err := payload.NewBytes(&loadtime.Payload{
		Id:    u[:],
		Time:  t1.Add(-4 * time.Second),
		Size_: 1024,
	})
	if err != nil {
		t.Fatalf("generating payload %s", err)
	}
	b3, err := payload.NewBytes(&loadtime.Payload{
		Id:    u[:],
		Time:  t1.Add(2 * time.Second),
		Size_: 1024,
	})
	fmt.Println(b3)
	t2 := t1.Add(time.Second)
	fmt.Println(t2)
	if err != nil {
		t.Fatalf("generating payload %s", err)
	}
	s := &mockBlockStore{
		blocks: []*types.Block{
			{
				Header: types.Header{
					Time: uint64(t1.UTC().UnixNano()),
				},
				Data: types.Data{
					Txs: []types.Tx{b1, b2},
				},
			},
			{
				Header: types.Header{
					Time: uint64(t1.UTC().UnixNano()),
				},
				Data: types.Data{
					Txs: []types.Tx{[]byte("error")},
				},
			},
			{
				Data: types.Data{
					Txs: []types.Tx{b3, b3},
				},
				Header: types.Header{
					Time: uint64(t2.UTC().UnixNano()),
				},
			},
			{
				Header: types.Header{
					Time: uint64(t2.UTC().UnixNano()),
				},
				Data: types.Data{
					Txs: []types.Tx{},
				},
			},
		},
	}
	rs, err := report.GenerateFromBlockStore(s)
	if err != nil {
		t.Fatalf("generating report %s", err)
	}
	if rs.ErrorCount() != 1 {
		t.Fatalf("ErrorCount did not match expected. Expected %d but contained %d", 1, rs.ErrorCount())
	}
	rl := rs.List()
	if len(rl) != 1 {
		t.Fatalf("number of reports did not match expected. Expected %d but contained %d", 1, len(rl))
	}
	r := rl[0]
	if len(r.All) != 4 {
		t.Fatalf("report contained different number of data points from expected. Expected %d but contained %d", 4, len(r.All))
	}
	if r.NegativeCount != 2 {
		t.Fatalf("NegativeCount did not match expected. Expected %d but contained %d", 2, r.NegativeCount)
	}
	if r.Avg != 3*time.Second {
		t.Fatalf("Avg did not match expected. Expected %s but contained %s", 3*time.Second, r.Avg)
	}
	if r.Min != -time.Second {
		t.Fatalf("Min did not match expected. Expected %s but contained %s", time.Second, r.Min)
	}
	if r.Max != 10*time.Second {
		t.Fatalf("Max did not match expected. Expected %s but contained %s", 10*time.Second, r.Max)
	}
	// Verified using online standard deviation calculator:
	// https://www.calculator.net/standard-deviation-calculator.html?numberinputs=10%2C+4%2C+-1%2C+-1&ctype=s&x=45&y=12
	expectedStdDev := 5228129047 * time.Nanosecond
	if r.StdDev != expectedStdDev {
		t.Fatalf("StdDev did not match expected. Expected %s but contained %s", expectedStdDev, r.StdDev)
	}
}
