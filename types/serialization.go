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

func (b *Block) MarshalBinary() ([]byte, error) {
	return b.ToProto().Marshal()
}

func (b *Batch) MarshalBinary() ([]byte, error) {
	return b.ToProto().Marshal()
}

func (b *Block) UnmarshalBinary(data []byte) error {
	var pBlock pb.Block
	err := pBlock.Unmarshal(data)
	if err != nil {
		return err
	}
	err = b.FromProto(&pBlock)
	return err
}

func (b *Batch) UnmarshalBinary(data []byte) error {
	var pBatch pb.Batch
	err := pBatch.Unmarshal(data)
	if err != nil {
		return err
	}
	err = b.FromProto(&pBatch)
	return err
}

func (h *Header) MarshalBinary() ([]byte, error) {
	return h.ToProto().Marshal()
}

func (h *Header) UnmarshalBinary(data []byte) error {
	var pHeader pb.Header
	err := pHeader.Unmarshal(data)
	if err != nil {
		return err
	}
	err = h.FromProto(&pHeader)
	return err
}

func (d *Data) MarshalBinary() ([]byte, error) {
	return d.ToProto().Marshal()
}

func (c *Commit) MarshalBinary() ([]byte, error) {
	return c.ToProto().Marshal()
}

func (c *Commit) UnmarshalBinary(data []byte) error {
	var pCommit pb.Commit
	err := pCommit.Unmarshal(data)
	if err != nil {
		return err
	}
	err = c.FromProto(&pCommit)
	return err
}

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

func safeCopy(dst, src []byte) bool {
	if len(src) != len(dst) {
		return false
	}
	_ = copy(dst, src)
	return true
}

func (b *Block) ToProto() *pb.Block {
	return &pb.Block{
		Header:     b.Header.ToProto(),
		Data:       b.Data.ToProto(),
		LastCommit: b.LastCommit.ToProto(),
	}
}

func (b *Batch) ToProto() *pb.Batch {
	return &pb.Batch{
		StartHeight: b.StartHeight(),
		EndHeight:   b.EndHeight(),
		Blocks:      blocksToProto(b.Blocks),
		Commits:     commitsToProto(b.Commits),
	}
}

func (d *Data) ToProto() *pb.Data {
	return &pb.Data{
		Txs:                    txsToByteSlices(d.Txs),
		IntermediateStateRoots: d.IntermediateStateRoots.RawRootsList,
		Evidence:               evidenceToProto(d.Evidence),
		ConsensusMessages:      d.ConsensusMessages,
	}
}

func (b *Block) FromProto(other *pb.Block) error {
	err := b.Header.FromProto(other.Header)
	if err != nil {
		return err
	}
	b.Data.Txs = byteSlicesToTxs(other.Data.Txs)
	b.Data.IntermediateStateRoots.RawRootsList = other.Data.IntermediateStateRoots
	b.Data.Evidence = evidenceFromProto(other.Data.Evidence)
	b.Data.ConsensusMessages = other.Data.ConsensusMessages
	if other.LastCommit != nil {
		err := b.LastCommit.FromProto(other.LastCommit)
		if err != nil {
			return err
		}
	}

	return nil
}

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

func (c *Commit) FromProto(other *pb.Commit) error {
	c.Height = other.Height
	if !safeCopy(c.HeaderHash[:], other.HeaderHash) {
		return errors.New("invalid length of HeaderHash")
	}
	c.Signatures = byteSlicesToSignatures(other.Signatures)

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

func (s *State) ToProto() (*pb.State, error) {
	var proposerProto *pb.Sequencer
	proposer := s.GetProposer()
	if proposer != nil {
		var err error
		proposerProto, err = proposer.ToProto()
		if err != nil {
			return nil, err
		}
	}

	return &pb.State{
		Version:             &s.Version,
		ChainId:             s.ChainID,
		InitialHeight:       int64(s.InitialHeight),
		LastBlockHeight:     int64(s.Height()),
		ConsensusParams:     s.ConsensusParams,
		LastResultsHash:     s.LastResultsHash[:],
		LastHeaderHash:      s.LastHeaderHash[:],
		AppHash:             s.AppHash[:],
		RollappParams:       s.RollappParams,
		Proposer:            proposerProto,
		RevisionStartHeight: int64(s.RevisionStartHeight),
	}, nil
}

func (s *State) FromProto(other *pb.State) error {
	s.Version = *other.Version
	s.ChainID = other.ChainId
	s.InitialHeight = uint64(other.InitialHeight)
	s.SetHeight(uint64(other.LastBlockHeight))
	s.RevisionStartHeight = uint64(other.RevisionStartHeight)
	if other.Proposer != nil {
		proposer, err := SequencerFromProto(other.Proposer)
		if err != nil {
			return err
		}
		s.SetProposer(proposer)
	} else {
		s.SetProposer(nil)
	}

	s.ConsensusParams = other.ConsensusParams
	copy(s.LastResultsHash[:], other.LastResultsHash)
	copy(s.LastHeaderHash[:], other.LastHeaderHash)
	copy(s.AppHash[:], other.AppHash)
	s.RollappParams = other.RollappParams
	return nil
}

func (s *Sequencer) ToProto() (*pb.Sequencer, error) {
	if s == nil {
		return nil, fmt.Errorf("nil sequencer")
	}
	protoVal, err := s.val.ToProto()
	if err != nil {
		return nil, fmt.Errorf("tendermint validator to proto: %w", err)
	}
	return &pb.Sequencer{
		SettlementAddress:   s.SettlementAddress,
		Validator:           *protoVal,
		RewardAddr:          s.RewardAddr,
		WhitelistedRelayers: s.WhitelistedRelayers,
	}, nil
}

func SequencerFromProto(seq *pb.Sequencer) (*Sequencer, error) {
	if seq == nil {
		return nil, fmt.Errorf("nil sequencer")
	}
	val, err := types.ValidatorFromProto(&seq.Validator)
	if err != nil {
		return nil, fmt.Errorf("tendermint validator from proto: %w", err)
	}
	return &Sequencer{
		SettlementAddress:   seq.SettlementAddress,
		RewardAddr:          seq.RewardAddr,
		WhitelistedRelayers: seq.WhitelistedRelayers,
		val:                 *val,
	}, nil
}

func (s Sequencers) ToProto() (*pb.SequencerSet, error) {
	seqs := make([]pb.Sequencer, len(s))
	for i, seq := range s {
		seqProto, err := seq.ToProto()
		if err != nil {
			return nil, fmt.Errorf("sequencer to proto: %w", err)
		}
		seqs[i] = *seqProto
	}
	return &pb.SequencerSet{Sequencers: seqs}, nil
}

func SequencersFromProto(s *pb.SequencerSet) (Sequencers, error) {
	if s == nil {
		return Sequencers{}, fmt.Errorf("nil sequencer set")
	}
	seqs := make([]Sequencer, len(s.Sequencers))
	for i, seq := range s.Sequencers {
		sequencer, err := SequencerFromProto(&seq)
		if err != nil {
			return Sequencers{}, fmt.Errorf("sequencer from proto: %w", err)
		}
		seqs[i] = *sequencer
	}
	return seqs, nil
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
			ret = append(ret, &ae)
		}
	}
	return ret
}

func evidenceFromProto([]*abci.Evidence) EvidenceData {
	var ret EvidenceData

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

func blocksToProto(blocks []*Block) []*pb.Block {
	pbBlocks := make([]*pb.Block, len(blocks))
	for i, b := range blocks {
		pbBlocks[i] = b.ToProto()
	}
	return pbBlocks
}

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

func commitsToProto(commits []*Commit) []*pb.Commit {
	pbCommits := make([]*pb.Commit, len(commits))
	for i, c := range commits {
		pbCommits[i] = c.ToProto()
	}
	return pbCommits
}

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
