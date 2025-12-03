package freightcom

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MockAPIClient is a mock implementation of APIClient for testing.
type MockAPIClient struct {
	SimulateErrors  bool
	SimulateLatency time.Duration

	OnGetRates       func(ctx context.Context, req *RatesRequest) (*RatesResponse, error)
	OnCreateShipment func(ctx context.Context, req *ShipmentRequest) (*ShipmentResponse, error)
	OnGetLabel       func(ctx context.Context, orderID string, format string) (*LabelResponse, error)
	OnCancelShipment func(ctx context.Context, orderID string, reason string) (*CancelResponse, error)
	OnGetTracking    func(ctx context.Context, trackingNumber string) (*TrackingResponse, error)
}

// NewMockAPIClient creates a new mock API client with default behavior.
func NewMockAPIClient() *MockAPIClient {
	return &MockAPIClient{}
}

// GetRates returns mock shipping rates.
func (m *MockAPIClient) GetRates(ctx context.Context, req *RatesRequest) (*RatesResponse, error) {
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	if m.SimulateErrors {
		return nil, &APIError{Code: "MOCK_ERROR", Message: "Simulated API error"}
	}

	if m.OnGetRates != nil {
		return m.OnGetRates(ctx, req)
	}

	requestID := "fc-req-" + uuid.New().String()[:8]
	deliveryDate := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	expiresAt := time.Now().Add(30 * time.Minute).Format(time.RFC3339)

	return &RatesResponse{
		RequestID: requestID,
		Status:    "complete",
		Rates: []Rate{
			{
				ID:                "rate-" + uuid.New().String()[:8],
				ServiceID:         101,
				CarrierCode:       "fedex",
				CarrierName:       "FedEx",
				ServiceCode:       "FEDEX_GROUND",
				ServiceName:       "FedEx Ground",
				BaseRate:          15.99,
				FuelSurcharge:     1.92,
				TotalTax:          2.33,
				TotalPrice:        20.24,
				Currency:          "CAD",
				TransitDays:       3,
				EstimatedDelivery: deliveryDate,
				Guaranteed:        false,
				ExpiresAt:         expiresAt,
			},
			{
				ID:                "rate-" + uuid.New().String()[:8],
				ServiceID:         102,
				CarrierCode:       "fedex",
				CarrierName:       "FedEx",
				ServiceCode:       "FEDEX_EXPRESS_SAVER",
				ServiceName:       "FedEx Express Saver",
				BaseRate:          28.99,
				FuelSurcharge:     3.48,
				TotalTax:          4.22,
				TotalPrice:        36.69,
				Currency:          "CAD",
				TransitDays:       2,
				EstimatedDelivery: time.Now().AddDate(0, 0, 2).Format("2006-01-02"),
				Guaranteed:        true,
				ExpiresAt:         expiresAt,
			},
			{
				ID:                "rate-" + uuid.New().String()[:8],
				ServiceID:         201,
				CarrierCode:       "ups",
				CarrierName:       "UPS",
				ServiceCode:       "UPS_GROUND",
				ServiceName:       "UPS Ground",
				BaseRate:          14.50,
				FuelSurcharge:     1.74,
				TotalTax:          2.11,
				TotalPrice:        18.35,
				Currency:          "CAD",
				TransitDays:       4,
				EstimatedDelivery: time.Now().AddDate(0, 0, 4).Format("2006-01-02"),
				Guaranteed:        false,
				ExpiresAt:         expiresAt,
			},
		},
	}, nil
}

// CreateShipment creates a mock shipment.
func (m *MockAPIClient) CreateShipment(ctx context.Context, req *ShipmentRequest) (*ShipmentResponse, error) {
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	if m.SimulateErrors {
		return nil, &APIError{Code: "MOCK_ERROR", Message: "Simulated API error"}
	}

	if m.OnCreateShipment != nil {
		return m.OnCreateShipment(ctx, req)
	}

	shipmentID := "fc-ship-" + uuid.New().String()[:8]
	trackingNumber := fmt.Sprintf("%d", 100000000000+time.Now().UnixNano()%900000000000)

	return &ShipmentResponse{
		ID:                shipmentID,
		UniqueID:          req.UniqueID,
		PreviouslyCreated: false,
		Status:            "booked",
		TrackingNumbers:   []string{trackingNumber},
		TrackingURL:       fmt.Sprintf("https://www.fedex.com/fedextrack/?trknbr=%s", trackingNumber),
		CarrierCode:       "fedex",
		ServiceName:       "FedEx Ground",
		TotalCharged:      20.24,
		Currency:          "CAD",
		EstimatedDelivery: time.Now().AddDate(0, 0, 3).Format("2006-01-02"),
		Labels: []Label{
			{
				Size:   "4x6",
				Format: "pdf",
				URL:    fmt.Sprintf("https://api.freightcom.com/shipment/%s/label.pdf", shipmentID),
			},
		},
	}, nil
}

// GetLabel retrieves a mock shipping label.
func (m *MockAPIClient) GetLabel(ctx context.Context, shipmentID string, format string) (*LabelResponse, error) {
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	if m.SimulateErrors {
		return nil, &APIError{Code: "MOCK_ERROR", Message: "Simulated API error"}
	}

	if m.OnGetLabel != nil {
		return m.OnGetLabel(ctx, shipmentID, format)
	}

	if format == "" {
		format = "pdf"
	}

	return &LabelResponse{
		ShipmentID: shipmentID,
		Labels: []Label{
			{
				Size:   "4x6",
				Format: format,
				URL:    fmt.Sprintf("https://api.freightcom.com/shipment/%s/label.%s", shipmentID, format),
			},
		},
	}, nil
}

// CancelShipment cancels a mock shipment.
func (m *MockAPIClient) CancelShipment(ctx context.Context, shipmentID string, reason string) (*CancelResponse, error) {
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	if m.SimulateErrors {
		return nil, &APIError{Code: "MOCK_ERROR", Message: "Simulated API error"}
	}

	if m.OnCancelShipment != nil {
		return m.OnCancelShipment(ctx, shipmentID, reason)
	}

	return &CancelResponse{
		ShipmentID:         shipmentID,
		Status:             "cancelled",
		RefundAmount:       20.24,
		Currency:           "CAD",
		ConfirmationNumber: fmt.Sprintf("CANCEL-%d", time.Now().UnixNano()%1000000),
	}, nil
}

// GetTracking retrieves mock tracking information.
func (m *MockAPIClient) GetTracking(ctx context.Context, shipmentID string) (*TrackingResponse, error) {
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	if m.SimulateErrors {
		return nil, &APIError{Code: "MOCK_ERROR", Message: "Simulated API error"}
	}

	if m.OnGetTracking != nil {
		return m.OnGetTracking(ctx, shipmentID)
	}

	now := time.Now()
	return &TrackingResponse{
		ShipmentID:     shipmentID,
		TrackingNumber: "123456789012",
		Status:         "in_transit",
		Events: []TrackingEvent{
			{
				Timestamp:   now.Add(-48 * time.Hour).Format(time.RFC3339),
				Description: "Shipment picked up",
				Location:    "Toronto, ON",
				Status:      "picked_up",
				Code:        "PU",
			},
			{
				Timestamp:   now.Add(-24 * time.Hour).Format(time.RFC3339),
				Description: "In transit to destination",
				Location:    "Mississauga, ON",
				Status:      "in_transit",
				Code:        "IT",
			},
		},
	}, nil
}

var _ APIClient = (*MockAPIClient)(nil)
