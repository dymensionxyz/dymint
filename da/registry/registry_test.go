package registry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dymensionxyz/dymint/da/registry"
)

func TestRegistery(t *testing.T) {
	assert := assert.New(t)

	expected := []string{"mock", "grpc", "celestia", "avail", "interchain"}
	actual := registry.RegisteredClients()

	assert.ElementsMatch(expected, actual)

	for _, e := range expected {
		dalc := registry.GetClient(e)
		assert.NotNil(dalc)
	}

	assert.Nil(registry.GetClient("nonexistent"))
}
