package shipper

import (
	"errors"
	"fmt"
)

// ShipperError represents an error from a shipping carrier.
type ShipperError struct {
	Carrier    string
	Code       string
	Message    string
	StatusCode int
	Retryable  bool
	Cause      error
}

// Error implements the error interface.
func (e *ShipperError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s error (%s): %s: %v", e.Carrier, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s error (%s): %s", e.Carrier, e.Code, e.Message)
}

// Unwrap returns the underlying cause.
func (e *ShipperError) Unwrap() error {
	return e.Cause
}

// Is implements errors.Is for ShipperError.
func (e *ShipperError) Is(target error) bool {
	t, ok := target.(*ShipperError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// NewShipperError creates a new ShipperError.
func NewShipperError(carrier, code, message string) *ShipperError {
	return &ShipperError{
		Carrier: carrier,
		Code:    code,
		Message: message,
	}
}

// WithCause adds a cause to the error.
func (e *ShipperError) WithCause(err error) *ShipperError {
	e.Cause = err
	return e
}

// WithStatusCode adds an HTTP status code to the error.
func (e *ShipperError) WithStatusCode(code int) *ShipperError {
	e.StatusCode = code
	return e
}

// WithRetryable marks the error as retryable.
func (e *ShipperError) WithRetryable(retryable bool) *ShipperError {
	e.Retryable = retryable
	return e
}

// Sentinel errors for common shipping scenarios.
var (
	// ErrInvalidAddress indicates the address is invalid or incomplete.
	ErrInvalidAddress = errors.New("invalid address")

	// ErrServiceUnavailable indicates the carrier service is temporarily unavailable.
	ErrServiceUnavailable = errors.New("service unavailable")

	// ErrQuoteExpired indicates the quote has expired and cannot be used.
	ErrQuoteExpired = errors.New("quote has expired")

	// ErrQuoteNotFound indicates the quote ID was not found.
	ErrQuoteNotFound = errors.New("quote not found")

	// ErrOrderNotFound indicates the order ID was not found.
	ErrOrderNotFound = errors.New("order not found")

	// ErrCancellationNotAllowed indicates the order cannot be cancelled.
	ErrCancellationNotAllowed = errors.New("cancellation not allowed")

	// ErrLabelNotAvailable indicates the label is not yet available.
	ErrLabelNotAvailable = errors.New("label not available")

	// ErrAuthenticationFailed indicates carrier authentication failed.
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrRateLimitExceeded indicates the carrier rate limit was exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrInvalidPackage indicates package dimensions or weight are invalid.
	ErrInvalidPackage = errors.New("invalid package")

	// ErrCarrierNotFound indicates the requested carrier is not registered.
	ErrCarrierNotFound = errors.New("carrier not found")
)

// IsRetryable returns true if the error is retryable.
func IsRetryable(err error) bool {
	var shipperErr *ShipperError
	if errors.As(err, &shipperErr) {
		return shipperErr.Retryable
	}
	return errors.Is(err, ErrServiceUnavailable) || errors.Is(err, ErrRateLimitExceeded)
}
