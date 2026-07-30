package server

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
)

// Middleware wraps a handler for a given route pattern, returning a new handler.
// The chain runs outermost-first: panic -> global -> route-specific -> handler.
type Middleware func(string, http.Handler) http.Handler

// PanicHandler turns a recovered panic value into an APIError. recover returns
// any, so the recovered value may be of any type. Returning nil is treated as a
// generic internal error.
type PanicHandler func(ctx context.Context, recovered any) APIError

func panicMiddleware(panicHandler PanicHandler, errorHandler ErrorHandler) Middleware {
	return func(_ string, next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			buf := newBufferedResponseWriter()

			defer func() {
				rec := recover()
				if rec == nil {
					buf.flush(w) // no panic: send the captured response as-is
					return
				}

				apiErr := panicHandler(req.Context(), rec)
				if apiErr == nil {
					apiErr = ErrInternal("panic handler internal error")
				}

				errorHandler(w, req, apiErr) // buffered partial output discarded
			}()

			next.ServeHTTP(buf, req)
		})
	}
}

func (o *serverOptions) defaultPanicHandler() {
	o.panicHandler = func(ctx context.Context, recovered any) APIError {
		if o.logger != nil {
			err, ok := recovered.(error)
			if !ok {
				err = fmt.Errorf("panic: %v", recovered)
			}
			ctx = o.logger.WithError(ctx, err)
			o.logger.Error(ctx, "recovered panic\n"+string(debug.Stack()))
		}
		return ErrInternal("internal server error")
	}
}
