// Package mock provides a mock shipper implementation for testing.
package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/tournevent/logistic/pkg/shipper"
)

// Client is a mock shipper for testing.
type Client struct {
	name string
}

// New creates a new mock shipper.
func New(name string) *Client {
	return &Client{name: name}
}

// Name returns the carrier name.
func (c *Client) Name() string {
	return c.name
}

// GetQuote returns mock shipping quotes.
func (c *Client) GetQuote(ctx context.Context, req *shipper.QuoteRequest) (*shipper.QuoteResponse, error) {
	now := time.Now()
	expiresAt := now.Add(30 * time.Minute)
	estimatedDelivery := now.Add(5 * 24 * time.Hour)

	return &shipper.QuoteResponse{
		QuoteID:   fmt.Sprintf("%s-quote-%d", c.name, now.UnixNano()),
		ExpiresAt: expiresAt,
		Rates: []shipper.RateOption{
			{
				RateID:      fmt.Sprintf("%s-rate-standard-%d", c.name, now.UnixNano()),
				Carrier:     c.name,
				ServiceCode: "STANDARD",
				ServiceName: fmt.Sprintf("%s Standard", c.name),
				ServiceType: shipper.ServiceStandard,
				BaseRate:    shipper.Money{Amount: 12.50, Currency: "CAD"},
				FuelSurcharge: shipper.Money{Amount: 1.50, Currency: "CAD"},
				Taxes:       shipper.Money{Amount: 1.82, Currency: "CAD"},
				TotalPrice:  shipper.Money{Amount: 15.82, Currency: "CAD"},
				TransitDays: 5,
				EstimatedDelivery: &estimatedDelivery,
				ExpiresAt:   expiresAt,
				Guaranteed:  false,
			},
			{
				RateID:      fmt.Sprintf("%s-rate-express-%d", c.name, now.UnixNano()),
				Carrier:     c.name,
				ServiceCode: "EXPRESS",
				ServiceName: fmt.Sprintf("%s Express", c.name),
				ServiceType: shipper.ServiceExpress,
				BaseRate:    shipper.Money{Amount: 24.00, Currency: "CAD"},
				FuelSurcharge: shipper.Money{Amount: 2.50, Currency: "CAD"},
				Taxes:       shipper.Money{Amount: 3.45, Currency: "CAD"},
				TotalPrice:  shipper.Money{Amount: 29.95, Currency: "CAD"},
				TransitDays: 2,
				EstimatedDelivery: func() *time.Time { t := now.Add(2 * 24 * time.Hour); return &t }(),
				ExpiresAt:   expiresAt,
				Guaranteed:  true,
			},
		},
	}, nil
}

// CreateOrder creates a mock shipping order.
func (c *Client) CreateOrder(ctx context.Context, req *shipper.CreateOrderRequest) (*shipper.CreateOrderResponse, error) {
	now := time.Now()
	orderID := fmt.Sprintf("%s-order-%d", c.name, now.UnixNano())
	trackingNumber := fmt.Sprintf("1Z%s%d", c.name[:3], now.UnixNano()%1000000000)
	estimatedDelivery := now.Add(5 * 24 * time.Hour)

	return &shipper.CreateOrderResponse{
		OrderID:        orderID,
		TrackingNumber: trackingNumber,
		TrackingURL:    fmt.Sprintf("https://track.%s.mock/track/%s", c.name, trackingNumber),
		Status:         shipper.StatusConfirmed,
		Carrier:        c.name,
		ServiceName:    fmt.Sprintf("%s Standard", c.name),
		TotalCharged:   shipper.Money{Amount: 15.82, Currency: "CAD"},
		EstimatedDelivery: &estimatedDelivery,
		LabelURL:       fmt.Sprintf("https://labels.%s.mock/%s.pdf", c.name, orderID),
	}, nil
}

// GetLabel returns a mock shipping label.
func (c *Client) GetLabel(ctx context.Context, req *shipper.GetLabelRequest) (*shipper.GetLabelResponse, error) {
	format := req.Format
	if format == "" {
		format = shipper.LabelPDF
	}

	return &shipper.GetLabelResponse{
		OrderID: req.OrderID,
		Label: shipper.Label{
			Format: format,
			URL:    fmt.Sprintf("https://labels.%s.mock/%s.%s", c.name, req.OrderID, format),
		},
	}, nil
}

// CancelOrder cancels a mock shipping order.
func (c *Client) CancelOrder(ctx context.Context, req *shipper.CancelOrderRequest) (*shipper.CancelOrderResponse, error) {
	return &shipper.CancelOrderResponse{
		OrderID:            req.OrderID,
		Status:             shipper.StatusCancelled,
		RefundAmount:       &shipper.Money{Amount: 15.82, Currency: "CAD"},
		ConfirmationNumber: fmt.Sprintf("CANCEL-%d", time.Now().UnixNano()),
	}, nil
}
