package event

import (
	"context"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"

	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
)




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
		err = fmt.Errorf("subscribe unbuffered: %w", err)
		if !errors.Is(err, context.Canceled) {
			logger.Error("Must subscribe.", "err", err)
			panic(err)
		}
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-subscription.Out():
			callback(event)
		case <-subscription.Cancelled():
			logger.Error("subscription cancelled", "clientID", clientID)
			return
		}
	}
}


func MustPublish(ctx context.Context, pubsubServer *pubsub.Server, msg interface{}, events map[string][]string) {
	err := pubsubServer.PublishWithEvents(ctx, msg, events)
	if err != nil && !errors.Is(err, context.Canceled) {
		panic(err)
	}
}


func QueryFor(eventTypeKey, eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", eventTypeKey, eventType))
}
