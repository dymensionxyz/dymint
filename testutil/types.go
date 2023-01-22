package testutil

import (
	"crypto/rand"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"

	abciconv "github.com/dymensionxyz/dymint/conv/abci"
	"github.com/dymensionxyz/dymint/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	version "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"
)

const (
	// DefaultBatchSize is the default batch size for testing
	DefaultBatchSize = 5
	// BlockVersion is the default block version for testing
	BlockVersion = 1
	// AppVersion is the default app version for testing
	AppVersion = 2
)

func createRandomHashes() [][32]byte {
	h := [][32]byte{}
	for i := 0; i < 8; i++ {
		var h1 [32]byte
		_, err := rand.Read(h1[:])
		if err != nil {
			panic(err)
		}
		h = append(h, h1)
	}
	return h
}

// GenerateBlocks generates random blocks.
func GenerateBlocks(startHeight uint64, num uint64, proposerKey crypto.PrivKey) ([]*types.Block, error) {
	blocks := make([]*types.Block, num)
	for i := uint64(0); i < num; i++ {
		h := createRandomHashes()
		block := &types.Block{
			Header: types.Header{
				Version: types.Version{
					Block: BlockVersion,
					App:   AppVersion,
				},
				NamespaceID:    [8]byte{0, 1, 2, 3, 4, 5, 6, 7},
				Height:         i + startHeight,
				Time:           4567,
				LastHeaderHash: h[0],
				LastCommitHash: h[1],
				DataHash:       h[2],
				ConsensusHash:  h[3],
				// AppHash:         h[4],
				AppHash:         [32]byte{},
				LastResultsHash: getEmptyLastResultsHash(),
				ProposerAddress: []byte{4, 3, 2, 1},
				AggregatorsHash: h[6],
			},
			Data: types.Data{
				Txs:                    nil,
				IntermediateStateRoots: types.IntermediateStateRoots{RawRootsList: [][]byte{{0x1}}},
				Evidence:               types.EvidenceData{Evidence: nil},
			},
			LastCommit: types.Commit{
				Height:     8,
				HeaderHash: h[7],
				Signatures: []types.Signature{},
			},
		}
		signature, err := generateSignature(proposerKey, &block.Header)
		if err != nil {
			return nil, err
		}
		block.LastCommit.Signatures = []types.Signature{signature}
		blocks[i] = block
	}
	return blocks, nil
}

// GenerateCommits generates commits based on passed blocks.
func GenerateCommits(blocks []*types.Block, proposerKey crypto.PrivKey) ([]*types.Commit, error) {
	commits := make([]*types.Commit, len(blocks))
	for i, block := range blocks {
		signature, err := generateSignature(proposerKey, &block.Header)
		if err != nil {
			return nil, err
		}
		commits[i] = &types.Commit{
			Height:     block.Header.Height,
			HeaderHash: block.Header.Hash(),
			Signatures: []types.Signature{signature},
		}
	}
	return commits, nil
}

func generateSignature(proposerKey crypto.PrivKey, header *types.Header) ([]byte, error) {
	abciHeaderPb := abciconv.ToABCIHeaderPB(header)
	abciHeaderBytes, err := abciHeaderPb.Marshal()
	if err != nil {
		return nil, err
	}
	sign, err := proposerKey.Sign(abciHeaderBytes)
	if err != nil {
		return nil, err
	}
	return sign, nil
}

// GenerateBatch generates a batch out of random blocks
func GenerateBatch(startHeight uint64, endHeight uint64, proposerKey crypto.PrivKey) (*types.Batch, error) {
	blocks, err := GenerateBlocks(startHeight, endHeight-startHeight+1, proposerKey)
	if err != nil {
		return nil, err
	}
	commits, err := GenerateCommits(blocks, proposerKey)
	if err != nil {
		return nil, err
	}
	batch := &types.Batch{
		StartHeight: startHeight,
		EndHeight:   endHeight,
		Blocks:      blocks,
		Commits:     commits,
	}
	return batch, nil
}

// GenerateRandomValidatorSet generates random validator sets
func GenerateRandomValidatorSet() *tmtypes.ValidatorSet {
	pubKey := ed25519.GenPrivKey().PubKey()
	return &tmtypes.ValidatorSet{
		Proposer: &tmtypes.Validator{PubKey: pubKey, Address: pubKey.Address()},
		Validators: []*tmtypes.Validator{
			{PubKey: pubKey, Address: pubKey.Address()},
		},
	}
}

// GenerateState generates an initial state for testing.
func GenerateState(initialHeight int64, lastBlockHeight int64) types.State {
	return types.State{
		ChainID:         "test-chain",
		InitialHeight:   initialHeight,
		AppHash:         [32]byte{},
		LastResultsHash: getEmptyLastResultsHash(),
		Version: tmstate.Version{
			Consensus: version.Consensus{
				Block: BlockVersion,
				App:   AppVersion,
			},
		},
		LastBlockHeight: lastBlockHeight,
		LastValidators:  GenerateRandomValidatorSet(),
		Validators:      GenerateRandomValidatorSet(),
		NextValidators:  GenerateRandomValidatorSet(),
	}
}

// GenerateGenesis generates a genesis for testing.
func GenerateGenesis(initialHeight int64) *tmtypes.GenesisDoc {
	return &tmtypes.GenesisDoc{
		ChainID:       "test-chain",
		InitialHeight: initialHeight,
		ConsensusParams: &tmproto.ConsensusParams{
			Block: tmproto.BlockParams{
				MaxBytes:   1024,
				MaxGas:     -1,
				TimeIotaMs: 1000,
			},
			Evidence: tmproto.EvidenceParams{
				MaxAgeNumBlocks: 100,
				MaxAgeDuration:  time.Second,
			},
			Validator: tmproto.ValidatorParams{
				PubKeyTypes: []string{"ed25519"},
			},
			Version: tmproto.VersionParams{
				AppVersion: AppVersion,
			},
		},
	}
}

func getEmptyLastResultsHash() [32]byte {
	lastResults := []*abci.ResponseDeliverTx{}
	return *(*[32]byte)(tmtypes.NewResults(lastResults).Hash())
}
