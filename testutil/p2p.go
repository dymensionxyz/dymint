package testutil

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"
	"go.uber.org/multierr"

	"github.com/dymensionxyz/dymint/config"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

type TestNet []*p2p.Client

func (tn TestNet) Close() (err error) {
	for i := range tn {
		err = multierr.Append(err, tn[i].Close())
	}
	return
}

func (tn TestNet) WaitForDHT() {
	for i := range tn {
		<-tn[i].DHT.RefreshRoutingTable()
	}
}

type HostDescr struct {
	ChainID string
	Conns   []int
	RealKey bool
}

// copied from libp2p net/mock
var blackholeIP6 = net.ParseIP("100::")

// copied from libp2p net/mock
func getAddr(sk crypto.PrivKey) (multiaddr.Multiaddr, error) {
	id, err := peer.IDFromPrivateKey(sk)
	if err != nil {
		return nil, err
	}
	suffix := id
	if len(id) > 8 {
		suffix = id[len(id)-8:]
	}
	ip := append(net.IP{}, blackholeIP6...)
	copy(ip[net.IPv6len-len(suffix):], suffix)
	a, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip6/%s/tcp/4242", ip))
	if err != nil {
		return nil, fmt.Errorf("create test multiaddr: %w", err)
	}
	return a, nil
}

func StartTestNetwork(ctx context.Context, t *testing.T, n int, conf map[int]HostDescr, validators []p2p.GossipValidator, logger types.Logger) TestNet {
	t.Helper()
	require := require.New(t)

	mnet := mocknet.New()
	for i := 0; i < n; i++ {
		var descr HostDescr
		if d, ok := conf[i]; ok {
			descr = d
		}
		if descr.RealKey {
			privKey, _, _ := crypto.GenerateEd25519Key(rand.Reader)
			addr, err := getAddr(privKey)
			require.NoError(err)
			_, err = mnet.AddPeer(privKey, addr)
			require.NoError(err)
		} else {
			_, err := mnet.GenPeer()
			require.NoError(err)
		}
	}

	err := mnet.LinkAll()
	require.NoError(err)

	// prepare seed node lists
	seeds := make([]string, n)
	for src, descr := range conf {
		require.Less(src, n)
		for _, dst := range descr.Conns {
			require.Less(dst, n)
			seeds[src] += mnet.Hosts()[dst].Addrs()[0].String() + "/p2p/" + mnet.Peers()[dst].String() + ","
		}
		seeds[src] = strings.TrimSuffix(seeds[src], ",")
	}

	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	require.NoError(err)

	clients := make([]*p2p.Client, n)
	store := store.New(store.NewDefaultInMemoryKVStore())
	for i := 0; i < n; i++ {
		client, err := p2p.NewClient(config.P2PConfig{
			BootstrapNodes:               seeds[i],
			GossipSubCacheSize:           50,
			BootstrapRetryTime:           30 * time.Second,
			BlockSyncRequestIntervalTime: 30 * time.Second,
			ListenAddress:                config.DefaultListenAddress,
			BlockSyncEnabled:             true,
		},
			mnet.Hosts()[i].Peerstore().PrivKey(mnet.Hosts()[i].ID()),
			conf[i].ChainID,
			store,
			pubsubServer, datastore.NewMapDatastore(),
			logger)
		require.NoError(err)
		require.NotNil(client)

		client.SetTxValidator(validators[i])
		clients[i] = client
	}

	for i, c := range clients {
		err := c.StartWithHost(ctx, mnet.Hosts()[i])
		require.NoError(err)
	}

	return clients
}
