package utils

import (
	"context"

	"github.com/dymensionxyz/dymint/log"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// SubscribeAndHandleEvents subscribes to events and sends back a callback
func SubscribeAndHandleEvents(ctx context.Context, pubsubServer *pubsub.Server, clientID string, eventQuery pubsub.Query, callback func(event pubsub.Message), logger log.Logger, outCapacity ...int) {
	subscription, err := pubsubServer.Subscribe(ctx, clientID, eventQuery, outCapacity...)
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
			logger.Info(clientID + " subscription canceled")
			return
		}
	}
}

// SubmitEventOrPanic submits an event or panics
func SubmitEventOrPanic(ctx context.Context, pubsubServer *pubsub.Server, msg interface{}, events map[string][]string) {
	err := pubsubServer.PublishWithEvents(ctx, msg, events)
	if err != nil {
		panic(err)
	}
}
