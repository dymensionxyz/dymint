package event

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"

	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
)

// MustSubscribe subscribes to events and sends back a callback
// clientID is essentially the subscriber id, see https://pkg.go.dev/github.com/tendermint/tendermint/libs/pubsub#pkg-overview
func MustSubscribe(
	ctx context.Context,
	pubsubServer *pubsub.Server,
	clientID string,
	eventQuery pubsub.Query,
	callback func(event pubsub.Message),
	logger types.Logger,
) {
	subscription, err := pubsubServer.SubscribeUnbuffered(ctx, clientID, eventQuery)
	if err != nil {
		logger.Error("subscribe to events")
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

// MustPublish submits an event or panics
func MustPublish(ctx context.Context, pubsubServer *pubsub.Server, msg any, events map[string][]string) {
	err := pubsubServer.PublishWithEvents(ctx, msg, events)
	if err != nil {
		panic(err)
	}
}

// QueryFor returns a query for the given event.
func QueryFor(eventTypeKey, eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", eventTypeKey, eventType))
}
