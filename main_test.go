package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
//  Test Helpers
// ============================================================================

// fakeAllowlistProvider is a mock implementation of AllowlistProvider for testing.
type fakeAllowlistProvider struct {
	data    []byte
	hash    string
	getErr  error
	hashErr error
}

// Get returns predefined data or error for testing purposes.
func (f *fakeAllowlistProvider) Get(_ context.Context) ([]byte, error) {
	return f.data, f.getErr
}

func (f *fakeAllowlistProvider) Hash() (string, error) {
	return f.hash, f.hashErr
}

// ============================================================================
//  Tests for ServerConfig
// ============================================================================

func TestDefaultServerConfig(t *testing.T) {
	t.Parallel()

	conf := DefaultServerConfig()

	require.NotEmpty(t, conf)
	assert.Equal(t, hostDefault, conf.Host)
	assert.Equal(t, portDefault, conf.Port)
	assert.Equal(t, readHeaderTimeout, conf.ReadHeaderTimeout)
	assert.Equal(t, readTimeout, conf.ReadTimeout)
	assert.Equal(t, writeTimeout, conf.WriteTimeout)
	assert.Equal(t, idleTimeout, conf.IdleTimeout)
	assert.Equal(t, shutdownTimeout, conf.ShutdownTimeout)
}

func TestServerConfig_Addr(t *testing.T) {
	t.Parallel()

	conf := DefaultServerConfig()
	conf.Host = "example.com"
	conf.Port = "8080"

	assert.Equal(t, "example.com:8080", conf.Addr())
}

// ============================================================================
//  Tests for StaticAllowlistProvider
// ============================================================================

func TestStaticAllowlistProvider_Get(t *testing.T) {
	t.Parallel()

	prov := new(StaticAllowlistProvider)

	data, err := prov.Get(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []byte(allowlist), data)
}

func TestStaticAllowlistProvider_Get_canceled_context(t *testing.T) {
	t.Parallel()

	prov := new(StaticAllowlistProvider)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	data, err := prov.Get(ctx)

	require.Error(t, err)
	assert.Nil(t, data)
	require.ErrorIs(t, err, context.Canceled, "expected context.Canceled")
	assert.Contains(t, err.Error(), "context retrieval failed")
}

// ============================================================================
//  Tests for newAllowlistHandler
// ============================================================================

func TestNewAllowlistHandler_success(t *testing.T) {
	t.Parallel()

	data := "example.com\ngoogle.com\n"

	prov := &fakeAllowlistProvider{
		data:    []byte(data),
		hash:    fastHash(data),
		getErr:  nil,
		hashErr: nil,
	}
	handler := newAllowlistHandler(prov)

	req := httptest.NewRequest(http.MethodGet, "/allowlist.txt", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "example.com\ngoogle.com\n", rec.Body.String())
}

var errDatabaseConnection = errors.New("database connection failed")

func TestNewAllowlistHandler_hash_error(t *testing.T) {
	t.Parallel()

	prov := &fakeAllowlistProvider{
		data:    nil,
		hash:    "",
		getErr:  nil,
		hashErr: errDatabaseConnection,
	}
	handler := newAllowlistHandler(prov)

	req := httptest.NewRequest(http.MethodGet, "/allowlist.txt", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to get ETag")
}

func TestNewAllowlistHandler_get_error(t *testing.T) {
	t.Parallel()

	prov := &fakeAllowlistProvider{
		data:    nil,
		hash:    "somehash",
		getErr:  errDatabaseConnection,
		hashErr: nil,
	}
	handler := newAllowlistHandler(prov)

	req := httptest.NewRequest(http.MethodGet, "/allowlist.txt", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to load allowlist")
}

func TestNewAllowlistHandler_method_not_allowed(t *testing.T) {
	t.Parallel()

	prov := &fakeAllowlistProvider{
		data:    []byte("example.com\n"),
		hash:    "somehash",
		getErr:  nil,
		hashErr: nil,
	}
	handler := newAllowlistHandler(prov)

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(method, "/allowlist.txt", nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
			assert.Contains(t, rec.Body.String(), "method not allowed")
		})
	}
}

func TestNewAllowlistHandler_if_none_match(t *testing.T) {
	t.Parallel()

	data := "example.com\n"
	hash := fastHash(data)

	prov := &fakeAllowlistProvider{
		data:    []byte(data),
		hash:    hash,
		getErr:  nil,
		hashErr: nil,
	}
	handler := newAllowlistHandler(prov)

	tests := []struct {
		name           string
		ifNoneMatch    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "ETag matches - return 304",
			ifNoneMatch:    `"` + hash + `"`,
			expectedStatus: http.StatusNotModified,
			expectedBody:   "",
		},
		{
			name:           "ETag does not match - return 200",
			ifNoneMatch:    `"different-hash"`,
			expectedStatus: http.StatusOK,
			expectedBody:   data,
		},
		{
			name:           "No If-None-Match header - return 200",
			ifNoneMatch:    "",
			expectedStatus: http.StatusOK,
			expectedBody:   data,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/allowlist.txt", nil)
			if test.ifNoneMatch != "" {
				req.Header.Set("If-None-Match", test.ifNoneMatch)
			}

			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.Equal(t, test.expectedStatus, rec.Code)
			assert.Equal(t, test.expectedBody, rec.Body.String())
		})
	}
}

// ============================================================================
//  Tests for wrapError
// ============================================================================

var errOriginal = errors.New("original error")

func TestWrapError_with_error(t *testing.T) {
	t.Parallel()

	wrappedErr := wrapError(errOriginal, "operation failed")

	require.Error(t, wrappedErr)
	assert.Contains(t, wrappedErr.Error(), "operation failed")
	assert.Contains(t, wrappedErr.Error(), "original error")
	assert.ErrorIs(t, wrappedErr, errOriginal)
}

func TestWrapError_nil_error(t *testing.T) {
	t.Parallel()

	wrappedErr := wrapError(nil, "operation failed")

	assert.NoError(t, wrappedErr)
}

// Error determination (error.Is) test for context.Canceled.
func TestWrapError_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel the context

	err := wrapError(ctx.Err(), "context retrieval failed")

	require.ErrorIs(t, err, context.Canceled, "expected context.Canceled")
}

// Error determination (error.Is) test for context.DeadlineExceeded.
func TestWrapError_DeadlineExceeded(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(1 * time.Millisecond)

	err := wrapError(ctx.Err(), "timeout")

	require.ErrorIs(t, err, context.DeadlineExceeded, "expected context.DeadlineExceeded")
}

// ============================================================================
//  Tests for run
// ============================================================================

func TestRun_graceful_shutdown(t *testing.T) {
	t.Parallel()

	prov := &fakeAllowlistProvider{
		data:    []byte("test.com\n"),
		hash:    fastHash("test.com\n"),
		getErr:  nil,
		hashErr: nil,
	}
	conf := DefaultServerConfig()
	conf.Port = "0" // Use random available port
	conf.ShutdownTimeout = 1 * time.Second

	quit := make(chan os.Signal, 1)

	done := make(chan error, 1)

	go func() {
		done <- run(prov, conf, quit)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Trigger shutdown
	quit <- os.Interrupt

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("run did not return in time")
	}
}
