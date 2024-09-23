package block

import "github.com/gogo/protobuf/proto"

type ConsensusMessagesStream interface {
	GetConsensusMessages() ([]proto.Message, error)
}
