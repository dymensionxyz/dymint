package types

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/math"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	proto "github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	"github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"
)

func TestBlock_ValidateWithState(t *testing.T) {
	proposer := NewSequencerFromValidator(*tmtypes.NewValidator(ed25519.GenPrivKey().PubKey(), 1))
	proposerHash := proposer.MustHash()
	currentTime := time.Now().UTC()
	validState := &State{
		Version: tmstate.Version{
			Consensus: version.Consensus{
				Block: 1,
				App:   1,
			},
		},
		LastBlockHeight: atomic.Uint64{},
		AppHash:         [32]byte{1, 2, 3},
		LastResultsHash: [32]byte{4, 5, 6},
		LastHeaderHash:  [32]byte{7, 8, 9},
		ChainID:         "chainID",
	}
	validState.LastBlockHeight.Store(9)
	validState.SetProposer(proposer)

	// TODO: tests should be written to mutate valid block
	validBlock := &Block{
		Header: Header{
			Version: Version{
				Block: 1,
				App:   1,
			},
			Height:                10,
			Time:                  currentTime.UnixNano(),
			AppHash:               [32]byte{1, 2, 3},
			LastResultsHash:       [32]byte{4, 5, 6},
			ProposerAddress:       []byte("proposer"),
			DataHash:              [32]byte{},
			LastHeaderHash:        [32]byte{7, 8, 9},
			ChainID:               "chainID",
			SequencerHash:         [32]byte(proposerHash),
			ConsensusMessagesHash: ConsMessagesHash(nil),
		},
		Data:       Data{},
		LastCommit: Commit{},
	}
	validBlock.Header.DataHash = [32]byte(GetDataHash(validBlock))

	tests := []struct {
		name            string
		block           *Block
		state           *State
		wantErr         bool
		theErr          error
		expectedErrType interface{}
		isFraud         bool
	}{
		{
			name:    "Valid block",
			block:   validBlock,
			state:   validState,
			wantErr: false,
			isFraud: false,
		},
		{
			name: "Invalid block version",
			block: &Block{
				Header: Header{
					Version: Version{
						Block: 2,
						App:   1,
					},
					Height:                10,
					Time:                  currentTime.UnixNano(),
					AppHash:               [32]byte{1, 2, 3},
					LastResultsHash:       [32]byte{4, 5, 6},
					ProposerAddress:       []byte("proposer"),
					DataHash:              [32]byte(GetDataHash(validBlock)),
					LastHeaderHash:        [32]byte{7, 8, 9},
					ChainID:               "chainID",
					ConsensusMessagesHash: ConsMessagesHash(nil),
				},
			},
			state:   validState,
			theErr:  ErrVersionMismatch,
			wantErr: true,
			isFraud: false,
		},
		{
			name: "Invalid app version",
			block: &Block{
				Header: Header{
					Version: Version{
						Block: 1,
						App:   2,
					},
					Height:                10,
					Time:                  currentTime.UnixNano(),
					AppHash:               [32]byte{1, 2, 3},
					LastResultsHash:       [32]byte{4, 5, 6},
					ProposerAddress:       []byte("proposer"),
					DataHash:              [32]byte(GetDataHash(validBlock)),
					LastHeaderHash:        [32]byte{7, 8, 9},
					ChainID:               "chainID",
					ConsensusMessagesHash: ConsMessagesHash(nil),
				},
			},
			state:   validState,
			wantErr: true,
			theErr:  ErrVersionMismatch,
			isFraud: false,
		},
		{
			name: "Invalid height",
			block: &Block{
				Header: Header{
					Version:               validBlock.Header.Version,
					Height:                11,
					Time:                  currentTime.UnixNano(),
					AppHash:               [32]byte{1, 2, 3},
					LastResultsHash:       [32]byte{4, 5, 6},
					ProposerAddress:       []byte("proposer"),
					DataHash:              [32]byte(GetDataHash(validBlock)),
					ConsensusMessagesHash: ConsMessagesHash(nil),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: &ErrFraudHeightMismatch{},
			isFraud:         true,
		},
		{
			name: "Invalid AppHash",
			block: &Block{
				Header: Header{
					Version:               validBlock.Header.Version,
					Height:                10,
					Time:                  currentTime.UnixNano(),
					AppHash:               [32]byte{9, 9, 9},
					LastResultsHash:       [32]byte{4, 5, 6},
					ProposerAddress:       []byte("proposer"),
					DataHash:              [32]byte(GetDataHash(validBlock)),
					ConsensusMessagesHash: ConsMessagesHash(nil),
				},
			},
			state:           validState,
			expectedErrType: &ErrFraudAppHashMismatch{},
			wantErr:         true,
			isFraud:         true,
		},
		{
			name: "Invalid LastResultsHash",
			block: &Block{
				Header: Header{
					Version:               validBlock.Header.Version,
					Height:                10,
					Time:                  currentTime.UnixNano(),
					AppHash:               [32]byte{1, 2, 3},
					LastResultsHash:       [32]byte{9, 9, 9},
					ProposerAddress:       []byte("proposer"),
					DataHash:              [32]byte(GetDataHash(validBlock)),
					ConsensusMessagesHash: ConsMessagesHash(nil),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: &ErrLastResultsHashMismatch{},
			isFraud:         true,
		},
		{
			name: "Future block time",
			block: &Block{
				Header: Header{
					Version:               validBlock.Header.Version,
					Height:                10,
					Time:                  currentTime.Add(2 * TimeFraudMaxDrift).UnixNano(),
					AppHash:               [32]byte{1, 2, 3},
					LastResultsHash:       [32]byte{4, 5, 6},
					ProposerAddress:       []byte("proposer"),
					DataHash:              [32]byte(GetDataHash(validBlock)),
					ConsensusMessagesHash: ConsMessagesHash(nil),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: &ErrTimeFraud{},
			isFraud:         true,
		},
		{
			name: "Invalid proposer address",
			block: &Block{
				Header: Header{
					Version:               validBlock.Header.Version,
					Height:                10,
					Time:                  currentTime.UnixNano(),
					AppHash:               [32]byte{1, 2, 3},
					LastResultsHash:       [32]byte{4, 5, 6},
					ProposerAddress:       []byte{},
					ConsensusMessagesHash: ConsMessagesHash(nil),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: ErrEmptyProposerAddress,
			isFraud:         false,
		},
		{
			name: "invalid last header hash",
			block: &Block{
				Header: Header{
					Version:               validBlock.Header.Version,
					Height:                10,
					Time:                  currentTime.UnixNano(),
					AppHash:               [32]byte{1, 2, 3},
					LastResultsHash:       [32]byte{4, 5, 6},
					ProposerAddress:       []byte("proposer"),
					DataHash:              [32]byte(GetDataHash(validBlock)),
					LastHeaderHash:        [32]byte{1, 2, 3},
					ConsensusMessagesHash: ConsMessagesHash(nil),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: &ErrLastHeaderHashMismatch{},
			isFraud:         true,
		},
		{
			name: "invalid chain ID",
			block: &Block{
				Header: Header{
					Version:               validBlock.Header.Version,
					Height:                10,
					Time:                  currentTime.UnixNano(),
					AppHash:               [32]byte{1, 2, 3},
					LastResultsHash:       [32]byte{4, 5, 6},
					ProposerAddress:       []byte("proposer"),
					DataHash:              [32]byte(GetDataHash(validBlock)),
					LastHeaderHash:        [32]byte{7, 8, 9},
					ChainID:               "invalidChainID",
					ConsensusMessagesHash: ConsMessagesHash(nil),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: &ErrInvalidChainID{},
			isFraud:         true,
		},
		{
			name: "invalid NextSequencersHash",
			block: &Block{
				Header: Header{
					Version:               validBlock.Header.Version,
					Height:                10,
					Time:                  currentTime.UnixNano(),
					AppHash:               [32]byte{1, 2, 3},
					LastResultsHash:       [32]byte{4, 5, 6},
					ProposerAddress:       []byte("proposer"),
					DataHash:              [32]byte(GetDataHash(validBlock)),
					LastHeaderHash:        [32]byte{7, 8, 9},
					ChainID:               "chainID",
					NextSequencersHash:    [32]byte{1, 2, 3},
					ConsensusMessagesHash: ConsMessagesHash(nil),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: &ErrInvalidNextSequencersHashFraud{},
			isFraud:         true,
		},
		{
			name: "invalid header data hash",
			block: &Block{
				Header: Header{
					Version:               validBlock.Header.Version,
					Height:                10,
					Time:                  currentTime.UnixNano(),
					AppHash:               [32]byte{1, 2, 3},
					LastResultsHash:       [32]byte{4, 5, 6},
					ProposerAddress:       []byte("proposer"),
					DataHash:              [32]byte{1, 2, 3},
					LastHeaderHash:        [32]byte{7, 8, 9},
					ChainID:               "chainID",
					ConsensusMessagesHash: ConsMessagesHash(nil),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: ErrInvalidHeaderDataHashFraud{},
			isFraud:         true,
		},
		{
			name: "invalid dym header",
			block: &Block{
				Header: Header{
					Version:               validBlock.Header.Version,
					Height:                10,
					Time:                  currentTime.UnixNano(),
					AppHash:               [32]byte{1, 2, 3},
					LastResultsHash:       [32]byte{4, 5, 6},
					ProposerAddress:       []byte("proposer"),
					DataHash:              [32]byte{1, 2, 3},
					LastHeaderHash:        [32]byte{7, 8, 9},
					ChainID:               "chainID",
					ConsensusMessagesHash: ConsMessagesHash([]*proto.Any{{}}),
				},
			},
			state:           validState,
			wantErr:         true,
			expectedErrType: ErrInvalidDymHeaderFraud{},
			isFraud:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.block.ValidateWithState(tt.state)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.isFraud {
					require.True(t, errors.Is(err, gerrc.ErrFault))
					if tt.expectedErrType != nil {
						assert.True(t, errors.As(err, &tt.expectedErrType),
							"expected error of type %T, got %T", tt.expectedErrType, err)
					}
				} else {
					require.False(t, errors.Is(err, gerrc.ErrFault))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCommit_ValidateWithHeader(t *testing.T) {
	// Generate keys for the proposer and another actor for invalid signatures
	proposerKey := ed25519.GenPrivKey()
	anotherKey := ed25519.GenPrivKey()

	// Helper function to create a valid commit
	createValidCommit := func() (*Commit, *Block, []byte, error) {
		seq := NewSequencerFromValidator(*tmtypes.NewValidator(proposerKey.PubKey(), 1))
		proposerHash := seq.MustHash()

		block := &Block{
			Header: Header{
				Version: Version{
					Block: 1,
					App:   1,
				},
				ChainID:               "test",
				Height:                1,
				Time:                  time.Now().UTC().UnixNano(),
				LastHeaderHash:        [32]byte{},
				DataHash:              [32]byte{},
				ConsensusHash:         [32]byte{},
				AppHash:               [32]byte{},
				LastResultsHash:       [32]byte{},
				ProposerAddress:       proposerKey.PubKey().Address(),
				SequencerHash:         [32]byte(proposerHash),
				ConsensusMessagesHash: ConsMessagesHash(nil),
			},
		}

		abciHeaderBytes, signature, err := signBlock(block, proposerKey)
		if err != nil {
			return nil, nil, nil, err
		}

		commit := &Commit{
			Height:     block.Header.Height,
			HeaderHash: block.Hash(),
			Signatures: []Signature{signature},
		}

		return commit, block, abciHeaderBytes, nil
	}

	t.Run("Valid commit", func(t *testing.T) {
		commit, block, _, err := createValidCommit()
		require.NoError(t, err, "Creating the valid commit should not fail")

		err = commit.ValidateWithHeader(proposerKey.PubKey(), &block.Header)
		assert.NoError(t, err, "Validation should pass without errors")
	})

	t.Run("ValidateBasic fails - invalid height", func(t *testing.T) {
		commit, block, _, err := createValidCommit()
		require.NoError(t, err, "Creating the valid commit should not fail")

		commit.Height = 0 // Set an invalid height so ValidateBasic fails

		err = commit.ValidateWithHeader(proposerKey.PubKey(), &block.Header)
		require.Error(t, err, "Validation should fail due to an invalid height")
		require.Equal(t, &ErrInvalidBlockHeightFraud{0, &block.Header}, err)
		require.True(t, errors.Is(err, gerrc.ErrFault), "The error should be a fraud error")
	})

	t.Run("Invalid signature", func(t *testing.T) {
		commit, block, abciHeaderBytes, err := createValidCommit()
		require.NoError(t, err, "Creating the valid commit should not fail")

		// Generate an invalid signature using another key
		invalidSignature, err := anotherKey.Sign(abciHeaderBytes)
		require.NoError(t, err, "Generating the invalid signature should not fail")

		commit.Signatures = []Signature{invalidSignature}

		err = commit.ValidateWithHeader(proposerKey.PubKey(), &block.Header)
		require.Error(t, err, "Validation should fail due to an invalid signature")
		assert.True(t, errors.Is(err, gerrc.ErrInvalidArgument))
	})

	t.Run("Fails with more than one signature", func(t *testing.T) {
		commit, block, abciHeaderBytes, err := createValidCommit()
		require.NoError(t, err, "Creating the valid commit should not fail")

		// Generate another valid signature using a different key
		anotherSignature, err := anotherKey.Sign(abciHeaderBytes)
		require.NoError(t, err, "Generating an additional signature should not fail")

		// Add the additional signature to the commit
		commit.Signatures = append(commit.Signatures, anotherSignature)

		// Ensure there are more than one signature
		require.Greater(t, len(commit.Signatures), 1, "Commit should have more than one signature")

		// Validate and expect an error due to multiple signatures
		err = commit.ValidateWithHeader(proposerKey.PubKey(), &block.Header)
		require.Error(t, err, "Validation should fail when there is more than one signature")
		assert.True(t, errors.Is(err, gerrc.ErrInvalidArgument))
	})

	t.Run("Fails when signature size exceeds MaxSignatureSize", func(t *testing.T) {
		commit, block, _, err := createValidCommit()
		require.NoError(t, err, "Creating the valid commit should not fail")

		// Define a signature that exceeds MaxSignatureSize
		invalidSignature := make([]byte, math.MaxInt(ed25519.SignatureSize, 64)+1)

		// Replace the valid signature with the invalid oversized signature
		commit.Signatures = []Signature{invalidSignature}

		// Ensure the signature size exceeds the maximum allowed size
		require.Greater(t, len(invalidSignature), math.MaxInt(ed25519.SignatureSize, 64), "Signature size should exceed MaxSignatureSize")

		// Validate and expect an error due to oversized signature
		err = commit.ValidateWithHeader(proposerKey.PubKey(), &block.Header)
		require.Error(t, err, "Validation should fail when the signature size exceeds the maximum allowed size")
		assert.True(t, errors.Is(err, gerrc.ErrInvalidArgument))
	})

	t.Run("Fails when proposerPubKey.Address() does not match Header.ProposerAddress", func(t *testing.T) {
		commit, block, _, err := createValidCommit()
		require.NoError(t, err, "Creating the valid commit should not fail")

		// Modify the block header's proposer address to simulate a mismatch
		block.Header.ProposerAddress = anotherKey.PubKey().Address() // Set to a different proposer's address

		// resign the block with the new proposer address
		_, signature, err := signBlock(block, proposerKey)
		if err != nil {
			return
		}

		commit.Signatures = []Signature{signature}

		// Ensure the proposer's address does not match the block header's proposer address
		require.NotEqual(t, proposerKey.PubKey().Address(), block.Header.ProposerAddress, "The proposer's public key address should not match the block header proposer address")

		// Validate and expect an error due to mismatching proposer addresses
		err = commit.ValidateWithHeader(proposerKey.PubKey(), &block.Header)
		require.Error(t, err, "Validation should fail when the proposer's address does not match the header's proposer address")
		assert.Equal(t, NewErrInvalidProposerAddressFraud(block.Header.ProposerAddress, proposerKey.PubKey().Address(), &block.Header), err)
		assert.True(t, errors.Is(err, gerrc.ErrFault), "The error should be a fraud error")
	})

	t.Run("Fails when SequencerHash does not match proposerHash", func(t *testing.T) {
		commit, block, _, err := createValidCommit()
		require.NoError(t, err, "Creating the valid commit should not fail")

		// Modify the block header's SequencerHash to simulate a mismatch
		block.Header.SequencerHash = [32]byte{1, 2, 3} // Set to an invalid hash

		// resign the block with the new SequencerHash
		_, signature, err := signBlock(block, proposerKey)
		if err != nil {
			return
		}

		commit.Signatures = []Signature{signature}

		// Ensure the SequencerHash does not match the proposer's hash
		hash := NewSequencerFromValidator(*tmtypes.NewValidator(proposerKey.PubKey(), 1)).MustHash()
		require.NoError(t, err, "Generating the SequencerHash should not fail")
		require.NotEqual(t, block.Header.SequencerHash, hash, "The SequencerHash should not match the proposer's hash")

		// Validate and expect an error due to mismatching SequencerHash
		err = commit.ValidateWithHeader(proposerKey.PubKey(), &block.Header)
		bytes := NewSequencerFromValidator(*tmtypes.NewValidator(proposerKey.PubKey(), 1)).MustHash()

		require.Equal(t, &ErrInvalidSequencerHashFraud{[32]byte{1, 2, 3}, bytes, &block.Header}, err)
		require.True(t, errors.Is(err, gerrc.ErrFault), "The error should be a fraud error")
	})

	t.Run("HeaderHash does not match Block Hash", func(t *testing.T) {
		commit, block, _, err := createValidCommit()
		require.NoError(t, err, "Creating the valid commit should not fail")

		commit.HeaderHash = [32]byte{1, 2, 3} // Introduce an invalid hash

		assert.NotEqual(t, block.Hash(), commit.HeaderHash, "The commit header hash should not match the block header hash")

		err = commit.ValidateWithHeader(proposerKey.PubKey(), &block.Header)
		require.Equal(t, &ErrInvalidHeaderHashFraud{[32]byte{1, 2, 3}, &block.Header}, err)
		require.True(t, errors.Is(err, gerrc.ErrFault), "The error should be a fraud error")
	})
}

func signBlock(block *Block, proposerKey ed25519.PrivKey) ([]byte, []byte, error) {
	abciHeaderPb := ToABCIHeaderPB(&block.Header)
	abciHeaderBytes, err := abciHeaderPb.Marshal()
	if err != nil {
		return nil, nil, err
	}

	signature, err := proposerKey.Sign(abciHeaderBytes)
	if err != nil {
		return nil, nil, err
	}

	return abciHeaderBytes, signature, nil
}
