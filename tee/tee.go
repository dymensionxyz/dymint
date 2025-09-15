package tee

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	rollapptypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
)

// TEEResponse represents the response from the full node's /tee endpoint
type TEEResponse struct {
	Token string                `json:"token"`
	Nonce rollapptypes.TEENonce `json:"nonce"`
}

// TEEFinalizer handles fast finalization using TEE attestations
// It runs on the sequencer and queries the full node for attestations
type TEEFinalizer struct {
	config        config.BlockManagerConfig
	logger        types.Logger
	sidecarClient *http.Client
	hubClient     settlement.ClientI
}

func NewTEEFinalizer(config config.BlockManagerConfig, logger types.Logger, slClient settlement.ClientI) *TEEFinalizer {
	return &TEEFinalizer{
		config: config,
		logger: logger,
		sidecarClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		hubClient: slClient,
	}
}

func (f *TEEFinalizer) Start(ctx context.Context) error {
	ticker := time.NewTicker(f.config.TeeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			f.logger.Info("Stopping TEE attestation client")
			return nil
		case <-ticker.C:
			if err := f.fetchAndSubmitAttestation(); err != nil {
				f.logger.Error("Attestation fetch and submit", "error", err)
			}
		}
	}
}

func (f *TEEFinalizer) fetchAndSubmitAttestation() error {
	attestation, err := f.queryFullNodeTEE()
	if err != nil {
		return fmt.Errorf("query full node TEE: %w", err)
	}

	latestFinalizedHeight, err := f.hubClient.GetLatestFinalizedHeight()
	if err != nil {
		return fmt.Errorf("get latest finalized height: %w", err)
	}
	if attestation.Nonce.CurrHeight <= latestFinalizedHeight {
		return fmt.Errorf("attestation height is not greater than latest finalized height")
	}

	err = f.hubClient.SubmitTEEAttestation(
		attestation.Token,
		attestation.Nonce,
	)
	if err != nil {
		return fmt.Errorf("submit attestation to hub: %w", err)
	}

	return nil
}

func (f *TEEFinalizer) queryFullNodeTEE() (*TEEResponse, error) {
	return queryFullNodeTEE(f.sidecarClient, f.config.TeeSidecarURL)
}

func queryFullNodeTEE(client *http.Client, url string) (*TEEResponse, error) {
	// JSON-RPC request structure
	type jsonRPCRequest struct {
		JSONRPC string         `json:"jsonrpc"`
		Method  string         `json:"method"`
		Params  map[string]any `json:"params"`
		ID      int            `json:"id"`
	}

	// JSON-RPC response structure
	type jsonRPCResponse struct {
		JSONRPC string       `json:"jsonrpc"`
		Result  *TEEResponse `json:"result"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		ID int `json:"id"`
	}

	// Create JSON-RPC request
	request := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "tee",
		Params:  map[string]any{"dry": true},
		ID:      1,
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Make POST request to RPC endpoint
	resp, err := client.Post(
		url,
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("request attestation: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("full node returned status %d: %s", resp.StatusCode, string(body))
	}

	var rpcResponse jsonRPCResponse
	if err := json.Unmarshal(body, &rpcResponse); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Check for JSON-RPC error
	if rpcResponse.Error != nil {
		return nil, fmt.Errorf("JSON-RPC error %d: %s", rpcResponse.Error.Code, rpcResponse.Error.Message)
	}

	if rpcResponse.Result == nil {
		return nil, fmt.Errorf("empty result in JSON-RPC response")
	}

	return rpcResponse.Result, nil
}

// required to make work with json-rpc framework
func (t *TEEResponse) UnmarshalJSON(data []byte) error {
	var aux struct {
		Token string `json:"token"`
		Nonce struct {
			RollappId       string `json:"rollapp_id"`
			CurrHeight      string `json:"curr_height"`
			FinalizedHeight string `json:"finalized_height"`
		} `json:"nonce"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	currHeight, err := strconv.ParseUint(aux.Nonce.CurrHeight, 10, 64)
	if err != nil {
		return fmt.Errorf("parse curr_height: %w", err)
	}

	finalizedHeight, err := strconv.ParseUint(aux.Nonce.FinalizedHeight, 10, 64)
	if err != nil {
		return fmt.Errorf("parse finalized_height: %w", err)
	}

	t.Token = aux.Token
	t.Nonce = rollapptypes.TEENonce{
		RollappId:       aux.Nonce.RollappId,
		CurrHeight:      currHeight,
		FinalizedHeight: finalizedHeight,
	}

	return nil
}
