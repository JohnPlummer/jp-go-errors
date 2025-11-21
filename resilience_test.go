package errors

import (
	"fmt"
	"testing"
)

// TestResilienceSentinels tests that sentinel errors are correctly defined
func TestResilienceSentinels(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
		wantMsg  string
	}{
		{"ErrCircuitOpen", ErrCircuitOpen, "circuit breaker open"},
		{"ErrCircuitHalfOpen", ErrCircuitHalfOpen, "circuit breaker half-open, too many requests"},
		{"ErrRetryExhausted", ErrRetryExhausted, "retry attempts exhausted"},
		{"ErrMaxAttemptsInvalid", ErrMaxAttemptsInvalid, "max retry attempts must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.sentinel.Error() != tt.wantMsg {
				t.Errorf("got %q, want %q", tt.sentinel.Error(), tt.wantMsg)
			}
		})
	}
}

// TestRetryError tests RetryError creation and methods
func TestRetryError(t *testing.T) {
	t.Run("basic creation", func(t *testing.T) {
		lastErr := fmt.Errorf("connection timeout")
		allErrs := []error{fmt.Errorf("err1"), fmt.Errorf("err2"), lastErr}

		err := NewRetryError(3, 3, lastErr, allErrs)

		if err.Attempts != 3 {
			t.Errorf("got Attempts=%d, want 3", err.Attempts)
		}
		if err.MaxAttempts != 3 {
			t.Errorf("got MaxAttempts=%d, want 3", err.MaxAttempts)
		}
		if err.LastError != lastErr {
			t.Error("LastError not set correctly")
		}
		if len(err.AllErrors) != 3 {
			t.Errorf("got %d AllErrors, want 3", len(err.AllErrors))
		}
	})

	t.Run("error message format", func(t *testing.T) {
		lastErr := fmt.Errorf("timeout")
		err := NewRetryError(3, 5, lastErr, nil, WithOperation("CallAPI"))

		msg := err.Error()
		if msg == "" {
			t.Error("error message is empty")
		}
		// Should contain attempts info
		if !containsSubstring(msg, "3/5") {
			t.Errorf("message should contain attempt counts, got: %s", msg)
		}
		// Should contain operation
		if !containsSubstring(msg, "CallAPI") {
			t.Errorf("message should contain operation, got: %s", msg)
		}
	})

	t.Run("error message with component", func(t *testing.T) {
		err := NewRetryError(2, 3, nil, nil,
			WithOperation("Fetch"),
			WithComponent("normaliser"))

		msg := err.Error()
		if !containsSubstring(msg, "normaliser/Fetch") {
			t.Errorf("message should contain component/operation, got: %s", msg)
		}
	})

	t.Run("unwrap returns sentinel", func(t *testing.T) {
		err := NewRetryError(3, 3, nil, nil)

		if !Is(err, ErrRetryExhausted) {
			t.Error("Unwrap should return ErrRetryExhausted sentinel")
		}
	})

	t.Run("is not retryable", func(t *testing.T) {
		err := NewRetryError(3, 3, nil, nil)

		if err.IsRetryable() {
			t.Error("RetryError should not be retryable")
		}
	})

	t.Run("with options", func(t *testing.T) {
		err := NewRetryError(2, 3, nil, nil,
			WithOperation("Process"),
			WithComponent("worker"))

		if err.Operation != "Process" {
			t.Errorf("got Operation=%s, want Process", err.Operation)
		}
		if err.Component != "worker" {
			t.Errorf("got Component=%s, want worker", err.Component)
		}
	})
}

// TestCircuitBreakerErrorSentinelUnwrap tests that CircuitBreakerError correctly unwraps to sentinels
func TestCircuitBreakerErrorSentinelUnwrap(t *testing.T) {
	t.Run("open state unwraps to ErrCircuitOpen", func(t *testing.T) {
		err := NewCircuitBreakerError("circuit tripped", "CallAPI", "open")

		if !Is(err, ErrCircuitOpen) {
			t.Error("open state should unwrap to ErrCircuitOpen")
		}
	})

	t.Run("half-open state unwraps to ErrCircuitHalfOpen", func(t *testing.T) {
		err := NewCircuitBreakerError("too many requests", "CallAPI", "half-open")

		if !Is(err, ErrCircuitHalfOpen) {
			t.Error("half-open state should unwrap to ErrCircuitHalfOpen")
		}
	})

	t.Run("closed state does not unwrap to sentinels", func(t *testing.T) {
		err := NewCircuitBreakerError("recording failure", "CallAPI", "closed")

		if Is(err, ErrCircuitOpen) {
			t.Error("closed state should not unwrap to ErrCircuitOpen")
		}
		if Is(err, ErrCircuitHalfOpen) {
			t.Error("closed state should not unwrap to ErrCircuitHalfOpen")
		}
	})

	t.Run("unwrap preserves cause error", func(t *testing.T) {
		cause := fmt.Errorf("underlying error")
		err := NewCircuitBreakerError("circuit tripped", "CallAPI", "open", WithCause(cause))

		if !Is(err, cause) {
			t.Error("cause error should be in error chain")
		}
		if !Is(err, ErrCircuitOpen) {
			t.Error("sentinel should also be in error chain")
		}
	})

	t.Run("errors.As extraction works", func(t *testing.T) {
		err := NewCircuitBreakerError("circuit tripped", "CallAPI", "open",
			WithCounts(CircuitCounts{
				ConsecutiveFailures: 5,
				TotalFailures:       10,
			}))

		var cbErr *CircuitBreakerError
		if !As(err, &cbErr) {
			t.Fatal("errors.As should extract CircuitBreakerError")
		}

		if cbErr.State != "open" {
			t.Errorf("got State=%s, want open", cbErr.State)
		}
		if cbErr.Counts.ConsecutiveFailures != 5 {
			t.Errorf("got ConsecutiveFailures=%d, want 5", cbErr.Counts.ConsecutiveFailures)
		}
	})
}

// TestCircuitCounts tests CircuitCounts struct
func TestCircuitCounts(t *testing.T) {
	counts := CircuitCounts{
		Requests:             100,
		TotalSuccesses:       80,
		TotalFailures:        20,
		ConsecutiveSuccesses: 0,
		ConsecutiveFailures:  5,
	}

	if counts.Requests != 100 {
		t.Errorf("Requests: got %d, want 100", counts.Requests)
	}
	if counts.TotalSuccesses != 80 {
		t.Errorf("TotalSuccesses: got %d, want 80", counts.TotalSuccesses)
	}
	if counts.TotalFailures != 20 {
		t.Errorf("TotalFailures: got %d, want 20", counts.TotalFailures)
	}
	if counts.ConsecutiveSuccesses != 0 {
		t.Errorf("ConsecutiveSuccesses: got %d, want 0", counts.ConsecutiveSuccesses)
	}
	if counts.ConsecutiveFailures != 5 {
		t.Errorf("ConsecutiveFailures: got %d, want 5", counts.ConsecutiveFailures)
	}
}

// TestWithCounts tests the WithCounts option
func TestWithCounts(t *testing.T) {
	counts := CircuitCounts{
		Requests:            50,
		ConsecutiveFailures: 3,
	}

	err := NewCircuitBreakerError("circuit open", "API", "open", WithCounts(counts))

	if err.Counts.Requests != 50 {
		t.Errorf("got Requests=%d, want 50", err.Counts.Requests)
	}
	if err.Counts.ConsecutiveFailures != 3 {
		t.Errorf("got ConsecutiveFailures=%d, want 3", err.Counts.ConsecutiveFailures)
	}
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
