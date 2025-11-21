// Package errors provides resilience-related error types for circuit breakers and retries.
// These types enable consistent error handling across jp-go-resilience consumers
// without exposing implementation details like gobreaker.
package errors

import (
	"fmt"
	"strings"
)

// Resilience sentinel errors for circuit breaker and retry failures.
// Use errors.Is() to check these without importing implementation packages.
var (
	// ErrCircuitHalfOpen indicates circuit breaker is half-open and rejecting excess requests.
	ErrCircuitHalfOpen = New("circuit breaker half-open, too many requests")

	// ErrRetryExhausted indicates all retry attempts have been exhausted.
	ErrRetryExhausted = New("retry attempts exhausted")

	// ErrMaxAttemptsInvalid indicates max retry attempts configuration is invalid.
	ErrMaxAttemptsInvalid = New("max retry attempts must be positive")
)

// CircuitCounts mirrors gobreaker.Counts without the dependency.
// Provides observability context for circuit breaker state.
type CircuitCounts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// RetryError provides structured context for retry exhaustion.
// Wraps ErrRetryExhausted sentinel so errors.Is() works.
type RetryError struct {
	Attempts    int
	MaxAttempts int
	LastError   error
	AllErrors   []error
	Operation   string
	Component   string
}

func (e *RetryError) Error() string {
	var sb strings.Builder

	opStr := e.Operation
	if e.Component != "" {
		opStr = fmt.Sprintf("%s/%s", e.Component, e.Operation)
	}

	sb.WriteString(fmt.Sprintf("retry exhausted after %d/%d attempts", e.Attempts, e.MaxAttempts))

	if opStr != "" {
		sb.WriteString(fmt.Sprintf(" for %s", opStr))
	}

	if e.LastError != nil {
		sb.WriteString(fmt.Sprintf(": %v", e.LastError))
	}

	return sb.String()
}

// Unwrap returns the sentinel error for errors.Is() compatibility.
func (e *RetryError) Unwrap() error {
	return ErrRetryExhausted
}

// IsRetryable returns false - retry exhaustion means no more retries should occur.
func (e *RetryError) IsRetryable() bool {
	return false
}

// NewRetryError creates a RetryError with the given context.
func NewRetryError(attempts, maxAttempts int, lastError error, allErrors []error, opts ...Option) *RetryError {
	err := &RetryError{
		Attempts:    attempts,
		MaxAttempts: maxAttempts,
		LastError:   lastError,
		AllErrors:   allErrors,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}
