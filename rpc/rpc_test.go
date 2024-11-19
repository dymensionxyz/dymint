package rpc_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/version"
)

func TestNodeHealthRPCPropagation(t *testing.T) {
	var err error
	version.DRS = "0"
	server, listener := testutil.CreateLocalServer(t)
	defer func() {
		err = server.Stop()
		require.NoError(t, err)
	}()
	// Wait for some blocks to be produced
	time.Sleep(1 * time.Second)

	// Create cases for the test
	cases := []struct {
		name               string
		endpoint           string
		health             error
		expectedStatusCode int
		expectedMessage    string
	}{
		{
			name:               "statusNodeHealthy",
			endpoint:           "/health",
			health:             nil,
			expectedStatusCode: http.StatusOK,
			expectedMessage:    "{\"jsonrpc\":\"2.0\",\"result\":{\"isHealthy\":true,\"error\":\"\"},\"id\":-1}",
		},
		{
			name:               "statusNodeUnhealthy",
			endpoint:           "/health",
			health:             errors.New("unhealthy"),
			expectedStatusCode: http.StatusOK,
			expectedMessage:    "{\"jsonrpc\":\"2.0\",\"result\":{\"isHealthy\":false,\"error\":\"unhealthy\"},\"id\":-1}",
		},
		{
			name:               "statusNodeHealthyAgain",
			endpoint:           "/health",
			health:             nil,
			expectedStatusCode: http.StatusOK,
			expectedMessage:    "{\"jsonrpc\":\"2.0\",\"result\":{\"isHealthy\":true,\"error\":\"\"},\"id\":-1}",
		},
	}
	{
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// Emit an event to make the node unhealthy
				pubsubServer := server.PubSubServer()
				err := pubsubServer.PublishWithEvents(context.Background(), &events.DataHealthStatus{Error: tc.health},
					map[string][]string{events.NodeTypeKey: {events.HealthStatus}})
				require.NoError(t, err)
				time.Sleep(1 * time.Second)
				// Make the request
				res, err := httpGetWithTimeout(fmt.Sprintf("http://%s", listener.Addr().String())+tc.endpoint, 5*time.Second)
				require.NoError(t, err)
				defer func() {
					err = res.Body.Close()
					require.NoError(t, err)
				}()
				// Check the response
				assert.Equal(t, tc.expectedStatusCode, res.StatusCode)
				body, _ := io.ReadAll(res.Body)
				assert.Equal(t, tc.expectedMessage, string(body))
			})
		}
	}
}

func httpGetWithTimeout(url string, timeout time.Duration) (*http.Response, error) {
	// Create a new context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	// Cancel the context when we're done
	defer cancel()

	// Create a new HTTP client
	client := &http.Client{}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}
