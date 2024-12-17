package dymension

import (
	"time"

	"github.com/dymensionxyz/dymint/settlement"
)

func WithCosmosClient(cosmosClient CosmosClient) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.cosmosClient = cosmosClient
	}
}

func WithRetryAttempts(batchRetryAttempts uint) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.retryAttempts = batchRetryAttempts
	}
}

func WithBatchAcceptanceTimeout(batchAcceptanceTimeout time.Duration) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.batchAcceptanceTimeout = batchAcceptanceTimeout
	}
}

func WithBatchAcceptanceAttempts(batchAcceptanceAttempts uint) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.batchAcceptanceAttempts = batchAcceptanceAttempts
	}
}

func WithRetryMinDelay(retryMinDelay time.Duration) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.retryMinDelay = retryMinDelay
	}
}

func WithRetryMaxDelay(retryMaxDelay time.Duration) settlement.Option {
	return func(c settlement.ClientI) {
		dlc, _ := c.(*Client)
		dlc.retryMaxDelay = retryMaxDelay
	}
}
