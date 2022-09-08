package registry

import (
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/settlement/mock"
)

// Client represents a settlement layer client
type Client string

const (
	// ClientMock is a mock client for the settlement layer
	ClientMock Client = "mock"
	// Dymension is a client for interacting with dymension settlement layer
	Dymension Client = "dymension"
)

// A central registry for all Settlement Layer Clients
var clients = map[Client]func() settlement.LayerClient{
	ClientMock: func() settlement.LayerClient { return &mock.SettlementLayerClient{} },
	Dymension:  func() settlement.LayerClient { return &settlement.DymensionLayerClient{} },
}

// GetClient returns client identified by name.
func GetClient(client Client) settlement.LayerClient {
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
