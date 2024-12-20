package rpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	weaveVMtypes "github.com/dymensionxyz/dymint/da/weave_vm/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
)

type Signer interface {
	GetAccount(ctx context.Context) (common.Address, error)
	SignTransaction(ctx context.Context, signData *weaveVMtypes.SignData) (string, error)
}

type Logger interface {
	Debug(msg string, keyvals ...interface{})
	Info(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})
}

// WeaveVM RPC client
type RPCClient struct {
	log     Logger
	client  *ethclient.Client
	chainID int64
	signer  Signer
}

func NewWvmRPCClient(log Logger, cfg *weaveVMtypes.Config, signer Signer) (*RPCClient, error) {
	client, err := ethclient.Dial(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the WeaveVM client: %w", err)
	}

	ethRPCClient := &RPCClient{
		log:     log,
		client:  client,
		chainID: cfg.ChainID,
		signer:  signer,
	}

	return ethRPCClient, nil
}

func (rpc *RPCClient) SendTransaction(ctx context.Context, to string, data []byte) (string, error) {
	gas, err := rpc.estimateGas(ctx, to, data)
	if err != nil {
		return "", fmt.Errorf("failed to store data in weaveVM: failed estimate gas: %w", err)
	}

	weaveVMRawTx, err := rpc.createRawTransaction(ctx, to, string(data), gas)
	if err != nil {
		return "", fmt.Errorf("failed to store data in weaveVM: failed create transaction: %w", err)
	}

	weaveVMTxHash, err := rpc.sendRawTransaction(ctx, weaveVMRawTx)
	if err != nil {
		return "", fmt.Errorf("failed to store data in weaveVM: failed to send transaction: %w", err)
	}

	return weaveVMTxHash, nil
}

// estimateGas tries estimates the suggested amount of gas that required to execute a given transaction.
func (rpc *RPCClient) estimateGas(ctx context.Context, to string, data []byte) (uint64, error) {
	var (
		toAddr    = common.HexToAddress(to)
		bytesData []byte
		err       error
	)

	fromAddress, err := rpc.signer.GetAccount(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate gas, no signer: %w", err)
	}

	var encoded string
	if string(data) != "" {
		if ok := strings.HasPrefix(string(data), "0x"); !ok {
			encoded = hexutil.Encode(data)
		}

		bytesData, err = hexutil.Decode(encoded)
		if err != nil {
			return 0, err
		}
	}

	msg := ethereum.CallMsg{
		From: fromAddress,
		To:   &toAddr,
		Gas:  0x00,
		Data: bytesData,
	}

	gas, err := rpc.client.EstimateGas(ctx, msg)
	if err != nil {
		return 0, err
	}

	rpc.log.Debug("weaveVM: estimated tx gas price", "price", gas)

	return gas, nil
}

// createRawTransaction creates a raw EIP-1559 transaction and returns it as a hex string.
func (rpc *RPCClient) createRawTransaction(ctx context.Context, to string, data string, gasLimit uint64) (string, error) {
	baseFee, err := rpc.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", err
	}

	fromAddress, err := rpc.signer.GetAccount(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get an account from signer: %w", err)
	}
	nonce, err := rpc.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return "", err
	}

	signData := weaveVMtypes.SignData{To: to, Data: data, GasLimit: gasLimit, GasFeeCap: baseFee, Nonce: nonce}
	return rpc.signer.SignTransaction(ctx, &signData)
}

func (rpc *RPCClient) sendRawTransaction(ctx context.Context, signedTxHex string) (string, error) {
	var err error
	var signedTxBytes []byte

	if strings.HasPrefix(signedTxHex, "0x") {
		signedTxBytes, err = hexutil.Decode(signedTxHex)
		if err != nil {
			return "", fmt.Errorf("failed to decode signed transaction: %w", err)
		}
	} else {
		signedTxBytes, err = hex.DecodeString(signedTxHex)
		if err != nil {
			return "", fmt.Errorf("failed to decode signed transaction: %w", err)
		}
	}

	tx := new(ethtypes.Transaction)
	err = tx.UnmarshalBinary(signedTxBytes)
	if err != nil {
		err = rlp.DecodeBytes(signedTxBytes, tx)
		if err != nil {
			return "", fmt.Errorf("failed to parse signed transaction: %w", err)
		}
	}

	err = rpc.client.SendTransaction(ctx, tx)
	if err != nil {
		return "", err
	}

	rpc.log.Info("weaveVM: successfully sent transaction", "tx hash", tx.Hash().String())

	err = rpc.logReceipt(tx)
	if err != nil {
		rpc.log.Error("failed to log sent transaction receipt", "error", err)
	}

	return tx.Hash().String(), nil
}

func (rpc *RPCClient) GetTransactionByHash(ctx context.Context, txHash string) (*ethtypes.Transaction, bool, error) {
	hash := common.HexToHash(txHash)
	tx, isPending, err := rpc.client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get transaction by hash: %w", err)
	}
	return tx, isPending, nil
}

func (rpc *RPCClient) GetTransactionReceipt(ctx context.Context, txHash string) (*ethtypes.Receipt, error) {
	hash := common.HexToHash(txHash)

	receipt, err := rpc.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	return receipt, nil
}

func (rpc *RPCClient) logReceipt(tx *ethtypes.Transaction) error {
	var txDetails Transaction
	txBytes, err := tx.MarshalJSON()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(txBytes, &txDetails); err != nil {
		return err
	}

	txDetails.TransactionCost = tx.Cost().String()

	convertFields := []string{"Nonce", "MaxPriorityFeePerGas", "MaxFeePerGas", "Value", "Type", "Gas"}
	for _, field := range convertFields {
		if err := convertHexField(&txDetails, field); err != nil {
			return err
		}
	}

	txJSON, err := json.MarshalIndent(txDetails, "", "\t")
	if err != nil {
		return err
	}

	rpc.log.Debug("weaveVM: transaction receipt", "tx receipt", string(txJSON))
	return nil
}

// Transaction represents the structure of the transaction JSON.
type Transaction struct {
	Type                 string   `json:"type"`
	ChainID              string   `json:"chainId"`
	Nonce                string   `json:"nonce"`
	To                   string   `json:"to"`
	Gas                  string   `json:"gas"`
	GasPrice             string   `json:"gasPrice,omitempty"`
	MaxPriorityFeePerGas string   `json:"maxPriorityFeePerGas"`
	MaxFeePerGas         string   `json:"maxFeePerGas"`
	Value                string   `json:"value"`
	Input                string   `json:"input"`
	AccessList           []string `json:"accessList"`
	V                    string   `json:"v"`
	R                    string   `json:"r"`
	S                    string   `json:"s"`
	YParity              string   `json:"yParity"`
	Hash                 string   `json:"hash"`
	TransactionTime      string   `json:"transactionTime,omitempty"`
	TransactionCost      string   `json:"transactionCost,omitempty"`
}

func convertHexField(tx *Transaction, field string) error {
	typeOfTx := reflect.TypeOf(*tx)
	txValue := reflect.ValueOf(tx).Elem()
	hexStr := txValue.FieldByName(field).String()
	intValue, err := strconv.ParseUint(hexStr[2:], 16, 64)
	if err != nil {
		return err
	}

	decimalStr := strconv.FormatUint(intValue, 10)
	_, ok := typeOfTx.FieldByName(field)
	if !ok {
		return fmt.Errorf("field %s does not exist in Transaction struct", field)
	}
	txValue.FieldByName(field).SetString(decimalStr)

	return nil
}
