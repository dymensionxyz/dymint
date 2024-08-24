package types

import (
	"errors"
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	abci "github.com/tendermint/tendermint/abci/types"
	prototypes "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/types"

	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
)

// MarshalBinary encodes Block into binary form and returns it.
func (b *Block) MarshalBinary() ([]byte, error) {
	return b.ToProto().Marshal()
}

// MarshalBinary encodes Batch into binary form and returns it.
func (b *Batch) MarshalBinary() ([]byte, error) {
	return b.ToProto().Marshal()
}

// UnmarshalBinary decodes binary form of Block into object.
func (b *Block) UnmarshalBinary(data []byte) error {
	var pBlock pb.Block
	err := pBlock.Unmarshal(data)
	if err != nil {
		return err
	}
	err = b.FromProto(&pBlock)
	return err
}

// UnmarshalBinary decodes binary form of Batch into object.
func (b *Batch) UnmarshalBinary(data []byte) error {
	var pBatch pb.Batch
	err := pBatch.Unmarshal(data)
	if err != nil {
		return err
	}
	err = b.FromProto(&pBatch)
	return err
}

// MarshalBinary encodes Header into binary form and returns it.
func (h *Header) MarshalBinary() ([]byte, error) {
	return h.ToProto().Marshal()
}

// UnmarshalBinary decodes binary form of Header into object.
func (h *Header) UnmarshalBinary(data []byte) error {
	var pHeader pb.Header
	err := pHeader.Unmarshal(data)
	if err != nil {
		return err
	}
	err = h.FromProto(&pHeader)
	return err
}

// MarshalBinary encodes Data into binary form and returns it.
func (d *Data) MarshalBinary() ([]byte, error) {
	return d.ToProto().Marshal()
}

// MarshalBinary encodes Commit into binary form and returns it.
func (c *Commit) MarshalBinary() ([]byte, error) {
	return c.ToProto().Marshal()
}

// UnmarshalBinary decodes binary form of Commit into object.
func (c *Commit) UnmarshalBinary(data []byte) error {
	var pCommit pb.Commit
	err := pCommit.Unmarshal(data)
	if err != nil {
		return err
	}
	err = c.FromProto(&pCommit)
	return err
}

// ToProto converts Header into protobuf representation and returns it.
func (h *Header) ToProto() *pb.Header {
	return &pb.Header{
		Version:           &pb.Version{Block: h.Version.Block, App: h.Version.App},
		NamespaceId:       []byte{},
		Height:            h.Height,
		Time:              h.Time,
		LastHeaderHash:    h.LastHeaderHash[:],
		LastCommitHash:    h.LastCommitHash[:],
		DataHash:          h.DataHash[:],
		ConsensusHash:     h.ConsensusHash[:],
		AppHash:           h.AppHash[:],
		LastResultsHash:   h.LastResultsHash[:],
		ProposerAddress:   h.ProposerAddress[:],
		SequencerHash:     h.SequencerHash[:],
		NextSequencerHash: h.NextSequencersHash[:],
		ChainId:           h.ChainID,
	}
}

// FromProto fills Header with data from its protobuf representation.
func (h *Header) FromProto(other *pb.Header) error {
	h.Version.Block = other.Version.Block
	h.Version.App = other.Version.App
	h.ChainID = other.ChainId
	h.Height = other.Height
	h.Time = other.Time
	if !safeCopy(h.LastHeaderHash[:], other.LastHeaderHash) {
		return errors.New("invalid length of 'LastHeaderHash'")
	}
	if !safeCopy(h.LastCommitHash[:], other.LastCommitHash) {
		return errors.New("invalid length of 'LastCommitHash'")
	}
	if !safeCopy(h.DataHash[:], other.DataHash) {
		return errors.New("invalid length of 'DataHash'")
	}
	if !safeCopy(h.ConsensusHash[:], other.ConsensusHash) {
		return errors.New("invalid length of 'ConsensusHash'")
	}
	if !safeCopy(h.AppHash[:], other.AppHash) {
		return errors.New("invalid length of 'AppHash'")
	}
	if !safeCopy(h.LastResultsHash[:], other.LastResultsHash) {
		return errors.New("invalid length of 'LastResultsHash'")
	}
	if !safeCopy(h.SequencerHash[:], other.SequencerHash) {
		return errors.New("invalid length of 'SequencerHash'")
	}
	if !safeCopy(h.NextSequencersHash[:], other.NextSequencerHash) {
		return errors.New("invalid length of 'SequencersHash'")
	}
	if len(other.ProposerAddress) > 0 {
		h.ProposerAddress = make([]byte, len(other.ProposerAddress))
		copy(h.ProposerAddress, other.ProposerAddress)
	}

	return nil
}

// safeCopy copies bytes from src slice into dst slice if both have same size.
// It returns true if sizes of src and dst are the same.
func safeCopy(dst, src []byte) bool {
	if len(src) != len(dst) {
		return false
	}
	_ = copy(dst, src)
	return true
}

// ToProto converts Block into protobuf representation and returns it.
func (b *Block) ToProto() *pb.Block {
	return &pb.Block{
		Header:     b.Header.ToProto(),
		Data:       b.Data.ToProto(),
		LastCommit: b.LastCommit.ToProto(),
	}
}

// ToProto converts Batch into protobuf representation and returns it.
func (b *Batch) ToProto() *pb.Batch {
	return &pb.Batch{
		StartHeight: b.StartHeight(),
		EndHeight:   b.EndHeight(),
		Blocks:      blocksToProto(b.Blocks),
		Commits:     commitsToProto(b.Commits),
	}
}

// ToProto converts Data into protobuf representation and returns it.
func (d *Data) ToProto() *pb.Data {
	return &pb.Data{
		Txs:                    txsToByteSlices(d.Txs),
		IntermediateStateRoots: d.IntermediateStateRoots.RawRootsList,
		Evidence:               evidenceToProto(d.Evidence),
	}
}

// FromProto fills Block with data from its protobuf representation.
func (b *Block) FromProto(other *pb.Block) error {
	err := b.Header.FromProto(other.Header)
	if err != nil {
		return err
	}
	b.Data.Txs = byteSlicesToTxs(other.Data.Txs)
	b.Data.IntermediateStateRoots.RawRootsList = other.Data.IntermediateStateRoots
	b.Data.Evidence = evidenceFromProto(other.Data.Evidence)
	if other.LastCommit != nil {
		err := b.LastCommit.FromProto(other.LastCommit)
		if err != nil {
			return err
		}
	}

	return nil
}

// FromProto fills Batch with data from its protobuf representation.
func (b *Batch) FromProto(other *pb.Batch) error {
	n := len(other.Blocks)
	start := other.StartHeight
	end := other.EndHeight
	if 0 < n && start != other.Blocks[0].Header.GetHeight() {
		return fmt.Errorf("start height does not match first block height: %w", gerrc.ErrInvalidArgument)
	}
	if 0 < n && end != other.Blocks[n-1].Header.GetHeight() {
		return fmt.Errorf("end height does not match last block height: %w", gerrc.ErrInvalidArgument)
	}
	b.Blocks = protoToBlocks(other.Blocks)
	b.Commits = protoToCommits(other.Commits)
	return nil
}

// ToProto converts Commit into protobuf representation and returns it.
func (c *Commit) ToProto() *pb.Commit {
	return &pb.Commit{
		Height:     c.Height,
		HeaderHash: c.HeaderHash[:],
		Signatures: signaturesToByteSlices(c.Signatures),
		TmSignature: &prototypes.CommitSig{
			BlockIdFlag:      prototypes.BlockIDFlag(c.TMSignature.BlockIDFlag),
			ValidatorAddress: c.TMSignature.ValidatorAddress,
			Timestamp:        c.TMSignature.Timestamp,
			Signature:        c.TMSignature.Signature,
		},
	}
}

// FromProto fills Commit with data from its protobuf representation.
func (c *Commit) FromProto(other *pb.Commit) error {
	c.Height = other.Height
	if !safeCopy(c.HeaderHash[:], other.HeaderHash) {
		return errors.New("invalid length of HeaderHash")
	}
	c.Signatures = byteSlicesToSignatures(other.Signatures)
	// For backwards compatibility with old state files that don't have this field.
	if other.TmSignature != nil {
		c.TMSignature = types.CommitSig{
			BlockIDFlag:      types.BlockIDFlag(other.TmSignature.BlockIdFlag),
			ValidatorAddress: other.TmSignature.ValidatorAddress,
			Timestamp:        other.TmSignature.Timestamp,
			Signature:        other.TmSignature.Signature,
		}
	}

	return nil
}

// ToProto converts State into protobuf representation and returns it.
func (s *State) ToProto() (*pb.State, error) {
	seqsProto, err := s.Sequencers.ToProto()
	if err != nil {
		return nil, err
	}

	return &pb.State{
		Version:                          &s.Version,
		ChainId:                          s.ChainID,
		InitialHeight:                    int64(s.InitialHeight),
		LastBlockHeight:                  int64(s.Height()),
		SequencerSet:                     *seqsProto,
		BaseHeight:                       s.BaseHeight,
		ConsensusParams:                  s.ConsensusParams,
		LastHeightConsensusParamsChanged: s.LastHeightConsensusParamsChanged,
		LastResultsHash:                  s.LastResultsHash[:],
		AppHash:                          s.AppHash[:],
	}, nil
}

// FromProto fills State with data from its protobuf representation.
func (s *State) FromProto(other *pb.State) error {
	var err error
	s.Version = *other.Version
	s.ChainID = other.ChainId
	s.InitialHeight = uint64(other.InitialHeight)
	s.SetHeight(uint64(other.LastBlockHeight))
	s.BaseHeight = other.BaseHeight

	err = s.Sequencers.FromProto(other.SequencerSet)
	if err != nil {
		return err
	}

	s.ConsensusParams = other.ConsensusParams
	s.LastHeightConsensusParamsChanged = other.LastHeightConsensusParamsChanged
	copy(s.LastResultsHash[:], other.LastResultsHash)
	copy(s.AppHash[:], other.AppHash)
	return nil
}

// ToProto converts SequencerSet into protobuf representation and returns it.
func (s *SequencerSet) ToProto() (*pb.SequencerSet, error) {
	protoSet := new(pb.SequencerSet)

	seqsProto := make([]*pb.Sequencer, len(s.Sequencers))
	for i := 0; i < len(s.Sequencers); i++ {
		valp, err := s.Sequencers[i].val.ToProto()
		if err != nil {
			return nil, fmt.Errorf("ToProto: SequencerSet: %w", err)
		}
		seq := new(pb.Sequencer)
		seq.SettlementAddress = s.Sequencers[i].SettlementAddress
		seq.Validator = valp
		seqsProto[i] = seq
	}
	protoSet.Sequencers = seqsProto

	if s.Proposer != nil {
		valp, err := s.Proposer.val.ToProto()
		if err != nil {
			return nil, fmt.Errorf("ToProto: SequencerSet: %w", err)
		}
		seq := new(pb.Sequencer)
		seq.Validator = valp
		seq.SettlementAddress = s.Proposer.SettlementAddress
		protoSet.Proposer = seq
	}

	return protoSet, nil
}

// FromProto fills SequencerSet with data from its protobuf representation.
func (s *SequencerSet) FromProto(protoSet pb.SequencerSet) error {
	seqs := make([]Sequencer, len(protoSet.Sequencers))
	for i, seqProto := range protoSet.Sequencers {
		val, err := types.ValidatorFromProto(seqProto.Validator)
		if err != nil {
			return fmt.Errorf("fromProto: SequencerSet: %w", err)
		}
		seqs[i].val = *val
		seqs[i].SettlementAddress = seqProto.SettlementAddress
	}
	s.Sequencers = seqs

	if protoSet.Proposer != nil {
		valProposer, err := types.ValidatorFromProto(protoSet.Proposer.Validator)
		if err != nil {
			return fmt.Errorf("fromProto: SequencerSet proposer: %w", err)
		}
		proposer := new(Sequencer)
		proposer.val = *valProposer
		proposer.SettlementAddress = protoSet.Proposer.SettlementAddress
		s.Proposer = proposer
	}
	return nil
}

func txsToByteSlices(txs Txs) [][]byte {
	bytes := make([][]byte, len(txs))
	for i := range txs {
		bytes[i] = txs[i]
	}
	return bytes
}

func byteSlicesToTxs(bytes [][]byte) Txs {
	if len(bytes) == 0 {
		return nil
	}
	txs := make(Txs, len(bytes))
	for i := range txs {
		txs[i] = bytes[i]
	}
	return txs
}

func evidenceToProto(evidence EvidenceData) []*abci.Evidence {
	var ret []*abci.Evidence
	for _, e := range evidence.Evidence {
		for _, ae := range e.ABCI() {
			ret = append(ret, &ae) //#nosec
		}
	}
	return ret
}

func evidenceFromProto([]*abci.Evidence) EvidenceData {
	var ret EvidenceData
	// TODO(tzdybal): right now Evidence is just an interface without implementations
	return ret
}

func signaturesToByteSlices(sigs []Signature) [][]byte {
	if sigs == nil {
		return nil
	}
	bytes := make([][]byte, len(sigs))
	for i := range sigs {
		bytes[i] = sigs[i]
	}
	return bytes
}

func byteSlicesToSignatures(bytes [][]byte) []Signature {
	if bytes == nil {
		return nil
	}
	sigs := make([]Signature, len(bytes))
	for i := range bytes {
		sigs[i] = bytes[i]
	}
	return sigs
}

// Convert a list of blocks to a list of protobuf blocks.
func blocksToProto(blocks []*Block) []*pb.Block {
	pbBlocks := make([]*pb.Block, len(blocks))
	for i, b := range blocks {
		pbBlocks[i] = b.ToProto()
	}
	return pbBlocks
}

// protoToBlocks converts a list of protobuf blocks to a list of go struct blocks.
func protoToBlocks(pbBlocks []*pb.Block) []*Block {
	blocks := make([]*Block, len(pbBlocks))
	for i, b := range pbBlocks {
		blocks[i] = new(Block)
		err := blocks[i].FromProto(b)
		if err != nil {
			panic(err)
		}
	}
	return blocks
}

// commitsToProto converts a list of commits to a list of protobuf commits.
func commitsToProto(commits []*Commit) []*pb.Commit {
	pbCommits := make([]*pb.Commit, len(commits))
	for i, c := range commits {
		pbCommits[i] = c.ToProto()
	}
	return pbCommits
}

// protoToCommits converts a list of protobuf commits to a list of go struct commits.
func protoToCommits(pbCommits []*pb.Commit) []*Commit {
	commits := make([]*Commit, len(pbCommits))
	for i, c := range pbCommits {
		commits[i] = new(Commit)
		err := commits[i].FromProto(c)
		if err != nil {
			panic(err)
		}
	}
	return commits
}
