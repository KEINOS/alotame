/*
Alotame ia a human-friendly local web app for allowlist-first Blocky DNS management.
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Sample allowlist data (will be replaced with actual data source in the future).
const allowlist = `exampleA.com
yahoo.co.jp
fonts.googleapis.com
`

// Default configuration values.
const (
	portDefault = "3965"
	hostDefault = "localhost"
)

// Server timeout configuration.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func allowlistHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(allowlist))
}

func main() {
	exitOnError(run())
}

func run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /allowlist.txt", allowlistHandler)

	addr := hostDefault + ":" + portDefault

	server := new(http.Server)

	server.Addr = addr
	server.Handler = mux
	server.ReadHeaderTimeout = readHeaderTimeout
	server.ReadTimeout = readTimeout
	server.WriteTimeout = writeTimeout
	server.IdleTimeout = idleTimeout

	// Start server in goroutine
	go func() {
		slog.Info("starting server", "addr", "http://"+addr+"/allowlist.txt")

		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error:", "error", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		return wrapError(err, "server forced to shutdown")
	}

	slog.Info("server stopped gracefully")

	return nil
}

func wrapError(err error, msg string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", msg, err)
	}

	return nil
}

func exitOnError(err error) {
	if err != nil {
		slog.Error("fatal error", "error", err)

		os.Exit(1)
	}
}
