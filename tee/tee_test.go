package tee

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryFullNodeTEEWithRealEndpoint(t *testing.T) {
	// t.Skip()

	// Test against the real endpoint
	url := "https://rpc.dan2.team.rollapp.network:443"
	client := &http.Client{}

	result, err := queryFullNodeTEE(client, url)
	require.NoError(t, err)
	require.NotNil(t, result)
}
