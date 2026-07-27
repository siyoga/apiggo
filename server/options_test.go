package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServerResolvesDefaults(t *testing.T) {
	srv := NewServer()

	require.NotNil(t, srv)
	assert.NotNil(t, srv.mux, "mux must be initialized")
	assert.NotNil(t, srv.panicMw, "panic middleware must be built")
	assert.NotNil(t, srv.options.logger, "logger default must be resolved")
	assert.NotNil(t, srv.options.panicHandler, "panic handler default must be resolved")
	assert.NotNil(t, srv.options.errorHandler, "error handler default must be resolved")
}

func TestWithTimeouts(t *testing.T) {
	srv := NewServer(
		WithReadTimeout(1*time.Second),
		WithWriteTimeout(2*time.Second),
		WithIdleTimeout(3*time.Second),
		WithShutdownTimeout(4*time.Second),
	)

	assert.Equal(t, 1*time.Second, srv.options.readTimeout)
	assert.Equal(t, 2*time.Second, srv.options.writeTimeout)
	assert.Equal(t, 3*time.Second, srv.options.idleTimeout)
	assert.Equal(t, 4*time.Second, srv.options.shutdownTimeout)
}

func TestWithMiddlewareStored(t *testing.T) {
	srv := NewServer(WithMiddleware(
		recordMiddleware("a", nil),
		recordMiddleware("b", nil),
	))

	require.Len(t, srv.options.middlewares, 2)
}

func TestWithRecommendedServerSettings(t *testing.T) {
	customLogger := &stubLogger{}
	var calledErrH bool
	errH := func(w http.ResponseWriter, r *http.Request, err error) { calledErrH = true }

	srv := NewServer(WithRecommendedServerSettings(RecommendedServerOptions{
		Logger:       customLogger,
		Middlewares:  []Middleware{recordMiddleware("m", nil)},
		ErrorHandler: errH,
	}))

	assert.Same(t, customLogger, srv.options.logger)
	require.Len(t, srv.options.middlewares, 1)

	// the custom error handler was applied, not the default
	srv.options.errorHandler(nil, nil, nil)
	assert.True(t, calledErrH, "custom error handler must be wired")
}
