# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.2] - 2026-08-06

### Fixed

- `WithMiddleware` now appends to the global middleware chain on repeated calls
  instead of replacing it, so a second `WithMiddleware` (or one used alongside
  `WithRecommendedServerSettings`) no longer silently drops earlier middleware.
- Response encoding failures are no longer silently swallowed. Generated HTTP
  adapters marshal the response body first and return a `500` through the error
  handler if it fails (instead of emitting a partial `200`); the runtime error
  handler logs an encode failure instead of ignoring it. **Regenerate** to pick
  up the adapter change.

## [v0.1.1] - 2026-08-06

### Changed

- Moved the library to the module root so it is importable as
  `github.com/siyoga/apiggo` (was `github.com/siyoga/apiggo/pkg/codegen`), giving
  the root pkg.go.dev page a proper symbol index.
- Moved the HTTP runtime from `pkg/server` to `server`; generated code now imports
  `github.com/siyoga/apiggo/server`. **Regenerate existing projects** (or update
  the import path by hand) after upgrading.
- Renamed the primary source file `codegen.go` → `apiggo.go` and added
  package-level documentation for both `apiggo` and `server`.

## [v0.1.0] - 2026-07-30

Initial public release.

### Added

- Contract-first code generator: OpenAPI **3.0.x** documents (YAML or JSON) →
  Go DTOs, HTTP adapters, and handler stubs.
- Typed end-to-end generation: scalar path/query/header parameters with automatic
  type conversion, `application/json` request/response bodies, enums, nested
  objects, arrays, and per-operation typed error responses.
- Handler stubs written once and never overwritten on regeneration; DTOs and
  HTTP adapters regenerated on every run.
- Thin `net/http` runtime (`server`): options-based configuration, middleware
  chain, panic recovery, unified error handling via the `APIError` interface, and
  graceful shutdown with a configurable budget.
- `apiggo` CLI with YAML config file (`-config`) and per-field flag overrides
  (`-schema`, `-module`, `-out`, `-dto-pkg`, `-router-pkg`, `-api-pkg`), plus
  `-output-config` to emit a starter config.

[Unreleased]: https://github.com/siyoga/apiggo/compare/v0.1.1...HEAD
[v0.1.1]: https://github.com/siyoga/apiggo/compare/v0.1.0...v0.1.1
[v0.1.0]: https://github.com/siyoga/apiggo/releases/tag/v0.1.0
