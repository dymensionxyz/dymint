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
			continue
		}
		_ = wsc.conn.SetWriteDeadline(time.Now().Add(writeWait))
		_, err = writer.Write(msg)
		if err != nil {
		}
		if err = writer.Close(); err != nil {
		}
	}
}

func (h *handler) wsHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	wsc, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
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
		}
	}()

	go ws.sendLoop()

	remoteAddr := ws.conn.RemoteAddr().String()

	for {
		mt, rdr, err := wsc.NextReader()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
			} else {
			}
			err := h.srv.client.EventBus.UnsubscribeAll(r.Context(), remoteAddr)
			if err != nil && !errors.Is(err, tmpubsub.ErrSubscriptionNotFound) {
			}
			break
		}

		if mt != websocket.TextMessage {
			continue
		}
		req, err := http.NewRequest(http.MethodGet, "", rdr)
		if err != nil {
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

type wsResponse struct {
	w io.Writer
}

var _ http.ResponseWriter = wsResponse{}

func (w wsResponse) Write(bytes []byte) (int, error) {
	return w.w.Write(bytes)
}

func (w wsResponse) Header() http.Header {
	return http.Header{}
}

func (w wsResponse) WriteHeader(statusCode int) {
}
