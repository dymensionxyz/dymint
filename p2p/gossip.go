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

// buffer size used by gossipSub router to consume received packets (blocks or txs). packets are dropped in case buffer overflows. in case of blocks, it can buffer up to 5 minutes (assuming 200ms block rate)
const pubsubBufferSize = 3000

// GossipMessage represents message gossiped via P2P network (e.g. transaction, Block etc).
type GossipMessage struct {
	Data []byte
	From peer.ID
}

// GossiperOption sets optional parameters of Gossiper.
type GossiperOption func(*Gossiper) error

type GossipMessageHandler func(ctx context.Context, gossipedBlock []byte)

// WithValidator options registers topic validator for Gossiper.
func WithValidator(validator GossipValidator) GossiperOption {
	return func(g *Gossiper) error {
		return g.ps.RegisterTopicValidator(g.topic.String(), wrapValidator(g, validator))
	}
}

// Gossiper is an abstraction of P2P publish subscribe mechanism.
type Gossiper struct {
	ownID peer.ID

	ps         *pubsub.PubSub
	topic      *pubsub.Topic
	sub        *pubsub.Subscription
	msgHandler GossipMessageHandler
	logger     types.Logger
}

// NewGossiper creates new, ready to use instance of Gossiper.
//
// Returned Gossiper object can be used for sending (Publishing) and receiving messages in topic identified by topicStr.
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

// Close is used to disconnect from topic and free resources used by Gossiper.
func (g *Gossiper) Close() error {
	err := g.ps.UnregisterTopicValidator(g.topic.String())
	g.sub.Cancel()
	return multierr.Combine(
		err,
		g.topic.Close(),
	)
}

// Publish publishes data to gossip topic.
func (g *Gossiper) Publish(ctx context.Context, data []byte) error {
	return g.topic.Publish(ctx, data)
}

// ProcessMessages waits for messages published in the topic and execute handler.
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

func wrapValidator(gossiper *Gossiper, validator GossipValidator) pubsub.ValidatorEx {
	return func(_ context.Context, _ peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
		// Make sure we don't process our own messages.
		// In this case we'll want to return true but not to actually handle the message.
		if msg.GetFrom() == gossiper.ownID {
			return pubsub.ValidationAccept
		}
		return validator(&GossipMessage{
			Data: msg.Data,
			From: msg.GetFrom(),
		})
	}
}
