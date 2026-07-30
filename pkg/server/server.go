package server

import (
	"context"
	"errors"
	"net"
	"net/http"
)

// Server hosts generated operations on a stdlib http.ServeMux, wiring each one
// through the configured middleware chain, panic recovery, and error handling.
type Server struct {
	options serverOptions
	mux     *http.ServeMux
	panicMw Middleware
}

// NewServer builds a Server from the given options and runs every registrar
// collected by WithOpenAPIMethod, wiring each operation onto the mux.
func NewServer(opts ...Option) *Server {
	servOpts := newServerOptions(opts...)

	srv := &Server{
		mux:     http.NewServeMux(),
		panicMw: panicMiddleware(servOpts.panicHandler, servOpts.errorHandler),
		options: servOpts,
	}

	// Run the registrars collected by WithOpenAPIMethod now that the *Server
	// exists: each builds its adapter handler and calls srv.Register to wire the
	// middleware chain and register on the mux.
	for _, register := range servOpts.methods {
		register(srv)
	}

	return srv
}

// Register wraps h in the middleware chain and registers it on the mux under a
// method-aware pattern ("POST /users/{id}"). The chain runs outside-in as
// panic -> global -> route-specific -> handler, so panic recovery catches
// panics from every middleware, not just the handler. The generated HTTP adapter
// builds h (parse in -> handler -> render out/APIError) and hands it here; the
// runtime owns middleware assembly, the adapter knows nothing about it.
func (s *Server) Register(method, pattern string, h http.Handler, routeMws ...Middleware) {
	// route-specific → global → panic (lifo order)
	for i := len(routeMws) - 1; i >= 0; i-- {
		h = routeMws[i](pattern, h)
	}
	for i := len(s.options.middlewares) - 1; i >= 0; i-- {
		h = s.options.middlewares[i](pattern, h)
	}
	h = s.panicMw(pattern, h) // panic outside of everything

	s.mux.Handle(method+" "+pattern, h)
}

// HandleError renders err as an HTTP response through the configured
// ErrorHandler (APIError typecast + unknown-error -> 500 fallback + render). The
// generated adapter calls this on any error a handler returns, so the normal-error
// path and the panic path go through the same handler.
func (s *Server) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	s.options.errorHandler(w, r, err)
}

// Serve starts an HTTP server on addr with the configured timeouts and blocks
// until ctx is cancelled or the server fails to start. On cancellation it
// gracefully shuts down, bounded by the shutdown timeout (unbounded if unset).
// Cancellation is the caller's job, e.g. signal.NotifyContext.
func (s *Server) Serve(ctx context.Context, addr string) error {
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  s.options.readTimeout,
		WriteTimeout: s.options.writeTimeout,
		IdleTimeout:  s.options.idleTimeout,
		// Every request context derives from ctx, so cancelling ctx signals all
		// in-flight handlers to wind down; Shutdown below then drains within the
		// budget. Handlers that respect ctx exit fast, slow ones ride out
		// shutdownTimeout.
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	serveErr := make(chan error, 1)
	go func() {
		err := httpSrv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil // graceful shutdown, not a failure
		}
		serveErr <- err
	}()

	select {
	case err := <-serveErr: // ListenAndServe returned before ctx was cancelled
		return err
	case <-ctx.Done():
	}

	shutdownCtx := context.Background()
	if s.options.shutdownTimeout > 0 {
		var cancel context.CancelFunc
		shutdownCtx, cancel = context.WithTimeout(shutdownCtx, s.options.shutdownTimeout)
		defer cancel()
	}

	return httpSrv.Shutdown(shutdownCtx)
}
