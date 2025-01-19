package types

import (
	"encoding/json"
	"math/big"
)

type SignData struct {
	To        string
	Data      string
	GasLimit  uint64
	GasFeeCap *big.Int
	Nonce     uint64
}

type SignerJSONRPCRequest struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int64       `json:"id"`
}

type SignerJSONRPCResponse struct {
	Version string              `json:"jsonrpc"`
	Result  json.RawMessage     `json:"result"`
	Error   *SignerJSONRPCError `json:"error,omitempty"`
	ID      int64               `json:"id"`
}

type SignerJSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
