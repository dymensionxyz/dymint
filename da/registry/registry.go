package registry

import (
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/avail"
	"github.com/dymensionxyz/dymint/da/celestia"
	"github.com/dymensionxyz/dymint/da/grpc"
	"github.com/dymensionxyz/dymint/da/local"
)


var clients = map[string]func() da.DataAvailabilityLayerClient{
	"mock":     func() da.DataAvailabilityLayerClient { return &local.DataAvailabilityLayerClient{} },
	"grpc":     func() da.DataAvailabilityLayerClient { return &grpc.DataAvailabilityLayerClient{} },
	"celestia": func() da.DataAvailabilityLayerClient { return &celestia.DataAvailabilityLayerClient{} },
	"avail":    func() da.DataAvailabilityLayerClient { return &avail.DataAvailabilityLayerClient{} },
}


func GetClient(name string) da.DataAvailabilityLayerClient {
	f, ok := clients[name]
	if !ok {
		return nil
	}
	return f()
}


func RegisteredClients() []string {
	registered := make([]string, 0, len(clients))
	for name := range clients {
		registered = append(registered, name)
	}
	return registered
}
