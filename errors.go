// Package errors provides structured error types with automatic stack traces.
// It wraps cockroachdb/errors to provide enterprise-grade error handling with
// typed sentinels, IsRetryable() interface, and structured error context.
//
// Key features:
//   - Automatic stack trace capture for all errors
//   - Typed error sentinels for type-safe error detection
//   - IsRetryable() interface for intelligent retry logic
//   - Structured error context (HTTP status codes, validation fields, etc.)
//   - Safe error details for production logging
package errors

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/cockroachdb/errors"
)

// Re-export commonly used cockroachdb/errors functions.
// These automatically include stack traces when creating or wrapping errors.
var (
	// New creates a new error with a stack trace.
	New = errors.New

	// Errorf creates a new error with formatted message and stack trace.
	Errorf = errors.Errorf

	// Wrap annotates an error with a message and stack trace.
	Wrap = errors.Wrap

	// Wrapf annotates an error with a formatted message and stack trace.
	Wrapf = errors.Wrapf

	// WithStack adds a stack trace to an error if it doesn't have one.
	WithStack = errors.WithStack

	// Is checks if an error matches a target using error chain traversal.
	Is = errors.Is

	// As finds the first error in the chain that matches target type.
	As = errors.As

	// Unwrap returns the result of calling Unwrap on err, if err's type contains
	// an Unwrap method returning error. Otherwise, Unwrap returns nil.
	Unwrap = errors.Unwrap

	// Cause returns the underlying cause of the error, if possible.
	Cause = errors.Cause
)

// Sentinel errors for common retryable conditions.
// Use these when wrapping errors to enable type-safe error detection.
var (
	// ErrRateLimited indicates API rate limiting.
	ErrRateLimited = errors.New("rate limited")

	// ErrNetworkTimeout indicates network timeout.
	ErrNetworkTimeout = errors.New("network timeout")

	// ErrServerError indicates 5xx HTTP server error.
	ErrServerError = errors.New("server error")

	// ErrConnectionError indicates network connection failure.
	ErrConnectionError = errors.New("connection error")

	// ErrDeadlock indicates database deadlock.
	ErrDeadlock = errors.New("database deadlock")

	// ErrCircuitOpen indicates circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker open")

	// ErrInvalidResponse indicates malformed or unexpected response.
	ErrInvalidResponse = errors.New("invalid response")
)

// HTTPError wraps HTTP-related errors with status code information.
// Automatically includes stack trace from creation point.
type HTTPError struct {
	StatusCode int
	Message    string
	Component  string
	Err        error
}

func (e *HTTPError) Error() string {
	msgStr := e.Message
	if e.Component != "" {
		msgStr = fmt.Sprintf("%s: %s", e.Component, e.Message)
	}

	if e.Err != nil {
		return fmt.Sprintf("HTTP %d: %s: %v", e.StatusCode, msgStr, e.Err)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, msgStr)
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true for 5xx errors and 429 (rate limit).
func (e *HTTPError) IsRetryable() bool {
	return e.StatusCode >= 500 || e.StatusCode == 429
}

// NewHTTPError creates an HTTPError with automatic stack trace.
func NewHTTPError(statusCode int, message string, cause error) error {
	httpErr := &HTTPError{
		StatusCode: statusCode,
		Message:    message,
		Err:        cause,
	}
	return httpErr
}

// IsHTTPError checks if err is an HTTPError and returns it.
func IsHTTPError(err error) (*HTTPError, bool) {
	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr, true
	}
	return nil, false
}

// GetHTTPStatusCode extracts HTTP status code from error, or 0 if not found.
func GetHTTPStatusCode(err error) int {
	if httpErr, ok := IsHTTPError(err); ok {
		return httpErr.StatusCode
	}
	return 0
}

// RateLimitError represents rate limiting with retry-after duration.
// Automatically includes stack trace from creation point.
type RateLimitError struct {
	Message    string
	Operation  string
	Component  string
	RetryAfter time.Duration
	Err        error
}

func (e *RateLimitError) Error() string {
	opStr := e.Operation
	if e.Component != "" {
		opStr = fmt.Sprintf("%s/%s", e.Component, e.Operation)
	}

	if e.Err != nil {
		return fmt.Sprintf("rate limited in %s (retry after %v): %s: %v",
			opStr, e.RetryAfter, e.Message, e.Err)
	}
	return fmt.Sprintf("rate limited in %s (retry after %v): %s",
		opStr, e.RetryAfter, e.Message)
}

func (e *RateLimitError) Unwrap() error {
	return e.Err
}

func (e *RateLimitError) IsRetryable() bool {
	return true
}

// NewRateLimitError creates a RateLimitError with automatic stack trace.
func NewRateLimitError(message, operation string, retryAfter time.Duration, opts ...Option) error {
	err := &RateLimitError{
		Message:    message,
		Operation:  operation,
		RetryAfter: retryAfter,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// RetryableError represents a generic retryable error with retry-after duration.
// More general than RateLimitError - can be used for any temporary failure.
// Automatically includes stack trace from creation point.
type RetryableError struct {
	Message    string
	Operation  string
	Component  string
	RetryAfter time.Duration
	Err        error
}

func (e *RetryableError) Error() string {
	opStr := e.Operation
	if e.Component != "" {
		opStr = fmt.Sprintf("%s/%s", e.Component, e.Operation)
	}

	if e.Err != nil {
		return fmt.Sprintf("retryable error in %s (retry after %v): %s: %v",
			opStr, e.RetryAfter, e.Message, e.Err)
	}
	return fmt.Sprintf("retryable error in %s (retry after %v): %s",
		opStr, e.RetryAfter, e.Message)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

func (e *RetryableError) IsRetryable() bool {
	return true
}

// NewRetryableError creates a RetryableError with automatic stack trace.
func NewRetryableError(message, operation string, retryAfter time.Duration, opts ...Option) error {
	err := &RetryableError{
		Message:    message,
		Operation:  operation,
		RetryAfter: retryAfter,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// TimeoutError represents an operation that exceeded its deadline.
// Automatically includes stack trace from creation point.
type TimeoutError struct {
	Message   string
	Operation string
	Component string
	Duration  time.Duration
	Err       error
}

func (e *TimeoutError) Error() string {
	opStr := e.Operation
	if e.Component != "" {
		opStr = fmt.Sprintf("%s/%s", e.Component, e.Operation)
	}

	if e.Err != nil {
		return fmt.Sprintf("timeout in %s after %v: %s: %v",
			opStr, e.Duration, e.Message, e.Err)
	}
	return fmt.Sprintf("timeout in %s after %v: %s",
		opStr, e.Duration, e.Message)
}

func (e *TimeoutError) Unwrap() error {
	return e.Err
}

func (e *TimeoutError) IsRetryable() bool {
	return true
}

// NewTimeoutError creates a TimeoutError with automatic stack trace.
func NewTimeoutError(message, operation string, duration time.Duration, opts ...Option) error {
	err := &TimeoutError{
		Message:   message,
		Operation: operation,
		Duration:  duration,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// IsTimeout checks if err is a timeout error (TimeoutError or net.Error with Timeout()).
func IsTimeout(err error) bool {
	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}

// ValidationError represents a data validation failure.
// Automatically includes stack trace from creation point.
type ValidationError struct {
	Message   string
	Field     string
	Component string
	Value     any
	Err       error
}

func (e *ValidationError) Error() string {
	baseMsg := ""
	if e.Component != "" {
		baseMsg = fmt.Sprintf("validation failed in %s for field '%s' (value: %v)",
			e.Component, e.Field, e.Value)
	} else {
		baseMsg = fmt.Sprintf("validation failed for field '%s' (value: %v)",
			e.Field, e.Value)
	}

	if e.Message != "" {
		if e.Err != nil {
			return fmt.Sprintf("%s: %s: %v", baseMsg, e.Message, e.Err)
		}
		return fmt.Sprintf("%s: %s", baseMsg, e.Message)
	}

	if e.Err != nil {
		return fmt.Sprintf("%s: %v", baseMsg, e.Err)
	}
	return baseMsg
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

func (e *ValidationError) IsRetryable() bool {
	return false
}

// NewValidationError creates a ValidationError with automatic stack trace.
func NewValidationError(message, field string, opts ...Option) error {
	err := &ValidationError{
		Message: message,
		Field:   field,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// IsValidation checks if err is a ValidationError.
func IsValidation(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// ProcessingError represents an error during data processing.
// Automatically includes stack trace from creation point.
type ProcessingError struct {
	Message   string
	Operation string
	ItemID    string
	Component string
	Retryable bool
	Err       error
}

func (e *ProcessingError) Error() string {
	retryStr := "not retryable"
	if e.Retryable {
		retryStr = "retryable"
	}

	opStr := e.Operation
	if e.Component != "" {
		opStr = fmt.Sprintf("%s/%s", e.Component, e.Operation)
	}

	if e.ItemID != "" {
		if e.Err != nil {
			return fmt.Sprintf("%s: %s failed for item %s (%s): %v", e.Message, opStr, e.ItemID, retryStr, e.Err)
		}
		return fmt.Sprintf("%s: %s failed for item %s (%s)", e.Message, opStr, e.ItemID, retryStr)
	}

	if e.Err != nil {
		return fmt.Sprintf("%s: %s failed (%s): %v", e.Message, opStr, retryStr, e.Err)
	}
	return fmt.Sprintf("%s: %s failed (%s)", e.Message, opStr, retryStr)
}

func (e *ProcessingError) Unwrap() error {
	return e.Err
}

func (e *ProcessingError) IsRetryable() bool {
	// Check explicit flag first
	if e.Retryable {
		return true
	}

	// If wrapped error is retryable, this is retryable
	if e.Err != nil && IsRetryable(e.Err) {
		return true
	}

	return false
}

// NewProcessingError creates a ProcessingError with automatic stack trace.
func NewProcessingError(message, operation string, opts ...Option) error {
	err := &ProcessingError{
		Message:   message,
		Operation: operation,
		Retryable: false,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// NewRetryableProcessingError creates a retryable ProcessingError with automatic stack trace.
func NewRetryableProcessingError(message, operation string, opts ...Option) error {
	allOpts := append([]Option{WithRetryable(true)}, opts...)
	return NewProcessingError(message, operation, allOpts...)
}

// NetworkError represents a network connectivity failure.
// Automatically includes stack trace from creation point.
type NetworkError struct {
	Message     string
	Operation   string
	Component   string
	IsTransient bool
	Err         error
}

func (e *NetworkError) Error() string {
	transientStr := "persistent"
	if e.IsTransient {
		transientStr = "transient"
	}

	opStr := e.Operation
	if e.Component != "" {
		opStr = fmt.Sprintf("%s/%s", e.Component, e.Operation)
	}

	if e.Err != nil {
		return fmt.Sprintf("network error in %s (%s): %s: %v",
			opStr, transientStr, e.Message, e.Err)
	}
	return fmt.Sprintf("network error in %s (%s): %s",
		opStr, transientStr, e.Message)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

func (e *NetworkError) IsRetryable() bool {
	return e.IsTransient
}

// NewNetworkError creates a NetworkError with automatic stack trace.
func NewNetworkError(message, operation string, opts ...Option) error {
	err := &NetworkError{
		Message:     message,
		Operation:   operation,
		IsTransient: true, // Default to transient for network errors
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// CircuitBreakerError represents circuit breaker protection.
// Automatically includes stack trace from creation point.
type CircuitBreakerError struct {
	Message   string
	Operation string
	Component string
	State     string
	Err       error
}

func (e *CircuitBreakerError) Error() string {
	opStr := e.Operation
	if e.Component != "" {
		opStr = fmt.Sprintf("%s/%s", e.Component, e.Operation)
	}

	if e.Err != nil {
		return fmt.Sprintf("circuit breaker %s for %s: %s: %v",
			e.State, opStr, e.Message, e.Err)
	}
	return fmt.Sprintf("circuit breaker %s for %s: %s",
		e.State, opStr, e.Message)
}

func (e *CircuitBreakerError) Unwrap() error {
	return e.Err
}

func (e *CircuitBreakerError) IsRetryable() bool {
	// Circuit breaker manages its own retry timing
	return false
}

// NewCircuitBreakerError creates a CircuitBreakerError with automatic stack trace.
func NewCircuitBreakerError(message, operation, state string, opts ...Option) error {
	err := &CircuitBreakerError{
		Message:   message,
		Operation: operation,
		State:     state,
	}
	for _, opt := range opts {
		opt(err)
	}
	return err
}

// IsNetworkError checks if err is a network error (NetworkError or net.Error).
func IsNetworkError(err error) bool {
	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return true
	}

	var stdNetErr net.Error
	return errors.As(err, &stdNetErr)
}

// IsContextError checks if err is a context error (DeadlineExceeded or Canceled).
func IsContextError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

// NewInternalError creates an HTTPError with status 500 (Internal Server Error).
// This is a convenience wrapper for API/backend services.
func NewInternalError(message string, cause error) error {
	return NewHTTPError(500, message, cause)
}

// Sentinel errors for common API/backend error conditions.
var (
	// ErrActivityNotFound indicates a requested activity was not found.
	ErrActivityNotFound = errors.New("activity not found")

	// ErrLocationNotFound indicates a requested location was not found.
	ErrLocationNotFound = errors.New("location not found")
)
