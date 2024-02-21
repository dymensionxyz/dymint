package middleware

import (
	"net/http"
	"strconv"

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
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			isHealthy, err := s.healthStatus.Get()
			//in case the endpoint is health we return health response
			if r.URL.Path == "/health" {

				w.WriteHeader(http.StatusOK)
				var error string
				if err != nil {
					error = err.Error()
				}
				json := `{"jsonrpc":"2.0","result":{"isHealthy":` + strconv.FormatBool(isHealthy) + `,:"error":"` + error + `"},"id":-1}`
				_, err = w.Write([]byte(json))
				if err != nil {
					return
				}
				return

			} else {
				h.ServeHTTP(w, r)
			}

		})
	}
}
