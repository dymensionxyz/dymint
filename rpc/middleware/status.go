package middleware

import (
	"net/http"
	"strconv"

	"github.com/tendermint/tendermint/libs/log"
)

type Status func() error

func (s Status) Handler(logger log.Logger) HandlerFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := s()
			isHealthy := err == nil
			// in case the endpoint is health we return health response
			if r.URL.Path == "/health" {

				w.WriteHeader(http.StatusOK)
				var errS string
				if err != nil {
					errS = err.Error()
				}
				json := `{"jsonrpc":"2.0","result":{"isHealthy":` + strconv.FormatBool(isHealthy) + `,:"error":"` + errS + `"},"id":-1}`
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
