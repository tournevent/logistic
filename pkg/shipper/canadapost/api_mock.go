package canadapost

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
	OnGetLabel       func(ctx context.Context, shipmentID string, format string) (*LabelResponse, error)
	OnVoidShipment   func(ctx context.Context, shipmentID string) (*VoidResponse, error)
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
		return nil, &APIError{Code: "MOCK_ERROR", Description: "Simulated API error"}
	}

	if m.OnGetRates != nil {
		return m.OnGetRates(ctx, req)
	}

	quoteID := "cp-quote-" + uuid.New().String()[:8]
	deliveryDate := time.Now().AddDate(0, 0, 5).Format("2006-01-02")

	return &RatesResponse{
		QuoteID: quoteID,
		Rates: []Rate{
			{
				ServiceCode:       "DOM.RP",
				ServiceName:       "Regular Parcel",
				BaseRate:          9.99,
				FuelSurcharge:     1.20,
				Taxes:             1.46,
				TotalPrice:        12.65,
				ExpectedTransit:   5,
				ExpectedDelivery:  deliveryDate,
				GuaranteedDelivery: false,
			},
			{
				ServiceCode:       "DOM.XP",
				ServiceName:       "Xpresspost",
				BaseRate:          19.99,
				FuelSurcharge:     2.40,
				Taxes:             2.91,
				TotalPrice:        25.30,
				ExpectedTransit:   2,
				ExpectedDelivery:  time.Now().AddDate(0, 0, 2).Format("2006-01-02"),
				GuaranteedDelivery: true,
			},
			{
				ServiceCode:       "DOM.PC",
				ServiceName:       "Priority",
				BaseRate:          34.99,
				FuelSurcharge:     4.20,
				Taxes:             5.10,
				TotalPrice:        44.29,
				ExpectedTransit:   1,
				ExpectedDelivery:  time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
				GuaranteedDelivery: true,
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

	shipmentID := "cp-ship-" + uuid.New().String()[:8]
	trackingPIN := fmt.Sprintf("%d", 1000000000000+time.Now().UnixNano()%9000000000000)

	return &ShipmentResponse{
		ShipmentID:       shipmentID,
		TrackingPIN:      trackingPIN,
		ShipmentStatus:   "created",
		ServiceName:      "Regular Parcel",
		TotalCharged:     12.65,
		ExpectedDelivery: time.Now().AddDate(0, 0, 5).Format("2006-01-02"),
		Links: []Link{
			{Rel: "label", Href: fmt.Sprintf("https://api.canadapost.ca/rs/artifact/%s/label", shipmentID), MediaType: "application/pdf"},
			{Rel: "tracking", Href: fmt.Sprintf("https://www.canadapost-postescanada.ca/track-reperage/en#/search?searchFor=%s", trackingPIN)},
		},
	}, nil
}

// GetLabel retrieves a mock shipping label.
func (m *MockAPIClient) GetLabel(ctx context.Context, shipmentID string, format string) (*LabelResponse, error) {
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	if m.SimulateErrors {
		return nil, &APIError{Code: "MOCK_ERROR", Description: "Simulated API error"}
	}

	if m.OnGetLabel != nil {
		return m.OnGetLabel(ctx, shipmentID, format)
	}

	if format == "" {
		format = "application/pdf"
	}

	// Return mock PDF data (just a placeholder)
	return &LabelResponse{
		ShipmentID: shipmentID,
		Format:     format,
		Data:       []byte("%PDF-1.4 mock label data"),
	}, nil
}

// VoidShipment cancels a mock shipment.
func (m *MockAPIClient) VoidShipment(ctx context.Context, shipmentID string) (*VoidResponse, error) {
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	if m.SimulateErrors {
		return nil, &APIError{Code: "MOCK_ERROR", Description: "Simulated API error"}
	}

	if m.OnVoidShipment != nil {
		return m.OnVoidShipment(ctx, shipmentID)
	}

	return &VoidResponse{
		ShipmentID: shipmentID,
		Status:     "voided",
	}, nil
}

// GetTracking retrieves mock tracking information.
func (m *MockAPIClient) GetTracking(ctx context.Context, trackingNumber string) (*TrackingResponse, error) {
	if m.SimulateLatency > 0 {
		time.Sleep(m.SimulateLatency)
	}

	if m.SimulateErrors {
		return nil, &APIError{Code: "MOCK_ERROR", Description: "Simulated API error"}
	}

	if m.OnGetTracking != nil {
		return m.OnGetTracking(ctx, trackingNumber)
	}

	now := time.Now()
	return &TrackingResponse{
		TrackingPIN: trackingNumber,
		Status:      "in_transit",
		Events: []TrackingEvent{
			{
				Timestamp:   now.Add(-48 * time.Hour).Format(time.RFC3339),
				Description: "Item accepted at post office",
				Location:    "Toronto, ON",
				Type:        "accepted",
			},
			{
				Timestamp:   now.Add(-24 * time.Hour).Format(time.RFC3339),
				Description: "Item in transit",
				Location:    "Mississauga, ON",
				Type:        "in_transit",
			},
		},
	}, nil
}

var _ APIClient = (*MockAPIClient)(nil)
