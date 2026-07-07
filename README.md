# go-embedder

A Go library for converting text into float32 embedding vectors using OpenAI-compatible APIs.

## Features

- **Interface-based design** — the `embedder.Embedder` interface lets you swap providers without changing consumer code
- **Automatic batching** — large requests are split into sub-batches within API limits
- **Rate limiting** — token-bucket limiter prevents overwhelming the API
- **Retry with backoff** — exponential backoff with jitter on 429, 5xx, and transient transport errors
- **Per-text failure isolation** — `EmbedBatchPartial` returns per-text errors so partial failures don't discard successful results
- **Concurrency control** — semaphore caps in-flight HTTP requests across all callers
- **Context cancellation** — every blocking point respects `context.Context`

## Installation

```bash
go get github.com/ieshan/go-embedder
```

## Quick Start

Embed a single text string:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ieshan/go-embedder/openai"
)

func main() {
    client, err := openai.New(openai.Options{
        BaseURL: "https://api.openai.com/v1",
        APIKey:  "sk-...",
        Model:   "text-embedding-3-small",
    })
    if err != nil {
        log.Fatal(err)
    }

    vec, err := client.Embed(context.Background(), "Hello, world!")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("vector dimensions: %d\n", len(vec))
}
```

## Batch Embedding

Embed multiple texts in a single call. The client automatically splits the input into sub-batches within `MaxBatchSize` and dispatches them concurrently:

```go
texts := []string{
    "The quick brown fox",
    "jumps over the lazy dog",
    "Machine learning is fun",
}

vectors, err := client.EmbedBatch(context.Background(), texts)
if err != nil {
    log.Fatal(err)
}
for i, vec := range vectors {
    fmt.Printf("text[%d]: %d dimensions\n", i, len(vec))
}
```

`EmbedBatch` is all-or-nothing — if any sub-batch fails, the combined error is returned and no vectors are returned.

## Partial Batch Embedding

When you want per-text failure isolation, use `EmbedBatchPartial`. It returns parallel `vectors` and `errs` slices where `vectors[i]` is `nil` iff `errs[i] != nil`. Successful texts are never discarded due to sibling failures:

```go
vectors, errs := client.EmbedBatchPartial(context.Background(), texts)
for i := range texts {
    if errs[i] != nil {
        fmt.Printf("text[%d] failed: %v\n", i, errs[i])
        continue
    }
    fmt.Printf("text[%d]: %d dimensions\n", i, len(vectors[i]))
}
// Also check ctx.Err() to distinguish caller cancellation from per-sub-batch failure.
if ctx.Err() != nil {
    log.Printf("context was cancelled: %v", ctx.Err())
}
```

## Using the Embedder Interface

The `embedder.Embedder` interface lets you write provider-agnostic code:

```go
import "github.com/ieshan/go-embedder"

func ProcessTexts(ctx context.Context, e embedder.Embedder, texts []string) error {
    vectors, err := e.EmbedBatch(ctx, texts)
    if err != nil {
        return err
    }
    // use vectors...
    return nil
}
```

The `openai.Client` satisfies this interface:

```go
var _ embedder.Embedder = (*openai.Client)(nil)
```

## Configuration Reference

| Field | Type | Default | Description |
|---|---|---|---|
| `BaseURL` | `string` | *(required)* | Base URL of the embedding API, e.g. `https://api.openai.com/v1` |
| `APIKey` | `string` | `""` | API key sent in the `Authorization` header |
| `Model` | `string` | *(required)* | Embedding model name, e.g. `text-embedding-3-small` |
| `Dimensions` | `int` | `0` | Embedding vector dimensions; 0 uses the model default |
| `InputType` | `string` | `""` | Provider-specific input type, e.g. `document` or `query` for Voyage AI |
| `MaxBatchSize` | `int` | `64` | Maximum texts per API call |
| `MaxRetries` | `int` | `5` | Maximum retry attempts on 429 and 5xx errors |
| `RateLimit` | `float64` | `0` | Sustained requests per second; 0 disables rate limiting |
| `RateBurst` | `int` | `10` | Maximum burst above the sustained rate (used only when `RateLimit > 0`) |
| `RetryBaseDelay` | `time.Duration` | `500ms` | Initial backoff delay; doubles each attempt, capped at 30s |
| `HTTPClient` | `*http.Client` | `nil` | Custom HTTP client; nil creates a pooled client |
| `HTTPTimeout` | `time.Duration` | `30s` | HTTP timeout (applied only when `HTTPClient` is nil) |
| `Concurrency` | `int` | `8` | Maximum concurrent sub-batch HTTP requests |

## Supported Providers

Any provider that follows the OpenAI `/v1/embeddings` API shape works out of the box:

| Provider | BaseURL | Notes |
|---|---|---|
| **OpenAI** | `https://api.openai.com/v1` | Default provider |
| **Voyage AI** | `https://api.voyageai.com/v1` | Set `InputType` to `document` or `query` |
| **Azure OpenAI** | `https://<resource>.openai.azure.com/openai/deployments/<deployment>` | Append `/embeddings?api-version=...` to `BaseURL` |
| **Ollama** | `http://localhost:11434/v1` | Local inference, no API key needed |
| **LM Studio** | `http://localhost:1234/v1` | Local inference, no API key needed |

### Example: Voyage AI

```go
client, err := openai.New(openai.Options{
    BaseURL:   "https://api.voyageai.com/v1",
    APIKey:    "va-...",
    Model:     "voyage-3",
    InputType: "document", // or "query"
})
```

### Example: Ollama (local)

```go
client, err := openai.New(openai.Options{
    BaseURL: "http://localhost:11434/v1",
    Model:   "nomic-embed-text",
    // APIKey omitted — not required for local providers
})
```

## Error Handling

The client defines two sentinel errors for use with `errors.Is`:

- **`openai.ErrRateLimited`** — API returned 429 and all retries were exhausted
- **`openai.ErrAPIError`** — API returned a non-transient 4xx error (not 429)

For HTTP error details, use `errors.As` to extract `*openai.APIError`:

```go
var apiErr *openai.APIError
if errors.As(err, &apiErr) {
    fmt.Printf("HTTP %d: %s\n", apiErr.StatusCode, apiErr.Body)
}
```

### Retry Behavior

The client retries on:
- HTTP 429 (Too Many Requests)
- HTTP 5xx (Server Error)
- Transient transport errors (EOF, connection reset, TLS `bad_record_mac`, `http.Client.Timeout` expiry)

It does **not** retry on:
- HTTP 4xx (non-429) — returned immediately wrapped in `ErrAPIError`
- Fatal transport errors (DNS resolution failure, TLS certificate verification failure)
- Caller context cancellation or deadline

## Rate Limiting

Set `RateLimit` to throttle requests to a sustained rate. `RateBurst` allows short bursts above that rate. A value of `0` disables rate limiting entirely:

```go
client, err := openai.New(openai.Options{
    BaseURL:   "https://api.openai.com/v1",
    APIKey:    "sk-...",
    Model:     "text-embedding-3-small",
    RateLimit: 50,  // 50 requests/sec sustained
    RateBurst: 10,  // allow bursts of 10 above the sustained rate
})
```

## Concurrency

`Concurrency` caps the number of in-flight HTTP requests across **all** `EmbedBatch` and `EmbedBatchPartial` calls on the same client. This prevents overwhelming the provider when multiple callers embed concurrently:

```go
client, err := openai.New(openai.Options{
    BaseURL:      "https://api.openai.com/v1",
    APIKey:       "sk-...",
    Model:        "text-embedding-3-small",
    MaxBatchSize: 100, // 100 texts per API call
    Concurrency:  4,   // max 4 concurrent HTTP requests
})
```

## Custom HTTP Client

Pass your own `*http.Client` for fine-grained control over transport, TLS, proxies, etc. When `HTTPClient` is set, `HTTPTimeout` is ignored:

```go
client, err := openai.New(openai.Options{
    BaseURL:    "https://api.openai.com/v1",
    APIKey:     "sk-...",
    Model:      "text-embedding-3-small",
    HTTPClient: &http.Client{
        Timeout: 60 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            TLSClientConfig:     &tls.Config{ /* ... */ },
        },
    },
})
```

## Testing

```bash
make test
# or
go test ./...
```
