package types

import crypto "github.com/cosmos/cosmos-sdk/crypto/types"

// SequencerStatus defines the operating status of a sequencer
type SequencerStatus int32

const (
	// Proposer defines a sequencer that is currently the proposer
	Proposer SequencerStatus = iota
	// Inactive defines a sequencer that is currently inactive
	Inactive
)

// Sequencer represents a sequencer of the rollapp
type Sequencer struct {
	// PublicKey is the public key of the sequencer
	PublicKey crypto.PubKey
	// Status is status of the sequencer
	Status SequencerStatus
}
