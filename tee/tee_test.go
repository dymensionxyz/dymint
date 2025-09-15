package tee

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryFullNodeTEEWithRealEndpoint(t *testing.T) {
	t.Skip("Requires real server endpoint")

	url := "https://rpc.dan2.team.rollapp.network:443"
	client := &http.Client{}

	result, err := queryFullNodeTEE(client, url)

	require.NoError(t, err)
	require.NotNil(t, result)

	t.Logf("Token: %s", result.Token)
	t.Logf("Nonce - RollappID: %s", result.Nonce.RollappId)
	t.Logf("Nonce - CurrentHeight: %d (uint64)", result.Nonce.CurrHeight)
	t.Logf("Nonce - FinalizedHeight: %d (uint64)", result.Nonce.FinalizedHeight)
}
