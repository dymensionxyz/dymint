package rpc

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	uevent "github.com/dymensionxyz/dymint/utils/event"

	"github.com/rs/cors"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub"
	"github.com/tendermint/tendermint/libs/service"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	"golang.org/x/net/netutil"

	"github.com/dymensionxyz/dymint/node"
	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/rpc/client"
	"github.com/dymensionxyz/dymint/rpc/json"
	"github.com/dymensionxyz/dymint/rpc/middleware"
)

type Server struct {
	*service.BaseService

	config   *config.RPCConfig
	client   *client.Client
	node     *node.Node
	listener net.Listener

	server http.Server

	health   error
	healthMU sync.RWMutex
}

const (
	onStopTimeout = 5 * time.Second

	readHeaderTimeout = 5 * time.Second
)

type Option func(*Server)

func WithListener(listener net.Listener) Option {
	return func(d *Server) {
		d.listener = listener
	}
}

func NewServer(node *node.Node, config *config.RPCConfig, logger log.Logger, options ...Option) *Server {
	srv := &Server{
		config: config,
		client: client.NewClient(node),
		node:   node,
	}
	srv.BaseService = service.NewBaseService(logger, "RPC", srv)

	for _, option := range options {
		option(srv)
	}
	return srv
}

func (s *Server) Client() rpcclient.Client {
	return s.client
}

func (s *Server) PubSubServer() *pubsub.Server {
	return s.node.PubSubServer()
}

func (s *Server) OnStart() error {
	s.startEventListener()
	return s.startRPC()
}

func (s *Server) OnStop() {
	ctx, cancel := context.WithTimeout(context.Background(), onStopTimeout)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		s.Logger.Error("while shutting down RPC server", "error", err)
	}
}

func (s *Server) startEventListener() {
	go uevent.MustSubscribe(context.Background(), s.PubSubServer(), "RPCNodeHealthStatusHandler", events.QueryHealthStatus, s.onNodeHealthUpdate, s.Logger)
}

func (s *Server) onNodeHealthUpdate(event pubsub.Message) {
	eventData, _ := event.Data().(*events.DataHealthStatus)
	if eventData.Error != nil {
		s.Logger.Error("node is unhealthy", "error", eventData.Error)
	}
	s.healthMU.Lock()
	defer s.healthMU.Unlock()
	s.health = eventData.Error
}

func (s *Server) getHealthStatus() error {
	s.healthMU.RLock()
	defer s.healthMU.RUnlock()

	return s.health
}

func (s *Server) startRPC() error {
	if s.config.ListenAddress == "" {
		s.Logger.Info("Listen address not specified - RPC will not be exposed")
		return nil
	}
	parts := strings.SplitN(s.config.ListenAddress, "://", 2)
	if len(parts) != 2 {
		return errors.New("invalid RPC listen address: expecting tcp://host:port")
	}
	proto := parts[0]
	addr := parts[1]

	var listener net.Listener
	if s.listener == nil {
		var err error
		listener, err = net.Listen(proto, addr)
		if err != nil {
			return err
		}
	} else {
		listener = s.listener
	}

	if s.config.MaxOpenConnections != 0 {
		s.Logger.Debug("limiting number of connections", "limit", s.config.MaxOpenConnections)
		listener = netutil.LimitListener(listener, s.config.MaxOpenConnections)
	}

	handler, err := json.GetHTTPHandler(s.client, s.Logger)
	if err != nil {
		return err
	}

	if s.config.IsCorsEnabled() {
		s.Logger.Debug("CORS enabled",
			"origins", s.config.CORSAllowedOrigins,
			"methods", s.config.CORSAllowedMethods,
			"headers", s.config.CORSAllowedHeaders,
		)
		c := cors.New(cors.Options{
			AllowedOrigins: s.config.CORSAllowedOrigins,
			AllowedMethods: s.config.CORSAllowedMethods,
			AllowedHeaders: s.config.CORSAllowedHeaders,
		})
		handler = c.Handler(handler)
	}

	reg := middleware.GetRegistry()
	reg.Register(middleware.Status{Err: s.getHealthStatus})
	middlewareClient := middleware.NewClient(*reg, s.Logger.With("module", "rpc/middleware"))
	handler = middlewareClient.Handle(handler)

	go func() {
		err := s.serve(listener, handler)
		if !errors.Is(err, http.ErrServerClosed) {
			s.Logger.Error("while serving HTTP", "error", err)
		}
	}()

	return nil
}

func (s *Server) serve(listener net.Listener, handler http.Handler) error {
	s.Logger.Info("serving HTTP", "listen address", listener.Addr())
	s.server = http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	if s.config.TLSCertFile != "" && s.config.TLSKeyFile != "" {
		return s.server.ServeTLS(listener, s.config.CertFile(), s.config.KeyFile())
	}
	return s.server.Serve(listener)
}
