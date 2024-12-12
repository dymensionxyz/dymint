package p2p

import (
	uevent "github.com/dymensionxyz/dymint/utils/event"
)





const (
	
	EventTypeKey = "p2p.event"
)

const (
	EventNewGossipedBlock  = "NewGossipedBlock"
	EventNewBlockSyncBlock = "NewBlockSyncBlock"
)






var EventQueryNewGossipedBlock = uevent.QueryFor(EventTypeKey, EventNewGossipedBlock)


var EventQueryNewBlockSyncBlock = uevent.QueryFor(EventTypeKey, EventNewBlockSyncBlock)
