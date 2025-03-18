package aptos_test

import (
	"testing"

	"github.com/dymensionxyz/dymint/da/aptos"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	c, err := aptos.NewClient()
	require.NoError(t, err)
	require.NotNil(t, c)

	err = c.TestSendTx()
	require.NoError(t, err)
}
