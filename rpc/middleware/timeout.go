package middleware

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

func NewTimeoutHandler(handler http.Handler, timeout time.Duration) http.Handler {
	return &timeoutHandler{
		handler: handler,
		timeout: timeout,
	}
}

type timeoutHandler struct {
	handler http.Handler
	timeout time.Duration
}

func (h *timeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	r = r.WithContext(ctx)
	h.handler.ServeHTTP(w, r)
}

func (h *timeoutHandler) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := h.handler.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("the handler does not support Hijacker interface")
	}
	return hijacker.Hijack()
}
