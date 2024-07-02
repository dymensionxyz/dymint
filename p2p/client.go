package p2p

import (
	"context"
	"encoding/hex"
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

type StatusCode uint64

// TODO(tzdybal): refactor to configuration parameters
const (
	// reAdvertisePeriod defines a period after which P2P client re-attempt advertising namespace in DHT.
	reAdvertisePeriod = 1 * time.Hour

	// peerLimit defines limit of number of peers returned during active peer discovery.
	peerLimit = 60

	// txTopicSuffix is added after namespace to create pubsub topic for TX gossiping.
	txTopicSuffix = "-tx"

	// blockTopicSuffix is added after namespace to create pubsub topic for block gossiping.
	blockTopicSuffix = "-block"

	// blockSyncProtocolSuffix is added after namespace to create block-sync protocol prefix.
	blockSyncProtocolPrefix = "block-sync"
)

// Client is a P2P client, implemented with libp2p.
//
// Initially, client connects to predefined seed nodes (aka bootnodes, bootstrap nodes).
// Those seed nodes serve Kademlia DHT protocol, and are agnostic to ORU chain. Using DHT
// peer routing and discovery clients find other peers within ORU network.
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

	// cancel is used to cancel context passed to libp2p functions
	// it's required because of discovery.Advertise call
	cancel context.CancelFunc

	localPubsubServer *tmpubsub.Server

	logger types.Logger

	blocksync *BlockSync

	blockSyncStore datastore.Datastore

	latestSeenHeight uint64

	blockAlreadyReceived map[uint64]struct{}

	store store.Store
}

// NewClient creates new Client object.
//
// Basic checks on parameters are done, and default parameters are provided for unset-configuration
// TODO(tzdybal): consider passing entire config, not just P2P config, to reduce number of arguments
func NewClient(conf config.P2PConfig, privKey crypto.PrivKey, chainID string, store store.Store, localPubsubServer *tmpubsub.Server, blockSyncStore datastore.Datastore, logger types.Logger) (*Client, error) {
	if privKey == nil {
		return nil, fmt.Errorf("private key: %w", gerrc.ErrNotFound)
	}
	if conf.ListenAddress == "" {
		conf.ListenAddress = config.DefaultListenAddress
	}

	return &Client{
		conf:                 conf,
		privKey:              privKey,
		chainID:              chainID,
		logger:               logger,
		localPubsubServer:    localPubsubServer,
		blockSyncStore:       blockSyncStore,
		latestSeenHeight:     uint64(0),
		store:                store,
		blockAlreadyReceived: make(map[uint64]struct{}),
	}, nil
}

// Start establish Client's P2P connectivity.
//
// Following steps are taken:
// 1. Setup libp2p host, start listening for incoming connections.
// 2. Setup gossibsub.
// 3. Setup DHT, establish connection to seed nodes and initialize peer discovery.
// 4. Use active peer discovery to look for peers from same ORU network.
func (c *Client) Start(ctx context.Context) error {
	// create new, cancelable context
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
		c.logger.Info("Listening on:", "address", fmt.Sprintf("%s/p2p/%s", a, c.Host.ID()))
	}

	c.logger.Debug("Setting up gossiping.")
	err := c.setupGossiping(ctx)
	if err != nil {
		return err
	}

	c.logger.Debug("Setting up DHT.")
	err = c.setupDHT(ctx)
	if err != nil {
		return err
	}

	c.logger.Debug("Setting up active peer discovery.")
	err = c.peerDiscovery(ctx)
	if err != nil {
		return err
	}

	c.logger.Debug("Setting up block sync protocol")
	err = c.startBlockSync(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Close gently stops Client.
func (c *Client) Close() error {
	c.cancel()

	return multierr.Combine(
		c.txGossiper.Close(),
		c.blockGossiper.Close(),
		c.DHT.Close(),
		c.Host.Close(),
	)
}

// GossipTx sends the transaction to the P2P network.
func (c *Client) GossipTx(ctx context.Context, tx []byte) error {
	c.logger.Debug("Gossiping transaction.", "len", len(tx))
	return c.txGossiper.Publish(ctx, tx)
}

// SetTxValidator sets the callback function, that will be invoked during message gossiping.
func (c *Client) SetTxValidator(val GossipValidator) {
	c.txValidator = val
}

// GossipBlock sends the block, and it's commit to the P2P network.
func (c *Client) GossipBlock(ctx context.Context, blockBytes []byte) error {
	c.logger.Debug("Gossiping block.", "len", len(blockBytes))
	return c.blockGossiper.Publish(ctx, blockBytes)
}

// GossipBlock sends the block, and it's commit to the P2P network.
func (c *Client) AddBlock(ctx context.Context, height uint64, blockBytes []byte) error {
	cid, err := c.blocksync.AddBlock(ctx, blockBytes)
	if err != nil {
		return fmt.Errorf("blocksync add block: %w", err)
	}
	_, err = c.store.SaveBlockCid(height, cid, nil)
	if err != nil {
		return fmt.Errorf("blocksync store block id: %w", err)
	}
	advErr := c.AdvertiseBlock(ctx, height, cid)
	if advErr != nil {
		return fmt.Errorf("blocksync advertise block %w", advErr)
	}
	return nil
}

func (c *Client) RemoveBlock(ctx context.Context, height uint64) error {
	cid, err := c.store.LoadBlockCid(height)
	if err != nil {
		return fmt.Errorf("load block id from store %d: %w", height, err)
	}
	err = c.blocksync.RemoveBlock(ctx, cid)
	if err != nil {
		return fmt.Errorf("remove block height %d: %w", height, err)
	}
	return nil
}

func (c *Client) AdvertiseBlock(ctx context.Context, height uint64, cid cid.Cid) error {
	err := c.DHT.PutValue(ctx, "/"+blockSyncProtocolPrefix+"/"+strconv.FormatUint(height, 10), []byte(cid.String()))
	return err
}

func (c *Client) GetBlockId(ctx context.Context, height uint64) (cid.Cid, error) {
	cidBytes, err := c.DHT.GetValue(ctx, "/"+blockSyncProtocolPrefix+"/"+strconv.FormatUint(height, 10))
	if err != nil {
		return cid.Undef, err
	}
	return cid.MustParse(string(cidBytes)), nil
}

// SetBlockValidator sets the callback function, that will be invoked after block is received from P2P network.
func (c *Client) SetBlockValidator(validator GossipValidator) {
	c.blockValidator = validator
}

// Addrs returns listen addresses of Client.
func (c *Client) Addrs() []multiaddr.Multiaddr {
	return c.Host.Addrs()
}

// Info returns p2p info
func (c *Client) Info() (p2p.ID, string, string) {
	return p2p.ID(hex.EncodeToString([]byte(c.Host.ID()))), c.conf.ListenAddress, c.chainID
}

// PeerConnection describe basic information about P2P connection.
// TODO(tzdybal): move it somewhere
type PeerConnection struct {
	NodeInfo         p2p.DefaultNodeInfo  `json:"node_info"`
	IsOutbound       bool                 `json:"is_outbound"`
	ConnectionStatus p2p.ConnectionStatus `json:"connection_status"`
	RemoteIP         string               `json:"remote_ip"`
}

// Peers returns list of peers connected to Client.
func (c *Client) Peers() []PeerConnection {
	conns := c.Host.Network().Conns()
	res := make([]PeerConnection, 0, len(conns))
	for _, conn := range conns {
		pc := PeerConnection{
			NodeInfo: p2p.DefaultNodeInfo{
				ListenAddr:    c.conf.ListenAddress,
				Network:       c.chainID,
				DefaultNodeID: p2p.ID(conn.RemotePeer().String()),
				// TODO(tzdybal): fill more fields
			},
			IsOutbound: conn.Stat().Direction == network.DirOutbound,
			ConnectionStatus: p2p.ConnectionStatus{
				Duration: time.Since(conn.Stat().Opened),
				// TODO(tzdybal): fill more fields
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
		c.logger.Info("No bootstrap nodes configured.")
		bootstrapNodes = c.GetSeedAddrInfo(c.conf.PersistentNodes)
		if len(bootstrapNodes) != 0 {
			c.logger.Info("Using persistent nodes as bootstrap nodes.")
		}
	}

	for _, sa := range bootstrapNodes {
		c.logger.Debug("Bootstrap node.", "addr", sa)
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
	// wait for DHT
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.DHT.RefreshRoutingTable():
	}
	c.disc = discovery.NewRoutingDiscovery(c.DHT)
	return nil
}

func (c *Client) startBlockSync(ctx context.Context) error {
	blocksync := SetupBlockSync(ctx, c.Host, c.blockSyncStore, c.blockSyncReceived, c.logger)
	c.blocksync = blocksync
	go c.retrieveBlockSyncLoop(ctx)
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

// tryConnect attempts to connect to a peer and logs error if necessary
func (c *Client) tryConnect(ctx context.Context, peer peer.AddrInfo) {
	c.logger.Debug("Trying to connect to peer.", "peer", peer)
	err := c.Host.Connect(ctx, peer)
	if err != nil {
		c.logger.Error("Connecting to peer.", "peer", peer, "error", err)
		return
	}
	c.logger.Debug("Connected to peer successfully.", "peer", peer)
}

func (c *Client) setupGossiping(ctx context.Context) error {
	pubsub.GossipSubHistoryGossip = c.conf.GossipSubCacheSize
	pubsub.GossipSubHistoryLength = c.conf.GossipSubCacheSize

	ps, err := pubsub.NewGossipSub(ctx, c.Host)
	if err != nil {
		return err
	}

	// tx gossiper receives the tx to add to the mempool through validation process, since it is a joint process
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
			c.logger.Error("Parse seed node.", "address", s, "error", err)
			continue
		}
		addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			c.logger.Error("Create addr info for seed.", "address", maddr, "error", err)
			continue
		}
		addrs = append(addrs, *addrInfo)
	}
	return addrs
}

// getNamespace returns unique string identifying ORU network.
//
// It is used to advertise/find peers in libp2p DHT.
// For now, chainID is used.
func (c *Client) getNamespace() string {
	return c.chainID
}

func (c *Client) getTxTopic() string {
	return c.getNamespace() + txTopicSuffix
}

func (c *Client) getBlockTopic() string {
	return c.getNamespace() + blockTopicSuffix
}

// NewTxValidator creates a pubsub validator that uses the node's mempool to check the
// transaction. If the transaction is valid, then it is added to the mempool
func (c *Client) NewTxValidator() GossipValidator {
	return func(g *GossipMessage) bool {
		return true
	}
}

// blockSyncReceived is called on reception of new block via block-sync protocol
func (c *Client) blockSyncReceived(block *P2PBlock) {
	err := c.localPubsubServer.PublishWithEvents(context.Background(), *block, map[string][]string{EventTypeKey: {EventNewBlockSyncBlock}})
	if err != nil {
		c.logger.Error("Publishing event.", "err", err)
	}
	// Received block is cached and  no longer needed to request using block-sync
	c.blockAlreadyReceived[block.Block.Header.Height] = struct{}{}
}

// blockSyncReceived is called on reception of new block via gossip protocol
func (c *Client) blockGossipReceived(ctx context.Context, block []byte) {
	var gossipedBlock P2PBlock
	if err := gossipedBlock.UnmarshalBinary(block); err != nil {
		c.logger.Error("Deserialize gossiped block", "error", err)
	}
	err := c.localPubsubServer.PublishWithEvents(context.Background(), gossipedBlock, map[string][]string{EventTypeKey: {EventNewGossipedBlock}})
	if err != nil {
		c.logger.Error("Publishing event.", "err", err)
	}
	err = c.AddBlock(ctx, gossipedBlock.Block.Header.Height, block)
	if err != nil {
		c.logger.Error("Adding  block to blocksync store.", "err", err, "height", gossipedBlock.Block.Header.Height)
	}
	c.setLatestSeenHeight(gossipedBlock.Block.Header.Height)

	// Received block is cached and no longer needed to request using block-sync
	c.blockAlreadyReceived[gossipedBlock.Block.Header.Height] = struct{}{}
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
					c.logger.Error("Re-bootstrap DHT: %w", err)
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

// latest seen height to be retrieved using block-sync
func (c *Client) setLatestSeenHeight(height uint64) {
	c.latestSeenHeight = max(height, c.latestSeenHeight)
}

// retrieveBlockSyncLoop checks if there is any missing block to be retrieved and requests it on demand
func (c *Client) retrieveBlockSyncLoop(ctx context.Context) {
	ticker := time.NewTicker(c.conf.BlockSyncRetrieveRetryTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state, err := c.store.LoadState()
			if err != nil {
				c.logger.Error("loading state", "err", err)
				continue
			}

			// this loop iterates and retrieves all the blocks between the last block applied and the greatest height received,
			// skipping any block cached, since are already received.
			for h := state.NextHeight(); h <= c.latestSeenHeight; h++ {
				_, ok := c.blockAlreadyReceived[h]
				if ok {
					continue
				}
				c.logger.Debug("Getting block", "height", h)
				bid, err := c.GetBlockId(ctx, h)
				if err != nil || bid == cid.Undef {
					c.logger.Error("unable to find cid", "height", h)
					continue
				}
				_, err = c.store.SaveBlockCid(h, bid, nil)
				if err != nil {
					c.logger.Error("storing block cid", "height", h, "cid", bid)
					continue
				}
				block, err := c.blocksync.GetBlock(ctx, bid)
				if err != nil {
					continue
				}
				if c.blocksync.msgHandler != nil {
					c.blocksync.msgHandler(&block)
				}
			}
			for h := range c.blockAlreadyReceived {
				if h < state.NextHeight() {
					delete(c.blockAlreadyReceived, h)
				}
			}
		}
	}
}

func (c *Client) advertiseBlockSyncCids(ctx context.Context) {
	state, err := c.store.LoadState()
	if err != nil {
		c.logger.Error("loading state", "err", err)
		return
	}
	for h := state.BaseHeight; h <= state.Height(); h++ {
		id, err := c.GetBlockId(ctx, h)
		if err == nil && id != cid.Undef {
			continue
		}

		id, err = c.store.LoadBlockCid(h)
		if err != nil || id == cid.Undef {
			continue
		}

		err = c.AdvertiseBlock(ctx, h, id)
		if err != nil {
			continue
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

// validates the content identifiers advertised in the DHT are valid
type blockIdValidator struct{}

func (blockIdValidator) Validate(_ string, id []byte) error {
	_, err := cid.Parse(string(id))
	return err
}

func (blockIdValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }
