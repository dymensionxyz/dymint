package json

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/dymensionxyz/dymint/types"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
)

type wsConn struct {
	conn   *websocket.Conn
	queue  chan []byte
	logger types.Logger
}

func (wsc *wsConn) sendLoop() {
	for msg := range wsc.queue {
		writer, err := wsc.conn.NextWriter(websocket.TextMessage)
		if err != nil {
			wsc.logger.Error("failed to create writer", "error", err)
			continue
		}
		_, err = writer.Write(msg)
		if err != nil {
			wsc.logger.Error("failed to write message", "error", err)
		}
		if err = writer.Close(); err != nil {
			wsc.logger.Error("failed to close writer", "error", err)
		}
	}
}

func (h *handler) wsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO(tzdybal): configuration options
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	wsc, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("failed to update to WebSocket connection", "error", err)
		return
	}
	remoteAddr := wsc.RemoteAddr().String()
	defer func() {
		err := wsc.Close()
		if err != nil {
			h.logger.Error("failed to close WebSocket connection", "err")
		}
	}()

	ws := &wsConn{
		conn:   wsc,
		queue:  make(chan []byte),
		logger: h.logger,
	}
	go ws.sendLoop()

	for {
		mt, r, err := wsc.NextReader()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				h.logger.Debug("WebSocket connection closed")
				err := h.srv.client.EventBus.UnsubscribeAll(context.Background(), remoteAddr)
				if err != nil && err != tmpubsub.ErrSubscriptionNotFound {
					h.logger.Error("Failed to unsubscribe addr from events", "addr", remoteAddr, "err", err)
				}
			} else {
				h.logger.Error("failed to read next WebSocket message", "error", err)
			}
			break
		}

		if mt != websocket.TextMessage {
			// TODO(tzdybal): https://github.com/dymensionxyz/dymint/issues/465
			h.logger.Debug("expected text message")
			continue
		}
		req, err := http.NewRequest(http.MethodGet, "", r)
		req.RemoteAddr = remoteAddr
		if err != nil {
			h.logger.Error("failed to create request", "error", err)
			continue
		}

		writer := new(bytes.Buffer)
		h.serveJSONRPCforWS(newResponseWriter(writer), req, ws)
		ws.queue <- writer.Bytes()
	}
}

func newResponseWriter(w io.Writer) http.ResponseWriter {
	return &wsResponse{w}
}

// wsResponse is a simple implementation of http.ResponseWriter
type wsResponse struct {
	w io.Writer
}

var _ http.ResponseWriter = wsResponse{}

// Write use underlying writer to write response to WebSocket
func (w wsResponse) Write(bytes []byte) (int, error) {
	return w.w.Write(bytes)
}

func (w wsResponse) Header() http.Header {
	return http.Header{}
}

func (w wsResponse) WriteHeader(statusCode int) {
}
