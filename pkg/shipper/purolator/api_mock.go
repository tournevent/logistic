package purolator

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
	OnGetLabel       func(ctx context.Context, shipmentPIN string, format string) (*LabelResponse, error)
	OnVoidShipment   func(ctx context.Context, shipmentPIN string) (*VoidResponse, error)
	OnGetTracking    func(ctx context.Context, trackingPIN string) (*TrackingResponse, error)
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
		return nil, &APIError{Code: "MOCK_ERROR", Description: "Simulated API error"}
	}

	if m.OnGetRates != nil {
		return m.OnGetRates(ctx, req)
	}

	quoteID := "puro-quote-" + uuid.New().String()[:8]
	deliveryGround := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	deliveryExpress := time.Now().AddDate(0, 0, 2).Format("2006-01-02")
	deliveryAM := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	return &RatesResponse{
		QuoteID: quoteID,
		ShipmentRates: []ShipmentRate{
			{
				ServiceCode:          "PurolatorGround",
				ServiceName:          "Purolator Ground",
				BasePrice:            16.75,
				FuelSurcharge:        2.01,
				Taxes:                2.44,
				TotalPrice:           21.20,
				ExpectedDeliveryDate: deliveryGround,
				EstimatedTransitDays: 5,
				GuaranteedDelivery:   false,
			},
			{
				ServiceCode:          "PurolatorExpress",
				ServiceName:          "Purolator Express",
				BasePrice:            28.50,
				FuelSurcharge:        3.42,
				Taxes:                4.15,
				TotalPrice:           36.07,
				ExpectedDeliveryDate: deliveryExpress,
				EstimatedTransitDays: 2,
				GuaranteedDelivery:   true,
			},
			{
				ServiceCode:          "PurolatorExpress9AM",
				ServiceName:          "Purolator Express 9AM",
				BasePrice:            45.00,
				FuelSurcharge:        5.40,
				Taxes:                6.55,
				TotalPrice:           56.95,
				ExpectedDeliveryDate: deliveryAM,
				EstimatedTransitDays: 1,
				GuaranteedDelivery:   true,
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
		return nil, &APIError{Code: "MOCK_ERROR", Description: "Simulated API error"}
	}

	if m.OnCreateShipment != nil {
		return m.OnCreateShipment(ctx, req)
	}

	shipmentPIN := "puro-ship-" + uuid.New().String()[:8]
	trackingNumber := fmt.Sprintf("329%012d", time.Now().UnixNano()%1000000000000)

	return &ShipmentResponse{
		ShipmentPIN:          shipmentPIN,
		TrackingNumber:       trackingNumber,
		TotalPrice:           21.20,
		ExpectedDeliveryDate: time.Now().AddDate(0, 0, 5).Format("2006-01-02"),
		PiecePINs:            []string{trackingNumber},
		DocumentLinks: []DocumentLink{
			{Type: "Label", URL: fmt.Sprintf("https://eship.purolator.com/shipment/%s/label.pdf", shipmentPIN)},
		},
	}, nil
}

// GetLabel retrieves a mock shipping label.
func (m *MockAPIClient) GetLabel(ctx context.Context, shipmentPIN string, format string) (*LabelResponse, error) {
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	if m.SimulateErrors {
		return nil, &APIError{Code: "MOCK_ERROR", Description: "Simulated API error"}
	}

	if m.OnGetLabel != nil {
		return m.OnGetLabel(ctx, shipmentPIN, format)
	}

	if format == "" {
		format = "application/pdf"
	}

	// Return mock PDF data (just a placeholder)
	return &LabelResponse{
		ShipmentPIN: shipmentPIN,
		Format:      format,
		Data:        []byte("%PDF-1.4 mock purolator label data"),
	}, nil
}

// VoidShipment cancels a mock shipment.
func (m *MockAPIClient) VoidShipment(ctx context.Context, shipmentPIN string) (*VoidResponse, error) {
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	if m.SimulateErrors {
		return nil, &APIError{Code: "MOCK_ERROR", Description: "Simulated API error"}
	}

	if m.OnVoidShipment != nil {
		return m.OnVoidShipment(ctx, shipmentPIN)
	}

	return &VoidResponse{
		ShipmentPIN: shipmentPIN,
		Status:      "voided",
		Message:     "Shipment successfully voided",
	}, nil
}

// GetTracking retrieves mock tracking information.
func (m *MockAPIClient) GetTracking(ctx context.Context, trackingPIN string) (*TrackingResponse, error) {
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	if m.SimulateErrors {
		return nil, &APIError{Code: "MOCK_ERROR", Description: "Simulated API error"}
	}

	if m.OnGetTracking != nil {
		return m.OnGetTracking(ctx, trackingPIN)
	}

	now := time.Now()
	return &TrackingResponse{
		TrackingPIN:    trackingPIN,
		Status:         "InTransit",
		DeliveryStatus: "On Schedule",
		Events: []TrackingEvent{
			{
				Timestamp:   now.Add(-48 * time.Hour).Format(time.RFC3339),
				Description: "Picked up by Purolator",
				Location:    "Toronto, ON",
				Type:        "PickedUp",
			},
			{
				Timestamp:   now.Add(-24 * time.Hour).Format(time.RFC3339),
				Description: "In transit to destination",
				Location:    "Mississauga, ON",
				Type:        "InTransit",
			},
		},
	}, nil
}

var _ APIClient = (*MockAPIClient)(nil)
