package types

import (
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
)

type Evidence interface {
	ABCI() []abci.Evidence
	Bytes() []byte
	Hash() []byte
	Height() int64
	String() string
	Time() time.Time
	ValidateBasic() error
}
