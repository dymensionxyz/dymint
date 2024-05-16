package dymension

import (
	"time"
)

// Option is a function that configures the LayerI.
type Option func(*LayerClient)

// WithCosmosClient is an option that sets the CosmosClient.
func WithCosmosClient(cosmosClient CosmosClient) Option {
	return func(d *LayerClient) {
		d.cosmosClient = cosmosClient
	}
}

// WithRetryAttempts is an option that sets the number of attempts to retry when interacting with the settlement layer.
func WithRetryAttempts(batchRetryAttempts uint) Option {
	return func(d *LayerClient) {
		d.retryAttempts = batchRetryAttempts
	}
}

// WithBatchAcceptanceTimeout is an option that sets the timeout for waiting for a batch to be accepted by the settlement layer.
func WithBatchAcceptanceTimeout(batchAcceptanceTimeout time.Duration) Option {
	return func(d *LayerClient) {
		d.batchAcceptanceTimeout = batchAcceptanceTimeout
	}
}

// WithRetryMinDelay is an option that sets the retry function mindelay between hub retry attempts.
func WithRetryMinDelay(retryMinDelay time.Duration) Option {
	return func(d *LayerClient) {
		d.retryMinDelay = retryMinDelay
	}
}

// WithRetryMaxDelay is an option that sets the retry function max delay between hub retry attempts.
func WithRetryMaxDelay(retryMaxDelay time.Duration) Option {
	return func(d *LayerClient) {
		d.retryMaxDelay = retryMaxDelay
	}
}
