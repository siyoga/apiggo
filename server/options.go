package server

import (
	"context"
	"time"
)

// optionFunc adapts a plain function into an Option.
type optionFunc func(*serverOptions)

func (optFunc optionFunc) server(servOpt *serverOptions) {
	optFunc(servOpt)
}

// Option configures a Server. Every With* constructor returns one.
type Option interface {
	server(*serverOptions)
}

type serverOptions struct {
	readTimeout     time.Duration
	idleTimeout     time.Duration
	writeTimeout    time.Duration
	shutdownTimeout time.Duration

	logger      Logger
	middlewares []Middleware
	methods     []func(*Server)

	panicHandler PanicHandler
	errorHandler ErrorHandler
}

func newServerOptions(opts ...Option) serverOptions {
	o := serverOptions{}
	o.defaultLogger()
	o.defaultPanicHandler()
	o.defaultErrorHandler()

	for _, opt := range opts {
		opt.server(&o)
	}

	return o
}

// RecommendedServerOptions bundles the settings applied by
// WithRecommendedServerSettings. Any nil/zero field is left at its default.
type RecommendedServerOptions struct {
	Logger       Logger
	Middlewares  []Middleware
	PanicHandler PanicHandler
	ErrorHandler ErrorHandler
}

// WithRecommendedServerSettings applies a bundle of options in one call, skipping
// any field left nil so the server keeps its default for that setting.
func WithRecommendedServerSettings(recOpts RecommendedServerOptions) Option {
	return optionFunc(func(servOpt *serverOptions) {
		if recOpts.Logger != nil {
			WithLogger(recOpts.Logger).server(servOpt)
		}

		if recOpts.Middlewares != nil {
			WithMiddleware(recOpts.Middlewares...).server(servOpt)
		}

		if recOpts.PanicHandler != nil {
			WithPanicHandler(recOpts.PanicHandler).server(servOpt)
		}

		if recOpts.ErrorHandler != nil {
			WithErrorHandler(recOpts.ErrorHandler).server(servOpt)
		}
	})
}

// WithLogger sets the logger the runtime uses to report recovered panics and
// internal errors.
func WithLogger(l Logger) Option {
	return optionFunc(func(servOpt *serverOptions) {
		servOpt.logger = l
	})
}

// WithMiddleware registers global middleware applied to every route, outermost
// first. Repeated calls append to the chain rather than replacing it.
func WithMiddleware(mw ...Middleware) Option {
	return optionFunc(func(servOpt *serverOptions) {
		servOpt.middlewares = append(servOpt.middlewares, mw...)
	})
}

// WithReadTimeout sets http.Server.ReadTimeout (zero means no timeout).
func WithReadTimeout(t time.Duration) Option {
	return optionFunc(func(servOpt *serverOptions) {
		servOpt.readTimeout = t
	})
}

// WithIdleTimeout sets http.Server.IdleTimeout (zero means no timeout).
func WithIdleTimeout(t time.Duration) Option {
	return optionFunc(func(servOpt *serverOptions) {
		servOpt.idleTimeout = t
	})
}

// WithWriteTimeout sets http.Server.WriteTimeout (zero means no timeout).
func WithWriteTimeout(t time.Duration) Option {
	return optionFunc(func(servOpt *serverOptions) {
		servOpt.writeTimeout = t
	})
}

// WithShutdownTimeout caps how long graceful shutdown waits for in-flight
// requests to drain (zero means wait indefinitely).
func WithShutdownTimeout(t time.Duration) Option {
	return optionFunc(func(servOpt *serverOptions) {
		servOpt.shutdownTimeout = t
	})
}

// WithPanicHandler overrides the handler invoked when a recovered panic reaches
// the panic middleware.
func WithPanicHandler(h PanicHandler) Option {
	return optionFunc(func(servOpt *serverOptions) {
		servOpt.panicHandler = h
	})
}

// WithErrorHandler overrides how errors returned by handlers are mapped to HTTP
// responses.
func WithErrorHandler(h ErrorHandler) Option {
	return optionFunc(func(servOpt *serverOptions) {
		servOpt.errorHandler = h
	})
}

// WithOpenAPIMethod registers one generated operation. method is the generated
// registrar: given the typed handler it returns a func(*Server) that builds the
// adapter http.Handler and calls s.Register. The registrar is stashed here and run
// in NewServer once the *Server exists.
func WithOpenAPIMethod[IN, OUT any](
	method func(func(ctx context.Context, in IN) (OUT, error)) func(*Server),
	handler func(ctx context.Context, in IN) (OUT, error),
) Option {
	return optionFunc(func(servOpt *serverOptions) {
		servOpt.methods = append(servOpt.methods, method(handler))
	})
}
