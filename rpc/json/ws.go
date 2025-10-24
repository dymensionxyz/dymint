package json

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/types"
)

type wsConn struct {
	conn   *websocket.Conn
	queue  chan []byte
	logger types.Logger
}

const writeWait = 10 * time.Second

func (wsc *wsConn) sendLoop() {
	for msg := range wsc.queue {
		writer, err := wsc.conn.NextWriter(websocket.TextMessage)
		if err != nil {
			wsc.logger.Error("create writer", "error", err)
			continue
		}
		_ = wsc.conn.SetWriteDeadline(time.Now().Add(writeWait))
		_, err = writer.Write(msg)
		if err != nil {
			wsc.logger.Error("write message", "error", err)
		}
		if err = writer.Close(); err != nil {
			wsc.logger.Error("close writer", "error", err)
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
		h.logger.Error("update to WebSocket connection", "error", err)
		return
	}

	ws := &wsConn{
		conn:   wsc,
		queue:  make(chan []byte),
		logger: h.logger,
	}

	defer func() {
		close(ws.queue)

		if err := ws.conn.Close(); err != nil {
			h.logger.Error("close WebSocket connection", "err")
		}
	}()

	go ws.sendLoop()

	remoteAddr := ws.conn.RemoteAddr().String()

	for {
		mt, rdr, err := wsc.NextReader()
		if err != nil {
			e := &websocket.CloseError{}
			if errors.As(err, &e) {
				h.logger.Debug("WebSocket connection closed", "reason", err)
			} else {
				h.logger.Error("read next WebSocket message", "error", err)
			}
			err := h.srv.client.EventBus.UnsubscribeAll(r.Context(), remoteAddr)
			if err != nil && !errors.Is(err, tmpubsub.ErrSubscriptionNotFound) {
				h.logger.Error("unsubscribe addr from events", "addr", remoteAddr, "err", err)
			}
			break
		}

		if mt != websocket.TextMessage {
			// TODO(tzdybal): https://github.com/dymensionxyz/dymint/issues/465
			h.logger.Debug("expected text message")
			continue
		}
		req, err := http.NewRequest(http.MethodGet, "", rdr)
		if err != nil {
			h.logger.Error("create request", "error", err)
			continue
		}

		req.RemoteAddr = remoteAddr

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
