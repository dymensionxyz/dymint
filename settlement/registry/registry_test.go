package registry_test

import (
	"testing"

	"github.com/dymensionxyz/dymint/settlement/registry"
	"github.com/stretchr/testify/assert"
)

func TestRegistery(t *testing.T) {
	assert := assert.New(t)

	expected := []registry.Client{registry.Mock, registry.Dymension, registry.Grpc}
	actual := registry.RegisteredClients()

	assert.ElementsMatch(expected, actual)

	for _, e := range expected {
		dalc := registry.GetClient(e)
		assert.NotNil(dalc)
	}

	assert.Nil(registry.GetClient("nonexistent"))
}
