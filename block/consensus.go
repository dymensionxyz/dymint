package block

import (
	"fmt"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	rdktypes "github.com/dymensionxyz/dymension-rdk/x/sequencers/types"
	"github.com/gogo/protobuf/proto"

	"github.com/dymensionxyz/dymint/types"
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

func ConsensusMsgSigner(m proto.Message) (sdk.AccAddress, error) {
	switch m.(type) {
	case *rdktypes.ConsensusMsgUpsertSequencer:
		const RdkSequencersModuleName = "sequencers"
		return authtypes.NewModuleAddress(RdkSequencersModuleName), nil
	default:
		return nil, fmt.Errorf("unknown consensus msg")
	}
}

// ConsensusMsgsOnSequencerSetUpdate forms a list of consensus messages to handle the sequencer set update.
func ConsensusMsgsOnSequencerSetUpdate(newSequencers []types.Sequencer) ([]proto.Message, error) {
	msgs := make([]proto.Message, 0, len(newSequencers))
	for _, s := range newSequencers {
		anyPK, err := s.AnyConsPubKey()
		if err != nil {
			return nil, fmt.Errorf("sequencer consensus public key: %w", err)
		}
		signer, err := ConsensusMsgSigner(new(rdktypes.ConsensusMsgUpsertSequencer))
		if err != nil {
			return nil, fmt.Errorf("consensus msg signer: %w", err)
		}
		msgs = append(msgs, &rdktypes.ConsensusMsgUpsertSequencer{
			Signer:     signer.String(),
			Operator:   s.SettlementAddress,
			ConsPubKey: anyPK,
			RewardAddr: s.RewardAddr,
			Relayers:   s.WhitelistedRelayers,
		})
	}
	return msgs, nil
}
