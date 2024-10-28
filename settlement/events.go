package settlement

import (
	"fmt"

	uevent "github.com/dymensionxyz/dymint/utils/event"
)

const (
	// EventTypeKey is a reserved composite key for event name.
	EventTypeKey = "settlement.event"

	// Event types
	EventNewBatchAccepted   = "NewBatchAccepted"
	EventNewBondedSequencer = "NewBondedSequencer"
	EventRotationStarted    = "RotationStarted"
	EventNewBatchFinalized  = "NewBatchFinalized"
)

// Convenience objects
var (
	EventNewBatchAcceptedList   = map[string][]string{EventTypeKey: {EventNewBatchAccepted}}
	EventNewBondedSequencerList = map[string][]string{EventTypeKey: {EventNewBondedSequencer}}
	EventRotationStartedList    = map[string][]string{EventTypeKey: {EventRotationStarted}}
	EventNewBatchFinalizedList  = map[string][]string{EventTypeKey: {EventNewBatchFinalized}}
)

// Queries
var (
	EventQueryNewSettlementBatchAccepted  = uevent.QueryFor(EventTypeKey, EventNewBatchAccepted)
	EventQueryNewSettlementBatchFinalized = uevent.QueryFor(EventTypeKey, EventNewBatchFinalized)
	EventQueryNewBondedSequencer          = uevent.QueryFor(EventTypeKey, EventNewBondedSequencer)
	EventQueryRotationStarted             = uevent.QueryFor(EventTypeKey, EventRotationStarted)
)

// Data

type EventDataNewBatch struct {
	StartHeight uint64
	// EndHeight is the height of the last accepted batch
	EndHeight uint64
	// StateIndex is the rollapp-specific index the batch was saved in the SL
	StateIndex uint64
}

func (e EventDataNewBatch) String() string {
	return fmt.Sprintf("EndHeight: %d, StateIndex: %d", e.EndHeight, e.StateIndex)
}

type EventDataNewBondedSequencer struct {
	SeqAddr string
}

func (e EventDataNewBondedSequencer) String() string {
	return fmt.Sprintf("SeqAddr: %s", e.SeqAddr)
}

type EventDataRotationStarted struct {
	NextSeqAddr string
}

func (e EventDataRotationStarted) String() string {
	return fmt.Sprintf("NextSeqAddr: %s", e.NextSeqAddr)
}
