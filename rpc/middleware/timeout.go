package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"
)

func NewHijackableTimeoutHandler(handler http.Handler, timeout time.Duration) *hijackableTimeoutHandler {
	return &hijackableTimeoutHandler{
		Handler:     http.TimeoutHandler(handler, timeout, "Server Timeout"),
		nextHandler: handler,
	}
}

type hijackableTimeoutHandler struct {
	http.Handler
	nextHandler http.Handler
}

func (h *hijackableTimeoutHandler) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := h.nextHandler.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("the handler does not support Hijacker interface")
	}
	return hijacker.Hijack()
}
