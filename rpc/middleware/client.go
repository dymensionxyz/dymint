package middleware

import (
	"net/http"

	"github.com/tendermint/tendermint/libs/log"
)

// Client is a struct that holds registered middlewares and provides methods
// to run these middlewares on an HTTP handler.
type Client struct {
	registry *Registry
	logger   log.Logger
}

// NewClient creates and returns a new Client instance.
func NewClient(reg Registry, logger log.Logger) *Client {
	return &Client{
		registry: &reg,
		logger:   logger,
	}
}

// Handle wraps the provided http.Handler with the registered middlewares and returns the final http.Handler.
func (mc *Client) Handle(h http.Handler) http.Handler {
	registeredMiddlewares := mc.registry.GetRegistered()

	finalHandler := h
	for i := len(registeredMiddlewares) - 1; i >= 0; i-- {
		finalHandler = registeredMiddlewares[i].Handler(mc.logger)(finalHandler)
	}
	return finalHandler
}
