/*
Alotame ia a human-friendly local web app for allowlist-first Blocky DNS management.
*/
package main

import (
	"context"
	"crypto/sha3"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/zeebo/xxh3"
)

// Sample allowlist data (will be replaced with actual data source in the future).
const allowlist = `
# Sample Allowlist
github.com
example.com
yahoo.com
`

// Default configuration values.
const (
	portDefault = "5963"
	// hostDefault = "localhost" // local run only.
	hostDefault = "0.0.0.0" // allow external access
)

// Server timeout configuration.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// ============================================================================
//  Types and Interfaces
// ============================================================================

// AllowlistProvider defines an interface for fetching allowlist data.
type AllowlistProvider interface {
	Get(ctx context.Context) ([]byte, error)
	Hash() (string, error)
}

// StaticAllowlistProvider returns a static allowlist data.
type StaticAllowlistProvider struct{}

// Get returns the static allowlist data.
func (prov *StaticAllowlistProvider) Get(ctx context.Context) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, wrapError(ctx.Err(), "context retrieval failed")
	}

	return []byte(allowlist), nil
}

// Hash returns the current hash of the allowlist data.
// This hash is used for cache validation and not for security purposes.
func (prov *StaticAllowlistProvider) Hash() (string, error) {
	return fastHash(allowlist), nil
}

// ServerConfig holds the configuration for the HTTP server.
type ServerConfig struct {
	Host              string
	Port              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// DefaultServerConfig returns the default server configuration.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:              hostDefault,
		Port:              portDefault,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ShutdownTimeout:   shutdownTimeout,
	}
}

// Addr returns the server address in "host:port" format.
func (c ServerConfig) Addr() string {
	return c.Host + ":" + c.Port
}

// ============================================================================
//  Main Function
// ============================================================================

func main() {
	prov := new(StaticAllowlistProvider)
	conf := DefaultServerConfig()
	quit := setupSignalHandler()

	exitOnError(run(prov, conf, quit))
}

// run starts the HTTP server and blocks until a quit signal is received or
// the server fails to start.
func run(prov AllowlistProvider, conf ServerConfig, quit <-chan os.Signal) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /allowlist.txt", newAllowlistHandler(prov))

	server := newHTTPServer(conf, mux)
	serverErr := make(chan error, 1)

	go startServer(server, conf.Addr(), serverErr)

	dummyLen := 16
	_ = secureHash("test", dummyLen) // dummy call to avoid unused function error. Will implement soon.

	select {
	case <-quit:
		slog.Info("shutting down server...")

		return shutdownServer(server, conf.ShutdownTimeout)
	case err := <-serverErr:
		return err
	}
}

// setupSignalHandler creates a channel that receives OS signals for graceful shutdown.
func setupSignalHandler() <-chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	return quit
}

// startServer runs the HTTP server and sends any error to the provided channel.
func startServer(server *http.Server, addr string, errCh chan<- error) {
	slog.Info("starting server", "addr", "http://"+addr+"/allowlist.txt")

	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server error:", "error", err)

		errCh <- err
	}

	close(errCh)
}

// shutdownServer gracefully shuts down the server with a timeout.
func shutdownServer(server *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		return wrapError(err, "server forced to shutdown")
	}

	slog.Info("server stopped gracefully")

	return nil
}

// ============================================================================
//  Helper Functions
// ============================================================================

func newHTTPServer(conf ServerConfig, handler http.Handler) *http.Server {
	srv := new(http.Server)

	srv.Addr = conf.Addr()
	srv.Handler = handler
	srv.ReadHeaderTimeout = conf.ReadHeaderTimeout
	srv.ReadTimeout = conf.ReadTimeout
	srv.WriteTimeout = conf.WriteTimeout
	srv.IdleTimeout = conf.IdleTimeout

	return srv
}

func newAllowlistHandler(prov AllowlistProvider) http.HandlerFunc {
	return func(respW http.ResponseWriter, req *http.Request) {
		respW.Header().Set("Content-Type", "text/plain; charset=utf-8")

		rawETag, err := prov.Hash()
		if err != nil {
			http.Error(respW, "failed to get ETag of allowlist",
				http.StatusInternalServerError)

			return
		}

		etag := `"` + rawETag + `"`

		// As of 2026-01-11, Blocky does not support ETag-based caching.
		// However, we implement it here for future compatibility.
		match := req.Header.Get("If-None-Match")
		if match == etag {
			respW.WriteHeader(http.StatusNotModified)

			return
		}

		respW.Header().Set("ETag", etag)
		respW.Header().Set("Cache-Control", "no-cache")

		data, err := prov.Get(req.Context())
		if err != nil {
			http.Error(respW, "failed to load allowlist",
				http.StatusInternalServerError)

			return
		}

		respW.WriteHeader(http.StatusOK)

		_, err = respW.Write(data)
		if err != nil {
			slog.Error("failed to write response", "error", err)

			return
		}

		slog.Info("served allowlist",
			"size", len(data), "remote_addr", req.RemoteAddr)
	}
}

// fastHash computes a fast hash of the given data using XXHash3.
func fastHash(data string) string {
	hashed := xxh3.HashString(data)

	return strconv.FormatUint(hashed, 16)
}

// secureHash computes a secure hash of the given data of specified length using
// SHAKE256 (SHA3 variant).
// If the given length is less than or equal to zero, it returns SHA3-256 hash
// (32 bytes) instead.
func secureHash(data string, length int) string {
	if length > 0 {
		return hex.EncodeToString(sha3.SumSHAKE256([]byte(data), length))
	}

	hashed := sha3.Sum256([]byte(data))

	return hex.EncodeToString(hashed[:])
}

func wrapError(err error, msg string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", msg, err)
}

func exitOnError(err error) {
	if err != nil {
		slog.Error("fatal error", "error", err)

		os.Exit(1)
	}
}
