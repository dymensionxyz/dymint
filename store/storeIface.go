package store

import (
	"github.com/ipfs/go-cid"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"

	"github.com/dymensionxyz/dymint/types"
)




type KV interface {
	Get(key []byte) ([]byte, error)          
	Set(key []byte, value []byte) error      
	Delete(key []byte) error                 
	NewBatch() KVBatch                       
	PrefixIterator(prefix []byte) KVIterator 
	Close() error                            
}


type KVBatch interface {
	Set(key, value []byte) error 
	Delete(key []byte) error     
	Commit() error               
	Discard()                    
}


type KVIterator interface {
	Valid() bool
	Next()
	Key() []byte
	Value() []byte
	Error() error
	Discard()
}


type Store interface {
	
	NewBatch() KVBatch

	
	SaveBlock(block *types.Block, commit *types.Commit, batch KVBatch) (KVBatch, error)

	
	LoadBlock(height uint64) (*types.Block, error)

	
	LoadBlockByHash(hash [32]byte) (*types.Block, error)

	
	SaveBlockResponses(height uint64, responses *tmstate.ABCIResponses, batch KVBatch) (KVBatch, error)

	
	LoadBlockResponses(height uint64) (*tmstate.ABCIResponses, error)

	
	LoadCommit(height uint64) (*types.Commit, error)

	
	LoadCommitByHash(hash [32]byte) (*types.Commit, error)

	
	
	SaveState(state *types.State, batch KVBatch) (KVBatch, error)

	
	LoadState() (*types.State, error)

	SaveProposer(height uint64, proposer types.Sequencer, batch KVBatch) (KVBatch, error)

	LoadProposer(height uint64) (types.Sequencer, error)

	PruneStore(to uint64, logger types.Logger) (uint64, error)

	SaveBlockCid(height uint64, cid cid.Cid, batch KVBatch) (KVBatch, error)

	LoadBlockCid(height uint64) (cid.Cid, error)

	SaveBlockSource(height uint64, source types.BlockSource, batch KVBatch) (KVBatch, error)

	LoadBlockSource(height uint64) (types.BlockSource, error)

	SaveValidationHeight(height uint64, batch KVBatch) (KVBatch, error)

	LoadValidationHeight() (uint64, error)

	LoadDRSVersion(height uint64) (uint32, error)

	SaveDRSVersion(height uint64, version uint32, batch KVBatch) (KVBatch, error)

	RemoveBlockCid(height uint64) error

	LoadBaseHeight() (uint64, error)

	SaveBaseHeight(height uint64) error

	LoadBlockSyncBaseHeight() (uint64, error)

	SaveBlockSyncBaseHeight(height uint64) error

	LoadIndexerBaseHeight() (uint64, error)

	SaveIndexerBaseHeight(height uint64) error

	SaveLastBlockSequencerSet(sequencers types.Sequencers, batch KVBatch) (KVBatch, error)

	LoadLastBlockSequencerSet() (types.Sequencers, error)

	Close() error
}
