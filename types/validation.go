package types

import (
	"bytes"
	"errors"

	tmtypes "github.com/tendermint/tendermint/types"
)

// ValidateBasic performs basic validation of a block.
func (b *Block) ValidateBasic() error {
	err := b.Header.ValidateBasic()
	if err != nil {
		return err
	}

	err = b.Data.ValidateBasic()
	if err != nil {
		return err
	}

	err = b.LastCommit.ValidateBasic()
	if err != nil {
		return err
	}

	return nil
}

func (b *Block) ValidateWithState(state State) error {
	err := b.ValidateBasic()
	if err != nil {
		return err
	}
	if b.Header.Version.App != state.Version.Consensus.App ||
		b.Header.Version.Block != state.Version.Consensus.Block {
		return errors.New("b version mismatch")
	}
	if state.LastBlockHeight <= 0 && b.Header.Height != uint64(state.InitialHeight) {
		return errors.New("initial b height mismatch")
	}
	if state.LastBlockHeight > 0 && b.Header.Height != uint64(state.LastStoreHeight)+1 {
		return errors.New("b height mismatch")
	}
	if !bytes.Equal(b.Header.AppHash[:], state.AppHash[:]) {
		return errors.New("AppHash mismatch")
	}
	if !bytes.Equal(b.Header.LastResultsHash[:], state.LastResultsHash[:]) {
		return errors.New("LastResultsHash mismatch")
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

// ValidateBasic performs basic validation of block data.
// Actually it's a placeholder, because nothing is checked.
func (d *Data) ValidateBasic() error {
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
func (c *Commit) Validate(proposer *Sequencer, msg []byte) error {
	if err := c.ValidateBasic(); err != nil {
		return err
	}
	if !proposer.PublicKey.VerifySignature(msg, c.Signatures[0]) {
		return ErrInvalidSignature
	}
	return nil
}
