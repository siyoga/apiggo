package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// recordMiddleware returns a Middleware that appends name to log (when non-nil)
// before delegating to next. Used to assert chain ordering.
func recordMiddleware(name string, log *[]string) Middleware {
	return func(_ string, next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if log != nil {
				*log = append(*log, name)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// stubLogger is a Logger that records error messages instead of writing them.
type stubLogger struct {
	mu     sync.Mutex
	errors []string
}

func (l *stubLogger) WithError(ctx context.Context, _ error) context.Context { return ctx }

func (l *stubLogger) Error(_ context.Context, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, fmt.Sprint(args...))
}

// freeAddr returns a currently-free loopback address to bind a test server to.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}

// waitServer polls url until it answers or the deadline passes.
func waitServer(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server did not become ready: %s", url)
}

// mustReceive fails the test if ch does not fire within timeout.
func mustReceive(t *testing.T, ch <-chan struct{}, timeout time.Duration, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(timeout):
		t.Fatal(msg)
	}
}
