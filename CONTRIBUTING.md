# Contributing to apiggo

Thanks for your interest in improving apiggo! Issues and pull requests are
welcome. This guide covers everything you need to get productive.

## Prerequisites

- **Go 1.26 or newer** (the toolchain version is pinned in [`go.mod`](go.mod)).
- No other external tooling is required to build or test.

## Development loop

Clone the repo and run the standard trio before sending a change:

```bash
go build ./...   # compiles the generator, runtime, and command
go vet ./...     # static checks
go test ./...    # unit + golden tests
```

Please make sure code is formatted before committing:

```bash
gofmt -w .
```

## Project layout

- `cmd/apiggo` — the CLI entry point (flag/config parsing).
- the root package `apiggo` — the OpenAPI → Go generator: IR, loader, naming, render
  (`ir.go`, `types.go`, `loader.go`, `names.go`, `render.go`, `config.go`, `apiggo.go`).
- `templates` — the `*.go.tmpl` files that produce DTOs, HTTP adapters, and handler stubs.
- `server` — the thin `net/http` runtime (options, middleware, error handling, graceful shutdown).

## Golden fixtures

The generator is tested against **golden fixtures** — a known-good snapshot of
its output. They live under `testdata/petstore-min`:

- `openapi.yaml` — the input contract.
- `golden/` — the exact files the generator is expected to emit.

If you change a template (`templates/*.tmpl`) or any codegen logic,
the golden output will change. When that happens:

1. Run `go test .` and confirm the diff in the failure output is **intended**.
2. Regenerate the golden files with `go test . -run TestGenerateGolden -update`
   (or `task golden`) so they match the new expected output.
3. Review the golden diff as part of your change — it is the most important
   signal that a generator change does what you think it does.

Never update golden files blindly to make a test pass — the diff *is* the review.

## Pull requests

- Keep PRs focused on a single change.
- Add or update tests for the behavior you change.
- Ensure `go build ./...`, `go vet ./...`, and `go test ./...` all pass.
- Update `README.md` / `CHANGELOG.md` when behavior or the public surface changes.

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By
participating, you agree to uphold it.

## License

By contributing, you agree that your contributions will be licensed under the
[Apache License 2.0](LICENSE).
