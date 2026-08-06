package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPanicMiddleware_NoPanic_FlushesResponseAsIs(t *testing.T) {
	o := newServerOptions()
	mw := panicMiddleware(o.panicHandler, o.errorHandler)

	h := mw("/x", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test", "1")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "hello")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "1", rec.Header().Get("X-Test"))
	assert.Equal(t, "hello", rec.Body.String())
}

func TestPanicMiddleware_Panic_DiscardsPartialAndMapsStatus(t *testing.T) {
	// Custom panic handler returns a Forbidden APIError to prove the status
	// comes from the panic handler, not a hardcoded 500.
	panicH := func(_ context.Context, _ any) APIError { return ErrForbidden("recovered") }
	o := newServerOptions()
	mw := panicMiddleware(panicH, o.errorHandler)

	h := mw("/x", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "partial output") // buffered, must be discarded
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.NotContains(t, rec.Body.String(), "partial output")
	assert.Contains(t, rec.Body.String(), "recovered")
}

func TestPanicMiddleware_NilFromHandler_FallsBackTo500(t *testing.T) {
	panicH := func(_ context.Context, _ any) APIError { return nil }
	o := newServerOptions()
	mw := panicMiddleware(panicH, o.errorHandler)

	h := mw("/x", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestDefaultPanicHandler_LogsAndReturns500(t *testing.T) {
	log := &stubLogger{}
	o := serverOptions{logger: log}
	o.defaultPanicHandler()

	ae := o.panicHandler(context.Background(), "boom")

	assert.Equal(t, http.StatusInternalServerError, ae.StatusCode())
	assert.NotEmpty(t, log.errors, "panic must be logged")
}
