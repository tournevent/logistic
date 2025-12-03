package shipper_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tournevent/logistic/pkg/shipper"
)

func TestShipperError_Error(t *testing.T) {
	err := shipper.NewShipperError("freightcom", "INVALID_ADDRESS", "Invalid postal code")
	assert.Equal(t, "freightcom error (INVALID_ADDRESS): Invalid postal code", err.Error())
}

func TestShipperError_ErrorWithCause(t *testing.T) {
	cause := errors.New("network timeout")
	err := shipper.NewShipperError("freightcom", "API_ERROR", "API call failed").WithCause(cause)
	assert.Contains(t, err.Error(), "API call failed")
	assert.Contains(t, err.Error(), "network timeout")
}

func TestShipperError_Unwrap(t *testing.T) {
	cause := errors.New("network timeout")
	err := shipper.NewShipperError("freightcom", "API_ERROR", "API call failed").WithCause(cause)
	assert.True(t, errors.Is(err, cause))
}

func TestShipperError_Is(t *testing.T) {
	err1 := shipper.NewShipperError("freightcom", "INVALID_ADDRESS", "Invalid postal code")
	err2 := shipper.NewShipperError("canadapost", "INVALID_ADDRESS", "Different message")

	// Same code should match
	assert.True(t, errors.Is(err1, err2))
}

func TestShipperError_IsNot(t *testing.T) {
	err1 := shipper.NewShipperError("freightcom", "INVALID_ADDRESS", "Invalid postal code")
	err2 := shipper.NewShipperError("freightcom", "DIFFERENT_CODE", "Different error")

	// Different codes should not match
	assert.False(t, errors.Is(err1, err2))
}

func TestShipperError_WithStatusCode(t *testing.T) {
	err := shipper.NewShipperError("freightcom", "AUTH_ERROR", "Unauthorized").WithStatusCode(401)
	assert.Equal(t, 401, err.StatusCode)
}

func TestShipperError_WithRetryable(t *testing.T) {
	err := shipper.NewShipperError("freightcom", "RATE_LIMIT", "Too many requests").WithRetryable(true)
	assert.True(t, err.Retryable)
}

func TestIsRetryable_ShipperError(t *testing.T) {
	err := shipper.NewShipperError("freightcom", "RATE_LIMIT", "Too many requests").WithRetryable(true)
	assert.True(t, shipper.IsRetryable(err))
}

func TestIsRetryable_ShipperErrorNotRetryable(t *testing.T) {
	err := shipper.NewShipperError("freightcom", "INVALID_ADDRESS", "Bad address").WithRetryable(false)
	assert.False(t, shipper.IsRetryable(err))
}

func TestIsRetryable_ServiceUnavailable(t *testing.T) {
	assert.True(t, shipper.IsRetryable(shipper.ErrServiceUnavailable))
}

func TestIsRetryable_RateLimitExceeded(t *testing.T) {
	assert.True(t, shipper.IsRetryable(shipper.ErrRateLimitExceeded))
}

func TestIsRetryable_InvalidAddress(t *testing.T) {
	assert.False(t, shipper.IsRetryable(shipper.ErrInvalidAddress))
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrInvalidAddress", shipper.ErrInvalidAddress},
		{"ErrServiceUnavailable", shipper.ErrServiceUnavailable},
		{"ErrQuoteExpired", shipper.ErrQuoteExpired},
		{"ErrQuoteNotFound", shipper.ErrQuoteNotFound},
		{"ErrOrderNotFound", shipper.ErrOrderNotFound},
		{"ErrCancellationNotAllowed", shipper.ErrCancellationNotAllowed},
		{"ErrLabelNotAvailable", shipper.ErrLabelNotAvailable},
		{"ErrAuthenticationFailed", shipper.ErrAuthenticationFailed},
		{"ErrRateLimitExceeded", shipper.ErrRateLimitExceeded},
		{"ErrInvalidPackage", shipper.ErrInvalidPackage},
		{"ErrCarrierNotFound", shipper.ErrCarrierNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}
