package proto_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmos "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/gogo/protobuf/proto"
	gogo "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/require"

	protoutils "github.com/dymensionxyz/dymint/utils/proto"
	cosmosseq "github.com/dymensionxyz/dymint/utils/proto/test_data/cosmos_proto"
	gogoseq "github.com/dymensionxyz/dymint/utils/proto/test_data/gogo_proto"
)

func TestConvertCosmosToGogo(t *testing.T) {
	setup()

	expectedPK, gogoProtoPK := GenerateGogoProtoPK(t)

	// create a sequencer with gogo any
	gogoSeq := &gogoseq.Sequencer{
		DymintPubKey: gogoProtoPK,
	}

	// serialize the sequencer with gogo any
	gogoSeqBytes, err := proto.Marshal(gogoSeq)
	require.NoError(t, err)

	// deserialize the sequencer, now it holds a cosmos any
	cosmosSeq := cosmosseq.Sequencer{}
	err = proto.Unmarshal(gogoSeqBytes, &cosmosSeq)
	require.NoError(t, err)

	// try to deserialize gogo to cosmos any
	var actualPK cryptotypes.PubKey
	err = cdc.UnpackAny(cosmosSeq.DymintPubKey, &actualPK)
	require.NoError(t, err)

	// check that the value matches the initial
	require.Equal(t, expectedPK, actualPK)
}

func TestConvertGogoToCosmos(t *testing.T) {
	setup()

	expectedPK, cosmosProtoPK := GenerateCosmosProtoPK(t)

	// create a sequencer with cosmos any
	cosmosSeq := &cosmosseq.Sequencer{
		DymintPubKey: cosmosProtoPK,
	}

	// serialize the sequencer with cosmos any
	cosmosSeqBytes, err := proto.Marshal(cosmosSeq)
	require.NoError(t, err)

	// deserialize the sequencer, now it holds gogo any
	gogoSeq := gogoseq.Sequencer{}
	err = proto.Unmarshal(cosmosSeqBytes, &gogoSeq)
	require.NoError(t, err)

	// convert gogo to cosmos any
	cosmosProtoPK1 := protoutils.GogoToCosmos(gogoSeq.DymintPubKey)

	// try to deserialize it to cosmos any
	var actualPK cryptotypes.PubKey
	err = cdc.UnpackAny(cosmosProtoPK1, &actualPK)
	require.NoError(t, err)

	// check that the value matches the initial
	require.Equal(t, expectedPK, actualPK)
}

func GenerateGogoProtoPK(t *testing.T) (cryptotypes.PubKey, *gogo.Any) {
	t.Helper()

	// generate the expected pubkey
	expectedPK := ed25519.GenPrivKey().PubKey()

	// convert it to cosmos any
	cosmosProtoPK, err := cdctypes.NewAnyWithValue(expectedPK)
	require.NoError(t, err)

	// convert cosmos to gogo any
	gogoProtoPK := protoutils.CosmosToGogo(cosmosProtoPK)

	return expectedPK, gogoProtoPK
}

func GenerateCosmosProtoPK(t *testing.T) (cryptotypes.PubKey, *cosmos.Any) {
	t.Helper()

	// generate the expected pubkey
	expectedPK := ed25519.GenPrivKey().PubKey()

	// convert it to cosmos any
	cosmosProtoPK, err := cdctypes.NewAnyWithValue(expectedPK)
	require.NoError(t, err)

	return expectedPK, cosmosProtoPK
}

var cdc *codec.ProtoCodec

func setup() {
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	cdc = codec.NewProtoCodec(interfaceRegistry)
}
