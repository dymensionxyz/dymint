package rpc

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dymensionxyz/dymint/utilevent"

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
	"github.com/dymensionxyz/dymint/rpc/sharedtypes"
)

// Server handles HTTP and JSON-RPC requests, exposing Tendermint-compatible API.
type Server struct {
	*service.BaseService

	config       *config.RPCConfig
	client       *client.Client
	node         *node.Node
	healthStatus sharedtypes.HealthStatus
	listener     net.Listener
	ctx          context.Context

	server http.Server
}

// Option is a function that configures the Server.
type Option func(*Server)

// WithListener is an option that sets the listener.
func WithListener(listener net.Listener) Option {
	return func(d *Server) {
		d.listener = listener
	}
}

// NewServer creates new instance of Server with given configuration.
func NewServer(node *node.Node, config *config.RPCConfig, logger log.Logger, options ...Option) *Server {
	srv := &Server{
		config: config,
		client: client.NewClient(node),
		node:   node,
		healthStatus: sharedtypes.HealthStatus{
			IsHealthy: true,
			Error:     nil,
		},
		ctx: context.Background(),
	}
	srv.BaseService = service.NewBaseService(logger, "RPC", srv)

	// Apply options
	for _, option := range options {
		option(srv)
	}
	return srv
}

// Client returns a Tendermint-compatible rpc Client instance.
//
// This method is called in cosmos-sdk.
func (s *Server) Client() rpcclient.Client {
	return s.client
}

func (s *Server) PubSubServer() *pubsub.Server {
	return s.node.PubSubServer()
}

// OnStart is called when Server is started (see service.BaseService for details).
func (s *Server) OnStart() error {
	s.startEventListener()
	return s.startRPC()
}

// OnStop is called when Server is stopped (see service.BaseService for details).
func (s *Server) OnStop() {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		s.Logger.Error("while shuting down RPC server", "error", err)
	}
}

// EventListener registers events to callbacks.
func (s *Server) startEventListener() {
	go utilevent.MustSubscribe(s.ctx, s.PubSubServer(), "RPCNodeHealthStatusHandler", events.EventQueryHealthStatus, s.onHealthStatus, s.Logger)
}

// onHealthStatus is a callback function that handles health status events.
func (s *Server) onHealthStatus(event pubsub.Message) {
	eventData := event.Data().(*events.EventDataHealthStatus)
	if eventData.Error != nil {
		s.Logger.Error("node is unhealthy: got error health check from sublayer", "error", eventData.Error)
	}
	s.healthStatus.Set(eventData.Error)
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

	// Apply Middleware
	reg := middleware.GetRegistry()
	reg.Register(
		middleware.NewStatusMiddleware(&s.healthStatus),
	)
	middlewareClient := middleware.NewClient(*reg, s.Logger.With("module", "rpc/middleware"))
	handler = middlewareClient.Handle(handler)

	// Start HTTP server
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
	s.server = http.Server{Handler: handler} // #nosec
	if s.config.TLSCertFile != "" && s.config.TLSKeyFile != "" {
		return s.server.ServeTLS(listener, s.config.CertFile(), s.config.KeyFile())
	}
	return s.server.Serve(listener)
}
