package server

import (
	"context"
	"fmt"
	"log/slog"
)

// Logger is the minimal logging surface the runtime needs; supply one via
// WithLogger, or rely on the slog-backed default.
type Logger interface {
	Error(ctx context.Context, args ...any)
	WithError(ctx context.Context, err error) context.Context
}

// ctxErrKey is the private context key under which WithError stashes an error so
// that a subsequent Error call can attach it to the log record.
type ctxErrKey struct{}

// defaultLogger is the Logger used when the caller does not provide one via
// WithLogger. It writes structured error records to slog's default handler.
type defaultLogger struct {
	l *slog.Logger
}

func (o *serverOptions) defaultLogger() {
	o.logger = defaultLogger{l: slog.Default()}
}

func (d defaultLogger) WithError(ctx context.Context, err error) context.Context {
	return context.WithValue(ctx, ctxErrKey{}, err)
}

func (d defaultLogger) Error(ctx context.Context, args ...any) {
	msg := fmt.Sprint(args...)

	if err, ok := ctx.Value(ctxErrKey{}).(error); ok && err != nil {
		d.l.ErrorContext(ctx, msg, slog.Any("error", err))
		return
	}

	d.l.ErrorContext(ctx, msg)
}
