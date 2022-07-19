package registry

import (
	"github.com/celestiaorg/optimint/settlement"
	"github.com/celestiaorg/optimint/settlement/mock"
)

// A central registry for all Settlement Layer Clients
var clients = map[string]func() settlement.SettlementLayerClient{
	"mock": func() settlement.SettlementLayerClient { return &mock.SettlementLayerClient{} },
}

// GetClient returns client identified by name.
func GetClient(name string) settlement.SettlementLayerClient {
	f, ok := clients[name]
	if !ok {
		return nil
	}
	return f()
}

// RegisteredClients returns names of all settlement clients in registry.
func RegisteredClients() []string {
	registered := make([]string, 0, len(clients))
	for name := range clients {
		registered = append(registered, name)
	}
	return registered
}
