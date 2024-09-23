package block

import "google.golang.org/protobuf/proto"

type ConsensusMessagesStream interface {
	GetConsensusMessages() ([]proto.Message, error)
}
