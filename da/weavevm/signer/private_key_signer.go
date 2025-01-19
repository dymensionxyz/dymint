package signer

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	weaveVMtypes "github.com/dymensionxyz/dymint/da/weavevm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type Logger interface {
	Debug(msg string, keyvals ...interface{})
	Info(msg string, keyvals ...interface{})
	Error(msg string, keyvals ...interface{})
}

type PrivateKeySigner struct {
	privateKey string
	chainID    int64

	log Logger
}

func NewPrivateKeySigner(privateKey string, log Logger, chainID int64) *PrivateKeySigner {
	return &PrivateKeySigner{privateKey: privateKey, log: log, chainID: chainID}
}

func (pks *PrivateKeySigner) SignTransaction(_ context.Context, signData *weaveVMtypes.SignData) (string, error) {
	return pks.signTxWithPrivateKey(signData.To, signData.Data, signData.GasFeeCap, signData.GasLimit, signData.Nonce)
}

func (pks *PrivateKeySigner) signTxWithPrivateKey(to string, data string, gasFeeCap *big.Int, gasLimit uint64, nonce uint64) (string, error) {
	pks.log.Info("sign transaction using private key")

	// Prepare data payload.
	var hexData string
	if strings.HasPrefix(data, "0x") {
		hexData = data
	} else {
		hexData = hexutil.Encode([]byte(data))
	}
	bytesData, err := hexutil.Decode(hexData)
	if err != nil {
		return "", err
	}

	toAddr := common.HexToAddress(to)
	txData := ethtypes.DynamicFeeTx{
		ChainID:   big.NewInt(pks.chainID),
		Nonce:     nonce,
		GasTipCap: big.NewInt(0),
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &toAddr,
		Data:      bytesData,
	}
	tx := ethtypes.NewTx(&txData)

	pKeyBytes, err := hexutil.Decode("0x" + pks.privateKey)
	if err != nil {
		return "", err
	}
	// Convert the private key bytes to an ECDSA private key.
	ecdsaPrivateKey, err := crypto.ToECDSA(pKeyBytes)
	if err != nil {
		return "", err
	}

	signedTx, err := ethtypes.SignTx(tx, ethtypes.LatestSignerForChainID(big.NewInt(pks.chainID)), ecdsaPrivateKey)
	if err != nil {
		return "", err
	}

	// Encode the signed transaction into RLP (Recursive Length Prefix) format for transmission.
	var buf bytes.Buffer
	err = signedTx.EncodeRLP(&buf)
	if err != nil {
		return "", err
	}

	// Return the RLP-encoded transaction as a hexadecimal string.
	rawTxRLPHex := hex.EncodeToString(buf.Bytes())

	return rawTxRLPHex, nil
}

func (pks *PrivateKeySigner) GetAccount(_ context.Context) (common.Address, error) {
	account, _, err := pks.getAddressFromPrivateKey()
	return account, err
}

func (pks *PrivateKeySigner) getAddressFromPrivateKey() (common.Address, *ecdsa.PrivateKey, error) {
	// Getting public address from private key
	pKeyBytes, err := hexutil.Decode("0x" + pks.privateKey)
	if err != nil {
		return common.Address{}, nil, err
	}
	// Convert the private key bytes to an ECDSA private key.
	ecdsaPrivateKey, err := crypto.ToECDSA(pKeyBytes)
	if err != nil {
		return common.Address{}, nil, err
	}
	// Extract the public key from the ECDSA private key.
	publicKey := ecdsaPrivateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, nil, fmt.Errorf("error casting public key to ECDSA")
	}

	// Compute the Ethereum address of the signer from the public key.
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return fromAddress, ecdsaPrivateKey, nil
}
