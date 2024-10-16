package block

import (
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/gogo/protobuf/proto"

	rdktypes "github.com/dymensionxyz/dymint/types/pb/rollapp/sequencers/types"
	protoutils "github.com/dymensionxyz/dymint/utils/proto"
)

type ConsensusMessagesStream interface {
	GetConsensusMessages() ([]proto.Message, error)
}

// consensusMsgsOnCreateBlock forms a list of consensus messages that need execution on rollapp's BeginBlock.
// Currently, we need to create a sequencer in the rollapp if it doesn't exist in the following cases:
//   - On the very first block after the genesis or
//   - On the last block of the current sequencer (eg, during the rotation).
func (m *Manager) consensusMsgsOnCreateBlock(
	nextProposerSettlementAddr string,
	lastSeqBlock bool, // Indicates that the block is the last for the current seq. True during the rotation.
) ([]proto.Message, error) {
	if !m.State.IsGenesis() && !lastSeqBlock {
		return nil, nil
	}

	nextSeq := m.State.Sequencers.GetByAddress(nextProposerSettlementAddr)
	// Sanity check. Must never happen in practice. The sequencer's existence is verified beforehand in Manager.CompleteRotation.
	if nextSeq == nil {
		return nil, fmt.Errorf("no sequencer found for address while creating a new block: %s", nextProposerSettlementAddr)
	}

	// Get proposer's consensus public key and convert it to proto.Any
	val, err := nextSeq.TMValidator()
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

	return []proto.Message{&rdktypes.ConsensusMsgUpsertSequencer{
		Operator:   nextProposerSettlementAddr,
		ConsPubKey: protoutils.CosmosToGogo(anyPK),
		RewardAddr: nextProposerSettlementAddr,
	}}, nil
}
