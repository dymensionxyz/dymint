package block

import (
	"encoding/base64"

	"github.com/libp2p/go-libp2p/core/crypto"
)

const (
	// fakes odysphere addr
	// ody seq: dym10dhjnv8r6l3a46j0zu8fhvfpy4vtkj60y5vt7l
	// hub q sequencer show-sequencer dym10dhjnv8r6l3a46j0zu8fhvfpy4vtkj60y5vt7l
	odySeqPub = "BctwsllAr1SaZf8ULbAOgT08hMDhjh8GeYUWRII3Guo="
)

type privKey struct {
	*crypto.Ed25519PrivateKey
	pubKeyOverride []byte
}

type pubKey struct {
	*crypto.Ed25519PublicKey
	val []byte
}

func (p *pubKey) Raw() ([]byte, error) {
	return p.val, nil
}

func (p *privKey) GetPublic() crypto.PubKey {
	return &pubKey{
		val: p.pubKeyOverride,
	}
}

func overrideKey(localKey crypto.PrivKey) crypto.PrivKey {
	key, err := base64.StdEncoding.DecodeString(odySeqPub)
	if err != nil {
		panic(err)
	}
	return &privKey{
		Ed25519PrivateKey: localKey.(*crypto.Ed25519PrivateKey),
		pubKeyOverride:    key,
	}
}
