package json

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/errors"
	"github.com/gorilla/rpc/v2/json2"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	rpctypes "github.com/tendermint/tendermint/rpc/jsonrpc/types"

	"github.com/dymensionxyz/dymint/rpc/client"
	"github.com/dymensionxyz/dymint/rpc/client/tee"
	"github.com/dymensionxyz/dymint/types"
)

const (
	// defaultSubscribeTimeout is the default timeout for a subscription.
	defaultSubscribeTimeout = 5 * time.Second
	// defaultSubscribeBufferSize is the default buffer size for a subscription.
	defaultSubscribeBufferSize = 100
)

// GetHTTPHandler returns handler configured to serve Tendermint-compatible RPC.
func GetHTTPHandler(l *client.Client, logger types.Logger, opts ...option) (http.Handler, error) {
	return newHandler(newService(l, logger, opts...), json2.NewCodec(), logger), nil
}

type option func(*service)

func WithSubscribeTimeout(timeout time.Duration) option {
	return func(s *service) {
		s.subscribeTimeout = timeout
	}
}

func WithSubscribeBufferSize(size int) option {
	return func(s *service) {
		s.subscribeBufferSize = size
	}
}

type method struct {
	m          reflect.Value
	argsType   reflect.Type
	returnType reflect.Type
	ws         bool
}

func newMethod(m interface{}) *method {
	mType := reflect.TypeOf(m)

	return &method{
		m:          reflect.ValueOf(m),
		argsType:   mType.In(1).Elem(),
		returnType: mType.Out(0).Elem(),
		ws:         mType.NumIn() == 4,
	}
}

type service struct {
	client  *client.Client
	methods map[string]*method
	logger  types.Logger

	subscribeTimeout    time.Duration
	subscribeBufferSize int
}

func newService(c *client.Client, l types.Logger, opts ...option) *service {
	s := service{
		client:              c,
		logger:              l,
		subscribeTimeout:    defaultSubscribeTimeout,
		subscribeBufferSize: defaultSubscribeBufferSize,
	}
	s.methods = map[string]*method{
		"abci_info":            newMethod(s.ABCIInfo),
		"abci_query":           newMethod(s.ABCIQuery),
		"block_by_hash":        newMethod(s.BlockByHash),
		"block_results":        newMethod(s.BlockResults),
		"block_search":         newMethod(s.BlockSearch),
		"block_validated":      newMethod(s.BlockValidated),
		"block":                newMethod(s.Block),
		"blockchain":           newMethod(s.BlockchainInfo),
		"broadcast_evidence":   newMethod(s.BroadcastEvidence),
		"broadcast_tx_async":   newMethod(s.BroadcastTxAsync),
		"broadcast_tx_commit":  newMethod(s.BroadcastTxCommit),
		"broadcast_tx_sync":    newMethod(s.BroadcastTxSync),
		"check_tx":             newMethod(s.CheckTx),
		"commit":               newMethod(s.Commit),
		"consensus_params":     newMethod(s.ConsensusParams),
		"consensus_state":      newMethod(s.GetConsensusState),
		"dump_consensus_state": newMethod(s.DumpConsensusState),
		"eth_blockNumber":      newMethod(s.EthBlockNumber),
		"eth_chainId":          newMethod(s.EthChainId),
		"eth_getBalance":       newMethod(s.EthGetBalance),
		"eth_getBlockByNumber": newMethod(s.EthGetBlockByNumber),
		"genesis_chunked":      newMethod(s.GenesisChunked),
		"genesis":              newMethod(s.Genesis),
		"health":               newMethod(s.Health),
		"net_info":             newMethod(s.NetInfo),
		"num_unconfirmed_txs":  newMethod(s.NumUnconfirmedTxs),
		"status":               newMethod(s.Status),
		"subscribe":            newMethod(s.Subscribe),
		"tee":                  newMethod(s.Tee),
		"tx_search":            newMethod(s.TxSearch),
		"tx":                   newMethod(s.Tx),
		"unconfirmed_txs":      newMethod(s.UnconfirmedTxs),
		"unsubscribe_all":      newMethod(s.UnsubscribeAll),
		"unsubscribe":          newMethod(s.Unsubscribe),
		"validators":           newMethod(s.Validators),
	}

	for _, opt := range opts {
		opt(&s)
	}

	return &s
}

func (s *service) Subscribe(req *http.Request, args *subscribeArgs, wsConn *wsConn, subscriptionID []byte) (*ctypes.ResultSubscribe, error) {
	addr := req.RemoteAddr

	if err := s.client.IsSubscriptionAllowed(addr); err != nil {
		return nil, errors.Wrap(err, "subscription not allowed")
	}

	s.logger.Debug("subscribe to query", "remote", addr, "query", args.Query)

	ctx, cancel := context.WithTimeout(req.Context(), s.subscribeTimeout)
	defer cancel()

	out, err := s.client.Subscribe(ctx, addr, args.Query, s.subscribeBufferSize)
	if err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}
	go func(subscriptionID []byte) {
		for msg := range out {
			// build the base response
			var resp rpctypes.RPCResponse
			// Check if subscriptionID is string or int and generate the rest of the response accordingly
			subscriptionIDInt, err := strconv.Atoi(string(subscriptionID))
			if err != nil {
				s.logger.Info("Failed to convert subscriptionID to int")
				resp = rpctypes.NewRPCSuccessResponse(rpctypes.JSONRPCStringID(subscriptionID), msg)
			} else {
				resp = rpctypes.NewRPCSuccessResponse(rpctypes.JSONRPCIntID(subscriptionIDInt), msg)
			}
			// Marshal response to JSON and send it to the websocket queue
			jsonBytes, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				s.logger.Error("marshal RPCResponse to JSON", "err", err)
				continue
			}
			if wsConn != nil {
				wsConn.queue <- jsonBytes
			}
		}
	}(subscriptionID)

	return &ctypes.ResultSubscribe{}, nil
}

func (s *service) Unsubscribe(req *http.Request, args *unsubscribeArgs) (*emptyResult, error) {
	s.logger.Debug("unsubscribe from query", "remote", req.RemoteAddr, "query", args.Query)
	err := s.client.Unsubscribe(req.Context(), req.RemoteAddr, args.Query)
	if err != nil {
		return nil, fmt.Errorf("unsubscribe: %w", err)
	}
	return &emptyResult{}, nil
}

func (s *service) UnsubscribeAll(req *http.Request, args *unsubscribeAllArgs) (*emptyResult, error) {
	s.logger.Debug("unsubscribe from all queries", "remote", req.RemoteAddr)
	err := s.client.UnsubscribeAll(req.Context(), req.RemoteAddr)
	if err != nil {
		return nil, fmt.Errorf("unsubscribe all: %w", err)
	}
	return &emptyResult{}, nil
}

// info API
func (s *service) Health(req *http.Request, args *healthArgs) (*ctypes.ResultHealth, error) {
	return s.client.Health(req.Context())
}

func (s *service) Status(req *http.Request, args *statusArgs) (*ctypes.ResultStatus, error) {
	return s.client.Status(req.Context())
}

func (s *service) NetInfo(req *http.Request, args *netInfoArgs) (*ctypes.ResultNetInfo, error) {
	return s.client.NetInfo(req.Context())
}

func (s *service) BlockchainInfo(req *http.Request, args *blockchainInfoArgs) (*ctypes.ResultBlockchainInfo, error) {
	return s.client.BlockchainInfo(req.Context(), int64(args.MinHeight), int64(args.MaxHeight))
}

func (s *service) Genesis(req *http.Request, args *genesisArgs) (*ctypes.ResultGenesis, error) {
	return s.client.Genesis(req.Context())
}

func (s *service) GenesisChunked(req *http.Request, args *genesisChunkedArgs) (*ctypes.ResultGenesisChunk, error) {
	return s.client.GenesisChunked(req.Context(), uint(args.ID)) //nolint:gosec // id is always positive
}

func (s *service) Block(req *http.Request, args *blockArgs) (*ctypes.ResultBlock, error) {
	return s.client.Block(req.Context(), (*int64)(&args.Height))
}

func (s *service) BlockByHash(req *http.Request, args *blockByHashArgs) (*ctypes.ResultBlock, error) {
	return s.client.BlockByHash(req.Context(), args.Hash)
}

func (s *service) BlockResults(req *http.Request, args *blockResultsArgs) (*ctypes.ResultBlockResults, error) {
	return s.client.BlockResults(req.Context(), (*int64)(&args.Height))
}

func (s *service) Commit(req *http.Request, args *commitArgs) (*ctypes.ResultCommit, error) {
	return s.client.Commit(req.Context(), (*int64)(&args.Height))
}

func (s *service) CheckTx(req *http.Request, args *checkTxArgs) (*ctypes.ResultCheckTx, error) {
	return s.client.CheckTx(req.Context(), args.Tx)
}

func (s *service) Tee(req *http.Request) (*tee.Response, error) {
	return s.client.Tee(req.Context())
}

func (s *service) Tx(req *http.Request, args *txArgs) (*ctypes.ResultTx, error) {
	return s.client.Tx(req.Context(), args.Hash, args.Prove)
}

func (s *service) TxSearch(req *http.Request, args *txSearchArgs) (*ctypes.ResultTxSearch, error) {
	return s.client.TxSearch(req.Context(), args.Query, args.Prove, (*int)(&args.Page), (*int)(&args.PerPage), args.OrderBy)
}

func (s *service) BlockSearch(req *http.Request, args *blockSearchArgs) (*ctypes.ResultBlockSearch, error) {
	return s.client.BlockSearch(req.Context(), args.Query, (*int)(&args.Page), (*int)(&args.PerPage), args.OrderBy)
}

func (s *service) Validators(req *http.Request, args *validatorsArgs) (*ctypes.ResultValidators, error) {
	return s.client.Validators(req.Context(), (*int64)(&args.Height), (*int)(&args.Page), (*int)(&args.PerPage))
}

func (s *service) DumpConsensusState(req *http.Request, args *dumpConsensusStateArgs) (*ctypes.ResultDumpConsensusState, error) {
	return s.client.DumpConsensusState(req.Context())
}

func (s *service) GetConsensusState(req *http.Request, args *getConsensusStateArgs) (*ctypes.ResultConsensusState, error) {
	return s.client.ConsensusState(req.Context())
}

func (s *service) ConsensusParams(req *http.Request, args *consensusParamsArgs) (*ctypes.ResultConsensusParams, error) {
	return s.client.ConsensusParams(req.Context(), (*int64)(&args.Height))
}

func (s *service) UnconfirmedTxs(req *http.Request, args *unconfirmedTxsArgs) (*ctypes.ResultUnconfirmedTxs, error) {
	return s.client.UnconfirmedTxs(req.Context(), (*int)(&args.Limit))
}

func (s *service) NumUnconfirmedTxs(req *http.Request, args *numUnconfirmedTxsArgs) (*ctypes.ResultUnconfirmedTxs, error) {
	return s.client.NumUnconfirmedTxs(req.Context())
}

// tx broadcast API
func (s *service) BroadcastTxCommit(req *http.Request, args *broadcastTxCommitArgs) (*ctypes.ResultBroadcastTxCommit, error) {
	return s.client.BroadcastTxCommit(req.Context(), args.Tx)
}

func (s *service) BroadcastTxSync(req *http.Request, args *broadcastTxSyncArgs) (*ctypes.ResultBroadcastTx, error) {
	return s.client.BroadcastTxSync(req.Context(), args.Tx)
}

func (s *service) BroadcastTxAsync(req *http.Request, args *broadcastTxAsyncArgs) (*ctypes.ResultBroadcastTx, error) {
	return s.client.BroadcastTxAsync(req.Context(), args.Tx)
}

// abci API
func (s *service) ABCIQuery(req *http.Request, args *ABCIQueryArgs) (*ctypes.ResultABCIQuery, error) {
	return s.client.ABCIQueryWithOptions(req.Context(), args.Path, args.Data, rpcclient.ABCIQueryOptions{
		Height: int64(args.Height),
		Prove:  args.Prove,
	})
}

func (s *service) ABCIInfo(req *http.Request, args *ABCIInfoArgs) (*ctypes.ResultABCIInfo, error) {
	return s.client.ABCIInfo(req.Context())
}

// evidence API
func (s *service) BroadcastEvidence(req *http.Request, args *broadcastEvidenceArgs) (*ctypes.ResultBroadcastEvidence, error) {
	return s.client.BroadcastEvidence(req.Context(), args.Evidence)
}

func (s *service) BlockValidated(req *http.Request, args *blockArgs) (*client.ResultBlockValidated, error) {
	fmt.Println(args)
	return s.client.BlockValidated((*int64)(&args.Height))
}

func (s *service) EthChainId(req *http.Request, args *blockArgs) (*client.ResultEthMethod, error) {
	return s.client.ChainID()
}

func (s *service) EthBlockNumber(req *http.Request, args *blockArgs) (*client.ResultEthMethod, error) {
	return s.client.EthBlockNumber(req.Context())
}

func (s *service) EthGetBlockByNumber(req *http.Request, args *ethBlockArgs) (*client.ResultEthMethod, error) {
	heightStr := strings.Replace(args.Height, "0x", "", -1)
	height, err := strconv.ParseInt(heightStr, 16, 64)
	if err != nil {
		return &client.ResultEthMethod{Result: ""}, nil
	}
	return s.client.EthGetBlockByNumber(req.Context(), &height)
}

func (s *service) EthGetBalance(req *http.Request, args *ethBalanceArgs) (*client.ResultEthMethod, error) {
	heightStr := strings.Replace(args.Height, "0x", "", -1)
	height, err := strconv.ParseInt(heightStr, 16, 64)
	if err != nil {
		height = 0
	}
	addrStr := strings.Replace(args.Address, "0x", "", -1)

	return s.client.EthGetBalance(req.Context(), addrStr, &height)
}
