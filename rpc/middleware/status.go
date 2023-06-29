package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/dymensionxyz/dymint/rpc/sharedtypes"
	"github.com/tendermint/tendermint/libs/log"
)

// StatusMiddleware is a struct that holds the health status of the node.
type StatusMiddleware struct {
	healthStatus *sharedtypes.HealthStatus
}

// NewStatusMiddleware creates and returns a new Status instance with the given healthStatus.
func NewStatusMiddleware(healthStatus *sharedtypes.HealthStatus) *StatusMiddleware {
	return &StatusMiddleware{
		healthStatus: healthStatus,
	}
}

// Handler returns a MiddlewareFunc that checks the node's health status.
func (s *StatusMiddleware) Handler(logger log.Logger) HandlerFunc {
	return func(h http.Handler) http.Handler {
		return status(s.healthStatus, h, logger)
	}
}

// status is a middleware that checks if the node is healthy.
// If the node is not healthy, it returns a 503 Service Unavailable.
func status(healthStatus *sharedtypes.HealthStatus, h http.Handler, logger log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isHealthy, err := healthStatus.Get()
		if !isHealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
			errorPrefix := "node is unhealthy"
			if err == nil {
				err = errors.New(errorPrefix)
			} else {
				err = fmt.Errorf("%s: %w", errorPrefix, err)
			}
			_, err := w.Write([]byte(err.Error()))
			if err != nil {
				logger.Error("failed to write response", "error", err)
			}
			return
		}
		h.ServeHTTP(w, r)
	})
}
