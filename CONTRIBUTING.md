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
- `pkg/codegen` — the OpenAPI → Go generator: IR, loader, naming, templates, render.
- `pkg/codegen/templates` — the `*.go.tmpl` files that produce DTOs, router glue, and handler stubs.
- `pkg/server` — the thin `net/http` runtime (options, middleware, error handling, graceful shutdown).

## Golden fixtures

The generator is tested against **golden fixtures** — a known-good snapshot of
its output. They live under `pkg/codegen/testdata/petstore-min`:

- `openapi.yaml` — the input contract.
- `golden/` — the exact files the generator is expected to emit.

If you change a template (`pkg/codegen/templates/*.tmpl`) or any codegen logic,
the golden output will change. When that happens:

1. Run `go test ./pkg/codegen/...` and confirm the diff in the failure output is
   **intended**.
2. Regenerate/update the golden files so they match the new expected output.
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
