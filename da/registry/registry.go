package registry

import (
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/avail"
	"github.com/dymensionxyz/dymint/da/celestia"
	"github.com/dymensionxyz/dymint/da/grpc"
	"github.com/dymensionxyz/dymint/da/loadnetwork"
	"github.com/dymensionxyz/dymint/da/local"
)

// this is a central registry for all Data Availability Layer Clients
var clients = map[string]func() da.DataAvailabilityLayerClient{
	"mock":        func() da.DataAvailabilityLayerClient { return &local.DataAvailabilityLayerClient{} },
	"grpc":        func() da.DataAvailabilityLayerClient { return &grpc.DataAvailabilityLayerClient{} },
	"celestia":    func() da.DataAvailabilityLayerClient { return &celestia.DataAvailabilityLayerClient{} },
	"avail":       func() da.DataAvailabilityLayerClient { return &avail.DataAvailabilityLayerClient{} },
	"loadnetwork": func() da.DataAvailabilityLayerClient { return &loadnetwork.DataAvailabilityLayerClient{} },
}

// GetClient returns client identified by name.
func GetClient(name string) da.DataAvailabilityLayerClient {
	f, ok := clients[name]
	if !ok {
		return nil
	}
	return f()
}

// RegisteredClients returns names of all DA clients in registry.
func RegisteredClients() []string {
	registered := make([]string, 0, len(clients))
	for name := range clients {
		registered = append(registered, name)
	}
	return registered
}
