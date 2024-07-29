package types

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	tmcrypto "github.com/tendermint/tendermint/crypto"
)

type BlockSource uint64

const (
	_ BlockSource = iota
	ProducedBlock
	GossipedBlock
	DABlock
	LocalDbBlock
)

func (s BlockSource) String() string {
	return AllSources[s]
}

var AllSources = []string{"none", "produced", "gossip", "da", "local_db"}

type BlockMetaData struct {
	Source   BlockSource
	DAHeight uint64
}

type CachedBlock struct {
	Block  *Block
	Commit *Commit
}

func GetAddress(key crypto.PrivKey) ([]byte, error) {
	rawKey, err := key.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	return tmcrypto.AddressHash(rawKey), nil
}
