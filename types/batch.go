package types

const (
	MaxBlockSizeAdjustment = 0.9
)

type Batch struct {
	Blocks  []*Block
	Commits []*Commit

	LastBatch  bool
	DRSVersion []uint32
	Revision   uint64
}

func (b Batch) StartHeight() uint64 {
	if len(b.Blocks) == 0 {
		return 0
	}
	return b.Blocks[0].Header.Height
}

func (b Batch) EndHeight() uint64 {
	if len(b.Blocks) == 0 {
		return 0
	}
	return b.Blocks[len(b.Blocks)-1].Header.Height
}

func (b Batch) NumBlocks() uint64 {
	return uint64(len(b.Blocks))
}

func (b Batch) SizeBlockAndCommitBytes() int {
	cnt := 0
	for _, block := range b.Blocks {
		cnt += block.SizeBytes()
	}
	for _, commit := range b.Commits {
		cnt += commit.SizeBytes()
	}
	return cnt
}

func (b Batch) SizeBytes() int {
	return b.ToProto().Size()
}
