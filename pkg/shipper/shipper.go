// Package shipper provides an abstraction layer for shipping carriers.
package shipper

import (
	"context"
)

// Shipper defines the interface that all shipping carriers must implement.
type Shipper interface {
	// Name returns the carrier identifier (e.g., "freightcom", "canadapost", "purolator").
	Name() string

	// GetQuote returns shipping rate quotes for a shipment.
	GetQuote(ctx context.Context, req *QuoteRequest) (*QuoteResponse, error)

	// CreateOrder creates a new shipment with the carrier.
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error)

	// GetLabel retrieves the shipping label for an order.
	GetLabel(ctx context.Context, req *GetLabelRequest) (*GetLabelResponse, error)

	// CancelOrder cancels an existing shipment.
	CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error)
}
