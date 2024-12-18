package p2p

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	cdiscovery "github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	discutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"github.com/multiformats/go-multiaddr"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/p2p"
	"go.uber.org/multierr"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/types"
)

const (
	reAdvertisePeriod = 1 * time.Hour

	peerLimit = 60

	txTopicSuffix = "-tx"

	blockTopicSuffix = "-block"

	blockSyncProtocolPrefix = "block-sync"
)

type Client struct {
	conf    config.P2PConfig
	chainID string
	privKey crypto.PrivKey

	Host host.Host
	DHT  *dht.IpfsDHT
	disc *discovery.RoutingDiscovery

	txGossiper  *Gossiper
	txValidator GossipValidator

	blockGossiper  *Gossiper
	blockValidator GossipValidator

	cancel context.CancelFunc

	localPubsubServer *tmpubsub.Server

	logger types.Logger

	blocksync *BlockSync

	blockSyncStore datastore.Datastore

	store store.Store

	blocksReceived *BlocksReceived
}

func NewClient(conf config.P2PConfig, privKey crypto.PrivKey, chainID string, store store.Store, localPubsubServer *tmpubsub.Server, blockSyncStore datastore.Datastore, logger types.Logger) (*Client, error) {
	if privKey == nil {
		return nil, fmt.Errorf("private key: %w", gerrc.ErrNotFound)
	}
	if conf.ListenAddress == "" {
		conf.ListenAddress = config.DefaultListenAddress
	}

	return &Client{
		conf:              conf,
		privKey:           privKey,
		chainID:           chainID,
		logger:            logger,
		localPubsubServer: localPubsubServer,
		blockSyncStore:    blockSyncStore,
		store:             store,
		blocksReceived: &BlocksReceived{
			blocksReceived: make(map[uint64]struct{}),
		},
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	ctx, c.cancel = context.WithCancel(ctx)
	host, err := c.listen()
	if err != nil {
		return err
	}
	return c.StartWithHost(ctx, host)
}

func (c *Client) StartWithHost(ctx context.Context, h host.Host) error {
	c.Host = h
	for _, a := range c.Host.Addrs() {
	}

	err := c.setupGossiping(ctx)
	if err != nil {
		return err
	}

	err = c.setupDHT(ctx)
	if err != nil {
		return err
	}

	err = c.peerDiscovery(ctx)
	if err != nil {
		return err
	}

	if !c.conf.BlockSyncEnabled {
		return nil
	}

	err = c.startBlockSync(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Close() error {
	c.cancel()

	return multierr.Combine(
		c.txGossiper.Close(),
		c.blockGossiper.Close(),
		c.DHT.Close(),
		c.Host.Close(),
	)
}

func (c *Client) GossipTx(ctx context.Context, tx []byte) error {
	return c.txGossiper.Publish(ctx, tx)
}

func (c *Client) SetTxValidator(val GossipValidator) {
	c.txValidator = val
}

func (c *Client) GossipBlock(ctx context.Context, blockBytes []byte) error {
	return c.blockGossiper.Publish(ctx, blockBytes)
}

func (c *Client) SaveBlock(ctx context.Context, height uint64, revision uint64, blockBytes []byte) error {
	if !c.conf.BlockSyncEnabled {
		return nil
	}
	_, err := c.store.LoadBlockSyncBaseHeight()
	if err != nil {
		err = c.store.SaveBlockSyncBaseHeight(height)
		if err != nil {
			return err
		}
	}

	cid, err := c.blocksync.SaveBlock(ctx, blockBytes)
	if err != nil {
		return fmt.Errorf("blocksync add block: %w", err)
	}
	_, err = c.store.SaveBlockCid(height, cid, nil)
	if err != nil {
		return fmt.Errorf("blocksync store block id: %w", err)
	}
	err = c.AdvertiseBlockIdToDHT(ctx, height, revision, cid)
	if err != nil {
	}
	return nil
}

func (c *Client) RemoveBlocks(ctx context.Context, to uint64) (uint64, error) {
	prunedBlocks := uint64(0)

	from, err := c.store.LoadBlockSyncBaseHeight()
	if errors.Is(err, gerrc.ErrNotFound) {
	} else if err != nil {
		return prunedBlocks, err
	}

	for h := from; h < to; h++ {
		cid, err := c.store.LoadBlockCid(h)
		if errors.Is(err, gerrc.ErrNotFound) {
			continue
		}
		if err != nil {
			continue
		}
		// blocksync update not atomic with store update
		err = c.blocksync.DeleteBlock(ctx, cid)
		if err != nil {
			continue
		}
		err = c.store.RemoveBlockCid(h)
		if err != nil {
			continue
		}
		prunedBlocks++
	}

	err = c.store.SaveBlockSyncBaseHeight(to)
	if err != nil {
		return prunedBlocks, err
	}

	return prunedBlocks, nil
}

func (c *Client) AdvertiseBlockIdToDHT(ctx context.Context, height uint64, revision uint64, cid cid.Cid) error {
	err := c.DHT.PutValue(ctx, getBlockSyncKeyByHeight(height, revision), []byte(cid.String()))
	return err
}

func (c *Client) GetBlockIdFromDHT(ctx context.Context, height uint64, revision uint64) (cid.Cid, error) {
	cidBytes, err := c.DHT.GetValue(ctx, getBlockSyncKeyByHeight(height, revision))
	if err != nil {
		return cid.Undef, err
	}
	return cid.MustParse(string(cidBytes)), nil
}

func (c *Client) UpdateLatestSeenHeight(height uint64) {
	c.blocksReceived.latestSeenHeight = max(height, c.blocksReceived.latestSeenHeight) // :(
}

func (c *Client) SetBlockValidator(validator GossipValidator) {
	c.blockValidator = validator
}

func (c *Client) Addrs() []multiaddr.Multiaddr {
	return c.Host.Addrs()
}

func (c *Client) Info() (p2p.ID, string, string) {
	return p2p.ID(hex.EncodeToString([]byte(c.Host.ID()))), c.conf.ListenAddress, c.chainID
}

type PeerConnection struct {
	NodeInfo         p2p.DefaultNodeInfo  `json:"node_info"`
	IsOutbound       bool                 `json:"is_outbound"`
	ConnectionStatus p2p.ConnectionStatus `json:"connection_status"`
	RemoteIP         string               `json:"remote_ip"`
}

func (c *Client) Peers() []PeerConnection {
	conns := c.Host.Network().Conns()
	res := make([]PeerConnection, 0, len(conns))
	for _, conn := range conns {
		pc := PeerConnection{
			NodeInfo: p2p.DefaultNodeInfo{
				ListenAddr:    c.conf.ListenAddress,
				Network:       c.chainID,
				DefaultNodeID: p2p.ID(conn.RemotePeer().String()),
			},
			IsOutbound: conn.Stat().Direction == network.DirOutbound,
			ConnectionStatus: p2p.ConnectionStatus{
				Duration: time.Since(conn.Stat().Opened),
			},
			RemoteIP: conn.RemoteMultiaddr().String(),
		}
		res = append(res, pc)
	}
	return res
}

func (c *Client) listen() (host.Host, error) {
	var err error
	maddr, err := multiaddr.NewMultiaddr(c.conf.ListenAddress)
	if err != nil {
		return nil, err
	}

	host, err := libp2p.New(libp2p.ListenAddrs(maddr), libp2p.Identity(c.privKey))
	if err != nil {
		return nil, err
	}

	return host, nil
}

func (c *Client) setupDHT(ctx context.Context) error {
	bootstrapNodes := c.GetSeedAddrInfo(c.conf.BootstrapNodes)
	if len(bootstrapNodes) == 0 {
		bootstrapNodes = c.GetSeedAddrInfo(c.conf.PersistentNodes)
		if len(bootstrapNodes) != 0 {
		}
	}

	for _, sa := range bootstrapNodes {
	}

	var err error

	val := dht.NamespacedValidator(blockSyncProtocolPrefix, blockIdValidator{})
	c.DHT, err = dht.New(ctx, c.Host, dht.Mode(dht.ModeServer), dht.ProtocolPrefix(blockSyncProtocolPrefix), val, dht.BootstrapPeers(bootstrapNodes...))
	if err != nil {
		return fmt.Errorf("create DHT: %w", err)
	}

	err = c.DHT.Bootstrap(ctx)
	if err != nil {
		return fmt.Errorf("bootstrap DHT: %w", err)
	}

	go c.bootstrapLoop(ctx)

	c.Host = routedhost.Wrap(c.Host, c.DHT)

	return nil
}

func (c *Client) peerDiscovery(ctx context.Context) error {
	err := c.setupPeerDiscovery(ctx)
	if err != nil {
		return err
	}

	if c.conf.AdvertisingEnabled {
		err = c.advertise(ctx)
		if err != nil {
			return err
		}
	}

	err = c.findPeers(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) setupPeerDiscovery(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	case <-c.DHT.RefreshRoutingTable():
	}
	c.disc = discovery.NewRoutingDiscovery(c.DHT)
	return nil
}

func (c *Client) startBlockSync(ctx context.Context) error {
	blocksync := SetupBlockSync(ctx, c.Host, c.blockSyncStore, c.logger)
	c.blocksync = blocksync
	go c.retrieveBlockSyncLoop(ctx, c.blockSyncReceived)
	go c.advertiseBlockSyncCids(ctx)
	return nil
}

func (c *Client) advertise(ctx context.Context) error {
	discutil.Advertise(ctx, c.disc, c.getNamespace(), cdiscovery.TTL(reAdvertisePeriod))
	return nil
}

func (c *Client) findPeers(ctx context.Context) error {
	peerCh, err := c.disc.FindPeers(ctx, c.getNamespace(), cdiscovery.Limit(peerLimit))
	if err != nil {
		return err
	}

	for peer := range peerCh {
		go c.tryConnect(ctx, peer)
	}

	return nil
}

func (c *Client) tryConnect(ctx context.Context, peer peer.AddrInfo) {
	err := c.Host.Connect(ctx, peer)
	if err != nil {
		return
	}
}

func (c *Client) setupGossiping(ctx context.Context) error {
	pubsub.GossipSubHistoryGossip = c.conf.GossipSubCacheSize
	pubsub.GossipSubHistoryLength = c.conf.GossipSubCacheSize

	ps, err := pubsub.NewGossipSub(ctx, c.Host)
	if err != nil {
		return err
	}

	c.txGossiper, err = NewGossiper(c.Host, ps, c.getTxTopic(), nil, c.logger, WithValidator(c.txValidator))
	if err != nil {
		return err
	}
	go c.txGossiper.ProcessMessages(ctx)

	c.blockGossiper, err = NewGossiper(c.Host, ps, c.getBlockTopic(), c.blockGossipReceived, c.logger,
		WithValidator(c.blockValidator))
	if err != nil {
		return err
	}
	go c.blockGossiper.ProcessMessages(ctx)

	return nil
}

func (c *Client) GetSeedAddrInfo(seedStr string) []peer.AddrInfo {
	if len(seedStr) == 0 {
		return []peer.AddrInfo{}
	}
	seeds := strings.Split(seedStr, ",")
	addrs := make([]peer.AddrInfo, 0, len(seeds))
	for _, s := range seeds {
		maddr, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			continue
		}
		addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			continue
		}
		addrs = append(addrs, *addrInfo)
	}
	return addrs
}

func (c *Client) getNamespace() string {
	return c.chainID
}

func (c *Client) getTxTopic() string {
	return c.getNamespace() + txTopicSuffix
}

func (c *Client) getBlockTopic() string {
	return c.getNamespace() + blockTopicSuffix
}

func (c *Client) NewTxValidator() GossipValidator {
	return func(g *GossipMessage) bool {
		return true
	}
}

func (c *Client) blockSyncReceived(block *BlockData) {
	err := c.localPubsubServer.PublishWithEvents(context.Background(), *block, map[string][]string{EventTypeKey: {EventNewBlockSyncBlock}})
	if err != nil {
	}

	c.blocksReceived.AddBlockReceived(block.Block.Header.Height)
}

func (c *Client) blockGossipReceived(ctx context.Context, block []byte) {
	var gossipedBlock BlockData
	if err := gossipedBlock.UnmarshalBinary(block); err != nil {
	}
	err := c.localPubsubServer.PublishWithEvents(context.Background(), gossipedBlock, map[string][]string{EventTypeKey: {EventNewGossipedBlock}})
	if err != nil {
	}
	if c.conf.BlockSyncEnabled {
		_, err := c.store.LoadBlockCid(gossipedBlock.Block.Header.Height)

		if err == nil {
			return
		}
		err = c.SaveBlock(ctx, gossipedBlock.Block.Header.Height, gossipedBlock.Block.GetRevision(), block)
		if err != nil {
		}

		c.blocksReceived.AddBlockReceived(gossipedBlock.Block.Header.Height)
	}
}

func (c *Client) bootstrapLoop(ctx context.Context) {
	ticker := time.NewTicker(c.conf.BootstrapRetryTime)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if len(c.Peers()) == 0 {
				err := c.DHT.Bootstrap(ctx)
				if err != nil {
				}
			}
			persistentNodes := c.GetSeedAddrInfo(c.conf.PersistentNodes)
			for _, peer := range persistentNodes {
				found := c.findConnection(peer)
				if !found {
					c.tryConnect(ctx, peer)
				}
			}
		}
	}
}

func (c *Client) retrieveBlockSyncLoop(ctx context.Context, msgHandler BlockSyncMessageHandler) {
	ticker := time.NewTicker(c.conf.BlockSyncRequestIntervalTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			if len(c.Peers()) == 0 {
				continue
			}
			state, err := c.store.LoadState()
			if err != nil {
				continue
			}

			for h := state.NextHeight(); h <= c.blocksReceived.latestSeenHeight; h++ {
				if ctx.Err() != nil {
					return
				}
				ok := c.blocksReceived.IsBlockReceived(h)
				if ok {
					continue
				}
				id, err := c.GetBlockIdFromDHT(ctx, h, state.GetRevision())
				if err != nil || id == cid.Undef {
					continue
				}
				_, err = c.store.SaveBlockCid(h, id, nil)
				if err != nil {
					continue
				}
				block, err := c.blocksync.LoadBlock(ctx, id)
				if err != nil {
					continue
				}

				state, err := c.store.LoadState()
				if err != nil {
					return
				}
				if err := block.Validate(state.GetProposerPubKey()); err != nil {
					continue
				}
				msgHandler(&block)
				h = max(h, state.NextHeight()-1)
			}
			c.blocksReceived.RemoveBlocksReceivedUpToHeight(state.NextHeight())

		}
	}
}

func (c *Client) advertiseBlockSyncCids(ctx context.Context) {
	ticker := time.NewTicker(c.conf.BlockSyncRequestIntervalTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:

			if len(c.Peers()) == 0 {
				continue
			}
			state, err := c.store.LoadState()
			if err != nil {
				return
			}
			baseHeight, err := c.store.LoadBlockSyncBaseHeight()
			if err != nil && !errors.Is(err, gerrc.ErrNotFound) {
				return
			}
			for h := baseHeight; h <= state.Height(); h++ {
				if ctx.Err() != nil {
					return
				}
				id, err := c.store.LoadBlockCid(h)
				if err != nil || id == cid.Undef {
					continue
				}
				block, err := c.store.LoadBlock(h)
				if err != nil {
					continue
				}
				err = c.AdvertiseBlockIdToDHT(ctx, h, block.GetRevision(), id)
				if err != nil {
					continue
				}

			}

			return
		}
	}
}

func (c *Client) findConnection(peer peer.AddrInfo) bool {
	for _, con := range c.Host.Network().Conns() {
		if peer.ID == con.RemotePeer() {
			return true
		}
	}
	return false
}

func getBlockSyncKeyByHeight(height uint64, revision uint64) string {
	return "/" + blockSyncProtocolPrefix + "/" + strconv.FormatUint(revision, 10) + "/" + strconv.FormatUint(height, 10)
}

type blockIdValidator struct{}

func (blockIdValidator) Validate(_ string, id []byte) error {
	_, err := cid.Parse(string(id))
	return err
}

func (blockIdValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }
