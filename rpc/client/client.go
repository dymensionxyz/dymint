package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	sdkerrors "cosmossdk.io/errors"
	teetypes "github.com/dymensionxyz/dymint/tee"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/ethereum/go-ethereum/common/hexutil"
	evmostypes "github.com/evmos/evmos/v12/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/config"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	tmmath "github.com/tendermint/tendermint/libs/math"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
	"github.com/tendermint/tendermint/p2p"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/proxy"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
	tm_version "github.com/tendermint/tendermint/version"

	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/rpc/client/tee"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/version"

	ethctypes "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmosrpctypes "github.com/evmos/evmos/v12/rpc/types"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	minttypes "github.com/dymensionxyz/dymension-rdk/x/mint/types"
)

const (
	defaultPerPage = 30
	maxPerPage     = 100

	// TODO(tzdybal): make this configurable
	subscribeTimeout = 5 * time.Second
)

type BlockValidationStatus int

const (
	NotValidated = iota
	P2PValidated
	SLValidated
)

// ErrConsensusStateNotAvailable is returned because Dymint doesn't use Tendermint consensus.
var ErrConsensusStateNotAvailable = errors.New("consensus state not available in Dymint")

var _ rpcclient.Client = &Client{}

// Client implements tendermint RPC client interface.
//
// This is the type that is used in communication between cosmos-sdk app and Dymint.
type Client struct {
	*tmtypes.EventBus
	config *config.RPCConfig
	node   *node.Node

	// cache of chunked genesis data.
	genChunks []string
}

// GetNode returns the underlying node instance.
func (c *Client) GetNode() *node.Node {
	return c.node
}

type ResultBlockValidated struct {
	ChainID string
	Result  BlockValidationStatus
}

type ResultEthMethod struct {
	Result string
}

// NewClient returns Client working with given node.
func NewClient(node *node.Node) *Client {
	return &Client{
		EventBus: node.EventBus(),
		config:   config.DefaultRPCConfig(),
		node:     node,
	}
}

// ABCIInfo returns basic information about application state.
func (c *Client) ABCIInfo(ctx context.Context) (*ctypes.ResultABCIInfo, error) {
	resInfo, err := c.Query().InfoSync(proxy.RequestInfo)
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultABCIInfo{Response: *resInfo}, nil
}

// ABCIQuery queries for data from application.
func (c *Client) ABCIQuery(ctx context.Context, path string, data tmbytes.HexBytes) (*ctypes.ResultABCIQuery, error) {
	return c.ABCIQueryWithOptions(ctx, path, data, rpcclient.DefaultABCIQueryOptions)
}

// ABCIQueryWithOptions queries for data from application.
func (c *Client) ABCIQueryWithOptions(ctx context.Context, path string, data tmbytes.HexBytes, opts rpcclient.ABCIQueryOptions) (*ctypes.ResultABCIQuery, error) {
	resQuery, err := c.Query().QuerySync(abci.RequestQuery{
		Path:   path,
		Data:   data,
		Height: opts.Height,
		Prove:  opts.Prove,
	})
	if err != nil {
		return nil, err
	}
	c.Logger.Debug("ABCIQuery", "path", path, "height", resQuery.Height)
	return &ctypes.ResultABCIQuery{Response: *resQuery}, nil
}

// BroadcastTxCommit returns with the responses from CheckTx and DeliverTx.
// More: https://docs.tendermint.com/master/rpc/#/Tx/broadcast_tx_commit
func (c *Client) BroadcastTxCommit(ctx context.Context, tx tmtypes.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	// This implementation corresponds to Tendermints implementation from rpc/core/mempool.go.
	// ctx.RemoteAddr godoc: If neither HTTPReq nor WSConn is set, an empty string is returned.
	// This code is a local client, so we can assume that subscriber is ""
	subscriber := "" // ctx.RemoteAddr()

	if err := c.IsSubscriptionAllowed(subscriber); err != nil {
		return nil, sdkerrors.Wrap(err, "subscription not allowed")
	}

	// Subscribe to tx being committed in block.
	subCtx, cancel := context.WithTimeout(ctx, subscribeTimeout)
	defer cancel()
	q := tmtypes.EventQueryTxFor(tx)
	deliverTxSub, err := c.EventBus.Subscribe(subCtx, subscriber, q)
	if err != nil {
		err = fmt.Errorf("subscribe to tx: %w", err)
		c.Logger.Error("on broadcast_tx_commit", "err", err)
		return nil, err
	}
	defer func() {
		if err := c.EventBus.Unsubscribe(context.Background(), subscriber, q); err != nil {
			c.Logger.Error("unsubscribing from eventBus", "err", err)
		}
	}()

	// add to mempool and wait for CheckTx result
	checkTxResCh := make(chan *abci.Response, 1)
	err = c.node.Mempool.CheckTx(tx, func(res *abci.Response) {
		select {
		case <-ctx.Done():
		case checkTxResCh <- res:
		}
	}, mempool.TxInfo{})
	if err != nil {
		c.Logger.Error("on broadcastTxCommit", "err", err)
		return nil, fmt.Errorf("on broadcastTxCommit: %w", err)
	}
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("broadcast confirmation not received: %w", ctx.Err())
	case checkTxResMsg := <-checkTxResCh:
		checkTxRes := checkTxResMsg.GetCheckTx()
		if checkTxRes.Code != abci.CodeTypeOK {
			return &ctypes.ResultBroadcastTxCommit{
				CheckTx:   *checkTxRes,
				DeliverTx: abci.ResponseDeliverTx{},
				Hash:      tx.Hash(),
			}, nil
		}

		// broadcast tx
		err = c.node.P2P.GossipTx(ctx, tx)
		if err != nil {
			return nil, fmt.Errorf("tx added to local mempool but failure to broadcast: %w", err)
		}

		// Wait for the tx to be included in a block or timeout.
		select {
		case msg := <-deliverTxSub.Out(): // The tx was included in a block.
			deliverTxRes, _ := msg.Data().(tmtypes.EventDataTx)
			return &ctypes.ResultBroadcastTxCommit{
				CheckTx:   *checkTxRes,
				DeliverTx: deliverTxRes.Result,
				Hash:      tx.Hash(),
				Height:    deliverTxRes.Height,
			}, nil
		case <-deliverTxSub.Cancelled():
			var reason string
			if deliverTxSub.Err() == nil {
				reason = "Dymint exited"
			} else {
				reason = deliverTxSub.Err().Error()
			}
			err = fmt.Errorf("deliverTxSub was cancelled (reason: %s)", reason)
			c.Logger.Error("on broadcastTxCommit", "err", err)
			return &ctypes.ResultBroadcastTxCommit{
				CheckTx:   *checkTxRes,
				DeliverTx: abci.ResponseDeliverTx{},
				Hash:      tx.Hash(),
			}, err
		case <-time.After(c.config.TimeoutBroadcastTxCommit):
			err = errors.New("timed out waiting for tx to be included in a block")
			c.Logger.Error("on broadcastTxCommit", "err", err)
			return &ctypes.ResultBroadcastTxCommit{
				CheckTx:   *checkTxRes,
				DeliverTx: abci.ResponseDeliverTx{},
				Hash:      tx.Hash(),
			}, err
		}
	}
}

// BroadcastTxAsync returns right away, with no response. Does not wait for
// CheckTx nor DeliverTx results.
// More: https://docs.tendermint.com/master/rpc/#/Tx/broadcast_tx_async
func (c *Client) BroadcastTxAsync(ctx context.Context, tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	err := c.node.Mempool.CheckTx(tx, nil, mempool.TxInfo{})
	if err != nil {
		return nil, err
	}
	// gossipTx optimistically
	err = c.node.P2P.GossipTx(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("tx added to local mempool but failed to gossip: %w", err)
	}
	return &ctypes.ResultBroadcastTx{Hash: tx.Hash()}, nil
}

// BroadcastTxSync returns with the response from CheckTx. Does not wait for
// DeliverTx result.
// More: https://docs.tendermint.com/master/rpc/#/Tx/broadcast_tx_sync
func (c *Client) BroadcastTxSync(ctx context.Context, tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	resCh := make(chan *abci.Response, 1)
	err := c.node.Mempool.CheckTx(tx, func(res *abci.Response) {
		resCh <- res
	}, mempool.TxInfo{})
	if err != nil {
		return nil, err
	}
	res := <-resCh
	r := res.GetCheckTx()

	// gossip the transaction if it's in the mempool.
	// Note: we have to do this here because, unlike the tendermint mempool reactor, there
	// is no routine that gossips transactions after they enter the pool
	if r.Code == abci.CodeTypeOK {
		err = c.node.P2P.GossipTx(ctx, tx)
		if err != nil {
			// the transaction must be removed from the mempool if it cannot be gossiped.
			// if this does not occur, then the user will not be able to try again using
			// this node, as the CheckTx call above will return an error indicating that
			// the tx is already in the mempool
			_ = c.node.Mempool.RemoveTxByKey(tx.Key())
			return nil, fmt.Errorf("gossip tx: %w", err)
		}
	}

	return &ctypes.ResultBroadcastTx{
		Code:      r.Code,
		Data:      r.Data,
		Log:       r.Log,
		Codespace: r.Codespace,
		Hash:      tx.Hash(),
	}, nil
}

// Subscribe subscribe given subscriber to a query.
func (c *Client) Subscribe(ctx context.Context, subscriber, query string, outCapacity ...int) (out <-chan ctypes.ResultEvent, err error) {
	q, err := tmquery.New(query)
	if err != nil {
		return nil, fmt.Errorf("parse query: %w", err)
	}

	outCap := 1
	if len(outCapacity) > 0 {
		outCap = outCapacity[0]
	}

	var sub tmtypes.Subscription
	if outCap > 0 {
		sub, err = c.EventBus.Subscribe(ctx, subscriber, q, outCap)
	} else {
		sub, err = c.SubscribeUnbuffered(ctx, subscriber, q)
	}
	if err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}

	outc := make(chan ctypes.ResultEvent, outCap)
	go c.eventsRoutine(sub, subscriber, q, outc)

	return outc, nil
}

// Unsubscribe unsubscribes given subscriber from a query.
func (c *Client) Unsubscribe(ctx context.Context, subscriber, query string) error {
	q, err := tmquery.New(query)
	if err != nil {
		return fmt.Errorf("parse query: %w", err)
	}
	return c.EventBus.Unsubscribe(ctx, subscriber, q)
}

// Genesis returns entire genesis.
func (c *Client) Genesis(_ context.Context) (*ctypes.ResultGenesis, error) {
	return &ctypes.ResultGenesis{Genesis: c.node.GetGenesis()}, nil
}

// GenesisChunked returns given chunk of genesis.
func (c *Client) GenesisChunked(_ context.Context, id uint) (*ctypes.ResultGenesisChunk, error) {
	genChunks, err := c.GetGenesisChunks()
	if err != nil {
		return nil, fmt.Errorf("while creating chunks of the genesis document: %w", err)
	}
	if genChunks == nil {
		return nil, fmt.Errorf("service configuration error, genesis chunks are not initialized")
	}

	chunkLen := len(genChunks)
	if chunkLen == 0 {
		return nil, fmt.Errorf("service configuration error, there are no chunks")
	}

	// it's safe to do uint(chunkLen)-1 (no overflow) since we always have at least one chunk here
	if id > uint(chunkLen)-1 {
		return nil, fmt.Errorf("there are %d chunks, %d is invalid", chunkLen-1, id)
	}

	return &ctypes.ResultGenesisChunk{
		TotalChunks: chunkLen,
		ChunkNumber: int(id), //nolint:gosec // id is always positive
		Data:        genChunks[id],
	}, nil
}

// BlockchainInfo gets block headers for minHeight <= height <= maxHeight.
//
// If maxHeight does not yet exist, blocks up to the current height will be
// returned. If minHeight does not exist (due to pruning), earliest existing
// height will be used.
//
// At most 20 items will be returned. Block headers are returned in descending
// order (highest first).
//
// More: https://docs.cometbft.com/v0.34/rpc/#/Info/blockchain
//
// See https://github.com/dymensionxyz/cometbft/blob/a059b062dcfc719406354e0a80f5f6d3cf7401e1/rpc/core/blocks.go#L26
// Note: used at least by self relay typescript client
func (c *Client) BlockchainInfo(ctx context.Context, minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	baseHeight, err := c.node.BlockManager.Store.LoadBaseHeight()
	if err != nil && !errors.Is(err, gerrc.ErrNotFound) {
		return nil, err
	}
	if errors.Is(err, gerrc.ErrNotFound) {
		baseHeight = 1
	}
	bmHeight := int64(c.node.GetBlockManagerHeight()) //nolint:gosec // height is non-negative and falls in int64

	minHeight, maxHeight, err = filterMinMax(
		int64(baseHeight), //nolint:gosec // height is non-negative and falls in int64
		bmHeight,
		minHeight,
		maxHeight,
		20, // hardcoded, return max 20 blocks
	)
	if err != nil {
		return nil, err
	}
	c.Logger.Debug("BlockchainInfo", "maxHeight", maxHeight, "minHeight", minHeight)

	blocks := make([]*tmtypes.BlockMeta, 0, maxHeight-minHeight+1)
	for height := maxHeight; height >= minHeight; height-- {
		block, err := c.node.Store.LoadBlock(uint64(height)) //nolint:gosec // height is non-negative and falls in int64
		if err != nil {
			return nil, err
		}
		if block != nil {
			tmblockmeta, err := types.ToABCIBlockMeta(block)
			if err != nil {
				return nil, err
			}
			blocks = append(blocks, tmblockmeta)
		}
	}

	return &ctypes.ResultBlockchainInfo{
		LastHeight: int64(c.node.GetBlockManagerHeight()), //nolint:gosec // height is non-negative and falls in int64
		BlockMetas: blocks,
	}, nil
}

// NetInfo returns basic information about client P2P connections.
func (c *Client) NetInfo(ctx context.Context) (*ctypes.ResultNetInfo, error) {
	res := ctypes.ResultNetInfo{
		Listening: true,
	}
	for _, ma := range c.node.P2P.Addrs() {
		res.Listeners = append(res.Listeners, ma.String())
	}
	peers := c.node.P2P.Peers()
	res.NPeers = len(peers)
	for _, peer := range peers {
		res.Peers = append(res.Peers, ctypes.Peer{
			NodeInfo:         peer.NodeInfo,
			IsOutbound:       peer.IsOutbound,
			ConnectionStatus: peer.ConnectionStatus,
			RemoteIP:         peer.RemoteIP,
		})
	}

	return &res, nil
}

// DumpConsensusState always returns error as there is no consensus state in Dymint.
func (c *Client) DumpConsensusState(ctx context.Context) (*ctypes.ResultDumpConsensusState, error) {
	return nil, ErrConsensusStateNotAvailable
}

// ConsensusState always returns error as there is no consensus state in Dymint.
func (c *Client) ConsensusState(ctx context.Context) (*ctypes.ResultConsensusState, error) {
	return nil, ErrConsensusStateNotAvailable
}

// ConsensusParams returns consensus params at given height.
//
// Currently, consensus params changes are not supported and this method returns params as defined in genesis.
func (c *Client) ConsensusParams(ctx context.Context, height *int64) (*ctypes.ResultConsensusParams, error) {
	// TODO(tzdybal): implement consensus params handling: https://github.com/dymensionxyz/dymint/issues/291
	params := c.node.GetGenesis().ConsensusParams
	return &ctypes.ResultConsensusParams{
		BlockHeight: int64(c.normalizeHeight(height)), //nolint:gosec // height is non-negative and falls in int64
		ConsensusParams: tmproto.ConsensusParams{
			Block: tmproto.BlockParams{
				MaxBytes:   params.Block.MaxBytes,
				MaxGas:     params.Block.MaxGas,
				TimeIotaMs: params.Block.TimeIotaMs,
			},
			Evidence: tmproto.EvidenceParams{
				MaxAgeNumBlocks: params.Evidence.MaxAgeNumBlocks,
				MaxAgeDuration:  params.Evidence.MaxAgeDuration,
				MaxBytes:        params.Evidence.MaxBytes,
			},
			Validator: tmproto.ValidatorParams{
				PubKeyTypes: params.Validator.PubKeyTypes,
			},
			Version: tmproto.VersionParams{
				AppVersion: params.Version.AppVersion,
			},
		},
	}, nil
}

// Health endpoint returns empty value. It can be used to monitor service availability.
func (c *Client) Health(ctx context.Context) (*ctypes.ResultHealth, error) {
	return &ctypes.ResultHealth{}, nil
}

// Block method returns BlockID and block itself for given height.
//
// If height is nil, it returns information about last known block.
func (c *Client) Block(ctx context.Context, height *int64) (*ctypes.ResultBlock, error) {
	heightValue := c.normalizeHeight(height)
	block, err := c.node.Store.LoadBlock(heightValue)
	if err != nil {
		return nil, err
	}
	hash := block.Hash()
	abciBlock, err := types.ToABCIBlock(block)
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultBlock{
		BlockID: tmtypes.BlockID{
			Hash: hash[:],
			PartSetHeader: tmtypes.PartSetHeader{
				Total: 1,
				Hash:  hash[:],
			},
		},
		Block: abciBlock,
	}, nil
}

// BlockByHash returns BlockID and block itself for given hash.
func (c *Client) BlockByHash(ctx context.Context, hash []byte) (*ctypes.ResultBlock, error) {
	var h [32]byte
	copy(h[:], hash)

	block, err := c.node.Store.LoadBlockByHash(h)
	if err != nil {
		return nil, err
	}

	abciBlock, err := types.ToABCIBlock(block)
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultBlock{
		BlockID: tmtypes.BlockID{
			Hash: h[:],
			PartSetHeader: tmtypes.PartSetHeader{
				Total: 1,
				Hash:  h[:],
			},
		},
		Block: abciBlock,
	}, nil
}

// BlockResults returns information about transactions, events and updates of validator set and consensus params.
func (c *Client) BlockResults(ctx context.Context, height *int64) (*ctypes.ResultBlockResults, error) {
	var h uint64
	if height == nil {
		h = c.node.GetBlockManagerHeight()
	} else {
		h = uint64(*height) //nolint:gosec // height is non-negative and falls in int64
	}
	resp, err := c.node.Store.LoadBlockResponses(h)
	if err != nil {
		return nil, err
	}

	return &ctypes.ResultBlockResults{
		Height:                int64(h), //nolint:gosec // height is non-negative and falls in int64
		TxsResults:            resp.DeliverTxs,
		BeginBlockEvents:      resp.BeginBlock.Events,
		EndBlockEvents:        resp.EndBlock.Events,
		ValidatorUpdates:      resp.EndBlock.ValidatorUpdates,
		ConsensusParamUpdates: resp.EndBlock.ConsensusParamUpdates,
	}, nil
}

// Commit returns signed header (aka commit) at given height.
func (c *Client) Commit(ctx context.Context, height *int64) (*ctypes.ResultCommit, error) {
	heightValue := c.normalizeHeight(height)
	com, err := c.node.Store.LoadCommit(heightValue)
	if err != nil {
		return nil, err
	}
	b, err := c.node.Store.LoadBlock(heightValue)
	if err != nil {
		return nil, err
	}
	commit := types.ToABCICommit(com)
	block, err := types.ToABCIBlock(b)
	if err != nil {
		return nil, err
	}

	return ctypes.NewResultCommit(&block.Header, commit, true), nil
}

// Validators returns paginated list of validators at given height.
func (c *Client) Validators(ctx context.Context, heightPtr *int64, _, _ *int) (*ctypes.ResultValidators, error) {
	height := c.normalizeHeight(heightPtr)

	proposer, err := c.node.Store.LoadProposer(height)
	if err != nil {
		return nil, fmt.Errorf("load validators: height %d: %w", height, err)
	}

	return &ctypes.ResultValidators{
		BlockHeight: int64(height), //nolint:gosec // height is non-negative and falls in int64
		Validators:  proposer.TMValidators(),
		Count:       1,
		Total:       1,
	}, nil
}

// Tx returns detailed information about transaction identified by its hash.
func (c *Client) Tx(ctx context.Context, hash []byte, prove bool) (*ctypes.ResultTx, error) {
	res, err := c.node.TxIndexer.Get(hash)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("tx (%X) not found", hash)
	}

	height := res.Height
	index := res.Index

	var proof tmtypes.TxProof
	if prove {
		block, _ := c.node.Store.LoadBlock(uint64(height)) //nolint:gosec // height is non-negative and falls in int64
		blockProof := block.Data.Txs.Proof(int(index))     // XXX: overflow on 32-bit machines
		proof = tmtypes.TxProof{
			RootHash: blockProof.RootHash,
			Data:     tmtypes.Tx(blockProof.Data),
			Proof:    blockProof.Proof,
		}
	}

	return &ctypes.ResultTx{
		Hash:     hash,
		Height:   height,
		Index:    index,
		TxResult: res.Result,
		Tx:       res.Tx,
		Proof:    proof,
	}, nil
}

// TxSearch returns detailed information about transactions matching query.
func (c *Client) TxSearch(ctx context.Context, query string, prove bool, pagePtr, perPagePtr *int, orderBy string) (*ctypes.ResultTxSearch, error) {
	q, err := tmquery.New(query)
	if err != nil {
		return nil, err
	}

	results, err := c.node.TxIndexer.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	// sort results (must be done before pagination)
	switch orderBy {
	case "desc":
		sort.Slice(results, func(i, j int) bool {
			if results[i].Height == results[j].Height {
				return results[i].Index > results[j].Index
			}
			return results[i].Height > results[j].Height
		})
	case "asc", "":
		sort.Slice(results, func(i, j int) bool {
			if results[i].Height == results[j].Height {
				return results[i].Index < results[j].Index
			}
			return results[i].Height < results[j].Height
		})
	default:
		return nil, errors.New("expected order_by to be either `asc` or `desc` or empty")
	}

	// paginate results
	totalCount := len(results)
	perPage := validatePerPage(perPagePtr)

	page, err := validatePage(pagePtr, perPage, totalCount)
	if err != nil {
		return nil, err
	}

	skipCount := validateSkipCount(page, perPage)
	pageSize := tmmath.MinInt(perPage, totalCount-skipCount)

	apiResults := make([]*ctypes.ResultTx, 0, pageSize)
	for i := skipCount; i < skipCount+pageSize; i++ {
		r := results[i]

		var proof tmtypes.TxProof
		/*if prove {
			block := nil                               //env.BlockStore.LoadBlock(r.Height)
			proof = block.Data.Txs.Proof(int(r.Index)) // XXX: overflow on 32-bit machines
		}*/

		apiResults = append(apiResults, &ctypes.ResultTx{
			Hash:     tmtypes.Tx(r.Tx).Hash(),
			Height:   r.Height,
			Index:    r.Index,
			TxResult: r.Result,
			Tx:       r.Tx,
			Proof:    proof,
		})
	}

	return &ctypes.ResultTxSearch{Txs: apiResults, TotalCount: totalCount}, nil
}

// BlockSearch defines a method to search for a paginated set of blocks by
// BeginBlock and EndBlock event search criteria.
func (c *Client) BlockSearch(ctx context.Context, query string, page, perPage *int, orderBy string) (*ctypes.ResultBlockSearch, error) {
	q, err := tmquery.New(query)
	if err != nil {
		return nil, err
	}

	results, err := c.node.BlockIndexer.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	// Sort the results
	switch orderBy {
	case "desc":
		sort.Slice(results, func(i, j int) bool {
			return results[i] > results[j]
		})

	case "asc", "":
		sort.Slice(results, func(i, j int) bool {
			return results[i] < results[j]
		})
	default:
		return nil, errors.New("expected order_by to be either `asc` or `desc` or empty")
	}

	// Paginate
	totalCount := len(results)
	perPageVal := validatePerPage(perPage)

	pageVal, err := validatePage(page, perPageVal, totalCount)
	if err != nil {
		return nil, err
	}

	skipCount := validateSkipCount(pageVal, perPageVal)
	pageSize := tmmath.MinInt(perPageVal, totalCount-skipCount)

	// Fetch the blocks
	blocks := make([]*ctypes.ResultBlock, 0, pageSize)
	for i := skipCount; i < skipCount+pageSize; i++ {
		b, err := c.node.Store.LoadBlock(uint64(results[i])) //nolint:gosec // height is non-negative and falls in int64
		if err != nil {
			return nil, err
		}
		block, err := types.ToABCIBlock(b)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, &ctypes.ResultBlock{
			Block: block,
			BlockID: tmtypes.BlockID{
				Hash: block.Hash(),
			},
		})
	}

	return &ctypes.ResultBlockSearch{Blocks: blocks, TotalCount: totalCount}, nil
}

// Status returns detailed information about current status of the node.
func (c *Client) Status(_ context.Context) (*ctypes.ResultStatus, error) {
	latest, err := c.node.Store.LoadBlock(c.node.GetBlockManagerHeight())
	if err != nil {
		// TODO(tzdybal): extract error
		return nil, fmt.Errorf("find latest block: %w", err)
	}

	latestBlockHash := latest.Header.DataHash
	latestAppHash := latest.Header.AppHash
	latestHeight := latest.Header.Height
	latestBlockTime := latest.Header.GetTimestamp()

	proposer, err := c.node.Store.LoadProposer(latest.Header.Height)
	if err != nil {
		return nil, fmt.Errorf("fetch the validator info at latest block: %w", err)
	}
	state, err := c.node.Store.LoadState()
	if err != nil {
		return nil, fmt.Errorf("load the last saved state: %w", err)
	}
	defaultProtocolVersion := p2p.NewProtocolVersion(
		tm_version.P2PProtocol,
		state.Version.Consensus.Block,
		state.Version.Consensus.App,
	)
	id, addr, network := c.node.P2P.Info()
	txIndexerStatus := "on"

	result := &ctypes.ResultStatus{
		// TODO(ItzhakBokris): update NodeInfo fields
		NodeInfo: p2p.DefaultNodeInfo{
			ProtocolVersion: defaultProtocolVersion,
			DefaultNodeID:   id,
			ListenAddr:      addr,
			Network:         network,
			Version:         version.Build,
			Channels:        []byte{0x1},
			Moniker:         config.DefaultBaseConfig().Moniker,
			Other: p2p.DefaultNodeInfoOther{
				TxIndex:    txIndexerStatus,
				RPCAddress: c.config.ListenAddress,
			},
		},
		SyncInfo: ctypes.SyncInfo{
			LatestBlockHash:   latestBlockHash[:],
			LatestAppHash:     latestAppHash[:],
			LatestBlockHeight: int64(latestHeight), //nolint:gosec // height is non-negative and falls in int64
			LatestBlockTime:   latestBlockTime,
			// CatchingUp is true if the node is not at the latest height received from p2p or da.
			CatchingUp: c.node.BlockManager.TargetHeight.Load() > latestHeight,
			// TODO(tzdybal): add missing fields
			// EarliestBlockHash:   earliestBlockHash,
			// EarliestAppHash:     earliestAppHash,
			// EarliestBlockHeight: earliestBloc
			// kHeight,
			// EarliestBlockTime:   time.Unix(0, earliestBlockTimeNano),
		},
		// TODO(ItzhakBokris): update ValidatorInfo fields
		ValidatorInfo: ctypes.ValidatorInfo{
			Address:     tmbytes.HexBytes(proposer.ConsAddress()),
			PubKey:      proposer.PubKey(),
			VotingPower: 1,
		},
		DymensionStatus: ctypes.DymensionStatus{
			DAPath:        c.node.BlockManager.GetActiveDAClient().RollappId(),
			RollappParams: types.RollappParamsToABCI(state.RollappParams),
		},
	}
	return result, nil
}

type DAInfo struct{}

// BroadcastEvidence is not yet implemented.
func (c *Client) BroadcastEvidence(ctx context.Context, evidence tmtypes.Evidence) (*ctypes.ResultBroadcastEvidence, error) {
	return &ctypes.ResultBroadcastEvidence{
		Hash: evidence.Hash(),
	}, nil
}

// NumUnconfirmedTxs returns information about transactions in mempool.
func (c *Client) NumUnconfirmedTxs(ctx context.Context) (*ctypes.ResultUnconfirmedTxs, error) {
	return &ctypes.ResultUnconfirmedTxs{
		Count:      c.node.Mempool.Size(),
		Total:      c.node.Mempool.Size(),
		TotalBytes: c.node.Mempool.SizeBytes(),
	}, nil
}

// UnconfirmedTxs returns transactions in mempool.
func (c *Client) UnconfirmedTxs(ctx context.Context, limitPtr *int) (*ctypes.ResultUnconfirmedTxs, error) {
	// reuse per_page validator
	limit := validatePerPage(limitPtr)

	txs := c.node.Mempool.ReapMaxTxs(limit)
	return &ctypes.ResultUnconfirmedTxs{
		Count:      len(txs),
		Total:      c.node.Mempool.Size(),
		TotalBytes: c.node.Mempool.SizeBytes(),
		Txs:        txs,
	}, nil
}

// CheckTx executes a new transaction against the application to determine its validity.
//
// If valid, the tx is automatically added to the mempool.
func (c *Client) CheckTx(ctx context.Context, tx tmtypes.Tx) (*ctypes.ResultCheckTx, error) {
	res, err := c.Mempool().CheckTxSync(abci.RequestCheckTx{Tx: tx})
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultCheckTx{ResponseCheckTx: *res}, nil
}

func (c *Client) ChainID() (*ResultEthMethod, error) {
	eip155ChainID, err := evmostypes.ParseChainID(c.node.BlockManager.State.ChainID)
	if err != nil {
		return &ResultEthMethod{}, nil
	}
	return &ResultEthMethod{Result: fmt.Sprintf("0x%x", eip155ChainID)}, nil
}

func (c *Client) BlockValidated(height *int64) (*ResultBlockValidated, error) {
	_, _, chainID := c.node.P2P.Info()
	// invalid height
	if height == nil || *height < 0 {
		return &ResultBlockValidated{Result: -1, ChainID: chainID}, nil
	}
	// node has not reached the height yet
	if uint64(*height) > c.node.BlockManager.State.Height() { //nolint:gosec // height is non-negative and falls in int64
		return &ResultBlockValidated{Result: NotValidated, ChainID: chainID}, nil
	}

	if uint64(*height) <= c.node.BlockManager.SettlementValidator.GetLastValidatedHeight() { //nolint:gosec // height is non-negative and falls in int64
		return &ResultBlockValidated{Result: SLValidated, ChainID: chainID}, nil
	}

	// block is applied, and therefore it is validated at block level but not at state update level
	return &ResultBlockValidated{Result: P2PValidated, ChainID: chainID}, nil
}

func (c *Client) eventsRoutine(sub tmtypes.Subscription, subscriber string, q tmpubsub.Query, outc chan<- ctypes.ResultEvent) {
	defer close(outc)
	for {
		select {
		case msg := <-sub.Out():
			result := ctypes.ResultEvent{Query: q.String(), Data: msg.Data(), Events: msg.Events()}
			if cap(outc) == 0 {
				outc <- result
			} else {
				select {
				case outc <- result:
				default:
					c.Logger.Error("wanted to publish ResultEvent, but out channel is full", "result", result, "query", result.Query)
				}
			}
		case <-sub.Cancelled():
			if errors.Is(sub.Err(), tmpubsub.ErrUnsubscribed) {
				return
			}

			c.Logger.Error("subscription was cancelled, resubscribing...", "err", sub.Err(), "query", q.String())
			sub = c.resubscribe(subscriber, q)
			if sub == nil { // client was stopped
				return
			}
		case <-c.Quit():
			return
		}
	}
}

// Try to resubscribe with exponential backoff.
func (c *Client) resubscribe(subscriber string, q tmpubsub.Query) tmtypes.Subscription {
	attempts := uint(0)
	for {
		if !c.IsRunning() {
			return nil
		}

		sub, err := c.EventBus.Subscribe(context.Background(), subscriber, q)
		if err == nil {
			return sub
		}

		attempts++
		time.Sleep((10 << attempts) * time.Millisecond) // 10ms -> 20ms -> 40ms
	}
}

func (c *Client) Consensus() proxy.AppConnConsensus {
	return c.node.ProxyApp().Consensus()
}

func (c *Client) Mempool() proxy.AppConnMempool {
	return c.node.ProxyApp().Mempool()
}

func (c *Client) Query() proxy.AppConnQuery {
	return c.node.ProxyApp().Query()
}

func (c *Client) Snapshot() proxy.AppConnSnapshot {
	return c.node.ProxyApp().Snapshot()
}

func (c *Client) normalizeHeight(height *int64) uint64 {
	var heightValue uint64
	if height == nil || *height == 0 {
		heightValue = c.node.GetBlockManagerHeight()
	} else {
		heightValue = uint64(*height) //nolint:gosec // height is non-negative and falls in int64
	}

	return heightValue
}

func (c *Client) IsSubscriptionAllowed(subscriber string) error {
	if c.NumClients() >= c.config.MaxSubscriptionClients {
		return fmt.Errorf("max_subscription_clients %d reached", c.config.MaxSubscriptionClients)
	} else if c.NumClientSubscriptions(subscriber) >= c.config.MaxSubscriptionsPerClient {
		return fmt.Errorf("max_subscriptions_per_client %d reached", c.config.MaxSubscriptionsPerClient)
	}

	return nil
}

func validatePerPage(perPagePtr *int) int {
	if perPagePtr == nil { // no per_page parameter
		return defaultPerPage
	}

	perPage := *perPagePtr
	if perPage < 1 {
		return defaultPerPage
	} else if perPage > maxPerPage {
		return maxPerPage
	}
	return perPage
}

func validatePage(pagePtr *int, perPage, totalCount int) (int, error) {
	if perPage < 1 {
		panic(fmt.Sprintf("zero or negative perPage: %d", perPage))
	}

	if pagePtr == nil || *pagePtr <= 0 { // no page parameter
		return 1, nil
	}

	pages := ((totalCount - 1) / perPage) + 1
	if pages == 0 {
		pages = 1 // one page (even if it's empty)
	}
	page := *pagePtr
	if page > pages {
		return 1, fmt.Errorf("page should be within [1, %d] range, given %d", pages, page)
	}

	return page, nil
}

func validateSkipCount(page, perPage int) int {
	skipCount := (page - 1) * perPage
	if skipCount < 0 {
		return 0
	}

	return skipCount
}

func filterMinMax(base, height, min, max, limit int64) (int64, int64, error) {
	if height < min {
		return 0, 0, fmt.Errorf("height must be greater than min: height %d, min: %d", height, min)
	}

	if max < base {
		return 0, 0, fmt.Errorf("max already pruned: base %d, max: %d", base, max)
	}

	// filter negatives
	if min < 0 || max < 0 {
		return min, max, errors.New("height must be greater than zero")
	}

	// adjust for default values
	if min == 0 {
		min = 1
	}
	if max == 0 {
		max = height
	}

	// limit max to the height
	max = tmmath.MaxInt64(base, tmmath.MinInt64(height, max))

	// limit min to the base
	min = tmmath.MaxInt64(base, min)

	// limit min to within `limit` of max
	// so the total number of blocks returned will be `limit`
	min = tmmath.MaxInt64(min, max-limit+1)

	if min > max {
		return min, max, fmt.Errorf("%w: min height %d can't be greater than max height %d",
			errors.New("invalid request"), min, max)
	}
	return min, max, nil
}

// EthBlockNumber returns the height of the block manager.
func (c *Client) EthBlockNumber(ctx context.Context) (*ResultEthMethod, error) {
	return &ResultEthMethod{Result: fmt.Sprintf("0x%x", c.node.GetBlockManagerHeight())}, nil
}

// EthGetBalance returns the block for the height provided.
func (c *Client) EthGetBlockByNumber(ctx context.Context, height *int64) (*ResultEthMethod, error) {
	resBlock, err := c.Block(ctx, height)
	// return if requested block height is greater than the current one
	if resBlock == nil || resBlock.Block == nil || err != nil {
		return nil, fmt.Errorf("failed to fetch block result from Tendermint. height:%d", height)
	}

	blockResults, err := c.BlockResults(ctx, height)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block result from Tendermint. height:%d error:%s", height, err.Error())
	}

	ethFormatBlock, err := c.RPCBlockFromTendermintBlock(resBlock, blockResults)
	if err != nil {
		return nil, err
	}
	result, err := json.Marshal(ethFormatBlock)
	if err != nil {
		return nil, err
	}
	return &ResultEthMethod{Result: string(result)}, nil
}

// EthGetBalance returns the provided account's balance (for the native denom) up to the provided block number.
func (c *Client) EthGetBalance(ctx context.Context, address string, height *int64) (*ResultEthMethod, error) {
	nativeDenom, err := c.getNativeDenom(ctx)
	if err != nil {
		return nil, err
	}
	if nativeDenom == "" {
		return nil, fmt.Errorf("no native denom found")
	}

	bz, err := hex.DecodeString(address)
	if err != nil {
		return nil, err
	}

	bech32Addr, err := bech32.ConvertAndEncode(sdktypes.GetConfig().GetBech32AccountAddrPrefix(), bz)
	if err != nil {
		return nil, err
	}

	req := &banktypes.QueryBalanceRequest{
		Address: bech32Addr,
		Denom:   nativeDenom,
	}
	data, err := req.Marshal()
	if err != nil {
		return nil, err
	}
	res, err := c.ABCIQueryWithOptions(ctx, "/cosmos.bank.v1beta1.Query/Balance", data, rpcclient.ABCIQueryOptions{
		Height: *height,
	})
	if err != nil {
		return nil, err
	}
	respBalance := &banktypes.QueryBalanceResponse{}
	err = respBalance.Unmarshal(res.Response.Value)
	if err != nil {
		return nil, err
	}

	return &ResultEthMethod{Result: hexutil.EncodeBig(respBalance.Balance.Amount.BigInt())}, nil
}

// RPCBlockFromTendermintBlock returns a JSON-RPC compatible Ethereum block from a
// given Tendermint block and its block result.
// Only basic header information is attached for metamask compatibility.
// It does not include txs info or base fees.
func (c *Client) RPCBlockFromTendermintBlock(
	resBlock *ctypes.ResultBlock,
	blockRes *ctypes.ResultBlockResults,
) (map[string]interface{}, error) {
	ethRPCTxs := []interface{}{}
	block := resBlock.Block

	formattedBlock := evmosrpctypes.FormatBlock(
		block.Header, block.Size(),
		c.node.BlockManager.State.ConsensusParams.Block.MaxGas, nil,
		ethRPCTxs, ethtypes.Bloom{}, ethctypes.BytesToAddress(block.ProposerAddress.Bytes()), nil,
	)
	return formattedBlock, nil
}

func (c *Client) getNativeDenom(ctx context.Context) (string, error) {
	mintDenom, err := c.getMintDenom(ctx)
	if err != nil {
		return "", err
	}
	if mintDenom != "" {
		return mintDenom, nil
	}

	return c.getStakingDenom(ctx)
}

func (c *Client) getMintDenom(ctx context.Context) (string, error) {
	reqDenom := &minttypes.QueryParamsRequest{}
	dataDenom, err := reqDenom.Marshal()
	if err != nil {
		return "", err
	}
	resValue, err := c.ABCIQueryWithOptions(ctx, "/rollapp.mint.v1beta1.Query/Params", dataDenom, rpcclient.DefaultABCIQueryOptions)
	if err != nil {
		return "", err
	}
	respDenom := &minttypes.QueryParamsResponse{}

	err = respDenom.Unmarshal(resValue.Response.Value)
	if err != nil {
		return "", err
	}
	return respDenom.MintDenom, nil
}

func (c *Client) getStakingDenom(ctx context.Context) (string, error) {
	reqStDenom := &stakingtypes.QueryParamsRequest{}
	dataDenom, err := reqStDenom.Marshal()
	if err != nil {
		return "", err
	}
	resValue, err := c.ABCIQueryWithOptions(ctx, "/cosmos.staking.v1beta1.Query/Params", dataDenom, rpcclient.DefaultABCIQueryOptions)
	if err != nil {
		return "", err
	}
	respStDenom := &stakingtypes.QueryParamsResponse{}

	err = respStDenom.Unmarshal(resValue.Response.Value)
	if err != nil {
		return "", err
	}
	return respStDenom.Params.BondDenom, nil
}

func (c *Client) Tee(ctx context.Context, dry bool) (*teetypes.TEEResponse, error) {
	ret, err := tee.GetToken(c.node, dry)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}
