package types

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	tmcrypto "github.com/tendermint/tendermint/crypto"
)

type BlockSource uint64

const (
	_ BlockSource = iota
	Produced
	Gossiped
	BlockSync
	DA
)

func (s BlockSource) String() string {
	return AllSources[s]
}

var AllSources = []string{"none", "produced", "gossip", "blocksync", "da", "local_db"}

type BlockMetaData struct {
	Source       BlockSource
	DAHeight     uint64
	SequencerSet Sequencers // The set of Rollapp sequencers that were present in the Hub while producing this block
}

type CachedBlock struct {
	Block  *Block
	Commit *Commit
	Source BlockSource
}

func GetAddress(key crypto.PrivKey) ([]byte, error) {
	rawKey, err := key.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	return tmcrypto.AddressHash(rawKey), nil
}
