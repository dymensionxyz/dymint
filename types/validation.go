package types

import (
	"errors"

	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmtypes "github.com/tendermint/tendermint/types"
)

// ValidateBasic performs basic validation of a block.
func (b *Block) ValidateBasic() error {
	err := b.Header.ValidateBasic()
	if err != nil {
		return err
	}

	abciData := tmtypes.Data{
		Txs: ToABCIBlockDataTxs(&b.Data),
	}
	if b.Header.DataHash != [32]byte(abciData.Hash()) {
		return ErrInvalidHeaderDataHash
	}

	err = b.LastCommit.ValidateBasic()
	if err != nil {
		return err
	}

	return nil
}

// ValidateBasic performs basic validation of a header.
func (h *Header) ValidateBasic() error {
	if len(h.ProposerAddress) == 0 {
		return errors.New("no proposer address")
	}

	return nil
}

// ValidateBasic performs basic validation of a commit.
func (c *Commit) ValidateBasic() error {
	if c.Height > 0 {
		if len(c.Signatures) != 1 {
			return errors.New("there should be 1 signature")
		}
		if len(c.Signatures[0]) > tmtypes.MaxSignatureSize {
			return errors.New("signature is too big")
		}
	}
	return nil
}

// Validate performs full validation of a commit.
func (c *Commit) Validate(proposerPubKey tmcrypto.PubKey, abciHeaderBytes []byte) error {
	if err := c.ValidateBasic(); err != nil {
		return err
	}
	if !proposerPubKey.VerifySignature(abciHeaderBytes, c.Signatures[0]) {
		return ErrInvalidSignature
	}
	return nil
}
