package types

// TODO: test SetProposer adds to the val set
/*
func TestSequencerSet_GetProposerPubKey(t *testing.T) {
	validator := &types.Validator{
		PubKey: crypto.Ed25519PublicKey([]byte("pubkey")),
	}
	sequencerSet := &SequencerSet{
		Proposer: validator,
	}
	expectedPubKey := crypto.Ed25519PublicKey([]byte("pubkey"))

	actualPubKey := sequencerSet.GetProposerPubKey()

	require.Equal(t, expectedPubKey, actualPubKey)
}

func TestSequencerSet_SetProposerByHash(t *testing.T) {
	validator1 := &types.Validator{
		Address: []byte("address1"),
	}
	validator2 := &types.Validator{
		Address: []byte("address2"),
	}
	sequencerSet := &SequencerSet{
		Validators: []*types.Validator{validator1, validator2},
	}
	hash := []byte("hash")

	err := sequencerSet.SetProposerByHash(hash)

	require.NoError(t, err)
	require.Equal(t, validator2, sequencerSet.Proposer)
}

func TestSequencerSet_SetProposer(t *testing.T) {
	validator := &types.Validator{
		Address: []byte("address"),
	}
	sequencerSet := &SequencerSet{}

	sequencerSet.SetProposer(validator)

	require.Equal(t, validator, sequencerSet.Proposer)
	require.Equal(t, []byte("address"), sequencerSet.ProposerHash)
	require.Equal(t, []*types.Validator{validator}, sequencerSet.Validators)
}

func TestSequencerSet_SetBondedSet(t *testing.T) {
	validator1 := &types.Validator{
		Address: []byte("address1"),
	}
	validator2 := &types.Validator{
		Address: []byte("address2"),
	}
	bondedSet := &types.ValidatorSet{
		Validators: []*types.Validator{validator1, validator2},
		Proposer:   validator1,
	}
	sequencerSet := &SequencerSet{}

	sequencerSet.SetBondedSet(bondedSet)

	require.Equal(t, []*types.Validator{validator1, validator2}, sequencerSet.Validators)
	require.Equal(t, validator1, sequencerSet.Proposer)
	require.Equal(t, []byte("address1"), sequencerSet.ProposerHash)
}

func TestSequencerSet_GetByAddress(t *testing.T) {
	validator1 := &types.Validator{
		Address: []byte("address1"),
	}
	validator2 := &types.Validator{
		Address: []byte("address2"),
	}
	sequencerSet := &SequencerSet{
		Validators: []*types.Validator{validator1, validator2},
	}

	actualValidator := sequencerSet.GetByAddress([]byte("address1"))

	require.Equal(t, validator1, actualValidator)
}

func TestSequencerSet_HasAddress(t *testing.T) {
	validator1 := &types.Validator{
		Address: []byte("address1"),
	}
	validator2 := &types.Validator{
		Address: []byte("address2"),
	}
	sequencerSet := &SequencerSet{
		Validators: []*types.Validator{validator1, validator2},
	}

	hasAddress := sequencerSet.HasAddress([]byte("address1"))

	require.True(t, hasAddress)
}

func TestSequencerSet_ToValSet(t *testing.T) {
	validator1 := &types.Validator{
		Address: []byte("address1"),
	}
	validator2 := &types.Validator{
		Address: []byte("address2"),
	}
	sequencerSet := &SequencerSet{
		Validators: []*types.Validator{validator1, validator2},
		Proposer:   validator1,
	}

	valSet := sequencerSet.ToValSet()

	require.Equal(t, []*types.Validator{validator1, validator2}, valSet.Validators)
	require.Equal(t, validator1, valSet.Proposer)
}

func TestSequencerSet_ToProto(t *testing.T) {
	validator1 := &types.Validator{
		Address: []byte("address1"),
	}
	validator2 := &types.Validator{
		Address: []byte("address2"),
	}
	sequencerSet := &SequencerSet{
		Validators: []*types.Validator{validator1, validator2},
		Proposer:   validator1,
	}

	protoSet, err := sequencerSet.ToProto()

	require.NoError(t, err)
	require.Equal(t, 2, len(protoSet.Sequencers))
	require.Equal(t, validator1, protoSet.Proposer)
}

func TestNewSequencerSetFromProto(t *testing.T) {
	valProto1 := &types.Validator{
		Address: []byte("address1"),
	}
	valProto2 := &types.Validator{
		Address: []byte("address2"),
	}
	protoSet := &pb.SequencerSet{
		Sequencers: []*cmtproto.Validator{valProto1, valProto2},
		Proposer:   valProto1,
	}

	sequencerSet, err := NewSequencerSetFromProto(*protoSet)

	require.NoError(t, err)
	require.Equal(t, 2, len(sequencerSet.Validators))
	require.Equal(t, valProto1, sequencerSet.Proposer)
}

func TestSequencerSet_Copy(t *testing.T) {
	validator1 := &types.Validator{
		Address: []byte("address1"),
	}
	validator2 := &types.Validator{
		Address: []byte("address2"),
	}
	sequencerSet := &SequencerSet{
		Validators:   []*types.Validator{validator1, validator2},
		Proposer:     validator1,
		ProposerHash: []byte("hash"),
	}

	copy := sequencerSet.Copy()

	require.Equal(t, sequencerSet.Validators, copy.Validators)
	require.Equal(t, sequencerSet.Proposer, copy.Proposer)
	require.Equal(t, sequencerSet.ProposerHash, copy.ProposerHash)
	require.False(t, bytes.Equal(sequencerSet.Validators[0].Address, copy.Validators[0].Address))
	require.False(t, bytes.Equal(sequencerSet.Proposer.Address, copy.Proposer.Address))
}
*/
