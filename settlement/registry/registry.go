package registry

import (
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/dymension"
	"github.com/dymensionxyz/dymint/settlement/grpc"
	"github.com/dymensionxyz/dymint/settlement/local"
)

type Client string

const (
	Local Client = "mock"

	Dymension Client = "dymension"

	Grpc Client = "grpc"
)

var clients = map[Client]func() settlement.ClientI{
	Local:     func() settlement.ClientI { return &local.Client{} },
	Dymension: func() settlement.ClientI { return &dymension.Client{} },
	Grpc:      func() settlement.ClientI { return &grpc.Client{} },
}

func GetClient(client Client) settlement.ClientI {
	f, ok := clients[client]
	if !ok {
		return nil
	}
	return f()
}

func RegisteredClients() []Client {
	registered := make([]Client, 0, len(clients))
	for client := range clients {
		registered = append(registered, client)
	}
	return registered
}
