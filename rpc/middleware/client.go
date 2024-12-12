package middleware

import (
	"net/http"

	"github.com/tendermint/tendermint/libs/log"
)

type Client struct {
	registry *Registry
	logger   log.Logger
}

func NewClient(reg Registry, logger log.Logger) *Client {
	return &Client{
		registry: &reg,
		logger:   logger,
	}
}

func (mc *Client) Handle(h http.Handler) http.Handler {
	registeredMiddlewares := mc.registry.GetRegistered()
	finalHandler := h
	for i := len(registeredMiddlewares) - 1; i >= 0; i-- {
		finalHandler = registeredMiddlewares[i].Handler(mc.logger)(finalHandler)
	}
	return finalHandler
}
