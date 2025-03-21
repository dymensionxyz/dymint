package registry_test

import (
	"testing"

	"github.com/dymensionxyz/dymint/da/registry"
	"github.com/stretchr/testify/assert"
)

func TestRegistery(t *testing.T) {
	assert := assert.New(t)

	expected := []string{"mock", "grpc", "celestia", "avail", "weavevm", "sui"}
	actual := registry.RegisteredClients()

	assert.ElementsMatch(expected, actual)

	for _, e := range expected {
		dalc := registry.GetClient(e)
		assert.NotNil(dalc)
	}

	assert.Nil(registry.GetClient("nonexistent"))
}
