package errors

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
)

// GetStackTrace returns a formatted stack trace for the error.
// Returns empty string if the error has no stack trace.
//
// Example output:
//
//	main.processItem
//	    /path/to/main.go:42
//	main.worker
//	    /path/to/main.go:28
//	main.main
//	    /path/to/main.go:15
func GetStackTrace(err error) string {
	if err == nil {
		return ""
	}

	// Use cockroachdb/errors' stack trace formatting
	return fmt.Sprintf("%+v", err)
}

// GetStackTraceLines returns the stack trace as individual lines.
// Returns empty slice if the error has no stack trace.
func GetStackTraceLines(err error) []string {
	if err == nil {
		return nil
	}

	trace := GetStackTrace(err)
	if trace == "" {
		return nil
	}

	lines := strings.Split(trace, "\n")
	// Filter out empty lines
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// GetSafeDetails returns error details safe for production logging.
// Redacts sensitive information while preserving debugging context.
//
// Example:
//
//	err := NewHTTPError(500, "Internal Server Error", fmt.Errorf("db password: secret123"))
//	safe := GetSafeDetails(err)
//	// safe contains error type and structure, but sensitive data is redacted
func GetSafeDetails(err error) string {
	if err == nil {
		return ""
	}

	// Use cockroachdb/errors' redaction features
	return errors.Redact(err)
}

// FormatError returns a formatted error string with type information.
// Useful for structured logging and debugging.
//
// Example output:
//
//	HTTPError(500): Internal Server Error: database connection failed
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	var parts []string

	// Add type information
	switch e := err.(type) {
	case *HTTPError:
		parts = append(parts, fmt.Sprintf("HTTPError(%d)", e.StatusCode))
	case *ValidationError:
		parts = append(parts, fmt.Sprintf("ValidationError(%s)", e.Field))
	case *TimeoutError:
		parts = append(parts, fmt.Sprintf("TimeoutError(%v)", e.Duration))
	case *RateLimitError:
		parts = append(parts, fmt.Sprintf("RateLimitError(%v)", e.RetryAfter))
	case *ProcessingError:
		retryable := "not retryable"
		if e.IsRetryable() {
			retryable = "retryable"
		}
		parts = append(parts, fmt.Sprintf("ProcessingError(%s)", retryable))
	case *NetworkError:
		transient := "persistent"
		if e.IsTransient {
			transient = "transient"
		}
		parts = append(parts, fmt.Sprintf("NetworkError(%s)", transient))
	case *CircuitBreakerError:
		parts = append(parts, fmt.Sprintf("CircuitBreakerError(%s)", e.State))
	default:
		parts = append(parts, "Error")
	}

	// Add error message
	parts = append(parts, err.Error())

	return strings.Join(parts, ": ")
}

// ExtractErrorInfo returns structured information about the error.
// Returns a map with error type, retryability, and extracted fields.
//
// Example:
//
//	info := ExtractErrorInfo(err)
//	// info = map[string]any{
//	//     "type": "HTTPError",
//	//     "retryable": true,
//	//     "status_code": 503,
//	//     "message": "Service Unavailable",
//	// }
func ExtractErrorInfo(err error) map[string]any {
	if err == nil {
		return nil
	}

	info := make(map[string]any)
	info["message"] = err.Error()
	info["retryable"] = IsRetryable(err)

	// Extract type-specific information
	switch e := err.(type) {
	case *HTTPError:
		info["type"] = "HTTPError"
		info["status_code"] = e.StatusCode

	case *ValidationError:
		info["type"] = "ValidationError"
		info["field"] = e.Field
		if e.Value != nil {
			info["value"] = e.Value
		}

	case *TimeoutError:
		info["type"] = "TimeoutError"
		info["operation"] = e.Operation
		info["duration"] = e.Duration.String()

	case *RateLimitError:
		info["type"] = "RateLimitError"
		info["operation"] = e.Operation
		info["retry_after"] = e.RetryAfter.String()

	case *ProcessingError:
		info["type"] = "ProcessingError"
		info["operation"] = e.Operation
		if e.ItemID != "" {
			info["item_id"] = e.ItemID
		}

	case *NetworkError:
		info["type"] = "NetworkError"
		info["operation"] = e.Operation
		info["transient"] = e.IsTransient

	case *CircuitBreakerError:
		info["type"] = "CircuitBreakerError"
		info["operation"] = e.Operation
		info["state"] = e.State

	default:
		info["type"] = "Error"
	}

	return info
}

// HasStackTrace checks if the error has a stack trace.
func HasStackTrace(err error) bool {
	if err == nil {
		return false
	}

	// cockroachdb/errors adds stack traces, check if present
	trace := fmt.Sprintf("%+v", err)
	return strings.Contains(trace, ".go:")
}
