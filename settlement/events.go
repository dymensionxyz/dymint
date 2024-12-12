package settlement

import (
	"fmt"

	uevent "github.com/dymensionxyz/dymint/utils/event"
)

const (
	EventTypeKey = "settlement.event"

	EventNewBatchAccepted   = "NewBatchAccepted"
	EventNewBondedSequencer = "NewBondedSequencer"
	EventRotationStarted    = "RotationStarted"
	EventNewBatchFinalized  = "NewBatchFinalized"
)

var (
	EventNewBatchAcceptedList   = map[string][]string{EventTypeKey: {EventNewBatchAccepted}}
	EventNewBondedSequencerList = map[string][]string{EventTypeKey: {EventNewBondedSequencer}}
	EventRotationStartedList    = map[string][]string{EventTypeKey: {EventRotationStarted}}
	EventNewBatchFinalizedList  = map[string][]string{EventTypeKey: {EventNewBatchFinalized}}
)

var (
	EventQueryNewSettlementBatchAccepted  = uevent.QueryFor(EventTypeKey, EventNewBatchAccepted)
	EventQueryNewSettlementBatchFinalized = uevent.QueryFor(EventTypeKey, EventNewBatchFinalized)
	EventQueryNewBondedSequencer          = uevent.QueryFor(EventTypeKey, EventNewBondedSequencer)
	EventQueryRotationStarted             = uevent.QueryFor(EventTypeKey, EventRotationStarted)
)

type EventDataNewBatch struct {
	StartHeight uint64

	EndHeight uint64

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
