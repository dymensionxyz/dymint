package p2p

import (
	"github.com/dymensionxyz/dymint/p2p/pb"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/utils/event"
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
/*                                 Event Data                                 */
/* -------------------------------------------------------------------------- */

// GossipedBlock defines the struct of the event data for the GossipedBlock
type GossipedBlock struct {
	// Block is the block that was gossiped
	Block types.Block
	// Commit is the commit that was gossiped
	Commit types.Commit
}

// MarshalBinary encodes GossipedBlock into binary form and returns it.
func (e *GossipedBlock) MarshalBinary() ([]byte, error) {
	return e.ToProto().Marshal()
}

// UnmarshalBinary decodes binary form of GossipedBlock into object.
func (e *GossipedBlock) UnmarshalBinary(data []byte) error {
	var pbGossipedBlock pb.GossipedBlock
	err := pbGossipedBlock.Unmarshal(data)
	if err != nil {
		return err
	}
	err = e.FromProto(&pbGossipedBlock)
	return err
}

// ToProto converts Data into protobuf representation and returns it.
func (e *GossipedBlock) ToProto() *pb.GossipedBlock {
	return &pb.GossipedBlock{
		Block:  e.Block.ToProto(),
		Commit: e.Commit.ToProto(),
	}
}

// FromProto fills GossipedBlock with data from its protobuf representation.
func (e *GossipedBlock) FromProto(other *pb.GossipedBlock) error {
	if err := e.Block.FromProto(other.Block); err != nil {
		return err
	}
	if err := e.Commit.FromProto(other.Commit); err != nil {
		return err
	}
	return nil
}

// Validate run basic validation on the gossiped block
func (e *GossipedBlock) Validate() error {
	if err := e.Block.ValidateBasic(); err != nil {
		return err
	}
	if err := e.Commit.ValidateBasic(); err != nil {
		return err
	}
	return nil
}

/* -------------------------------------------------------------------------- */
/*                                   Queries                                  */
/* -------------------------------------------------------------------------- */

// EventQueryNewNewGossipedBlock is the query used for getting EventNewGossipedBlock
var EventQueryNewNewGossipedBlock = event.QueryFor(EventTypeKey, EventNewGossipedBlock)
