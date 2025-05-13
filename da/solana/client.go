package solana

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type SolanaClient interface {
	SubmitBlob(blob []byte) (common.Hash, []byte, []byte, error)
	GetBlob(txHash string) ([]byte, error)
	GetAccountAddress() string
	GetSignerBalance() (*big.Int, error)
	ValidateInclusion(txHash string, commitment []byte, proof []byte) error
}
