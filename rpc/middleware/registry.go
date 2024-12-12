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

type HandlerFunc func(http.Handler) http.Handler

type Middleware interface {
	Handler(logger log.Logger) HandlerFunc
}

type Registry struct {
	middlewareList []Middleware
}

func GetRegistry() *Registry {
	once.Do(func() {
		instance = &Registry{}
	})
	return instance
}

func (r *Registry) Register(m Middleware) {
	r.middlewareList = append(r.middlewareList, m)
}

func (r *Registry) GetRegistered() []Middleware {
	return r.middlewareList
}
