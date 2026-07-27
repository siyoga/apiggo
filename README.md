# gopigen

A thin `net/http` runtime plus a code generator that turns an OpenAPI contract
into typed DTOs, transport glue, and handler stubs — so you only write business
logic inside `Handle(ctx, in) (out, error)`.

## Graceful shutdown

The server is started with:

```go
func (s *Server) Serve(ctx context.Context, addr string) error
```

`Serve` blocks until either the server fails to start or `ctx` is cancelled, and
then shuts down gracefully. **Cancellation is the caller's responsibility** —
typically a signal-bound context:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

srv := server.NewServer(
	server.WithShutdownTimeout(10*time.Second),
	server.WithOpenApiMethod(service.HelloWorld, h.Handle),
)

if err := srv.Serve(ctx, ":8080"); err != nil {
	log.Fatal(err)
}
```

### How it works

`Serve` wires two independent mechanisms off the same `ctx`:

1. **`http.Server.BaseContext` is set to `ctx`.** Every request's
   `r.Context()` — and therefore the `ctx` your handler receives in
   `Handle(ctx, in)` — is a child of it. Cancelling `ctx` **immediately signals
   every in-flight handler to wind down**. A handler that observes `ctx.Done()`
   returns at once.

2. **`http.Server.Shutdown` is called with a budget.** When `ctx` is cancelled,
   `Serve` calls `Shutdown` bounded by `WithShutdownTimeout`. This stops
   accepting new connections and waits for outstanding requests to drain within
   the budget. If the timeout is left unset (`0`), the drain is unbounded.

The combination gives you the best of both: handlers that respect `ctx` exit
fast, while slow ones are allowed to finish up to `shutdownTimeout`.

> Note: Go cannot forcibly kill a goroutine. "Cancelling" a handler means its
> `context.Context` is cancelled — the handler code must observe `ctx` and
> return. A handler that ignores `ctx` keeps running until it completes or the
> shutdown budget expires.

### Lifecycle and return values

```
Serve(ctx, addr)
  ├─ ListenAndServe runs in a goroutine
  │
  ├─ startup failure (e.g. address in use)  ──▶ returned immediately
  │
  └─ ctx cancelled
        ├─ BaseContext(ctx) cancels all in-flight request contexts
        └─ Shutdown(shutdownTimeout) drains connections
              ├─ drained in time     ──▶ returns nil
              └─ budget expired       ──▶ returns context.DeadlineExceeded
```

- A startup error from `ListenAndServe` is returned as-is.
- `http.ErrServerClosed` is the normal result of a graceful shutdown and is
  treated as success (not surfaced as an error).
- If the drain does not complete within `shutdownTimeout`, the deadline error
  from `Shutdown` is propagated to the caller.

### Related timeouts

These are configured as functional options and passed straight to the
`http.Server`; unset values mean "no timeout" (standard `net/http` semantics):

| Option | Effect |
|--------|--------|
| `WithReadTimeout(d)`  | max time to read the entire request |
| `WithWriteTimeout(d)` | max time to write the response |
| `WithIdleTimeout(d)`  | max keep-alive idle time between requests |
| `WithShutdownTimeout(d)` | budget for draining in-flight requests on shutdown |
