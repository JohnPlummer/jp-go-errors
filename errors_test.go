package errors

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

// TestHTTPError tests HTTPError creation and methods
func TestHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		cause      error
		retryable  bool
	}{
		{
			name:       "500 error is retryable",
			statusCode: 500,
			message:    "Internal Server Error",
			cause:      nil,
			retryable:  true,
		},
		{
			name:       "503 error is retryable",
			statusCode: 503,
			message:    "Service Unavailable",
			cause:      nil,
			retryable:  true,
		},
		{
			name:       "429 rate limit is retryable",
			statusCode: 429,
			message:    "Too Many Requests",
			cause:      nil,
			retryable:  true,
		},
		{
			name:       "404 error is not retryable",
			statusCode: 404,
			message:    "Not Found",
			cause:      nil,
			retryable:  false,
		},
		{
			name:       "400 error is not retryable",
			statusCode: 400,
			message:    "Bad Request",
			cause:      nil,
			retryable:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewHTTPError(tt.statusCode, tt.message, tt.cause)

			// Check error message contains status code
			errMsg := err.Error()
			if errMsg == "" {
				t.Fatal("Error message is empty")
			}

			// Check IsRetryable
			httpErr, ok := IsHTTPError(err)
			if !ok {
				t.Fatal("Expected HTTPError")
			}

			if httpErr.IsRetryable() != tt.retryable {
				t.Errorf("Expected retryable=%v, got %v", tt.retryable, httpErr.IsRetryable())
			}

			// Check status code extraction
			if GetHTTPStatusCode(err) != tt.statusCode {
				t.Errorf("Expected status code %d, got %d", tt.statusCode, GetHTTPStatusCode(err))
			}
		})
	}
}

// TestRateLimitError tests RateLimitError creation and methods
func TestRateLimitError(t *testing.T) {
	retryAfter := 60 * time.Second
	err := NewRateLimitError("API rate limit exceeded", "CallAPI", retryAfter)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check retryable
	if !IsRetryable(err) {
		t.Error("Rate limit error should be retryable")
	}

	// Check error message
	errMsg := err.Error()
	if errMsg == "" {
		t.Fatal("Error message is empty")
	}
}

// TestTimeoutError tests TimeoutError creation and methods
func TestTimeoutError(t *testing.T) {
	duration := 30 * time.Second
	err := NewTimeoutError("Operation timed out", "ProcessData", duration)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check retryable
	if !IsRetryable(err) {
		t.Error("Timeout error should be retryable")
	}

	// Check timeout detection
	if !IsTimeout(err) {
		t.Error("Expected IsTimeout to return true")
	}
}

// TestValidationError tests ValidationError creation and methods
func TestValidationError(t *testing.T) {
	err := NewValidationError("Price must be positive", "price",
		WithValue(-10))

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Validation errors should NOT be retryable
	if IsRetryable(err) {
		t.Error("Validation error should not be retryable")
	}

	// Check validation detection
	if !IsValidation(err) {
		t.Error("Expected IsValidation to return true")
	}

	// Check permanent error detection
	if !IsPermanentError(err) {
		t.Error("Validation error should be permanent")
	}
}

// TestProcessingError tests ProcessingError creation and methods
func TestProcessingError(t *testing.T) {
	t.Run("not retryable by default", func(t *testing.T) {
		err := NewProcessingError("Failed to process", "ProcessItem")

		if IsRetryable(err) {
			t.Error("Processing error should not be retryable by default")
		}
	})

	t.Run("explicitly retryable", func(t *testing.T) {
		err := NewProcessingError("Failed to process", "ProcessItem",
			WithRetryable(true))

		if !IsRetryable(err) {
			t.Error("Processing error should be retryable when explicitly set")
		}
	})

	t.Run("with item ID", func(t *testing.T) {
		err := NewProcessingError("Failed to process", "ProcessItem",
			WithItemID("item-123"))

		errMsg := err.Error()
		if errMsg == "" {
			t.Fatal("Error message is empty")
		}
	})

	t.Run("retryable constructor", func(t *testing.T) {
		err := NewRetryableProcessingError("Failed to process", "ProcessItem")

		if !IsRetryable(err) {
			t.Error("NewRetryableProcessingError should create retryable error")
		}
	})
}

// TestNetworkError tests NetworkError creation and methods
func TestNetworkError(t *testing.T) {
	t.Run("transient by default", func(t *testing.T) {
		err := NewNetworkError("Connection failed", "Connect")

		if !IsRetryable(err) {
			t.Error("Network error should be retryable by default")
		}

		if !IsNetworkError(err) {
			t.Error("Expected IsNetworkError to return true")
		}
	})

	t.Run("explicit persistent", func(t *testing.T) {
		err := NewNetworkError("DNS lookup failed", "Connect",
			WithTransient(false))

		if IsRetryable(err) {
			t.Error("Persistent network error should not be retryable")
		}
	})
}

// TestCircuitBreakerError tests CircuitBreakerError creation and methods
func TestCircuitBreakerError(t *testing.T) {
	err := NewCircuitBreakerError("Too many failures", "CallAPI", "open")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Circuit breaker errors should NOT be retryable (managed externally)
	if IsRetryable(err) {
		t.Error("Circuit breaker error should not be retryable")
	}
}

// TestIsRetryable tests the IsRetryable function with various error types
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "context.DeadlineExceeded",
			err:       context.DeadlineExceeded,
			retryable: false,
		},
		{
			name:      "context.Canceled",
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "ErrRateLimited sentinel",
			err:       ErrRateLimited,
			retryable: true,
		},
		{
			name:      "ErrNetworkTimeout sentinel",
			err:       ErrNetworkTimeout,
			retryable: true,
		},
		{
			name:      "ErrServerError sentinel",
			err:       ErrServerError,
			retryable: true,
		},
		{
			name:      "ErrConnectionError sentinel",
			err:       ErrConnectionError,
			retryable: true,
		},
		{
			name:      "ErrDeadlock sentinel",
			err:       ErrDeadlock,
			retryable: true,
		},
		{
			name:      "ErrCircuitOpen sentinel",
			err:       ErrCircuitOpen,
			retryable: true,
		},
		{
			name:      "wrapped context.DeadlineExceeded",
			err:       Wrap(context.DeadlineExceeded, "operation failed"),
			retryable: false,
		},
		{
			name:      "rate limit string pattern",
			err:       fmt.Errorf("API rate limit exceeded"),
			retryable: true,
		},
		{
			name:      "generic error",
			err:       fmt.Errorf("something went wrong"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.retryable {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

// TestIsRetryableTimeout tests the IsRetryableTimeout function
func TestIsRetryableTimeout(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "context.DeadlineExceeded",
			err:       context.DeadlineExceeded,
			retryable: false,
		},
		{
			name:      "TimeoutError",
			err:       NewTimeoutError("timeout", "operation", 30*time.Second),
			retryable: true,
		},
		{
			name:      "ErrNetworkTimeout sentinel",
			err:       ErrNetworkTimeout,
			retryable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryableTimeout(tt.err)
			if got != tt.retryable {
				t.Errorf("IsRetryableTimeout() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

// TestIsTransientError tests the IsTransientError function
func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		transient bool
	}{
		{
			name:      "nil error",
			err:       nil,
			transient: false,
		},
		{
			name:      "network error",
			err:       NewNetworkError("connection failed", "Connect"),
			transient: true,
		},
		{
			name:      "rate limit",
			err:       ErrRateLimited,
			transient: true,
		},
		{
			name:      "server error",
			err:       ErrServerError,
			transient: true,
		},
		{
			name:      "context canceled",
			err:       context.Canceled,
			transient: false,
		},
		{
			name:      "validation error",
			err:       NewValidationError("invalid", "field"),
			transient: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTransientError(tt.err)
			if got != tt.transient {
				t.Errorf("IsTransientError() = %v, want %v", got, tt.transient)
			}
		})
	}
}

// TestIsPermanentError tests the IsPermanentError function
func TestIsPermanentError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		permanent bool
	}{
		{
			name:      "nil error",
			err:       nil,
			permanent: false,
		},
		{
			name:      "validation error",
			err:       NewValidationError("invalid", "field"),
			permanent: true,
		},
		{
			name:      "context canceled",
			err:       context.Canceled,
			permanent: true,
		},
		{
			name:      "circuit breaker",
			err:       ErrCircuitOpen,
			permanent: true,
		},
		{
			name:      "404 HTTP error",
			err:       NewHTTPError(404, "Not Found", nil),
			permanent: true,
		},
		{
			name:      "429 rate limit (not permanent)",
			err:       NewHTTPError(429, "Too Many Requests", nil),
			permanent: false,
		},
		{
			name:      "500 server error (not permanent)",
			err:       NewHTTPError(500, "Internal Server Error", nil),
			permanent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPermanentError(tt.err)
			if got != tt.permanent {
				t.Errorf("IsPermanentError() = %v, want %v", got, tt.permanent)
			}
		})
	}
}

// TestErrorWrapping tests error chain traversal with Is and As
func TestErrorWrapping(t *testing.T) {
	t.Run("wrapped sentinel", func(t *testing.T) {
		baseErr := ErrRateLimited
		wrapped := Wrap(baseErr, "API call failed")

		if !Is(wrapped, ErrRateLimited) {
			t.Error("Expected Is to find ErrRateLimited in chain")
		}

		if !IsRetryable(wrapped) {
			t.Error("Wrapped rate limit error should be retryable")
		}
	})

	t.Run("wrapped HTTPError", func(t *testing.T) {
		baseErr := NewHTTPError(503, "Service Unavailable", nil)
		wrapped := Wrap(baseErr, "Request failed")

		var httpErr *HTTPError
		if !As(wrapped, &httpErr) {
			t.Fatal("Expected As to find HTTPError in chain")
		}

		if httpErr.StatusCode != 503 {
			t.Errorf("Expected status code 503, got %d", httpErr.StatusCode)
		}

		if !IsRetryable(wrapped) {
			t.Error("Wrapped 503 error should be retryable")
		}
	})

	t.Run("context error wrapped in TimeoutError", func(t *testing.T) {
		// This tests the critical case: context.DeadlineExceeded must NOT be retryable
		// even when wrapped in a TimeoutError that implements IsRetryable()
		baseErr := context.DeadlineExceeded
		timeoutErr := NewTimeoutError("operation timed out", "Process", 30*time.Second,
			WithCause(baseErr))

		// The error itself is a TimeoutError (which is retryable)
		var tErr *TimeoutError
		if !As(timeoutErr, &tErr) {
			t.Fatal("Expected TimeoutError")
		}

		// But IsRetryable should return false because context.DeadlineExceeded is in chain
		if IsRetryable(timeoutErr) {
			t.Error("TimeoutError wrapping context.DeadlineExceeded should NOT be retryable")
		}
	})
}

// TestOptions tests functional options
func TestOptions(t *testing.T) {
	t.Run("WithCause", func(t *testing.T) {
		cause := fmt.Errorf("database error")
		err := NewProcessingError("Failed", "Process", WithCause(cause))

		if !Is(err, cause) {
			t.Error("Expected cause to be in error chain")
		}
	})

	t.Run("WithRetryable", func(t *testing.T) {
		err := NewProcessingError("Failed", "Process", WithRetryable(true))

		if !IsRetryable(err) {
			t.Error("Expected error to be retryable")
		}
	})

	t.Run("WithItemID", func(t *testing.T) {
		err := NewProcessingError("Failed", "Process", WithItemID("item-123"))

		var procErr *ProcessingError
		if !As(err, &procErr) {
			t.Fatal("Expected ProcessingError")
		}

		if procErr.ItemID != "item-123" {
			t.Errorf("Expected ItemID=item-123, got %s", procErr.ItemID)
		}
	})

	t.Run("WithValue", func(t *testing.T) {
		err := NewValidationError("Invalid", "price", WithValue(-10))

		var valErr *ValidationError
		if !As(err, &valErr) {
			t.Fatal("Expected ValidationError")
		}

		if valErr.Value != -10 {
			t.Errorf("Expected Value=-10, got %v", valErr.Value)
		}
	})
}

// TestStackTrace tests stack trace functionality
func TestStackTrace(t *testing.T) {
	t.Run("error has stack trace", func(t *testing.T) {
		err := New("test error")

		if !HasStackTrace(err) {
			t.Error("Expected error to have stack trace")
		}

		trace := GetStackTrace(err)
		if trace == "" {
			t.Error("Expected non-empty stack trace")
		}
	})

	t.Run("wrapped error preserves stack trace", func(t *testing.T) {
		baseErr := New("base error")
		wrapped := Wrap(baseErr, "wrapped")

		if !HasStackTrace(wrapped) {
			t.Error("Expected wrapped error to have stack trace")
		}
	})

	t.Run("stack trace lines", func(t *testing.T) {
		err := New("test error")
		lines := GetStackTraceLines(err)

		if len(lines) == 0 {
			t.Error("Expected non-empty stack trace lines")
		}
	})
}

// TestFormatError tests error formatting
func TestFormatError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "HTTPError",
			err:  NewHTTPError(500, "Internal Server Error", nil),
		},
		{
			name: "ValidationError",
			err:  NewValidationError("invalid", "field"),
		},
		{
			name: "TimeoutError",
			err:  NewTimeoutError("timeout", "operation", 30*time.Second),
		},
		{
			name: "RateLimitError",
			err:  NewRateLimitError("rate limited", "operation", 60*time.Second),
		},
		{
			name: "ProcessingError",
			err:  NewProcessingError("failed", "operation"),
		},
		{
			name: "NetworkError",
			err:  NewNetworkError("connection failed", "operation"),
		},
		{
			name: "CircuitBreakerError",
			err:  NewCircuitBreakerError("circuit open", "operation", "open"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := FormatError(tt.err)
			if formatted == "" {
				t.Error("Expected non-empty formatted error")
			}
		})
	}
}

// TestExtractErrorInfo tests error information extraction
func TestExtractErrorInfo(t *testing.T) {
	err := NewHTTPError(503, "Service Unavailable", nil)
	info := ExtractErrorInfo(err)

	if info == nil {
		t.Fatal("Expected non-nil error info")
	}

	if info["type"] != "HTTPError" {
		t.Errorf("Expected type=HTTPError, got %v", info["type"])
	}

	if info["status_code"] != 503 {
		t.Errorf("Expected status_code=503, got %v", info["status_code"])
	}

	if !info["retryable"].(bool) {
		t.Error("Expected retryable=true")
	}
}

// TestIsTimeout tests timeout detection
func TestIsTimeout(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		isTimeout bool
	}{
		{
			name:      "nil error",
			err:       nil,
			isTimeout: false,
		},
		{
			name:      "TimeoutError",
			err:       NewTimeoutError("timeout", "op", 30*time.Second),
			isTimeout: true,
		},
		{
			name:      "net timeout error",
			err:       &net.DNSError{IsTimeout: true},
			isTimeout: true,
		},
		{
			name:      "generic error",
			err:       fmt.Errorf("not a timeout"),
			isTimeout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTimeout(tt.err)
			if got != tt.isTimeout {
				t.Errorf("IsTimeout() = %v, want %v", got, tt.isTimeout)
			}
		})
	}
}

// TestIsNetworkError tests network error detection
func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		isNetworkError bool
	}{
		{
			name:           "nil error",
			err:            nil,
			isNetworkError: false,
		},
		{
			name:           "NetworkError",
			err:            NewNetworkError("connection failed", "Connect"),
			isNetworkError: true,
		},
		{
			name:           "net.OpError",
			err:            &net.OpError{Op: "dial"},
			isNetworkError: true,
		},
		{
			name:           "generic error",
			err:            fmt.Errorf("not a network error"),
			isNetworkError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNetworkError(tt.err)
			if got != tt.isNetworkError {
				t.Errorf("IsNetworkError() = %v, want %v", got, tt.isNetworkError)
			}
		})
	}
}

// TestIsContextError tests context error detection
func TestIsContextError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		isContextError bool
	}{
		{
			name:           "nil error",
			err:            nil,
			isContextError: false,
		},
		{
			name:           "context.DeadlineExceeded",
			err:            context.DeadlineExceeded,
			isContextError: true,
		},
		{
			name:           "context.Canceled",
			err:            context.Canceled,
			isContextError: true,
		},
		{
			name:           "wrapped context.DeadlineExceeded",
			err:            Wrap(context.DeadlineExceeded, "timeout"),
			isContextError: true,
		},
		{
			name:           "generic error",
			err:            fmt.Errorf("not a context error"),
			isContextError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsContextError(tt.err)
			if got != tt.isContextError {
				t.Errorf("IsContextError() = %v, want %v", got, tt.isContextError)
			}
		})
	}
}

// TestGetSafeDetails tests safe error details extraction
func TestGetSafeDetails(t *testing.T) {
	err := NewHTTPError(500, "Internal Server Error", fmt.Errorf("database password: secret123"))
	safe := GetSafeDetails(err)

	if safe == "" {
		t.Error("Expected non-empty safe details")
	}
}

// TestReExports tests that re-exported functions work correctly
func TestReExports(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		err := New("test error")
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})

	t.Run("Errorf", func(t *testing.T) {
		err := Errorf("test error: %d", 42)
		if err == nil {
			t.Fatal("Expected error, got nil")
		}
	})

	t.Run("Wrap", func(t *testing.T) {
		base := fmt.Errorf("base error")
		wrapped := Wrap(base, "wrapped")
		if !Is(wrapped, base) {
			t.Error("Expected Is to find base error")
		}
	})

	t.Run("Wrapf", func(t *testing.T) {
		base := fmt.Errorf("base error")
		wrapped := Wrapf(base, "wrapped: %d", 42)
		if !Is(wrapped, base) {
			t.Error("Expected Is to find base error")
		}
	})

	t.Run("WithStack", func(t *testing.T) {
		base := fmt.Errorf("base error")
		stacked := WithStack(base)
		if !HasStackTrace(stacked) {
			t.Error("Expected error to have stack trace")
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		base := fmt.Errorf("base error")
		wrapped := Wrap(base, "wrapped")
		unwrapped := Unwrap(wrapped)
		if unwrapped == nil {
			t.Error("Expected unwrapped error")
		}
	})

	t.Run("Cause", func(t *testing.T) {
		base := fmt.Errorf("base error")
		wrapped := Wrap(base, "wrapped")
		cause := Cause(wrapped)
		if cause == nil {
			t.Error("Expected cause")
		}
	})
}

// TestAllOptions tests all option functions for coverage
func TestAllOptions(t *testing.T) {
	t.Run("WithMessage", func(t *testing.T) {
		err := NewHTTPError(500, "", nil)
		WithMessage("custom message")(err.(*HTTPError))
		if err.(*HTTPError).Message != "custom message" {
			t.Error("WithMessage did not set message")
		}
	})

	t.Run("WithStatusCode", func(t *testing.T) {
		err := NewHTTPError(0, "error", nil)
		WithStatusCode(503)(err.(*HTTPError))
		if err.(*HTTPError).StatusCode != 503 {
			t.Error("WithStatusCode did not set status code")
		}
	})

	t.Run("WithField", func(t *testing.T) {
		err := NewValidationError("error", "")
		WithField("email")(err.(*ValidationError))
		if err.(*ValidationError).Field != "email" {
			t.Error("WithField did not set field")
		}
	})

	t.Run("WithOperation", func(t *testing.T) {
		err := NewTimeoutError("", "OldOp", 30*time.Second)
		WithOperation("NewOp")(err.(*TimeoutError))
		if err.(*TimeoutError).Operation != "NewOp" {
			t.Error("WithOperation did not set operation")
		}
	})

	t.Run("WithState", func(t *testing.T) {
		err := NewCircuitBreakerError("error", "op", "")
		WithState("open")(err.(*CircuitBreakerError))
		if err.(*CircuitBreakerError).State != "open" {
			t.Error("WithState did not set state")
		}
	})

	t.Run("WithTransient", func(t *testing.T) {
		err := NewNetworkError("error", "op", WithTransient(false))
		if err.(*NetworkError).IsTransient {
			t.Error("WithTransient(false) did not work")
		}
	})
}

// TestErrorsWithNilCause tests errors with nil causes
func TestErrorsWithNilCause(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"HTTPError", NewHTTPError(500, "error", nil)},
		{"RateLimitError", NewRateLimitError("error", "op", 60*time.Second)},
		{"TimeoutError", NewTimeoutError("error", "op", 30*time.Second)},
		{"ValidationError", NewValidationError("error", "field")},
		{"NetworkError", NewNetworkError("error", "op")},
		{"CircuitBreakerError", NewCircuitBreakerError("error", "op", "open")},
		{"ProcessingError", NewProcessingError("error", "op")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("Expected error, got nil")
			}
			// Check error message is not empty
			if tt.err.Error() == "" {
				t.Error("Error message is empty")
			}
		})
	}
}

// TestErrorsWithCause tests errors with causes
func TestErrorsWithCause(t *testing.T) {
	cause := fmt.Errorf("underlying cause")

	tests := []struct {
		name string
		err  error
	}{
		{"HTTPError", NewHTTPError(500, "error", cause)},
		{"RateLimitError", NewRateLimitError("error", "op", 60*time.Second, WithCause(cause))},
		{"TimeoutError", NewTimeoutError("error", "op", 30*time.Second, WithCause(cause))},
		{"ValidationError", NewValidationError("error", "field", WithCause(cause))},
		{"NetworkError", NewNetworkError("error", "op", WithCause(cause))},
		{"CircuitBreakerError", NewCircuitBreakerError("error", "op", "open", WithCause(cause))},
		{"ProcessingError", NewProcessingError("error", "op", WithCause(cause))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("Expected error, got nil")
			}

			// Check that cause is in error chain
			if !Is(tt.err, cause) {
				t.Error("Cause not found in error chain")
			}
		})
	}
}

// TestExtractErrorInfoAllTypes tests ExtractErrorInfo for all error types
func TestExtractErrorInfoAllTypes(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType string
	}{
		{"HTTPError", NewHTTPError(503, "error", nil), "HTTPError"},
		{"ValidationError", NewValidationError("error", "field", WithValue("test")), "ValidationError"},
		{"TimeoutError", NewTimeoutError("error", "op", 30*time.Second), "TimeoutError"},
		{"RateLimitError", NewRateLimitError("error", "op", 60*time.Second), "RateLimitError"},
		{"ProcessingError", NewProcessingError("error", "op", WithItemID("123")), "ProcessingError"},
		{"NetworkError", NewNetworkError("error", "op"), "NetworkError"},
		{"CircuitBreakerError", NewCircuitBreakerError("error", "op", "open"), "CircuitBreakerError"},
		{"GenericError", fmt.Errorf("generic"), "Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ExtractErrorInfo(tt.err)
			if info == nil {
				t.Fatal("Expected non-nil info")
			}

			if info["type"] != tt.expectedType {
				t.Errorf("Expected type=%s, got %v", tt.expectedType, info["type"])
			}

			if info["message"] == "" {
				t.Error("Expected non-empty message")
			}
		})
	}
}

// TestProcessingErrorWithWrappedRetryable tests ProcessingError with retryable wrapped error
func TestProcessingErrorWithWrappedRetryable(t *testing.T) {
	// Create a retryable error and wrap it in ProcessingError
	retryableErr := NewRateLimitError("rate limited", "op", 60*time.Second)
	procErr := NewProcessingError("processing failed", "process", WithCause(retryableErr))

	if !IsRetryable(procErr) {
		t.Error("ProcessingError should be retryable when wrapping retryable error")
	}
}
