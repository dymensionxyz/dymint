package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
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
				w.Write([]byte(json))
				return

			} else {
				//in case it is not we append health status to any response
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, r)

				for k, v := range rec.Header() {
					w.Header()[k] = v
				}

				healthResponse := `,"isHealthy":` + strconv.FormatBool(isHealthy) + `}`
				data := []byte(healthResponse)

				clen, _ := strconv.Atoi(r.Header.Get("Content-Length"))
				clen += len(data)
				r.Header.Set("Content-Length", strconv.Itoa(clen))

				b, err := io.ReadAll(rec.Body)
				if err != nil {
					return
				}

				jsonResponse := string(b)

				if len(jsonResponse) > 0 {
					jsonResponse = jsonResponse[:len(jsonResponse)-2] + healthResponse
				}

				_, err = w.Write([]byte(jsonResponse))
				if err != nil {
					return
				}

			}

		})
	}
}
