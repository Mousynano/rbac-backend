// Package httpserver constructs and manages the HTTP server lifecycle.
// All timeouts and limits are set per the design specification.
package httpserver

import (
	"context"
	"net/http"
	"time"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
	maxHeaderBytes    = 16 * 1024 // 16 KiB
)

// New returns an *http.Server configured with all required timeouts and limits.
// The caller is responsible for calling ListenAndServe and Shutdown.
func New(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
	}
}

// GracefulShutdown drains in-flight requests with a 10-second deadline.
func GracefulShutdown(srv *http.Server) error {
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutCtx)
}
