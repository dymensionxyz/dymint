package block

import (
	"github.com/dymensionxyz/dymint/types"

	"github.com/libp2p/go-libp2p/core/crypto"
	tmcrypto "github.com/tendermint/tendermint/crypto"
)

// TODO: move to types package

type BlockSource uint64

const (
	producedBlock BlockSource = iota
	gossipedBlock
	daBlock
	localDbBlock
)

func (s BlockSource) String() string {
	return []string{"produced", "gossip", "da", "local_db"}[s]
}

type blockMetaData struct {
	source   BlockSource
	daHeight uint64
}

type CachedBlock struct {
	Block  *types.Block
	Commit *types.Commit
}

func getAddress(key crypto.PrivKey) ([]byte, error) {
	rawKey, err := key.GetPublic().Raw()
	if err != nil {
		return nil, err
	}
	return tmcrypto.AddressHash(rawKey), nil
}
