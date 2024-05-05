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
	EventNewGossipedBlock = "NewGossipedBlock"
)

/* -------------------------------------------------------------------------- */
/*                                   Queries                                  */
/* -------------------------------------------------------------------------- */

// EventQueryNewNewGossipedBlock is the query used for getting EventNewGossipedBlock
var EventQueryNewNewGossipedBlock = uevent.QueryFor(EventTypeKey, EventNewGossipedBlock)
