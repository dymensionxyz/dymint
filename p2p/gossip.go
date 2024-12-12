package p2p

import (
	"context"
	"errors"

	"go.uber.org/multierr"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/dymensionxyz/dymint/types"
)


const pubsubBufferSize = 3000


type GossipMessage struct {
	Data []byte
	From peer.ID
}


type GossiperOption func(*Gossiper) error

type GossipMessageHandler func(ctx context.Context, gossipedBlock []byte)


func WithValidator(validator GossipValidator) GossiperOption {
	return func(g *Gossiper) error {
		return g.ps.RegisterTopicValidator(g.topic.String(), wrapValidator(g, validator))
	}
}


type Gossiper struct {
	ownID peer.ID

	ps         *pubsub.PubSub
	topic      *pubsub.Topic
	sub        *pubsub.Subscription
	msgHandler GossipMessageHandler
	logger     types.Logger
}




func NewGossiper(host host.Host, ps *pubsub.PubSub, topicStr string, msgHandler GossipMessageHandler, logger types.Logger, options ...GossiperOption) (*Gossiper, error) {
	topic, err := ps.Join(topicStr)
	if err != nil {
		return nil, err
	}
	subscription, err := topic.Subscribe(pubsub.WithBufferSize(max(pubsub.GossipSubHistoryGossip, pubsubBufferSize)))
	if err != nil {
		return nil, err
	}
	g := &Gossiper{
		ownID:      host.ID(),
		ps:         ps,
		topic:      topic,
		sub:        subscription,
		logger:     logger,
		msgHandler: msgHandler,
	}

	for _, option := range options {
		err := option(g)
		if err != nil {
			return nil, err
		}
	}

	return g, nil
}


func (g *Gossiper) Close() error {
	err := g.ps.UnregisterTopicValidator(g.topic.String())
	g.sub.Cancel()
	return multierr.Combine(
		err,
		g.topic.Close(),
	)
}


func (g *Gossiper) Publish(ctx context.Context, data []byte) error {
	return g.topic.Publish(ctx, data)
}


func (g *Gossiper) ProcessMessages(ctx context.Context) {
	for {
		msg, err := g.sub.Next(ctx)
		if errors.Is(err, context.Canceled) {
			return
		}
		if err != nil {
			g.logger.Error("read message", "error", err)
			return
		}
		if g.msgHandler != nil {
			g.msgHandler(ctx, msg.Data)
		}
	}
}

func wrapValidator(gossiper *Gossiper, validator GossipValidator) pubsub.Validator {
	return func(_ context.Context, _ peer.ID, msg *pubsub.Message) bool {
		
		
		if msg.GetFrom() == gossiper.ownID {
			return true
		}
		return validator(&GossipMessage{
			Data: msg.Data,
			From: msg.GetFrom(),
		})
	}
}
