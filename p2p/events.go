package p2p

import (
	uevent "github.com/dymensionxyz/dymint/utils/event"
)

/* -------------------------------------------------------------------------- */
/*                                 Event types                                */
/* -------------------------------------------------------------------------- */

const (
	// EventTypeKey is a reserved composite key for event name.
	EventTypeKey = "p2p.event"
)

const (
	EventNewGossipedBlock  = "NewGossipedBlock"
	EventNewBlockSyncBlock = "NewBlockSyncBlock"
)

/* -------------------------------------------------------------------------- */
/*                                   Queries                                  */
/* -------------------------------------------------------------------------- */

// EventQueryNewGossipedBlock is the query used for getting EventNewGossipedBlock
var EventQueryNewGossipedBlock = uevent.QueryFor(EventTypeKey, EventNewGossipedBlock)

// EventQueryNewBlockSyncBlock is the query used for getting EventNewBlockSyncBlock
var EventQueryNewBlockSyncBlock = uevent.QueryFor(EventTypeKey, EventNewBlockSyncBlock)
