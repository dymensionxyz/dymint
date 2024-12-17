package main

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/informalsystems/tm-load-test/pkg/loadtest"

	"github.com/dymensionxyz/dymint/test/loadtime/payload"
	"github.com/dymensionxyz/dymint/test/pb/loadtime"
)

var (
	_ loadtest.ClientFactory = (*ClientFactory)(nil)
	_ loadtest.Client        = (*TxGenerator)(nil)
)

type ClientFactory struct {
	ID []byte
}

type TxGenerator struct {
	id    []byte
	conns uint64
	rate  uint64
	size  uint64
}

func main() {
	u := [16]byte(uuid.New())
	if err := loadtest.RegisterClientFactory("loadtime-client", &ClientFactory{ID: u[:]}); err != nil {
		panic(err)
	}
	loadtest.Run(&loadtest.CLIConfig{
		AppName:              "loadtime",
		AppShortDesc:         "Generate timestamped transaction load.",
		AppLongDesc:          "loadtime generates transaction load for the purpose of measuring the end-to-end latency of a transaction from submission to execution in a Dymint network.",
		DefaultClientFactory: "loadtime-client",
	})
}

func (f *ClientFactory) ValidateConfig(cfg loadtest.Config) error {
	psb, err := payload.MaxUnpaddedSize()
	if err != nil {
		return err
	}
	if psb > cfg.Size {
		return fmt.Errorf("payload size exceeds configured size")
	}
	return nil
}

func (f *ClientFactory) NewClient(cfg loadtest.Config) (loadtest.Client, error) {
	return &TxGenerator{
		id:    f.ID,
		conns: uint64(cfg.Connections),
		rate:  uint64(cfg.Rate),
		size:  uint64(cfg.Size),
	}, nil
}

func (c *TxGenerator) GenerateTx() ([]byte, error) {
	return payload.NewBytes(&loadtime.Payload{
		Connections: c.conns,
		Rate:        c.rate,
		Size_:       c.size,
		Id:          c.id,
	})
}
