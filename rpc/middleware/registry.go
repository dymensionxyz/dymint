package middleware

import (
	"net/http"
	"sync"

	"github.com/tendermint/tendermint/libs/log"
)

var (
	once     sync.Once
	instance *Registry
)

// HandlerFunc is a type alias for a function that takes an http.Handler and returns a new http.Handler.
type HandlerFunc func(http.Handler) http.Handler

// Middleware is an interface representing a middleware with a Handler method.
type Middleware interface {
	Handler(logger log.Logger) HandlerFunc
}

// Registry is a struct that holds a list of registered middlewares.
type Registry struct {
	middlewareList []Middleware
}

// GetRegistry returns a singleton instance of the Registry.
func GetRegistry() *Registry {
	once.Do(func() {
		instance = &Registry{}
	})
	return instance
}

// Register adds a Middleware to the list of registered middlewares in the Registry.
func (r *Registry) Register(m Middleware) {
	r.middlewareList = append(r.middlewareList, m)
}

// GetRegistered returns a list of registered middlewares.
func (r *Registry) GetRegistered() []Middleware {
	return r.middlewareList
}
