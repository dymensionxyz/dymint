package signer

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"

	weaveVMtypes "github.com/dymensionxyz/dymint/da/weavevm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Web3SignerClient struct {
	chainID  int64
	endpoint string
	log      Logger
	client   *http.Client
}

func NewWeb3SignerClient(cfg *weaveVMtypes.Config, log Logger) (*Web3SignerClient, error) {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: cfg.Timeout,
		}).DialContext,
		TLSHandshakeTimeout: cfg.Timeout,
	}

	// Configure TLS if cert and key files are provided
	if cfg.Web3SignerTLSCertFile != "" && cfg.Web3SignerTLSKeyFile != "" {
		err := configureTransportTLS(transport, cfg, log)
		if err != nil {
			return nil, err
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	return &Web3SignerClient{
		endpoint: cfg.Web3SignerEndpoint,
		log:      log,
		client:   client,
		chainID:  cfg.ChainID,
	}, nil
}

func configureTransportTLS(transport *http.Transport, cfg *weaveVMtypes.Config, log Logger) error {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	cert, err := tls.LoadX509KeyPair(cfg.Web3SignerTLSCertFile, cfg.Web3SignerTLSKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	// Load CA certificate if provided
	if cfg.Web3SignerTLSCACertFile != "" {
		caCert, err := os.ReadFile(cfg.Web3SignerTLSCACertFile)
		if err != nil {
			return fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = caCertPool
	} else {
		// If no CA cert provided, skip verification
		log.Info("No CA certificate provided, TLS verification will be skipped",
			"endpoint", cfg.Web3SignerEndpoint)
		tlsConfig.InsecureSkipVerify = true
	}

	transport.TLSClientConfig = tlsConfig
	log.Info("TLS configuration enabled for Web3Signer client",
		"cert_file", cfg.Web3SignerTLSCertFile,
		"key_file", cfg.Web3SignerTLSKeyFile,
		"ca_file", cfg.Web3SignerTLSCACertFile)

	return nil
}

func (web3s *Web3SignerClient) SignTransaction(ctx context.Context, signData *weaveVMtypes.SignData) (string, error) {
	return web3s.signTxWithWeb3Signer(ctx, signData.To, signData.Data, signData.GasFeeCap, signData.GasLimit, signData.Nonce)
}

func (web3s *Web3SignerClient) signTxWithWeb3Signer(ctx context.Context, to string, data string, gasFeeCap *big.Int, gasLimit, nonce uint64) (string, error) {
	web3s.log.Info("sign transaction using web3signer")

	fromAddress, err := web3s.GetAccount(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get an account from web3signer rpc: %w", err)
	}

	toAddr := common.HexToAddress(to)
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

	// Prepare transaction for Web3Signer
	tx := map[string]interface{}{
		"from":                 fromAddress.String(),
		"to":                   toAddr.String(),
		"gas":                  fmt.Sprintf("0x%x", gasLimit),
		"maxFeePerGas":         fmt.Sprintf("0x%x", gasFeeCap),
		"maxPriorityFeePerGas": "0x0",
		"value":                "0x0",
		"data":                 bytesData,
		"nonce":                fmt.Sprintf("0x%x", nonce),
		"chainId":              fmt.Sprintf("0x%x", web3s.chainID),
	}

	// Sign transaction using Web3Signer
	signedTx, err := web3s.signTransaction(ctx, tx)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction with web3signer: %w", err)
	}

	return signedTx, nil
}

func (web3s *Web3SignerClient) GetAccount(ctx context.Context) (common.Address, error) {
	accounts, err := web3s.getAccounts(ctx)
	if err != nil {
		return common.Address{}, err
	}

	return accounts[0], nil
}

// GetAccount retrieves the account addresses from Web3Signer
func (web3s *Web3SignerClient) getAccounts(ctx context.Context) ([]common.Address, error) {
	request := weaveVMtypes.SignerJSONRPCRequest{
		Version: "2.0",
		Method:  "eth_accounts",
		Params:  []interface{}{},
		ID:      1,
	}

	response, err := web3s.doRequest(ctx, &request)
	if err != nil {
		return nil, fmt.Errorf("web3signer request failed: %w", err)
	}

	// Parse the raw JSON result into a string array
	var result []string
	if err := json.Unmarshal(response.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal addresses: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("web3signer returned no accounts")
	}

	addresses := make([]common.Address, len(result))
	for i, address := range result {
		if !common.IsHexAddress(address) {
			return nil, fmt.Errorf("invalid address format received: %s", address)
		}
		addresses[i] = common.HexToAddress(address)
	}

	return addresses, nil
}

// SignTransaction signs a transaction using Web3Signer
func (web3s *Web3SignerClient) signTransaction(ctx context.Context, tx interface{}) (string, error) {
	request := weaveVMtypes.SignerJSONRPCRequest{
		Version: "2.0",
		Method:  "eth_signTransaction",
		Params:  []interface{}{tx},
		ID:      1,
	}

	response, err := web3s.doRequest(ctx, &request)
	if err != nil {
		return "", fmt.Errorf("web3signer request failed: %w", err)
	}

	// Parse the raw JSON result into a string
	var signedTx string
	if err := json.Unmarshal(response.Result, &signedTx); err != nil {
		return "", fmt.Errorf("failed to unmarshal signed transaction: %w", err)
	}

	if signedTx == "" {
		return "", fmt.Errorf("web3signer returned empty signature")
	}

	return signedTx, nil
}

// doRequest performs the HTTP request to the Web3Signer endpoint
func (web3s *Web3SignerClient) doRequest(ctx context.Context, request *weaveVMtypes.SignerJSONRPCRequest) (*weaveVMtypes.SignerJSONRPCResponse, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, web3s.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := web3s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var jsonRPCResponse weaveVMtypes.SignerJSONRPCResponse
	if err := json.Unmarshal(body, &jsonRPCResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if jsonRPCResponse.Error != nil {
		return nil, fmt.Errorf("web3signer error: %s (code: %d)",
			jsonRPCResponse.Error.Message,
			jsonRPCResponse.Error.Code)
	}

	return &jsonRPCResponse, nil
}
