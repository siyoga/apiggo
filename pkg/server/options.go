package server

import (
	"context"
	"time"
)

type optionFunc func(*serverOptions)

func (optFunc optionFunc) server(servOpt *serverOptions) {
	optFunc(servOpt)
}

type Option interface {
	server(*serverOptions)
}

type option struct {
	optionFunc
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

	enableServerLogs bool
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

type RecommendedServerOptions struct {
	Logger       Logger
	Middlewares  []Middleware
	PanicHandler PanicHandler
	ErrorHandler ErrorHandler
}

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

func WithLogger(l Logger) Option {
	return option{
		optionFunc: func(servOpt *serverOptions) {
			servOpt.enableServerLogs = true
			servOpt.logger = l

		},
	}
}

func WithMiddleware(mw ...Middleware) Option {
	return option{
		optionFunc: func(servOpt *serverOptions) {
			servOpt.middlewares = mw
		},
	}
}

func WithReadTimeout(t time.Duration) Option {
	return option{
		optionFunc: func(servOpt *serverOptions) {
			servOpt.readTimeout = t
		},
	}
}

func WithIdleTimeout(t time.Duration) Option {
	return option{
		optionFunc: func(servOpt *serverOptions) {
			servOpt.idleTimeout = t
		},
	}
}

func WithWriteTimeout(t time.Duration) Option {
	return option{
		optionFunc: func(servOpt *serverOptions) {
			servOpt.writeTimeout = t
		},
	}
}

func WithShutdownTimeout(t time.Duration) Option {
	return option{
		optionFunc: func(servOpt *serverOptions) {
			servOpt.shutdownTimeout = t
		},
	}
}

func WithPanicHandler(h PanicHandler) Option {
	return option{
		optionFunc: func(servOpt *serverOptions) {
			servOpt.panicHandler = h

		},
	}
}

func WithErrorHandler(h ErrorHandler) Option {
	return option{
		optionFunc: func(servOpt *serverOptions) {
			servOpt.errorHandler = h
		},
	}
}

// WithOpenApiMethod registers one generated operation. method is the generated
// registrar: given the typed handler it returns a func(*Server) that builds the
// adapter http.Handler and calls s.Register. The registrar is stashed here and run
// in NewServer once the *Server exists.
func WithOpenApiMethod[IN, OUT any](
	method func(func(ctx context.Context, in IN) (OUT, error)) func(*Server),
	handler func(ctx context.Context, in IN) (OUT, error),
) Option {
	return option{
		optionFunc: func(servOpt *serverOptions) {
			servOpt.methods = append(servOpt.methods, method(handler))
		},
	}
}
