package block

import (
	"github.com/dymensionxyz/dymint/types"

	"github.com/libp2p/go-libp2p/core/crypto"
	tmcrypto "github.com/tendermint/tendermint/crypto"
)

// TODO: move to types package

type blockSource string

const (
	producedBlock blockSource = "produced"
	gossipedBlock blockSource = "gossip"
	daBlock       blockSource = "da"
	localDbBlock  blockSource = "local_db"
)

type blockMetaData struct {
	source   blockSource
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
