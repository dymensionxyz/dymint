package sequencer

const (
	EventTypeCreateSequencer = "create_sequencer"
	AttributeKeyRollappId    = "rollapp_id"
	AttributeKeySequencer    = "sequencer"
	AttributeKeyBond         = "bond"
	AttributeKeyProposer     = "proposer"

	EventTypeUnbonding         = "unbonding"
	AttributeKeyCompletionTime = "completion_time"

	EventTypeNoBondedSequencer = "no_bonded_sequencer"

	EventTypeProposerRotated = "proposer_rotated"

	EventTypeUnbonded = "unbonded"

	EventTypeSlashed = "slashed"
)
