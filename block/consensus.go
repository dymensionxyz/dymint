package block

import (
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"

	"github.com/dymensionxyz/dymint/types"
	rdktypes "github.com/dymensionxyz/dymint/types/pb/rollapp/sequencers/types"
	protoutils "github.com/dymensionxyz/dymint/utils/proto"
	"github.com/dymensionxyz/dymint/utils/queue"
)

type ConsensusMsgQueue struct {
	mu    sync.Mutex
	queue *queue.Queue[proto.Message]
}

func NewConsensusMsgQueue() *ConsensusMsgQueue {
	return &ConsensusMsgQueue{
		mu:    sync.Mutex{},
		queue: queue.New[proto.Message](),
	}
}

func (q *ConsensusMsgQueue) Add(message ...proto.Message) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue.Enqueue(message...)
}

func (q *ConsensusMsgQueue) Get() []proto.Message {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.queue.DequeueAll()
}

// ConsensusMsgsOnSequencerSetUpdate forms a list of consensus messages to handle the sequencer set update.
func ConsensusMsgsOnSequencerSetUpdate(newSequencers []types.Sequencer) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0, len(newSequencers))
	for _, s := range newSequencers {
		anyPK, err := s.AnyConsPubKey()
		if err != nil {
			return nil, fmt.Errorf("sequencer consensus public key: %w", err)
		}
		msgs = append(msgs, &rdktypes.ConsensusMsgUpsertSequencer{
			Operator:   s.SettlementAddress,
			ConsPubKey: protoutils.CosmosToGogo(anyPK),
			RewardAddr: s.RewardAddr,
			Relayers:   s.WhitelistedRelayers,
		})
	}
	return msgs, nil
}
