package types

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmtypes "github.com/tendermint/tendermint/types"
)

func ValidateProposedTransition(state *State, block *Block, commit *Commit, proposerPubKey tmcrypto.PubKey) error {
	if err := block.ValidateWithState(state); err != nil {
		return fmt.Errorf("block: %w", err)
	}

	if err := commit.ValidateWithHeader(proposerPubKey, &block.Header); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ValidateBasic performs basic validation of a block.
func (b *Block) ValidateBasic() error {
	err := b.Header.ValidateBasic()
	if err != nil {
		return fmt.Errorf("header: %w", err)
	}

	err = b.Data.ValidateBasic()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}

	err = b.LastCommit.ValidateBasic()
	if err != nil {
		return fmt.Errorf("last commit: %w", err)
	}

	if b.Header.DataHash != [32]byte(GetDataHash(b)) {
		return ErrInvalidHeaderDataHash
	}

	if err := b.validateDymHeader(); err != nil {
		return fmt.Errorf("dym header: %w", err)
	}

	return nil
}

func (b *Block) validateDymHeader() error {
	exp := b.Header.DymHash()
	got := MakeDymHeader(b.Data.ConsensusMessages).Hash()
	if !bytes.Equal(exp, got) {
		return ErrInvalidDymHeaderHash
	}
	return nil
}

func (b *Block) ValidateWithState(state *State) error {
	err := b.ValidateBasic()
	if err != nil {
		if errors.Is(err, ErrInvalidHeaderDataHash) {
			return NewErrInvalidHeaderDataHashFraud(b)
		}
		if errors.Is(err, ErrInvalidDymHeader) {
			return NewErrInvalidDymHeaderFraud(b, err)
		}

		return err
	}

	if b.Header.ChainID != state.ChainID {
		return NewErrInvalidChainID(state.ChainID, b)
	}

	if b.Header.LastHeaderHash != state.LastHeaderHash {
		return NewErrLastHeaderHashMismatch(state.LastHeaderHash, b)
	}

	currentTime := time.Now().UTC()
	if currentTime.Add(TimeFraudMaxDrift).Before(b.Header.GetTimestamp()) {
		return NewErrTimeFraud(b, currentTime)
	}

	if err = validateRevision(b.Header, state); err != nil {
		return ErrVersionMismatch
	}

	nextHeight := state.NextHeight()
	if b.Header.Height != nextHeight {
		return NewErrFraudHeightMismatch(state.NextHeight(), &b.Header)
	}

	proposerHash := state.GetProposerHash()
	if !bytes.Equal(b.Header.SequencerHash[:], proposerHash) {
		return NewErrInvalidSequencerHashFraud([32]byte(proposerHash), b.Header.SequencerHash[:], &b.Header)
	}

	if !bytes.Equal(b.Header.AppHash[:], state.AppHash[:]) {
		return NewErrFraudAppHashMismatch(state.AppHash, &b.Header)
	}

	if !bytes.Equal(b.Header.LastResultsHash[:], state.LastResultsHash[:]) {
		return NewErrLastResultsHashMismatch(state.LastResultsHash, &b.Header)
	}

	return nil
}

// ValidateBasic performs basic validation of a header.
func (h *Header) ValidateBasic() error {
	if len(h.ProposerAddress) == 0 {
		return ErrEmptyProposerAddress
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

func (c *Commit) ValidateWithHeader(proposerPubKey tmcrypto.PubKey, header *Header) error {
	if err := c.ValidateBasic(); err != nil {
		return NewErrInvalidSignatureFraud(err, header, c)
	}

	abciHeaderPb := ToABCIHeaderPB(header)
	abciHeaderBytes, err := abciHeaderPb.Marshal()
	if err != nil {
		return err
	}

	// commit is validated to have single signature
	if !proposerPubKey.VerifySignature(abciHeaderBytes, c.Signatures[0]) {
		return NewErrInvalidSignatureFraud(ErrInvalidSignature, header, c)
	}

	if c.Height != header.Height {
		return NewErrInvalidCommitBlockHeightFraud(c.Height, header)
	}

	if !bytes.Equal(header.ProposerAddress, proposerPubKey.Address()) {
		return NewErrInvalidProposerAddressFraud(header.ProposerAddress, proposerPubKey.Address(), header)
	}

	seq := NewSequencerFromValidator(*tmtypes.NewValidator(proposerPubKey, 1))
	proposerHash, err := seq.Hash()
	if err != nil {
		return err
	}

	if !bytes.Equal(header.SequencerHash[:], proposerHash) {
		return NewErrInvalidSequencerHashFraud(header.SequencerHash, proposerHash[:], header)
	}

	if c.HeaderHash != header.Hash() {
		return NewErrInvalidHeaderHashFraud(c.HeaderHash, header)
	}

	return nil
}

func validateRevision(header Header, state *State) error {
	revision := state.GetRevisionByHeight(header.Height)
	if header.Version.App != revision.Revision.Consensus.App || header.Version.Block != revision.Revision.Consensus.Block {
		return ErrVersionMismatch
	}
	return nil
}
