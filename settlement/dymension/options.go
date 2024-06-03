package dymension

import (
	"time"

	"github.com/dymensionxyz/dymint/settlement"
)

// WithCosmosClient is an option that sets the CosmosClient.
func WithCosmosClient(cosmosClient CosmosClient) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.cosmosClient = cosmosClient
	}
}

// WithRetryAttempts is an option that sets the number of attempts to retry when interacting with the settlement layer.
func WithRetryAttempts(batchRetryAttempts uint) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.retryAttempts = batchRetryAttempts
	}
}

// WithBatchAcceptanceTimeout is an option that sets the timeout for waiting for a batch to be accepted by the settlement layer.
func WithBatchAcceptanceTimeout(batchAcceptanceTimeout time.Duration) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.batchAcceptanceTimeout = batchAcceptanceTimeout
	}
}

// WithBatchAcceptanceAttempts is an option that sets the number of attempts to check if a batch has been accepted by the settlement layer.
func WithBatchAcceptanceAttempts(batchAcceptanceAttempts uint) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.batchAcceptanceAttempts = batchAcceptanceAttempts
	}
}

// WithRetryMinDelay is an option that sets the retry function mindelay between hub retry attempts.
func WithRetryMinDelay(retryMinDelay time.Duration) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.retryMinDelay = retryMinDelay
	}
}

// WithRetryMaxDelay is an option that sets the retry function max delay between hub retry attempts.
func WithRetryMaxDelay(retryMaxDelay time.Duration) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.retryMaxDelay = retryMaxDelay
	}
}
