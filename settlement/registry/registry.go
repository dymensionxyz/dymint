package registry

import (
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/dymension"
	"github.com/dymensionxyz/dymint/settlement/grpc"
	"github.com/dymensionxyz/dymint/settlement/local"
)

// Client represents a settlement layer client
type Client string

const (
	// Local is a mock client for the settlement layer
	Local Client = "mock"
	// Dymension is a client for interacting with dymension settlement layer
	Dymension Client = "dymension"
	// Mock client using grpc for a shared use
	Grpc Client = "grpc"
)

// A central registry for all Settlement Layer Clients
var clients = map[Client]func() settlement.ClientI{
	Local:     func() settlement.ClientI { return &local.Client{} },
	Dymension: func() settlement.ClientI { return &dymension.Client{} },
	Grpc:      func() settlement.ClientI { return &grpc.Client{} },
}

// GetClient returns client identified by name.
func GetClient(client Client) settlement.ClientI {
	f, ok := clients[client]
	if !ok {
		return nil
	}
	return f()
}

// RegisteredClients returns names of all settlement clients in registry.
func RegisteredClients() []Client {
	registered := make([]Client, 0, len(clients))
	for client := range clients {
		registered = append(registered, client)
	}
	return registered
}
