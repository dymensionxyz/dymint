package client

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	sdkerrors "cosmossdk.io/errors"
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

	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/dymensionxyz/dymint/mempool"
	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/version"
)

const (
	defaultPerPage = 30
	maxPerPage     = 100

	subscribeTimeout = 5 * time.Second
)

type BlockValidationStatus int

const (
	NotValidated = iota
	P2PValidated
	SLValidated
)

var ErrConsensusStateNotAvailable = errors.New("consensus state not available in Dymint")

var _ rpcclient.Client = &Client{}

type Client struct {
	*tmtypes.EventBus
	config *config.RPCConfig
	node   *node.Node

	genChunks []string
}

type ResultBlockValidated struct {
	ChainID string
	Result  BlockValidationStatus
}

func NewClient(node *node.Node) *Client {
	return &Client{
		EventBus: node.EventBus(),
		config:   config.DefaultRPCConfig(),
		node:     node,
	}
}

func (c *Client) ABCIInfo(ctx context.Context) (*ctypes.ResultABCIInfo, error) {
	resInfo, err := c.Query().InfoSync(proxy.RequestInfo)
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultABCIInfo{Response: *resInfo}, nil
}

func (c *Client) ABCIQuery(ctx context.Context, path string, data tmbytes.HexBytes) (*ctypes.ResultABCIQuery, error) {
	return c.ABCIQueryWithOptions(ctx, path, data, rpcclient.DefaultABCIQueryOptions)
}

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
	return &ctypes.ResultABCIQuery{Response: *resQuery}, nil
}

func (c *Client) BroadcastTxCommit(ctx context.Context, tx tmtypes.Tx) (*ctypes.ResultBroadcastTxCommit, error) {
	subscriber := ""

	if err := c.IsSubscriptionAllowed(subscriber); err != nil {
		return nil, sdkerrors.Wrap(err, "subscription not allowed")
	}

	subCtx, cancel := context.WithTimeout(ctx, subscribeTimeout)
	defer cancel()
	q := tmtypes.EventQueryTxFor(tx)
	deliverTxSub, err := c.EventBus.Subscribe(subCtx, subscriber, q)
	if err != nil {
		err = fmt.Errorf("subscribe to tx: %w", err)
		return nil, err
	}
	defer func() {
		if err := c.EventBus.Unsubscribe(context.Background(), subscriber, q); err != nil {
		}
	}()

	checkTxResCh := make(chan *abci.Response, 1)
	err = c.node.Mempool.CheckTx(tx, func(res *abci.Response) {
		select {
		case <-ctx.Done():
		case checkTxResCh <- res:
		}
	}, mempool.TxInfo{})
	if err != nil {
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

		err = c.node.P2P.GossipTx(ctx, tx)
		if err != nil {
			return nil, fmt.Errorf("tx added to local mempool but failure to broadcast: %w", err)
		}

		select {
		case msg := <-deliverTxSub.Out():
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
			return &ctypes.ResultBroadcastTxCommit{
				CheckTx:   *checkTxRes,
				DeliverTx: abci.ResponseDeliverTx{},
				Hash:      tx.Hash(),
			}, err
		case <-time.After(c.config.TimeoutBroadcastTxCommit):
			err = errors.New("timed out waiting for tx to be included in a block")
			return &ctypes.ResultBroadcastTxCommit{
				CheckTx:   *checkTxRes,
				DeliverTx: abci.ResponseDeliverTx{},
				Hash:      tx.Hash(),
			}, err
		}
	}
}

func (c *Client) BroadcastTxAsync(ctx context.Context, tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	err := c.node.Mempool.CheckTx(tx, nil, mempool.TxInfo{})
	if err != nil {
		return nil, err
	}

	err = c.node.P2P.GossipTx(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("tx added to local mempool but failed to gossip: %w", err)
	}
	return &ctypes.ResultBroadcastTx{Hash: tx.Hash()}, nil
}

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

	if r.Code == abci.CodeTypeOK {
		err = c.node.P2P.GossipTx(ctx, tx)
		if err != nil {

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
		sub, err = c.EventBus.SubscribeUnbuffered(ctx, subscriber, q)
	}
	if err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}

	outc := make(chan ctypes.ResultEvent, outCap)
	go c.eventsRoutine(sub, subscriber, q, outc)

	return outc, nil
}

func (c *Client) Unsubscribe(ctx context.Context, subscriber, query string) error {
	q, err := tmquery.New(query)
	if err != nil {
		return fmt.Errorf("parse query: %w", err)
	}
	return c.EventBus.Unsubscribe(ctx, subscriber, q)
}

func (c *Client) Genesis(_ context.Context) (*ctypes.ResultGenesis, error) {
	return &ctypes.ResultGenesis{Genesis: c.node.GetGenesis()}, nil
}

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

	if id > uint(chunkLen)-1 {
		return nil, fmt.Errorf("there are %d chunks, %d is invalid", chunkLen-1, id)
	}

	return &ctypes.ResultGenesisChunk{
		TotalChunks: chunkLen,
		ChunkNumber: int(id),
		Data:        genChunks[id],
	}, nil
}

func (c *Client) BlockchainInfo(ctx context.Context, minHeight, maxHeight int64) (*ctypes.ResultBlockchainInfo, error) {
	const limit int64 = 20

	baseHeight, err := c.node.BlockManager.Store.LoadBaseHeight()
	if err != nil && !errors.Is(err, gerrc.ErrNotFound) {
		return nil, err
	}
	if errors.Is(err, gerrc.ErrNotFound) {
		baseHeight = 1
	}
	minHeight, maxHeight, err = filterMinMax(
		int64(baseHeight),
		int64(c.node.GetBlockManagerHeight()),
		minHeight,
		maxHeight,
		limit)
	if err != nil {
		return nil, err
	}

	blocks := make([]*tmtypes.BlockMeta, 0, maxHeight-minHeight+1)
	for height := maxHeight; height >= minHeight; height-- {
		block, err := c.node.Store.LoadBlock(uint64(height))
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
		LastHeight: int64(c.node.GetBlockManagerHeight()),
		BlockMetas: blocks,
	}, nil
}

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

func (c *Client) DumpConsensusState(ctx context.Context) (*ctypes.ResultDumpConsensusState, error) {
	return nil, ErrConsensusStateNotAvailable
}

func (c *Client) ConsensusState(ctx context.Context) (*ctypes.ResultConsensusState, error) {
	return nil, ErrConsensusStateNotAvailable
}

func (c *Client) ConsensusParams(ctx context.Context, height *int64) (*ctypes.ResultConsensusParams, error) {
	params := c.node.GetGenesis().ConsensusParams
	return &ctypes.ResultConsensusParams{
		BlockHeight: int64(c.normalizeHeight(height)),
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

func (c *Client) Health(ctx context.Context) (*ctypes.ResultHealth, error) {
	return &ctypes.ResultHealth{}, nil
}

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

func (c *Client) BlockResults(ctx context.Context, height *int64) (*ctypes.ResultBlockResults, error) {
	var h uint64
	if height == nil {
		h = c.node.GetBlockManagerHeight()
	} else {
		h = uint64(*height)
	}
	resp, err := c.node.Store.LoadBlockResponses(h)
	if err != nil {
		return nil, err
	}

	return &ctypes.ResultBlockResults{
		Height:                int64(h),
		TxsResults:            resp.DeliverTxs,
		BeginBlockEvents:      resp.BeginBlock.Events,
		EndBlockEvents:        resp.EndBlock.Events,
		ValidatorUpdates:      resp.EndBlock.ValidatorUpdates,
		ConsensusParamUpdates: resp.EndBlock.ConsensusParamUpdates,
	}, nil
}

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
	commit := types.ToABCICommit(com, &b.Header)
	block, err := types.ToABCIBlock(b)
	if err != nil {
		return nil, err
	}

	return ctypes.NewResultCommit(&block.Header, commit, true), nil
}

func (c *Client) Validators(ctx context.Context, heightPtr *int64, _, _ *int) (*ctypes.ResultValidators, error) {
	height := c.normalizeHeight(heightPtr)

	proposer, err := c.node.Store.LoadProposer(height)
	if err != nil {
		return nil, fmt.Errorf("load validators: height %d: %w", height, err)
	}

	return &ctypes.ResultValidators{
		BlockHeight: int64(height),
		Validators:  proposer.TMValidators(),
		Count:       1,
		Total:       1,
	}, nil
}

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
		block, _ := c.node.Store.LoadBlock(uint64(height))
		blockProof := block.Data.Txs.Proof(int(index))
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

func (c *Client) TxSearch(ctx context.Context, query string, prove bool, pagePtr, perPagePtr *int, orderBy string) (*ctypes.ResultTxSearch, error) {
	q, err := tmquery.New(query)
	if err != nil {
		return nil, err
	}

	results, err := c.node.TxIndexer.Search(ctx, q)
	if err != nil {
		return nil, err
	}

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

func (c *Client) BlockSearch(ctx context.Context, query string, page, perPage *int, orderBy string) (*ctypes.ResultBlockSearch, error) {
	q, err := tmquery.New(query)
	if err != nil {
		return nil, err
	}

	results, err := c.node.BlockIndexer.Search(ctx, q)
	if err != nil {
		return nil, err
	}

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

	totalCount := len(results)
	perPageVal := validatePerPage(perPage)

	pageVal, err := validatePage(page, perPageVal, totalCount)
	if err != nil {
		return nil, err
	}

	skipCount := validateSkipCount(pageVal, perPageVal)
	pageSize := tmmath.MinInt(perPageVal, totalCount-skipCount)

	blocks := make([]*ctypes.ResultBlock, 0, pageSize)
	for i := skipCount; i < skipCount+pageSize; i++ {
		b, err := c.node.Store.LoadBlock(uint64(results[i]))
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

func (c *Client) Status(_ context.Context) (*ctypes.ResultStatus, error) {
	latest, err := c.node.Store.LoadBlock(c.node.GetBlockManagerHeight())
	if err != nil {
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
	txIndexerStatus := "on"

	result := &ctypes.ResultStatus{
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
			LatestBlockHeight: int64(latestHeight),
			LatestBlockTime:   latestBlockTime,

			CatchingUp: c.node.BlockManager.TargetHeight.Load() > latestHeight,
		},

		ValidatorInfo: ctypes.ValidatorInfo{
			Address:     tmbytes.HexBytes(proposer.ConsAddress()),
			PubKey:      proposer.PubKey(),
			VotingPower: 1,
		},
	}
	return result, nil
}

func (c *Client) BroadcastEvidence(ctx context.Context, evidence tmtypes.Evidence) (*ctypes.ResultBroadcastEvidence, error) {
	return &ctypes.ResultBroadcastEvidence{
		Hash: evidence.Hash(),
	}, nil
}

func (c *Client) NumUnconfirmedTxs(ctx context.Context) (*ctypes.ResultUnconfirmedTxs, error) {
	return &ctypes.ResultUnconfirmedTxs{
		Count:      c.node.Mempool.Size(),
		Total:      c.node.Mempool.Size(),
		TotalBytes: c.node.Mempool.SizeBytes(),
	}, nil
}

func (c *Client) UnconfirmedTxs(ctx context.Context, limitPtr *int) (*ctypes.ResultUnconfirmedTxs, error) {
	limit := validatePerPage(limitPtr)

	txs := c.node.Mempool.ReapMaxTxs(limit)
	return &ctypes.ResultUnconfirmedTxs{
		Count:      len(txs),
		Total:      c.node.Mempool.Size(),
		TotalBytes: c.node.Mempool.SizeBytes(),
		Txs:        txs,
	}, nil
}

func (c *Client) CheckTx(ctx context.Context, tx tmtypes.Tx) (*ctypes.ResultCheckTx, error) {
	res, err := c.Mempool().CheckTxSync(abci.RequestCheckTx{Tx: tx})
	if err != nil {
		return nil, err
	}
	return &ctypes.ResultCheckTx{ResponseCheckTx: *res}, nil
}

func (c *Client) BlockValidated(height *int64) (*ResultBlockValidated, error) {
	if height == nil || *height < 0 {
		return &ResultBlockValidated{Result: -1, ChainID: chainID}, nil
	}

	if uint64(*height) > c.node.BlockManager.State.Height() {
		return &ResultBlockValidated{Result: NotValidated, ChainID: chainID}, nil
	}

	if uint64(*height) <= c.node.BlockManager.SettlementValidator.GetLastValidatedHeight() {
		return &ResultBlockValidated{Result: SLValidated, ChainID: chainID}, nil
	}

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
				}
			}
		case <-sub.Cancelled():
			if errors.Is(sub.Err(), tmpubsub.ErrUnsubscribed) {
				return
			}

			sub = c.resubscribe(subscriber, q)
			if sub == nil {
				return
			}
		case <-c.Quit():
			return
		}
	}
}

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
		time.Sleep((10 << attempts) * time.Millisecond)
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
		heightValue = uint64(*height)
	}

	return heightValue
}

func (c *Client) IsSubscriptionAllowed(subscriber string) error {
	if c.EventBus.NumClients() >= c.config.MaxSubscriptionClients {
		return fmt.Errorf("max_subscription_clients %d reached", c.config.MaxSubscriptionClients)
	} else if c.EventBus.NumClientSubscriptions(subscriber) >= c.config.MaxSubscriptionsPerClient {
		return fmt.Errorf("max_subscriptions_per_client %d reached", c.config.MaxSubscriptionsPerClient)
	}

	return nil
}

func validatePerPage(perPagePtr *int) int {
	if perPagePtr == nil {
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

	if pagePtr == nil || *pagePtr <= 0 {
		return 1, nil
	}

	pages := ((totalCount - 1) / perPage) + 1
	if pages == 0 {
		pages = 1
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
	if min < 0 || max < 0 {
		return min, max, errors.New("height must be greater than zero")
	}

	if min == 0 {
		min = 1
	}
	if max == 0 {
		max = height
	}

	max = tmmath.MinInt64(height, max)

	min = tmmath.MaxInt64(base, min)

	min = tmmath.MaxInt64(min, max-limit+1)

	if min > max {
		return min, max, fmt.Errorf("%w: min height %d can't be greater than max height %d",
			errors.New("invalid request"), min, max)
	}
	return min, max, nil
}
