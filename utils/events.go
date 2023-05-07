package utils

import (
	"context"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// SubscribeAndHandleEvents subscribes to events and sends back a callback
func SubscribeAndHandleEvents(ctx context.Context, pubsubServer *pubsub.Server, clientID string, eventQuery pubsub.Query, callback func(event pubsub.Message), logger log.Logger) {
	subscription, err := pubsubServer.Subscribe(ctx, clientID, eventQuery)
	if err != nil {
		logger.Error("failed to subscribe to events")
		panic(err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-subscription.Out():
			callback(event)
		case <-subscription.Cancelled():
			logger.Info("Subscription canceled")
		}
	}
}
