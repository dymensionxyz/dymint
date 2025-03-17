package testutil

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	dymintversion "github.com/dymensionxyz/dymint/version"
	"github.com/libp2p/go-libp2p/core/crypto"
	abci "github.com/tendermint/tendermint/abci/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	version "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dymint"
)

const (
	// BlockVersion is the default block version for testing
	BlockVersion = 1
	// AppVersion is the default app version for testing
	AppVersion = 0

	SettlementAccountPrefix = "dym"
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
	size := n.Uint64() + 100
	return types.Tx(GetRandomBytes(size))
}

func GetRandomBytes(n uint64) []byte {
	data := make([]byte, n)
	_, _ = rand.Read(data)
	return data
}

func GenerateSettlementAddress() string {
	addrBytes := ed25519.GenPrivKey().PubKey().Address().Bytes()
	addr, err := bech32.ConvertAndEncode(SettlementAccountPrefix, addrBytes)
	if err != nil {
		panic(err)
	}
	return addr
}

// generateBlock generates random blocks.
func generateBlock(height uint64, proposerHash []byte, lastHeaderHash [32]byte) *types.Block {
	h := createRandomHashes()

	block := &types.Block{
		Header: types.Header{
			Version: types.Version{
				Block: BlockVersion,
				App:   AppVersion,
			},
			Height:                height,
			Time:                  4567,
			LastHeaderHash:        lastHeaderHash,
			LastCommitHash:        h[0],
			DataHash:              h[1],
			ConsensusHash:         h[2],
			AppHash:               [32]byte{},
			LastResultsHash:       GetEmptyLastResultsHash(),
			ProposerAddress:       []byte{4, 3, 2, 1},
			SequencerHash:         [32]byte(proposerHash),
			NextSequencersHash:    [32]byte(proposerHash),
			ChainID:               "test-chain",
			ConsensusMessagesHash: types.ConsMessagesHash(nil),
		},
		Data: types.Data{
			Txs: nil,
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
	proposerHash := seq.MustHash()

	blocks := make([]*types.Block, num)
	lastHeaderHash := [32]byte{}
	for i := uint64(0); i < num; i++ {

		block := generateBlock(i+startHeight, proposerHash, lastHeaderHash)

		block.Data = types.Data{
			Txs: make(types.Txs, nTxs),
		}

		for i := 0; i < nTxs; i++ {
			block.Data.Txs[i] = GetRandomTx()
		}

		signature, err := generateSignature(proposerKey, &block.Header)
		if err != nil {
			return nil, err
		}
		block.LastCommit.Signatures = []types.Signature{signature}
		blocks[i] = block
		lastHeaderHash = block.Header.Hash()
	}
	return blocks, nil
}

// GenerateBlocks generates random blocks.
func GenerateBlocks(startHeight uint64, num uint64, proposerKey crypto.PrivKey, lastBlockHeader [32]byte) ([]*types.Block, error) {
	r, _ := proposerKey.Raw()
	seq := types.NewSequencerFromValidator(*tmtypes.NewValidator(ed25519.PrivKey(r).PubKey(), 1))
	proposerHash := seq.MustHash()

	blocks := make([]*types.Block, num)
	lastHeaderHash := lastBlockHeader
	for i := uint64(0); i < num; i++ {
		block := generateBlock(i+startHeight, proposerHash, lastHeaderHash)
		copy(block.Header.DataHash[:], types.GetDataHash(block))
		if i > 0 {
			copy(block.Header.LastCommitHash[:], types.GetLastCommitHash(&blocks[i-1].LastCommit))
		}

		signature, err := generateSignature(proposerKey, &block.Header)
		if err != nil {
			return nil, err
		}
		block.LastCommit.Signatures = []types.Signature{signature}
		block.Header.ProposerAddress = ed25519.PrivKey(r).PubKey().Address()
		block.Header.Time = time.Now().UTC().UnixNano()
		blocks[i] = block
		lastHeaderHash = block.Header.Hash()
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

func GenerateDRS(blocks int) []uint32 {
	drsVersion, _ := dymintversion.GetDRSVersion()
	drs := make([]uint32, blocks)
	for i := 0; i < blocks; i++ {
		drs[i] = drsVersion
	}
	return drs
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
func GenerateBatch(startHeight uint64, endHeight uint64, proposerKey crypto.PrivKey, lastBlockHeader [32]byte) (*types.Batch, error) {
	blocks, err := GenerateBlocks(startHeight, endHeight-startHeight+1, proposerKey, lastBlockHeader)
	if err != nil {
		return nil, err
	}
	commits, err := GenerateCommits(blocks, proposerKey)
	if err != nil {
		return nil, err
	}
	batch := &types.Batch{
		Blocks:     blocks,
		Commits:    commits,
		DRSVersion: GenerateDRS(len(blocks)),
	}
	return batch, nil
}

// GenerateLastBatch generates a final batch with LastBatch flag set to true and different NextSequencerHash
func GenerateLastBatch(startHeight uint64, endHeight uint64, proposerKey crypto.PrivKey, nextSequencerKey crypto.PrivKey, lastHeaderHash [32]byte) (*types.Batch, error) {
	nextSequencerRaw, _ := nextSequencerKey.Raw()
	nextSeq := types.NewSequencerFromValidator(*tmtypes.NewValidator(ed25519.PrivKey(nextSequencerRaw).PubKey(), 1))
	nextSequencerHash := nextSeq.MustHash()

	blocks, err := GenerateLastBlocks(startHeight, endHeight-startHeight+1, proposerKey, lastHeaderHash, [32]byte(nextSequencerHash))
	if err != nil {
		return nil, err
	}

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
func GenerateLastBlocks(startHeight uint64, num uint64, proposerKey crypto.PrivKey, lastHeaderHash [32]byte, nextSequencerHash [32]byte) ([]*types.Block, error) {
	r, _ := proposerKey.Raw()
	seq := types.NewSequencerFromValidator(*tmtypes.NewValidator(ed25519.PrivKey(r).PubKey(), 1))
	proposerHash := seq.MustHash()
	blocks := make([]*types.Block, num)

	for i := uint64(0); i < num; i++ {
		if i > 0 {
			lastHeaderHash = blocks[i-1].Header.Hash()
		}
		block := generateBlock(i+startHeight, proposerHash, lastHeaderHash)

		if i == num-1 {
			copy(block.Header.NextSequencersHash[:], nextSequencerHash[:])
		}

		copy(block.Header.DataHash[:], types.GetDataHash(block))
		if i > 0 {
			copy(block.Header.LastCommitHash[:], types.GetLastCommitHash(&blocks[i-1].LastCommit))
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

func MustGenerateBatch(startHeight uint64, endHeight uint64, proposerKey crypto.PrivKey) *types.Batch {
	blocks, err := GenerateBlocks(startHeight, endHeight-startHeight+1, proposerKey, [32]byte{})
	if err != nil {
		panic(err)
	}
	commits, err := GenerateCommits(blocks, proposerKey)
	if err != nil {
		panic(err)
	}
	return &types.Batch{
		Blocks:     blocks,
		Commits:    commits,
		DRSVersion: GenerateDRS(len(blocks)),
	}
}

func MustGenerateBatchAndKey(startHeight uint64, endHeight uint64) *types.Batch {
	proposerKey, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		panic(err)
	}
	return MustGenerateBatch(startHeight, endHeight, proposerKey)
}

// GenerateRandomValidatorSet generates random validator sets
func GenerateRandomValidatorSet() *tmtypes.ValidatorSet {
	return tmtypes.NewValidatorSet([]*tmtypes.Validator{
		tmtypes.NewValidator(ed25519.GenPrivKey().PubKey(), 1),
	})
}

func GenerateSequencer() types.Sequencer {
	return *types.NewSequencer(
		tmtypes.NewValidator(ed25519.GenPrivKey().PubKey(), 1).PubKey,
		GenerateSettlementAddress(),
		GenerateSettlementAddress(),
		[]string{GenerateSettlementAddress(), GenerateSettlementAddress()},
	)
}

// GenerateStateWithSequencer generates an initial state for testing.
func GenerateStateWithSequencer(initialHeight int64, lastBlockHeight int64, pubkey tmcrypto.PubKey) *types.State {
	s := &types.State{
		ChainID:         "test-chain",
		InitialHeight:   uint64(initialHeight), //nolint:gosec // height is non-negative and falls in int64
		AppHash:         [32]byte{},
		LastResultsHash: GetEmptyLastResultsHash(),
		Version: tmstate.Version{
			Consensus: version.Consensus{
				Block: BlockVersion,
				App:   AppVersion,
			},
		},
		RollappParams: dymint.RollappParams{
			Da:         "mock",
			DrsVersion: 0,
		},
		ConsensusParams: tmproto.ConsensusParams{
			Block: tmproto.BlockParams{
				MaxBytes: 100,
				MaxGas:   100,
			},
		},
	}
	s.SetProposer(types.NewSequencer(
		pubkey,
		GenerateSettlementAddress(),
		GenerateSettlementAddress(),
		[]string{GenerateSettlementAddress()},
	))
	s.SetHeight(uint64(lastBlockHeight)) //nolint:gosec // height is non-negative and falls in int64
	return s
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
		AppState: []byte("{\"rollappparams\": {\"params\": {\"da\": \"mock\",\"version\": 0}}}"),
	}
}

func GetEmptyLastResultsHash() [32]byte {
	lastResults := []*abci.ResponseDeliverTx{}
	return *(*[32]byte)(tmtypes.NewResults(lastResults).Hash())
}

func GetRandomBlock(height uint64, nTxs int) *types.Block {
	block := &types.Block{
		Header: types.Header{
			Height:                height,
			ConsensusMessagesHash: types.ConsMessagesHash(nil),
		},
		Data: types.Data{
			Txs: make(types.Txs, nTxs),
		},
	}

	for i := 0; i < nTxs; i++ {
		block.Data.Txs[i] = GetRandomTx()
	}

	return block
}

func GenerateBatchWithBlocks(numBlocks uint64, proposerKey crypto.PrivKey) *types.Batch {
	if numBlocks == 0 {
		return &types.Batch{
			Blocks:     []*types.Block{},
			Commits:    []*types.Commit{},
			DRSVersion: []uint32{},
		}
	}
	return MustGenerateBatch(0, numBlocks, proposerKey)
}
