package types

import (
	"math"
)





const (
	
	NamespaceVersionSize = 1
	
	
	NamespaceVersionMaxValue = math.MaxUint8

	
	NamespaceIDSize = 28

	
	NamespaceSize = NamespaceVersionSize + NamespaceIDSize

	
	ShareSize = 512

	
	
	ShareInfoBytes = 1

	
	
	SequenceLenBytes = 4

	
	ShareVersionZero = uint8(0)

	
	
	DefaultShareVersion = ShareVersionZero

	
	
	CompactShareReservedBytes = 4

	
	
	FirstCompactShareContentSize = ShareSize - NamespaceSize - ShareInfoBytes - SequenceLenBytes - CompactShareReservedBytes

	
	
	ContinuationCompactShareContentSize = ShareSize - NamespaceSize - ShareInfoBytes - CompactShareReservedBytes

	
	
	FirstSparseShareContentSize = ShareSize - NamespaceSize - ShareInfoBytes - SequenceLenBytes

	
	
	ContinuationSparseShareContentSize = ShareSize - NamespaceSize - ShareInfoBytes

	
	MinSquareSize = 1

	
	
	MinShareCount = MinSquareSize * MinSquareSize

	
	MaxShareVersion = 127

	
	DefaultGovMaxSquareSize = 64

	
	DefaultMaxBytes = DefaultGovMaxSquareSize * DefaultGovMaxSquareSize * ContinuationSparseShareContentSize
)
