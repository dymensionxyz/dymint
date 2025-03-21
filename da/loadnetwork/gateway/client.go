package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	loadnetworktypes "github.com/dymensionxyz/dymint/da/loadnetwork/types"
	"github.com/dymensionxyz/dymint/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Gateway struct {
	config *loadnetworktypes.Config
	logger types.Logger
}

func NewGatewayClient(config *loadnetworktypes.Config, logger types.Logger) *Gateway {
	return &Gateway{config, logger}
}

const loadnetworkGatewayURL = "https://gateway.load.network/v1/calldata/%s"

// Modified get function with improved error handling
func (g *Gateway) RetrieveFromGateway(ctx context.Context, txHash string) (*loadnetworktypes.LNDymintBlob, error) {
	type LNRetrieverResponse struct {
		ArweaveBlockHash   string `json:"arweave_block_hash"`
		Calldata           string `json:"calldata"`
		WarDecodedCalldata string `json:"war_decoded_calldata"`
		LNBlockHash        string `json:"weavevm_block_hash"`
		LNBlockNumber      uint64 `json:"wvm_block_id"`
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf(loadnetworkGatewayURL,
			txHash), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	r.Header.Set("Accept", "application/json")
	client := &http.Client{
		Timeout: g.config.Timeout,
	}

	g.logger.Debug("sending request to loadnetwork data retriever",
		"url", r.URL.String(),
		"headers", r.Header)

	resp, err := client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("failed to call loadnetwork-data-retriever: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if err := validateResponse(resp, body); err != nil {
		g.logger.Error("invalid response from loadnetwork data retriever",
			"status", resp.Status,
			"content_type", resp.Header.Get("Content-Type"),
			"body", string(body))
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	var loadnetworkData LNRetrieverResponse
	if err = json.Unmarshal(body, &loadnetworkData); err != nil {
		g.logger.Error("failed to unmarshal response",
			"error", err,
			"body", string(body),
			"content_type", resp.Header.Get("Content-Type"))
		return nil, fmt.Errorf("failed to unmarshal response: %w, body: %s", err, string(body))
	}

	g.logger.Info("loadnetwork backend: get data from LoadNetwork",
		"arweave_block_hash", loadnetworkData.ArweaveBlockHash,
		"loadnetwork_block_hash", loadnetworkData.LNBlockHash,
		"calldata_length", len(loadnetworkData.Calldata))

	blob, err := hexutil.Decode(loadnetworkData.Calldata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode calldata: %w", err)
	}

	if len(blob) == 0 {
		return nil, fmt.Errorf("decoded blob has length zero")
	}

	return &loadnetworktypes.LNDymintBlob{ArweaveBlockHash: loadnetworkData.ArweaveBlockHash, LNBlockHash: loadnetworkData.LNBlockHash, LNTxHash: txHash, Blob: blob}, nil
}

func validateResponse(resp *http.Response, body []byte) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("unexpected content type: %s, body: %s", contentType, string(body))
	}

	return nil
}
