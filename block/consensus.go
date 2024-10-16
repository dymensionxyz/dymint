package block

import (
	"fmt"
	"sync"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/gogo/protobuf/proto"

	"github.com/dymensionxyz/dymint/types"
	rdktypes "github.com/dymensionxyz/dymint/types/pb/rollapp/sequencers/types"
	protoutils "github.com/dymensionxyz/dymint/utils/proto"
	"github.com/dymensionxyz/dymint/utils/queue"
)

type ConsensusMessagesStream interface {
	Add(...proto.Message)
	Get() []proto.Message
}

type ConsensusMessagesQueue struct {
	mu    sync.Mutex
	queue *queue.Queue[proto.Message]
}

func NewConsensusMessagesQueue() *ConsensusMessagesQueue {
	return &ConsensusMessagesQueue{
		mu:    sync.Mutex{},
		queue: queue.New[proto.Message](),
	}
}

func (q *ConsensusMessagesQueue) Add(message ...proto.Message) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue.Enqueue(message...)
}

func (q *ConsensusMessagesQueue) Get() []proto.Message {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.queue.DequeueAll()
}

// ConsensusMsgsOnSequencerSetUpdate forms a list of consensus messages to handle the sequencer set update.
func ConsensusMsgsOnSequencerSetUpdate(newSequencers []types.Sequencer) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0, len(newSequencers))
	for _, s := range newSequencers {
		// Get proposer's consensus public key and convert it to proto.Any
		val, err := s.TMValidator()
		if err != nil {
			return nil, fmt.Errorf("convert next squencer to tendermint validator: %w", err)
		}
		pubKey, err := cryptocodec.FromTmPubKeyInterface(val.PubKey)
		if err != nil {
			return nil, fmt.Errorf("convert tendermint pubkey to cosmos: %w", err)
		}
		anyPK, err := codectypes.NewAnyWithValue(pubKey)
		if err != nil {
			return nil, fmt.Errorf("convert cosmos pubkey to any: %w", err)
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
