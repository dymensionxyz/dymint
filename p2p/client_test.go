package p2p_test

import (
	"context"
	"crypto/rand"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/store"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/testutil"
	p2ppubsub "github.com/libp2p/go-libp2p-pubsub"
)

func TestClientStartup(t *testing.T) {
	privKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
	pubsubServer := pubsub.NewServer()
	err := pubsubServer.Start()
	require.NoError(t, err)
	store := store.New(store.NewDefaultInMemoryKVStore())
	client, err := p2p.NewClient(config.P2PConfig{
		ListenAddress:                config.DefaultListenAddress,
		GossipSubCacheSize:           50,
		BootstrapRetryTime:           30 * time.Second,
		BlockSyncRequestIntervalTime: 30 * time.Second,
	}, privKey, "TestChain", store, pubsubServer, datastore.NewMapDatastore(), log.TestingLogger())
	assert := assert.New(t)
	assert.NoError(err)
	assert.NotNil(client)

	err = client.Start(context.Background())
	defer func() {
		_ = client.Close()
	}()
	assert.NoError(err)
}

// TestBootstrapping TODO: this test is flaky in main. Try running it 100 times to reproduce.
// https://github.com/dymensionxyz/dymint/issues/869
func TestBootstrapping(t *testing.T) {
	assert := assert.New(t)
	logger := log.TestingLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clients := testutil.StartTestNetwork(ctx, t, 4, map[int]testutil.HostDescr{
		1: {Conns: []int{0}},
		2: {Conns: []int{0, 1}},
		3: {Conns: []int{0}},
	}, make([]p2p.GossipValidator, 4), logger)

	// wait for clients to finish refreshing routing tables
	clients.WaitForDHT()

	for _, client := range clients {
		assert.Equal(3, len(client.Host.Network().Peers()))
	}
}

func TestDiscovery(t *testing.T) {
	assert := assert.New(t)
	logger := log.TestingLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	clients := testutil.StartTestNetwork(ctx, t, 5, map[int]testutil.HostDescr{
		1: {Conns: []int{0}, ChainID: "ORU2"},
		2: {Conns: []int{0}, ChainID: "ORU2"},
		3: {Conns: []int{1}, ChainID: "ORU1"},
		4: {Conns: []int{2}, ChainID: "ORU1"},
	}, make([]p2p.GossipValidator, 5), logger)

	// wait for clients to finish refreshing routing tables
	clients.WaitForDHT()

	assert.Contains(clients[3].Host.Network().Peers(), clients[4].Host.ID())
	assert.Contains(clients[4].Host.Network().Peers(), clients[3].Host.ID())
}

func TestGossiping(t *testing.T) {
	assert := assert.New(t)
	logger := log.TestingLogger()

	ctx := context.Background()

	expectedMsg := []byte("foobar")
	var wg sync.WaitGroup

	wg.Add(2)

	// ensure that Tx is delivered to client
	assertRecv := func(tx *p2p.GossipMessage) p2ppubsub.ValidationResult {
		logger.Debug("received tx", "body", string(tx.Data), "from", tx.From)
		assert.Equal(expectedMsg, tx.Data)
		wg.Done()
		return p2ppubsub.ValidationAccept
	}

	// ensure that Tx is not delivered to client
	assertNotRecv := func(*p2p.GossipMessage) p2ppubsub.ValidationResult {
		t.Fatal("unexpected Tx received")
		return p2ppubsub.ValidationReject
	}

	validators := []p2p.GossipValidator{assertRecv, assertNotRecv, assertNotRecv, assertRecv, assertNotRecv}

	// network connections topology: 3<->1<->0<->2<->4
	clients := testutil.StartTestNetwork(ctx, t, 5, map[int]testutil.HostDescr{
		0: {Conns: []int{}, ChainID: "2"},
		1: {Conns: []int{0}, ChainID: "1", RealKey: true},
		2: {Conns: []int{0}, ChainID: "1", RealKey: true},
		3: {Conns: []int{1}, ChainID: "2", RealKey: true},
		4: {Conns: []int{2}, ChainID: "2", RealKey: true},
	}, validators, logger)

	// wait for clients to finish refreshing routing tables
	clients.WaitForDHT()

	// this sleep is required for pubsub to "propagate" subscription information
	// TODO(tzdybal): is there a better way to wait for readiness?
	time.Sleep(1 * time.Second)

	// gossip from client 4. The validation of client 4 won't be called as we're not
	// validating messages sent by ourselves.
	err := clients[4].GossipTx(ctx, expectedMsg)
	assert.NoError(err)

	// wait for clients that should receive Tx
	wg.Wait()
}

// Test that advertises and retrieves a CID for a block height in the DHT
func TestAdvertiseBlock(t *testing.T) {
	logger := log.TestingLogger()

	ctx := context.Background()

	// required for tx validator
	assertRecv := func(tx *p2p.GossipMessage) p2ppubsub.ValidationResult {
		return p2ppubsub.ValidationAccept
	}

	// Create a cid manually by specifying the 'prefix' parameters
	pref := &cid.Prefix{
		Codec:    cid.DagProtobuf,
		MhLength: -1,
		MhType:   mh.SHA2_256,
		Version:  1,
	}

	// And then feed it some data
	expectedCid, err := pref.Sum([]byte("test"))
	require.NoError(t, err)

	// validators required
	validators := []p2p.GossipValidator{assertRecv, assertRecv, assertRecv, assertRecv, assertRecv}

	// network connections topology: 3<->1<->0<->2<->4
	clients := testutil.StartTestNetwork(ctx, t, 3, map[int]testutil.HostDescr{
		0: {Conns: []int{}, ChainID: "1"},
		1: {Conns: []int{0}, ChainID: "1"},
		2: {Conns: []int{1}, ChainID: "1"},
	}, validators, logger)

	// wait for clients to finish refreshing routing tables
	clients.WaitForDHT()

	// this sleep is required for pubsub to "propagate" subscription information
	// TODO(tzdybal): is there a better way to wait for readiness?
	time.Sleep(1 * time.Second)

	// advertise cid for height 1
	err = clients[2].AdvertiseBlockIdToDHT(ctx, 1, 1, expectedCid)
	require.NoError(t, err)

	// get cid for height 1
	receivedCid, err := clients[0].GetBlockIdFromDHT(ctx, 1, 1)
	require.NoError(t, err)
	require.Equal(t, expectedCid, receivedCid)
}

// Test that advertises an invalid CID in the DHT
func TestAdvertiseWrongCid(t *testing.T) {
	logger := log.TestingLogger()

	ctx := context.Background()

	// required for tx validator
	assertRecv := func(tx *p2p.GossipMessage) p2ppubsub.ValidationResult {
		return p2ppubsub.ValidationAccept
	}

	validators := []p2p.GossipValidator{assertRecv, assertRecv, assertRecv, assertRecv, assertRecv}

	// network connections topology: 3<->1<->0<->2<->4
	clients := testutil.StartTestNetwork(ctx, t, 3, map[int]testutil.HostDescr{
		0: {Conns: []int{}, ChainID: "1"},
		1: {Conns: []int{0}, ChainID: "1"},
		2: {Conns: []int{1}, ChainID: "1"},
	}, validators, logger)

	// wait for clients to finish refreshing routing tables
	clients.WaitForDHT()

	// this sleep is required for pubsub to "propagate" subscription information
	// TODO(tzdybal): is there a better way to wait for readiness?
	time.Sleep(1 * time.Second)

	// advertise cid for height 1
	receivedError := clients[2].DHT.PutValue(ctx, "/block-sync/"+strconv.FormatUint(1, 10), []byte("test"))

	require.Error(t, cid.ErrInvalidCid{}, receivedError)
}

func TestSeedStringParsing(t *testing.T) {
	t.Parallel()

	privKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)

	seed1 := "/ip4/127.0.0.1/tcp/7676/p2p/12D3KooWM1NFkZozoatQi3JvFE57eBaX56mNgBA68Lk5MTPxBE4U"
	seed1MA, err := multiaddr.NewMultiaddr(seed1)
	require.NoError(t, err)
	seed1AI, err := peer.AddrInfoFromP2pAddr(seed1MA)
	require.NoError(t, err)

	seed2 := "/ip4/127.0.0.1/tcp/7677/p2p/12D3KooWAPRFbmWF5dAXvxLnEDxiHWhUuApVDpNNZwShiFAiJqrj"
	seed2MA, err := multiaddr.NewMultiaddr(seed2)
	require.NoError(t, err)
	seed2AI, err := peer.AddrInfoFromP2pAddr(seed2MA)
	require.NoError(t, err)

	// this one is a valid multiaddr, but can't be converted to PeerID (because there is no ID)
	seed3 := "/ip4/127.0.0.1/tcp/12345"

	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(t, err)

	cases := []struct {
		name     string
		input    string
		expected []peer.AddrInfo
		nErrors  int
	}{
		{"empty input", "", []peer.AddrInfo{}, 0},
		{"one correct seed", seed1, []peer.AddrInfo{*seed1AI}, 0},
		{"two correct seeds", seed1 + "," + seed2, []peer.AddrInfo{*seed1AI, *seed2AI}, 0},
		{"one wrong, two correct", "/ip4/," + seed1 + "," + seed2, []peer.AddrInfo{*seed1AI, *seed2AI}, 1},
		{"empty, two correct", "," + seed1 + "," + seed2, []peer.AddrInfo{*seed1AI, *seed2AI}, 1},
		{"empty, correct, empty, correct ", "," + seed1 + ",," + seed2, []peer.AddrInfo{*seed1AI, *seed2AI}, 2},
		{"invalid id, two correct", seed3 + "," + seed1 + "," + seed2, []peer.AddrInfo{*seed1AI, *seed2AI}, 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			logger := &testutil.MockLogger{}
			store := store.New(store.NewDefaultInMemoryKVStore())
			client, err := p2p.NewClient(config.P2PConfig{
				GossipSubCacheSize: 50,
				BootstrapRetryTime: 30 * time.Second,
			}, privKey, "TestNetwork", store, pubsubServer, datastore.NewMapDatastore(), logger)
			require.NoError(err)
			require.NotNil(client)
			actual := client.GetSeedAddrInfo(c.input)
			assert.NotNil(actual)
			assert.Equal(c.expected, actual)
			// ensure that errors are logged
			assert.Equal(c.nErrors, len(logger.ErrLines))
		})
	}
}
