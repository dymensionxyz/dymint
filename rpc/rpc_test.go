package rpc_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/dymensionxyz/dymint/node/events"
	rpctestutils "github.com/dymensionxyz/dymint/rpc/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeHealthRPCPropogation(t *testing.T) {

	server, listener := rpctestutils.CreateLocalServer(t)
	defer server.Stop()
	// Wait for some blocks to be produced
	time.Sleep(1 * time.Second)

	// Create cases for the test
	cases := []struct {
		name               string
		endpoint           string
		isNodeHealthy      bool
		expectedStatusCode int
	}{
		{
			name:               "statusNodeHealthy",
			endpoint:           "/status",
			isNodeHealthy:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "statusNodeUnhealthy",
			endpoint:           "/status",
			isNodeHealthy:      false,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "statusNodeHealthyAgain",
			endpoint:           "/status",
			isNodeHealthy:      true,
			expectedStatusCode: http.StatusOK,
		},
	}
	{
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// Emit an event to make the node unhealthy
				pubsubServer := server.PubSubServer()
				err := pubsubServer.PublishWithEvents(context.Background(), &events.EventDataHealthStatus{Healthy: tc.isNodeHealthy},
					map[string][]string{events.EventNodeTypeKey: {events.EventHealthStatus}})
				require.NoError(t, err)
				time.Sleep(1 * time.Second)
				// Make the request
				res, err := http.Get(fmt.Sprintf("http://%s", listener.Addr().String()) + tc.endpoint)
				require.NoError(t, err)
				defer res.Body.Close()
				require.NoError(t, err)
				// Check the response
				assert.Equal(t, tc.expectedStatusCode, res.StatusCode)

			})
		}
	}
}
