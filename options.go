package errors

// Option is a functional option for configuring error creation.
// Use with error constructor functions to specify optional fields.
//
// Example:
//
//	err := NewProcessingError("Failed to process item", "ProcessItem",
//	    WithCause(dbErr),
//	    WithItemID("item-123"),
//	    WithRetryable(true))
type Option func(any)

// WithCause sets the underlying cause for an error.
// Use this to wrap lower-level errors while maintaining the error chain.
//
// Example:
//
//	dbErr := db.Query(...)
//	err := NewProcessingError("Failed to load user", "LoadUser",
//	    WithCause(dbErr))
func WithCause(cause error) Option {
	return func(err any) {
		switch e := err.(type) {
		case *HTTPError:
			e.Err = cause
		case *ValidationError:
			e.Err = cause
		case *TimeoutError:
			e.Err = cause
		case *RateLimitError:
			e.Err = cause
		case *ProcessingError:
			e.Err = cause
		case *NetworkError:
			e.Err = cause
		case *CircuitBreakerError:
			e.Err = cause
		}
	}
}

// WithRetryable sets whether a processing error is retryable.
// Only applies to ProcessingError types, ignored for others.
//
// Example:
//
//	err := NewProcessingError("Database connection lost", "SaveData",
//	    WithRetryable(true))
func WithRetryable(retryable bool) Option {
	return func(err any) {
		if e, ok := err.(*ProcessingError); ok {
			e.Retryable = retryable
		}
	}
}

// WithItemID sets the item ID for processing errors.
// Only applies to ProcessingError types, ignored for others.
//
// Example:
//
//	err := NewProcessingError("Failed to process activity", "ProcessActivity",
//	    WithItemID("activity-123"))
func WithItemID(itemID string) Option {
	return func(err any) {
		if e, ok := err.(*ProcessingError); ok {
			e.ItemID = itemID
		}
	}
}

// WithValue sets the value field for validation errors.
// Only applies to ValidationError types, ignored for others.
//
// Example:
//
//	err := NewValidationError("Price must be positive", "price",
//	    WithValue(-10))
func WithValue(value any) Option {
	return func(err any) {
		if e, ok := err.(*ValidationError); ok {
			e.Value = value
		}
	}
}

// WithOperation sets the operation name for errors that support it.
// Applies to TimeoutError, RateLimitError, ProcessingError, NetworkError, and CircuitBreakerError.
//
// Example:
//
//	err := NewTimeoutError("API call timed out", "GetUser", 30*time.Second)
func WithOperation(operation string) Option {
	return func(err any) {
		switch e := err.(type) {
		case *TimeoutError:
			e.Operation = operation
		case *RateLimitError:
			e.Operation = operation
		case *ProcessingError:
			e.Operation = operation
		case *NetworkError:
			e.Operation = operation
		case *CircuitBreakerError:
			e.Operation = operation
		}
	}
}

// WithMessage sets the message for errors that support it.
// Applies to most error types.
//
// Example:
//
//	err := NewHTTPError(500, "Internal Server Error", nil,
//	    WithMessage("Database connection failed"))
func WithMessage(message string) Option {
	return func(err any) {
		switch e := err.(type) {
		case *HTTPError:
			e.Message = message
		case *ValidationError:
			e.Message = message
		case *TimeoutError:
			e.Message = message
		case *RateLimitError:
			e.Message = message
		case *ProcessingError:
			e.Message = message
		case *NetworkError:
			e.Message = message
		case *CircuitBreakerError:
			e.Message = message
		}
	}
}

// WithStatusCode sets the HTTP status code.
// Only applies to HTTPError types, ignored for others.
//
// Example:
//
//	err := NewHTTPError(0, "Server error", cause,
//	    WithStatusCode(503))
func WithStatusCode(statusCode int) Option {
	return func(err any) {
		if e, ok := err.(*HTTPError); ok {
			e.StatusCode = statusCode
		}
	}
}

// WithField sets the field name for validation errors.
// Only applies to ValidationError types, ignored for others.
//
// Example:
//
//	err := NewValidationError("Invalid format", "",
//	    WithField("email"))
func WithField(field string) Option {
	return func(err any) {
		if e, ok := err.(*ValidationError); ok {
			e.Field = field
		}
	}
}

// WithTransient sets whether a network error is transient.
// Only applies to NetworkError types, ignored for others.
//
// Example:
//
//	err := NewNetworkError("DNS lookup failed", "Connect",
//	    WithTransient(false))
func WithTransient(transient bool) Option {
	return func(err any) {
		if e, ok := err.(*NetworkError); ok {
			e.IsTransient = transient
		}
	}
}

// WithState sets the circuit breaker state.
// Only applies to CircuitBreakerError types, ignored for others.
//
// Example:
//
//	err := NewCircuitBreakerError("Too many failures", "CallAPI", "",
//	    WithState("open"))
func WithState(state string) Option {
	return func(err any) {
		if e, ok := err.(*CircuitBreakerError); ok {
			e.State = state
		}
	}
}
