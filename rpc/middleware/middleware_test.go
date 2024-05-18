package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dymensionxyz/dymint/rpc/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

// Create a custom test middleware that implements the Middleware interface
type testMiddleware struct{}

func (t testMiddleware) Handler(log.Logger) middleware.HandlerFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test-Header", "true")
			h.ServeHTTP(w, r)
		})
	}
}

// TestRegistry checks if the middlewares are correctly registered in the registry.
func TestRegistry(t *testing.T) {
	reg := middleware.GetRegistry()

	m1 := middleware.Status{Err: func() error {
		return nil
	}}
	reg.Register(&m1)
	registeredMiddlewares := reg.GetRegistered()

	assert.Equal(t, 1, len(registeredMiddlewares), "Expected 1 middleware registered")
	assert.Equal(t, &m1, registeredMiddlewares[0], "Expected the first middleware to be the registered one")
}

// TestMiddlewareClient checks if the MiddlewareClient correctly handles the registered middlewares.
func TestMiddlewareClient(t *testing.T) {
	reg := middleware.GetRegistry()

	// Create a test middleware that sets a custom header
	testMiddleware := testMiddleware{}

	// Create a test handler that sets a response header
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	})

	// Register the test middleware and handle the request
	reg.Register(testMiddleware)
	mc := middleware.NewClient(*reg, log.TestingLogger())
	finalHandler := mc.Handle(handler)

	// Create a test request and response recorder
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	// Serve the request and check the headers
	finalHandler.ServeHTTP(rr, req)
	assert.Equal(t, "true", rr.Header().Get("X-Test-Header"), "Expected X-Test-Header to be set by the test middleware")
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "Expected Content-Type to be set by the handler")
}

// TestStatusMiddleware checks if the Status middleware behaves correctly based on the node health status.
func TestStatusMiddleware(t *testing.T) {
	// Test cases for both healthy and unhealthy status
	testCases := []struct {
		healthy         error
		expectedStatus  int
		expectedMessage string
	}{
		{
			healthy:         nil,
			expectedStatus:  http.StatusOK,
			expectedMessage: "{\"jsonrpc\":\"2.0\",\"result\":{\"isHealthy\":true,\"error\":\"\"},\"id\":-1}",
		},
		{
			healthy:         errors.New("node unhealthy"),
			expectedStatus:  http.StatusOK,
			expectedMessage: "{\"jsonrpc\":\"2.0\",\"result\":{\"isHealthy\":false,\"error\":\"node unhealthy\"},\"id\":-1}",
		},
	}

	for _, tc := range testCases {
		statusMiddleware := middleware.Status{Err: func() error {
			return tc.healthy
		}}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("Node healthy"))
			require.NoError(t, err, "Expected no error when writing the response body")
		})

		finalHandler := statusMiddleware.Handler(log.TestingLogger())(handler)

		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()

		finalHandler.ServeHTTP(rr, req)

		assert.Equal(t, tc.expectedStatus, rr.Code, "Expected status code to match the expected value")
		body := rr.Body.String()
		assert.Equal(t, tc.expectedMessage, body, "Expected response body to match the expected value")
	}
}
