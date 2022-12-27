package p2p

import (
	"fmt"

	"github.com/dymensionxyz/dymint/types"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
)

// Define the event type keys
const (
	// EventTypeKey is a reserved composite key for event name.
	EventTypeKey = "p2p.event"
)

// Define the event types
const (
	EventNewGossipedBlock = "NewGossipedBlock"
)

// EventDataNewGossipedBlock defines the struct of the event data for the EventDataNewGossipedBlock
type EventDataNewGossipedBlock struct {
	// Block is the block that was gossiped
	Block types.Block
	// Commit is the commit that was gossiped
	Commit types.Commit
}

func (e EventDataNewGossipedBlock) String() string {
	return fmt.Sprintf("EventDataNewGossipedBlock{Block: %v, Commit: %v}", e.Block, e.Commit)
}

// Define queries
var (
	EventQueryNewNewGossipedBlock = QueryForEvent(EventNewGossipedBlock)
)

// QueryForEvent returns a query for the given event.
func QueryForEvent(eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", EventTypeKey, eventType))
}
