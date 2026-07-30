# apiggo

[![Go Reference](https://pkg.go.dev/badge/github.com/siyoga/apiggo.svg)](https://pkg.go.dev/github.com/siyoga/apiggo)
[![Go Version](https://img.shields.io/badge/go-1.26-00ADD8?logo=go)](go.mod)
[![License](https://img.shields.io/badge/license-Apache_2.0-blue)](LICENSE)

<img align="right" width="150" src="_logo/logo.svg" alt="apiggo logo">

**Contract-first OpenAPI → Go code generator with a thin `net/http` runtime.**

Point it at an OpenAPI document and it emits typed DTOs, transport glue, and
handler stubs — so the only code you write is business logic inside
`Handle(ctx, in) (out, error)`.

## Features

- **Contract-first.** The OpenAPI document is the single source of truth — the
  spec and the implementation can't silently drift.
- **Typed end to end.** Path/query/header params, JSON bodies, responses, enums,
  and per-operation error responses all become concrete Go types.
- **You write only the interesting part.** Parameter parsing, type conversion,
  JSON (de)serialization, and error mapping are generated — your job is the body
  of `Handle`.
- **Safe regeneration.** DTOs and router glue are regenerated on every run;
  handler stubs are written **once and never overwritten**, so re-running after a
  spec change never clobbers your logic.
- **Zero framework.** Generated and runtime code use only stdlib `net/http`
  (Go 1.22+ method-aware `ServeMux`) — no router dependency, no reflection magic.
- **Batteries-included runtime.** Graceful shutdown, panic recovery, a middleware
  chain, and unified error handling.

## Overview

`apiggo` treats your OpenAPI spec as the single source of truth. From one
document it generates three cooperating layers of Go code and ships a small
runtime that hosts them:

- **DTOs** — request/response structs, enums, and typed errors derived from the
  schema.
- **Router glue** — HTTP handlers that parse parameters and bodies, call your
  code, and serialize the result.
- **Handler stubs** — minimal `Handle(ctx, in) (out, error)` scaffolds where your
  logic lives.

Everything the generator and the runtime produce depends only on the standard
library `net/http` (using Go 1.22+ method-aware `ServeMux` patterns) plus the
tiny `apiggo/pkg/server` package.

## How it works

Every operation in the spec flows through three layers:

```
OpenAPI spec ──apiggo──▶  DTOs        (types you pass around)
                           Router glue (transport: parse → call → serialize)
                           Handler stub(your business logic)
```

A generated run writes a tree like this:

```
.
├── generated/
│   ├── dto/dto.go          # types, enums, typed errors  (regenerated)
│   └── router/router.go    # one registrar per operation (regenerated)
└── api/
    ├── getpet/handler.go   # your logic — written once, never overwritten
    └── createpet/handler.go
```

The core idea: **the generator owns transport, you own logic.** The router glue
knows how to decode a request into a typed `In`, hand it to your `Handle`, and
turn the returned `Out` (or `error`) into an HTTP response. It calls into the
runtime for wiring — the glue itself knows nothing about middleware, panic
recovery, or how errors become status codes. That keeps generated code small and
your code free of transport concerns.

Errors are just values. A handler returns any `error`; if it satisfies the
runtime's `APIError` interface (`StatusCode() int` + `Body() any`), the runtime
maps it to that status and JSON body. The typed error structs generated from your
spec's error responses implement exactly that interface, so returning
`&dto.GetPetNotFound{...}` yields a real `404`. Anything else falls back to `500`.

## Install

```bash
go install github.com/siyoga/apiggo/cmd/apiggo@latest
```

## Quick start

### 1. Describe your API

```yaml
# api/openapi.yaml
openapi: 3.0.3
info: { title: petstore, version: 1.0.0 }
paths:
  /pets:
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/PetCreate' }
      responses:
        '200':
          description: created
          content:
            application/json:
              schema: { $ref: '#/components/schemas/Pet' }
components:
  schemas:
    Pet:
      type: object
      required: [id, name]
      properties:
        id:   { type: integer, format: int64 }
        name: { type: string }
    PetCreate:
      type: object
      required: [name]
      properties:
        name: { type: string }
```

### 2. Generate

```bash
apiggo -schema api/openapi.yaml -module github.com/acme/svc
```

This writes `generated/dto`, `generated/router`, and an `api/<operation>` stub
per operation.

### 3. Implement the handler

Fill in the generated stub — this file is yours from now on:

```go
// api/createpet/handler.go
package createpet

import (
	"context"

	dto "github.com/acme/svc/generated/dto"
)

type Handler struct{}

func New() *Handler { return &Handler{} }

func (h *Handler) Handle(ctx context.Context, in dto.CreatePetIn) (dto.CreatePetOut, error) {
	pet := dto.Pet{Id: 1, Name: in.Body.Name}
	return pet, nil
}
```

### 4. Wire it up and serve

`WithOpenApiMethod` pairs a generated registrar with your handler; `Serve` blocks
until the context is cancelled, then shuts down gracefully.

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/siyoga/apiggo/pkg/server"

	"github.com/acme/svc/api/createpet"
	"github.com/acme/svc/generated/router"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	h := createpet.New()

	srv := server.NewServer(
		server.WithShutdownTimeout(10*time.Second),
		server.WithOpenApiMethod(router.CreatePet, h.Handle),
	)

	if err := srv.Serve(ctx, ":8080"); err != nil {
		log.Fatal(err)
	}
}
```

## Generated code at a glance

You never edit these, but it helps to see the shape. **DTOs** — typed structs,
enums, and errors:

```go
type CreatePetIn struct {
	Body PetCreate `json:"-"`
}

type CreatePetOut = Pet

type GetPetNotFound struct {
	Message *string `json:"message,omitempty"`
}

func (e *GetPetNotFound) Error() string   { return "Not Found" }
func (e *GetPetNotFound) StatusCode() int { return 404 }
func (e *GetPetNotFound) Body() any       { return e }
```

**Router glue** — one registrar per operation that parses the request, calls your
handler, and serializes the response:

```go
// GetPet registers the GET /pets/{id} operation.
func GetPet(h func(ctx context.Context, in dto.GetPetIn) (dto.GetPetOut, error)) func(*server.Server) {
	return func(s *server.Server) {
		s.Register(http.MethodGet, "/pets/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in dto.GetPetIn
			{
				raw := r.PathValue("id")
				v, err := strconv.ParseInt(raw, 10, 64)
				if err != nil {
					s.HandleError(w, r, server.ErrBadRequest("invalid parameter: id"))
					return
				}
				in.Id = v
			}
			in.XTrace = r.Header.Get("X-Trace")
			out, err := h(r.Context(), in)
			if err != nil {
				s.HandleError(w, r, err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
		}))
	}
}
```

## Graceful shutdown

The runtime is built around clean shutdown. `Serve` blocks until either the
server fails to start or the context is cancelled — and cancellation is the
caller's responsibility, typically a signal-bound context:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

srv := server.NewServer(server.WithShutdownTimeout(10 * time.Second))
if err := srv.Serve(ctx, ":8080"); err != nil {
	log.Fatal(err)
}
```

Under the hood `Serve` wires two mechanisms off the same `ctx`:

1. **`http.Server.BaseContext` is set to `ctx`.** Every request context — and the
   `ctx` your handler receives — descends from it, so cancelling `ctx`
   immediately signals in-flight handlers to wind down.
2. **`http.Server.Shutdown` runs with a budget.** On cancellation, `Serve` stops
   accepting connections and drains outstanding requests within
   `WithShutdownTimeout` (unbounded if unset).

Handlers that observe `ctx` exit fast; slow ones are allowed to finish up to the
shutdown budget.

```
Serve(ctx, addr)
  ├─ startup failure (e.g. address in use)  ──▶ returned immediately
  └─ ctx cancelled
        ├─ BaseContext(ctx) cancels all in-flight request contexts
        └─ Shutdown(shutdownTimeout) drains connections
              ├─ drained in time  ──▶ returns nil
              └─ budget expired    ──▶ returns context.DeadlineExceeded
```

`http.ErrServerClosed` is the normal result of a graceful shutdown and is treated
as success. Read/write/idle timeouts are configured the same way, via
`WithReadTimeout`, `WithWriteTimeout`, and `WithIdleTimeout` (unset means "no
timeout", standard `net/http` semantics).

> Go cannot forcibly kill a goroutine — "cancelling" a handler means its
> `context.Context` is cancelled and the handler must observe it and return. A
> handler that ignores `ctx` runs until it finishes or the shutdown budget
> expires.

## Status

apiggo is early-stage and evolving. Current scope:

- OpenAPI **3.0.x** specs (YAML or JSON).
- `application/json` request and response bodies.
- Scalar path, query, and header parameters with automatic type conversion.
- Enums, nested objects, arrays, and per-operation typed error responses.
- Standard-library `net/http` output (no external router).

Every operation must declare an `operationId` — it's used to derive stable Go
identifiers and package names.

## Contributing

Issues and pull requests are welcome. Run the test suite with `go test ./...`,
and see [CONTRIBUTING.md](CONTRIBUTING.md) for the development loop and the
golden-fixture workflow. Participation is governed by our
[Code of Conduct](CODE_OF_CONDUCT.md).

## License

Licensed under the [Apache License 2.0](LICENSE).
