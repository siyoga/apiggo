package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister_RoutesByMethodAndPattern(t *testing.T) {
	srv := NewServer()
	srv.Register(http.MethodGet, "/hello", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	// matching method + path
	resp, err := http.Get(ts.URL + "/hello")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "ok", string(body))

	// wrong method on a method-aware pattern -> 405
	resp2, err := http.Post(ts.URL+"/hello", "text/plain", nil)
	require.NoError(t, err)
	_ = resp2.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp2.StatusCode)
}

func TestRegister_MiddlewareChainOrder(t *testing.T) {
	var order []string

	srv := NewServer(WithMiddleware(
		recordMiddleware("global1", &order),
		recordMiddleware("global2", &order),
	))

	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})
	srv.Register(http.MethodGet, "/x", h, recordMiddleware("route", &order))

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/x")
	require.NoError(t, err)
	_ = resp.Body.Close()

	// outer -> inner: global(in order) -> route -> handler
	assert.Equal(t, []string{"global1", "global2", "route", "handler"}, order)
}

// helloIn/helloOut and helloDescriptor emulate the generated dto + descriptor.
type helloIn struct{ Name string }
type helloOut struct {
	Message string `json:"message"`
}

func helloDescriptor(h func(context.Context, helloIn) (helloOut, error)) func(*Server) {
	return func(s *Server) {
		s.Register(http.MethodGet, "/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			out, err := h(r.Context(), helloIn{Name: r.URL.Query().Get("name")})
			if err != nil {
				s.HandleError(w, r, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}))
	}
}

func TestWithOpenAPIMethod_SuccessPath(t *testing.T) {
	var gotName string
	handler := func(_ context.Context, in helloIn) (helloOut, error) {
		gotName = in.Name
		return helloOut{Message: "hi " + in.Name}, nil
	}

	srv := NewServer(WithOpenAPIMethod(helloDescriptor, handler))

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/hello?name=Bob")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Bob", gotName)
	assert.JSONEq(t, `{"message":"hi Bob"}`, string(body))
}

func TestWithOpenAPIMethod_ErrorPath(t *testing.T) {
	handler := func(_ context.Context, in helloIn) (helloOut, error) {
		if in.Name == "" {
			return helloOut{}, ErrNotFound("name is required")
		}
		return helloOut{Message: in.Name}, nil
	}

	srv := NewServer(WithOpenAPIMethod(helloDescriptor, handler))

	ts := httptest.NewServer(srv.mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/hello")
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, string(body), "name is required")
}

func TestHandleError_UsesConfiguredHandler(t *testing.T) {
	srv := NewServer(WithErrorHandler(func(w http.ResponseWriter, _ *http.Request, err error) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = io.WriteString(w, err.Error())
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	srv.HandleError(rec, req, errors.New("boom"))

	assert.Equal(t, http.StatusTeapot, rec.Code)
	assert.Equal(t, "boom", rec.Body.String())
}

func TestServe_StartupErrorReturned(t *testing.T) {
	// Occupy a port, then try to Serve on the same address -> bind error.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	srv := NewServer()
	err = srv.Serve(context.Background(), l.Addr().String())

	require.Error(t, err, "Serve must return the ListenAndServe bind error")
}

func TestServe_GracefulShutdownOnCtxCancel(t *testing.T) {
	addr := freeAddr(t)

	srv := NewServer(WithShutdownTimeout(2 * time.Second))
	srv.Register(http.MethodGet, "/ping", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ctx, addr) }()

	waitServer(t, "http://"+addr+"/ping")
	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err, "graceful shutdown must return nil")
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after ctx cancel")
	}
}

func TestServe_BaseContextCancelsInFlightHandler(t *testing.T) {
	addr := freeAddr(t)

	entered := make(chan struct{})
	cancelled := make(chan struct{})

	srv := NewServer(WithShutdownTimeout(2 * time.Second))
	srv.Register(http.MethodGet, "/ping", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv.Register(http.MethodGet, "/block", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done() // BaseContext == Serve ctx, so cancel() unblocks this
		close(cancelled)
		w.WriteHeader(http.StatusOK)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ctx, addr) }()

	waitServer(t, "http://"+addr+"/ping")

	// fire the blocking request; it stays in the handler until ctx is cancelled
	go func() {
		resp, err := http.Get("http://" + addr + "/block")
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	mustReceive(t, entered, 3*time.Second, "handler was never entered")
	cancel()
	mustReceive(t, cancelled, 3*time.Second, "handler's request context was not cancelled by BaseContext")

	select {
	case <-errCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after ctx cancel")
	}
}
