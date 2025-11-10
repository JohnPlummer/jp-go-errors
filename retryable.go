package errors

import (
	"context"
	"strings"

	"github.com/cockroachdb/errors"
)

// Retryable interface defines errors that can be retried.
// Implement this interface on custom error types to enable
// intelligent retry logic without string parsing.
type Retryable interface {
	IsRetryable() bool
}

// IsRetryable checks if an error should trigger a retry.
// It checks in priority order:
// 1. Context errors (DeadlineExceeded, Canceled) - NOT retryable
// 2. Any error implementing Retryable interface (generic check)
// 3. Typed sentinel errors (ErrRateLimited, ErrNetworkTimeout, etc.)
// 4. HTTPError with retryable status codes (429, 5xx)
// 5. Defensive fallback for untyped rate limit messages
//
// CRITICAL: Context errors are checked FIRST because some error types
// implement IsRetryable() but may wrap context errors. If context.DeadlineExceeded
// is wrapped, retrying with the same context will fail immediately - these
// operations should be abandoned, not retried.
//
// The generic Retryable interface check (step 2) works with error types from
// any package, not just go-errors. External packages can define their own
// error types with IsRetryable() methods, and they will be properly detected.
//
// Example usage:
//
//	if err != nil {
//	    if IsRetryable(err) {
//	        // Use exponential backoff
//	        time.Sleep(backoff)
//	        continue
//	    }
//	    return err // Permanent failure
//	}
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Context errors are NOT retryable - must check BEFORE interface check.
	// When context.DeadlineExceeded or context.Canceled occurs, the parent
	// context is already exceeded or canceled. Retrying with the same context
	// will fail immediately. These indicate the operation should be abandoned.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}

	// Generic check for ANY error implementing Retryable interface.
	// This catches both go-errors package types and external error types
	// (e.g., deduplicator.comparisonTimeoutError) that implement IsRetryable().
	// Use errors.As() to traverse error chains (handles wrapped errors).
	var r Retryable
	if errors.As(err, &r) {
		return r.IsRetryable()
	}

	// Check for typed sentinel errors
	if errors.Is(err, ErrRateLimited) ||
		errors.Is(err, ErrNetworkTimeout) ||
		errors.Is(err, ErrServerError) ||
		errors.Is(err, ErrConnectionError) ||
		errors.Is(err, ErrDeadlock) ||
		errors.Is(err, ErrCircuitOpen) {
		return true
	}

	// Check for HTTPError with retryable status codes
	if httpErr, ok := IsHTTPError(err); ok {
		return httpErr.IsRetryable()
	}

	// Defensive: Check for rate limit patterns from external APIs we don't control.
	// This is a fallback for third-party libraries that don't use typed errors.
	// Prefer wrapping external errors with our typed errors at API boundaries.
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "rate limit") {
		return true
	}

	// Default to not retryable for safety
	return false
}

// IsRetryableTimeout checks if err is a retryable timeout.
// Returns false for context.DeadlineExceeded (parent context expired).
// Returns true for other timeout errors (network timeouts, API timeouts, etc.).
func IsRetryableTimeout(err error) bool {
	if err == nil {
		return false
	}

	// Context deadline is NOT retryable
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Check for TimeoutError type
	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return timeoutErr.IsRetryable()
	}

	// Check sentinel timeout errors
	return errors.Is(err, ErrNetworkTimeout)
}

// IsTransientError checks if err represents a transient failure.
// Transient failures are temporary and should be retried.
// Examples: network errors, rate limits, server errors.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Context errors are not transient
	if IsContextError(err) {
		return false
	}

	// Network errors are typically transient
	if IsNetworkError(err) {
		return true
	}

	// Rate limits are transient
	if errors.Is(err, ErrRateLimited) {
		return true
	}

	// Server errors are transient
	if errors.Is(err, ErrServerError) {
		return true
	}

	// Connection errors are transient
	if errors.Is(err, ErrConnectionError) {
		return true
	}

	// Deadlocks are transient
	if errors.Is(err, ErrDeadlock) {
		return true
	}

	return false
}

// IsPermanentError checks if err represents a permanent failure.
// Permanent failures should not be retried.
// Examples: validation errors, authentication errors, not found errors.
func IsPermanentError(err error) bool {
	if err == nil {
		return false
	}

	// Validation errors are permanent
	if IsValidation(err) {
		return true
	}

	// Context errors are permanent (operation abandoned)
	if IsContextError(err) {
		return true
	}

	// Circuit breaker errors are managed externally
	if errors.Is(err, ErrCircuitOpen) {
		return true
	}

	// 4xx HTTP errors (except 429 rate limit) are permanent
	if httpErr, ok := IsHTTPError(err); ok {
		return httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 && httpErr.StatusCode != 429
	}

	return false
}
