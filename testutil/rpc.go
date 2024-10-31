package testutil

import (
	"fmt"
	"net"
	"testing"

	"github.com/dymensionxyz/dymint/rpc"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"golang.org/x/net/nettest"
)

func CreateLocalServer(t *testing.T) (*rpc.Server, net.Listener) {
	// Create a new local listener
	listener, err := nettest.NewLocalListener("tcp")
	require.NoError(t, err)
	serverReadyCh := make(chan bool, 1)

	var server *rpc.Server
	// Start server with listener
	go func() {
		node, err := CreateNode(true, nil, GenerateGenesis(0))
		require.NoError(t, err)
		err = node.Start()
		require.NoError(t, err)
		rpcConfig := &config.RPCConfig{ListenAddress: fmt.Sprintf("tcp://%s", listener.Addr().String())}
		options := []rpc.Option{rpc.WithListener(listener)}
		server = rpc.NewServer(node, rpcConfig, log.TestingLogger(), options...)
		err = server.Start()
		require.NoError(t, err)
		serverReadyCh <- true
	}()

	<-serverReadyCh
	return server, listener
}
