# jp-go-errors

Structured error handling package wrapping [cockroachdb/errors](https://github.com/cockroachdb/errors) with enterprise-grade error types, automatic stack traces, and intelligent retry detection.

## Why This Package?

Production debugging requires stack traces. Type-safe retry logic requires structured errors. This package provides both:

- **Automatic stack traces** via cockroachdb/errors
- **Typed error sentinels** for reliable error detection (no string parsing)
- **IsRetryable() interface** for intelligent retry logic
- **Structured context** (HTTP status codes, validation fields, timeouts)
- **Safe error details** for production logging

## Installation

```bash
go get github.com/JohnPlummer/jp-go-errors
```

## Quick Start

```go
import "github.com/JohnPlummer/jp-go-errors"

// Create errors with automatic stack traces
err := errors.New("database connection failed")

// Get formatted stack trace for debugging
trace := errors.GetStackTrace(err)
fmt.Printf("Stack trace:\n%s\n", trace)

// Check if error is retryable
if errors.IsRetryable(err) {
    // Retry with exponential backoff
    time.Sleep(backoff)
    continue
}
```

## Error Types

### HTTPError - HTTP API Failures

```go
// 5xx and 429 are automatically retryable
err := errors.NewHTTPError(503, "Service Unavailable", cause)

if errors.IsRetryable(err) {
    // Will return true for 429, 500-599
}

statusCode := errors.GetHTTPStatusCode(err)  // 503
```

### ValidationError - Input Validation

```go
err := errors.NewValidationError(
    "Price must be positive",
    "price",
    errors.WithValue(-10),
)

// Validation errors are NEVER retryable
if errors.IsPermanentError(err) {
    return err  // Don't retry
}
```

### TimeoutError - Operation Timeouts

```go
err := errors.NewTimeoutError(
    "API call timed out",
    "GetUser",
    30*time.Second,
)

// Timeouts are retryable UNLESS context.DeadlineExceeded
if errors.IsRetryable(err) {
    // Safe to retry
}
```

### RateLimitError - API Rate Limiting

```go
err := errors.NewRateLimitError(
    "API rate limit exceeded",
    "CallAPI",
    60*time.Second,  // retry after
)

// Always retryable, includes suggested retry delay
```

### ProcessingError - Data Processing

```go
err := errors.NewProcessingError(
    "Failed to process activity",
    "ProcessActivity",
    errors.WithItemID("activity-123"),
    errors.WithRetryable(true),
)

// Or use the convenience constructor
err = errors.NewRetryableProcessingError(
    "Database deadlock",
    "SaveData",
    errors.WithCause(dbErr),
)
```

### NetworkError - Network Failures

```go
err := errors.NewNetworkError(
    "Connection refused",
    "Connect",
    // errors.WithTransient(false) for DNS failures
)

// Network errors are transient by default
if errors.IsTransientError(err) {
    // Retry allowed
}
```

### CircuitBreakerError - Circuit Breaker Protection

```go
err := errors.NewCircuitBreakerError(
    "Too many failures",
    "CallExternalAPI",
    "open",
)

// Circuit breaker manages its own retry timing
// These errors are NOT retryable
```

## IsRetryable() Logic

The `IsRetryable()` function implements sophisticated retry detection:

```go
func processWithRetry(ctx context.Context) error {
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        err := doWork(ctx)
        if err == nil {
            return nil
        }

        // CRITICAL: context.DeadlineExceeded is NOT retryable
        // even if wrapped in TimeoutError
        if !errors.IsRetryable(err) {
            return err  // Permanent failure
        }

        // Wait with exponential backoff
        backoff := time.Duration(attempt) * time.Second
        select {
        case <-time.After(backoff):
            continue
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    return errors.New("max retries exceeded")
}
```

### What Is Retryable?

- ✅ `ErrRateLimited`, `ErrNetworkTimeout`, `ErrServerError`
- ✅ `ErrConnectionError`, `ErrDeadlock`, `ErrCircuitOpen`
- ✅ HTTP 429, 500-599 status codes
- ✅ `TimeoutError`, `RateLimitError`, `NetworkError` (transient)
- ✅ `ProcessingError` with `Retryable: true`
- ❌ `context.DeadlineExceeded`, `context.Canceled`
- ❌ `ValidationError`
- ❌ HTTP 400-499 (except 429)
- ❌ `CircuitBreakerError`

### Why context.DeadlineExceeded Is NOT Retryable

When `context.DeadlineExceeded` occurs, the parent context has expired. Retrying with the same context will fail immediately. These errors indicate the operation should be **abandoned**, not retried.

## Error Wrapping

Preserve error chains while adding context:

```go
dbErr := db.Query(...)
if dbErr != nil {
    // Wrap with additional context
    return errors.Wrap(dbErr, "failed to load user")
}

// Check wrapped errors
if errors.Is(err, sql.ErrNoRows) {
    // Handle specific error
}

// Extract typed errors from chain
var httpErr *errors.HTTPError
if errors.As(err, &httpErr) {
    log.Printf("HTTP %d error", httpErr.StatusCode)
}
```

## Stack Traces

```go
err := errors.New("something failed")

// Get formatted stack trace
trace := errors.GetStackTrace(err)
// Output:
// github.com/myorg/myapp.processItem
//     /path/to/code/processor.go:42
// github.com/myorg/myapp.worker
//     /path/to/code/worker.go:28

// Check if error has stack trace
if errors.HasStackTrace(err) {
    // Stack trace available
}

// Get safe details for logging (redacts sensitive info)
safe := errors.GetSafeDetails(err)
```

## Structured Error Information

```go
err := errors.NewHTTPError(503, "Service Unavailable", nil)

// Extract structured information
info := errors.ExtractErrorInfo(err)
// map[string]any{
//     "type": "HTTPError",
//     "retryable": true,
//     "status_code": 503,
//     "message": "HTTP 503: Service Unavailable",
// }

// Format for logging
formatted := errors.FormatError(err)
// "HTTPError(503): HTTP 503: Service Unavailable"
```

## Functional Options

All error constructors support optional configuration:

```go
err := errors.NewProcessingError(
    "Failed to process",
    "ProcessItem",
    errors.WithCause(dbErr),
    errors.WithItemID("item-123"),
    errors.WithRetryable(true),
)

err = errors.NewValidationError(
    "Invalid email format",
    "email",
    errors.WithValue("not-an-email"),
)

err = errors.NewHTTPError(
    500,
    "Internal Server Error",
    cause,
)
```

## Migration from String-Based Detection

**Before:**

```go
// Brittle string parsing
if strings.Contains(err.Error(), "rate limit") {
    // Retry...
}
```

**After:**

```go
// Type-safe detection
if errors.IsRetryable(err) {
    // Retry...
}

// Or check specific types
if errors.Is(err, errors.ErrRateLimited) {
    // Handle rate limit
}
```

## Best Practices

### 1. Wrap External Errors at Boundaries

```go
// Wrap third-party library errors with typed errors
resp, err := http.Get(url)
if err != nil {
    return errors.NewNetworkError("HTTP GET failed", "FetchData",
        errors.WithCause(err))
}

if resp.StatusCode >= 500 {
    return errors.NewHTTPError(resp.StatusCode, "Server error", nil)
}
```

### 2. Use Typed Errors, Not Strings

```go
// Bad
if strings.Contains(err.Error(), "timeout") {
    // Fragile
}

// Good
if errors.IsTimeout(err) {
    // Type-safe
}
```

### 3. Check context.DeadlineExceeded Early

```go
func process(ctx context.Context) error {
    err := doWork(ctx)
    if err != nil {
        // Context errors are not retryable
        if errors.IsContextError(err) {
            return err
        }

        if errors.IsRetryable(err) {
            // Safe to retry
        }
    }
}
```

### 4. Preserve Stack Traces

```go
// Use Wrap/Wrapf to add context while preserving stack trace
if err := validateInput(data); err != nil {
    return errors.Wrap(err, "validation failed")
}
```

## License

MIT License - see [LICENSE](LICENSE) for details
