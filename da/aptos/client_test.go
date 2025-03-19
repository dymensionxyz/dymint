package aptos_test

import (
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/da/aptos"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	c, err := aptos.NewClient()
	require.NoError(t, err)
	require.NotNil(t, c)

	data := []byte("hello world")

	hash, err := c.TestSendTx(data)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	err = c.TestCheckBatchAvailable(hash)
	require.NoError(t, err)
	batch, err := c.TestRetrieveBatch(hash)
	require.NoError(t, err)

	require.Equal(t, data, batch)
}
