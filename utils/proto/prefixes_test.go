package proto_test

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"

	prefix1 "github.com/dymensionxyz/dymint/utils/proto/test_data/prefix_1"
	prefix2 "github.com/dymensionxyz/dymint/utils/proto/test_data/prefix_2"
)

// TestDifferentPrefixes ensures that using different package prefixes for the same
// type doesn't affect binary serialization/deserialization.
func TestDifferentPrefixes(t *testing.T) {
	_, cosmosProtoPK := GenerateGogoProtoPK(t)

	// Create two fully equal messages:
	//  - the first has 'dymension_rdk' prefix
	//  - the second has 'rollapp' prefix
	msgPrefix1 := &prefix1.ConsensusMsgUpsertSequencer{
		Signer:     "1234567890",
		Operator:   "1234567890",
		ConsPubKey: cosmosProtoPK,
		RewardAddr: "1234567890",
		Relayers:   []string{"1234567890", "1234567890"},
	}
	msgPrefix2 := &prefix2.ConsensusMsgUpsertSequencer{
		Signer:     "1234567890",
		Operator:   "1234567890",
		ConsPubKey: cosmosProtoPK,
		RewardAddr: "1234567890",
		Relayers:   []string{"1234567890", "1234567890"},
	}

	// Serialize both messages to bytes
	bytesMsg1, err := proto.Marshal(msgPrefix1)
	require.NoError(t, err)
	bytesMsg2, err := proto.Marshal(msgPrefix2)
	require.NoError(t, err)

	// Ensure that the bytes representation is the same
	require.Equal(t, bytesMsg1, bytesMsg2)

	// Try to deserialize the second bytes into the first message
	actualMsgPrefix1 := new(prefix1.ConsensusMsgUpsertSequencer)
	err = proto.Unmarshal(bytesMsg2, actualMsgPrefix1)

	// Try to deserialize the first bytes into the second message
	actualMsgPrefix2 := new(prefix2.ConsensusMsgUpsertSequencer)
	err = proto.Unmarshal(bytesMsg1, actualMsgPrefix2)

	// Verify that the messages are the same as the initial
	require.Equal(t, msgPrefix1, actualMsgPrefix1)
	require.Equal(t, msgPrefix2, actualMsgPrefix2)

	// Verify that all fields match
	require.Equal(t, actualMsgPrefix1.Signer, actualMsgPrefix2.Signer)
	require.Equal(t, actualMsgPrefix1.Operator, actualMsgPrefix2.Operator)
	require.Equal(t, actualMsgPrefix1.ConsPubKey, actualMsgPrefix2.ConsPubKey)
	require.Equal(t, actualMsgPrefix1.RewardAddr, actualMsgPrefix2.RewardAddr)
	require.Equal(t, actualMsgPrefix1.Relayers, actualMsgPrefix2.Relayers)
}
