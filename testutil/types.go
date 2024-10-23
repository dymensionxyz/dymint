package testutil

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dymint"
	dymintversion "github.com/dymensionxyz/dymint/version"
	"github.com/libp2p/go-libp2p/core/crypto"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	version "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"
)

const (
	// BlockVersion is the default block version for testing
	BlockVersion = 1
	// AppVersion is the default app version for testing
	AppVersion = 2
)

func createRandomHashes() [][32]byte {
	h := [][32]byte{}
	for i := 0; i < 4; i++ {
		var h1 [32]byte
		_, err := rand.Read(h1[:])
		if err != nil {
			panic(err)
		}
		h = append(h, h1)
	}
	return h
}

func GetRandomTx() types.Tx {
	n, _ := rand.Int(rand.Reader, big.NewInt(100))
	size := uint64(n.Int64()) + 100
	return types.Tx(GetRandomBytes(size))
}

func GetRandomBytes(n uint64) []byte {
	data := make([]byte, n)
	_, _ = rand.Read(data)
	return data
}

// generateBlock generates random blocks.
func generateBlock(height uint64, lastHeaderHash [32]byte, proposerHash []byte, chainId string, proposerAddress []byte) *types.Block {
	h := createRandomHashes()

	block := &types.Block{
		Header: types.Header{
			Version: types.Version{
				Block: BlockVersion,
				App:   AppVersion,
			},
			Height:             height,
			Time:               4567,
			LastHeaderHash:     lastHeaderHash,
			LastCommitHash:     h[0],
			DataHash:           h[1],
			ConsensusHash:      h[2],
			AppHash:            [32]byte{},
			LastResultsHash:    GetEmptyLastResultsHash(),
			ProposerAddress:    proposerAddress,
			SequencerHash:      [32]byte(proposerHash),
			NextSequencersHash: [32]byte(proposerHash),
			ChainID:            chainId,
		},
		Data: types.Data{
			Txs:                    nil,
			IntermediateStateRoots: types.IntermediateStateRoots{RawRootsList: [][]byte{{0x1}}},
			Evidence:               types.EvidenceData{Evidence: nil},
		},
		LastCommit: types.Commit{
			Height:     8,
			HeaderHash: h[3],
			Signatures: []types.Signature{},
		},
	}

	return block
}

func GenerateBlocksWithTxs(startHeight uint64, num uint64, proposerKey crypto.PrivKey, nTxs int, chainId string) ([]*types.Block, error) {
	r, _ := proposerKey.Raw()
	seq := types.NewSequencerFromValidator(*tmtypes.NewValidator(ed25519.PrivKey(r).PubKey(), 1))
	proposerHash := seq.Hash()

	blocks := make([]*types.Block, num)
	for i := uint64(0); i < num; i++ {
		lastHeaderHash := [32]byte{}
		if i > 0 {
			lastHeaderHash = blocks[i-1].Header.Hash()
		}
		block := generateBlock(i+startHeight, lastHeaderHash, proposerHash, chainId, seq.PubKey().Address())

		block.Data = types.Data{
			Txs: make(types.Txs, nTxs),
			IntermediateStateRoots: types.IntermediateStateRoots{
				RawRootsList: make([][]byte, nTxs),
			},
		}

		for i := 0; i < nTxs; i++ {
			block.Data.Txs[i] = GetRandomTx()
			block.Data.IntermediateStateRoots.RawRootsList[i] = GetRandomBytes(32)
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

// GenerateBlocks generates random blocks.
func GenerateBlocks(startHeight uint64, num uint64, proposerKey crypto.PrivKey, chainId string, lastHeaderHash [32]byte) ([]*types.Block, error) {
	r, _ := proposerKey.Raw()
	seq := types.NewSequencerFromValidator(*tmtypes.NewValidator(ed25519.PrivKey(r).PubKey(), 1))
	proposerHash := seq.Hash()

	blocks := make([]*types.Block, num)
	for i := uint64(0); i < num; i++ {
		if i > 0 {
			lastHeaderHash = blocks[i-1].Header.Hash()
		}
		block := generateBlock(i+startHeight, lastHeaderHash, proposerHash, chainId, seq.PubKey().Address())
		copy(block.Header.DataHash[:], types.GetDataHash(block))
		if i > 0 {
			copy(block.Header.LastCommitHash[:], types.GetLastCommitHash(&blocks[i-1].LastCommit, &block.Header))
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

	num := uint64(len(blocks))
	for i := uint64(0); i < num; i++ {
		block := blocks[i]
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
	abciHeaderPb := types.ToABCIHeaderPB(header)
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
func GenerateBatch(startHeight uint64, endHeight uint64, proposerKey crypto.PrivKey, chainId string, lastHeaderHash [32]byte) (*types.Batch, error) {
	blocks, err := GenerateBlocks(startHeight, endHeight-startHeight+1, proposerKey, chainId, lastHeaderHash)
	if err != nil {
		return nil, err
	}
	commits, err := GenerateCommits(blocks, proposerKey)
	if err != nil {
		return nil, err
	}
	batch := &types.Batch{
		Blocks:  blocks,
		Commits: commits,
	}
	return batch, nil
}

// GenerateLastBatch generates a final batch with LastBatch flag set to true and different NextSequencerHash
func GenerateLastBatch(startHeight uint64, endHeight uint64, proposerKey crypto.PrivKey, nextSequencerKey crypto.PrivKey, chainId string, lastHeaderHash [32]byte) (*types.Batch, error) {
	// Primero obtenemos el hash del siguiente sequencer
	nextSequencerRaw, _ := nextSequencerKey.Raw()
	nextSeq := types.NewSequencerFromValidator(*tmtypes.NewValidator(ed25519.PrivKey(nextSequencerRaw).PubKey(), 1))
	nextSequencerHash := nextSeq.Hash()

	// Generamos los bloques con el nuevo NextSequencerHash
	blocks, err := GenerateLastBlocks(startHeight, endHeight-startHeight+1, proposerKey, chainId, lastHeaderHash, [32]byte(nextSequencerHash))
	if err != nil {
		return nil, err
	}

	// Generamos los commits
	commits, err := GenerateCommits(blocks, proposerKey)
	if err != nil {
		return nil, err
	}

	batch := &types.Batch{
		Blocks:    blocks,
		Commits:   commits,
		LastBatch: true,
	}

	return batch, nil
}

// GenerateLastBlocks es similar a GenerateBlocks pero incluye el NextSequencerHash
func GenerateLastBlocks(startHeight uint64, num uint64, proposerKey crypto.PrivKey, chainId string, lastHeaderHash [32]byte, nextSequencerHash [32]byte) ([]*types.Block, error) {
	r, _ := proposerKey.Raw()
	seq := types.NewSequencerFromValidator(*tmtypes.NewValidator(ed25519.PrivKey(r).PubKey(), 1))
	proposerHash := seq.Hash()
	blocks := make([]*types.Block, num)

	for i := uint64(0); i < num; i++ {
		if i > 0 {
			lastHeaderHash = blocks[i-1].Header.Hash()
		}
		block := generateBlock(i+startHeight, lastHeaderHash, proposerHash, chainId, seq.PubKey().Address())

		if i == num-1 {
			copy(block.Header.NextSequencersHash[:], nextSequencerHash[:])
		}

		copy(block.Header.DataHash[:], types.GetDataHash(block))
		if i > 0 {
			copy(block.Header.LastCommitHash[:], types.GetLastCommitHash(&blocks[i-1].LastCommit, &block.Header))
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

func MustGenerateBatch(startHeight uint64, endHeight uint64, proposerKey crypto.PrivKey, chainId string, lastHeaderHash [32]byte) *types.Batch {
	blocks, err := GenerateBlocks(startHeight, endHeight-startHeight+1, proposerKey, chainId, lastHeaderHash)
	if err != nil {
		panic(err)
	}
	commits, err := GenerateCommits(blocks, proposerKey)
	if err != nil {
		panic(err)
	}
	return &types.Batch{
		Blocks:  blocks,
		Commits: commits,
	}
}

func MustGenerateBatchAndKey(startHeight uint64, endHeight uint64, chainId string, lastHeaderHash [32]byte) *types.Batch {
	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		panic(err)
	}
	return MustGenerateBatch(startHeight, endHeight, proposerKey, chainId, lastHeaderHash)
}

// GenerateRandomValidatorSet generates random validator sets
func GenerateRandomValidatorSet() *tmtypes.ValidatorSet {
	return tmtypes.NewValidatorSet([]*tmtypes.Validator{
		tmtypes.NewValidator(ed25519.GenPrivKey().PubKey(), 1),
	})
}

// GenerateStateWithSequencer generates an initial state for testing.
func GenerateStateWithSequencer(initialHeight int64, lastBlockHeight int64, pubkey tmcrypto.PubKey) *types.State {
	s := &types.State{
		ChainID:         "test-chain",
		InitialHeight:   uint64(initialHeight),
		BaseHeight:      uint64(initialHeight),
		AppHash:         [32]byte{},
		LastResultsHash: GetEmptyLastResultsHash(),
		Version: tmstate.Version{
			Consensus: version.Consensus{
				Block: BlockVersion,
				App:   AppVersion,
			},
		},
		RollappParams: dymint.RollappParams{
			Da:      "mock",
			Version: dymintversion.Commit,
		},
		ConsensusParams: tmproto.ConsensusParams{
			Block: tmproto.BlockParams{
				MaxBytes: 100,
				MaxGas:   100,
			},
		},
	}
	s.Sequencers.SetProposer(types.NewSequencer(pubkey, ""))
	s.SetHeight(uint64(lastBlockHeight))
	return s
}

// GenerateGenesis generates a genesis for testing.
func GenerateGenesis(chainId string, initialHeight int64) *tmtypes.GenesisDoc {
	return &tmtypes.GenesisDoc{
		ChainID:       chainId,
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
		AppState: []byte("{\"rollappparams\": {\"params\": {\"da\": \"mock\",\"version\": \"" + dymintversion.Commit + "\"}}}"),
	}
}

func GetEmptyLastResultsHash() [32]byte {
	lastResults := []*abci.ResponseDeliverTx{}
	return *(*[32]byte)(tmtypes.NewResults(lastResults).Hash())
}

func GetRandomBlock(height uint64, nTxs int) *types.Block {
	block := &types.Block{
		Header: types.Header{
			Height: height,
		},
		Data: types.Data{
			Txs: make(types.Txs, nTxs),
			IntermediateStateRoots: types.IntermediateStateRoots{
				RawRootsList: make([][]byte, nTxs),
			},
		},
	}

	for i := 0; i < nTxs; i++ {
		block.Data.Txs[i] = GetRandomTx()
		block.Data.IntermediateStateRoots.RawRootsList[i] = GetRandomBytes(32)
	}

	return block
}
