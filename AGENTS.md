# AGENTS.md

## Project Overview

go-embedder is a Go library for converting text into float32 embedding vectors using OpenAI-compatible APIs. It provides an interface-based design (`embedder.Embedder`) with a concrete implementation in the `openai` sub-package that handles batching, rate limiting, retry with backoff, per-text failure isolation, and concurrency control.

## Tech Stack

- **Language:** Go 1.26
- **Module:** `github.com/ieshan/go-embedder`
- **Dependencies:** `golang.org/x/sync` (semaphore), `golang.org/x/time` (rate limiter)
- **No external frameworks** — stdlib + two x/* packages only

## Project Structure

```
.
├── embedder.go          # Package embedder: the Embedder interface (public API contract)
├── openai/              # OpenAI-compatible client implementation
│   ├── client.go        # Client struct, New(), Embed/EmbedBatch/EmbedBatchPartial
│   ├── request.go       # HTTP request/response types, CallAPI
│   ├── retry.go         # CallWithRetry, transient error classification
│   ├── errors.go        # Sentinel errors: ErrRateLimited, ErrAPIError
│   ├── client_test.go   # Client tests
│   └── retry_test.go    # Retry logic tests
├── go.mod / go.sum      # Module definition and dependency checksums
├── Makefile             # Task targets (test, fmt, vet, lint, tidy, clean)
└── README.md            # User-facing documentation
```

## Commands

All commands are defined in the `Makefile`. Run them from the project root.

| Task | Command | Description |
|---|---|---|
| **Test** | `make test` | Run all tests (`go test ./...`) |
| **Format** | `make fmt` | Format all code (`go fmt -w .`) |
| **Vet** | `make vet` | Run static analysis (`go vet ./...`) |
| **Lint** | `make lint` | Run golangci-lint (`golangci-lint run ./...`) |
| **Tidy** | `make tidy` | Tidy modules (`go mod tidy`) |
| **Clean** | `make clean` | Clean build cache (`go clean ./...`) |

Run a single test:

```bash
go test ./openai/ -run TestEmbedBatch -v
```

Additional verification commands (not in Makefile):

```bash
go test -race ./...          # Race detector — run for concurrency changes
go test -cover ./...         # Coverage report
govulncheck ./...            # Vulnerability scan
go fix ./...                 # Apply Go 1.26 modernizers
```

## Code Style

Follow standard Go conventions. See `GO_BEST_PRACTICES.md` for the full reference (Go 1.20–1.26 features, error handling playbook, testing guidance).

- Run `go fmt` before committing — never commit unformatted code.
- Keep changes minimal and idiomatic; preserve public API contracts unless asked to break them.
- Exported identifiers must have doc comments.
- `context.Context` is the first parameter where required; never pass nil (`context.TODO()` if needed). Do not store context in long-lived structs.
- Error strings: lowercase, punctuation-light. Use `%w` for wrapping when callers should inspect causes.
- Use `errors.Is` / `errors.As` / `errors.AsType` for error inspection — never string matching. This project uses `errors.AsType` (Go 1.26) in `retry.go` for typed error extraction.
- Keep interfaces in consumer packages, small and composable — never define an interface in the same package as its implementation. The `embedder.Embedder` interface lives in the root package; `openai.Client` implements it in a sub-package.
- Every goroutine must have a clear stop condition; avoid leaks via blocked sends/receives. This project uses `sync.WaitGroup` with `wg.Go` and context cancellation for goroutine lifecycle.

### Code style example

```go
// ✅ Good — descriptive names, explicit error wrapping, context-first
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
    vectors, err := c.EmbedBatch(ctx, []string{text})
    if err != nil {
        return nil, err
    }
    return vectors[0], nil
}

// ❌ Bad — ignores context, swallows error, vague naming
func (c *Client) embed(t string) []float32 {
    v, _ := c.EmbedBatch(nil, []string{t})
    return v[0]
}
```

## Testing

- Add or update tests for every code change, even if nobody asked.
- Use table-driven tests with `t.Run` for coverage and maintainability.
- Test behavior and boundaries, not just happy paths — include error paths, rate limiting, retry exhaustion, and context cancellation.
- Keep tests deterministic and isolated. Use `httptest.NewServer` for HTTP mocking (see existing tests for patterns).
- Run `make test` (or `go test ./...`) before finishing any change.
- Run `go test -race ./...` for concurrency-related changes — this project uses goroutines and semaphores, so race detection is critical.
- Existing tests: `openai/client_test.go`, `openai/retry_test.go`.

## Dependency Hygiene

- `go.mod` is the source of truth for module path, Go version, and dependency graph.
- Prefer `go get` / `go mod tidy` over manual editing of `go.mod`.
- Keep `go.mod` minimal — remove dead dependencies with `make tidy`.
- Do not commit `go.sum` changes without a corresponding `go.mod` change.
- This project has only two dependencies (`golang.org/x/sync`, `golang.org/x/time`). Adding new ones requires explicit approval.

## Git Workflow

- Keep commits focused and atomic — one logical change per commit.
- Run `make fmt && make vet && make test` before pushing or opening a PR.
- Do not commit secrets, API keys, or credentials.

## Security

- Never hardcode API keys, endpoints, or credentials in source code.
- API keys are passed via `openai.Options.APIKey` at runtime — do not log or persist them.
- Run `govulncheck ./...` periodically to scan for known vulnerabilities in dependencies.
- Treat API responses as untrusted input — the client already caps response size at 64 MB via `io.LimitReader`.

## Do Not Modify

- `go.sum` — only change via `go mod tidy` or `go get`, never edit by hand.
- `embedder.go` interface contract — the `Embedder` interface is the public API; changing method signatures breaks all consumers.
- Existing tests — do not delete or weaken tests without explicit direction.
- `GO_BEST_PRACTICES.md` — project-agnostic reference, not project-specific to edit.

## Boundaries

- ✅ **Always do:** Run `make fmt`, `make vet`, and `make test` after code changes. Add tests for new behavior. Keep deltas minimal and style-consistent. Preserve the `embedder.Embedder` interface contract.
- ⚠️ **Ask first:** Adding new dependencies to `go.mod`. Changing public API signatures (`Embedder` interface, `openai.New`, `openai.Options` fields). Modifying retry/backoff behavior or rate limiter semantics.
- 🚫 **Never do:** Commit secrets or API keys. Hardcode API endpoints or keys in source. Delete or weaken existing tests without explicit direction. Add comments unless explicitly requested. Create unnecessary helper scripts or files.
