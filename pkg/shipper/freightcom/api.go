package freightcom

import (
	"context"
)

// APIClient defines the interface for Freightcom API operations.
// This abstraction allows for mock implementations during testing
// and real implementations in production.
type APIClient interface {
	// GetRates fetches shipping rates from Freightcom API
	GetRates(ctx context.Context, req *RatesRequest) (*RatesResponse, error)

	// CreateShipment creates a new shipment order
	CreateShipment(ctx context.Context, req *ShipmentRequest) (*ShipmentResponse, error)

	// GetLabel retrieves the shipping label for an order
	GetLabel(ctx context.Context, orderID string, format string) (*LabelResponse, error)

	// CancelShipment cancels an existing shipment
	CancelShipment(ctx context.Context, orderID string, reason string) (*CancelResponse, error)

	// GetTracking retrieves tracking information
	GetTracking(ctx context.Context, trackingNumber string) (*TrackingResponse, error)
}

// ============================================================================
// API Request/Response Types (match Freightcom REST API v2 structure)
// ============================================================================

// RatesRequest represents a Freightcom rate quote request.
// POST /rate endpoint
type RatesRequest struct {
	Services         []int           `json:"services,omitempty"`          // Service IDs to query (all if omitted)
	ExcludedServices []int           `json:"excluded_services,omitempty"` // Services to exclude
	Details          ShippingDetails `json:"details"`
}

// ShippingDetails contains shipping information for rate requests.
type ShippingDetails struct {
	Origin      Location      `json:"origin"`
	Destination Location      `json:"destination"`
	Packaging   PackagingInfo `json:"packaging"`
	CustomsData *CustomsData  `json:"customs_data,omitempty"` // For international
}

// Location represents origin or destination.
type Location struct {
	Name        string `json:"name,omitempty"`
	Company     string `json:"company,omitempty"`
	Address1    string `json:"address_1"`
	Address2    string `json:"address_2,omitempty"`
	City        string `json:"city"`
	Province    string `json:"province"`
	PostalCode  string `json:"postal_code"`
	Country     string `json:"country"` // ISO 3166-1 alpha-2 code
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	Residential bool   `json:"residential,omitempty"`
}

// PackagingInfo contains package details.
type PackagingInfo struct {
	Type     string    `json:"type"` // "package", "envelope", "pallet"
	Packages []Package `json:"packages"`
}

// Package represents a single package.
type Package struct {
	Length      float64 `json:"length"`       // cm
	Width       float64 `json:"width"`        // cm
	Height      float64 `json:"height"`       // cm
	Weight      float64 `json:"weight"`       // kg
	Description string  `json:"description,omitempty"`
	Quantity    int     `json:"quantity,omitempty"`
}

// CustomsData for international shipments.
type CustomsData struct {
	Description     string        `json:"description"`
	ReasonForExport string        `json:"reason_for_export"` // "sale", "gift", "sample", etc.
	Currency        string        `json:"currency"`          // CAD, USD
	Items           []CustomsItem `json:"items"`
}

// CustomsItem represents a customs declaration item.
type CustomsItem struct {
	Description   string  `json:"description"`
	Quantity      int     `json:"quantity"`
	Value         float64 `json:"value"`
	Weight        float64 `json:"weight"`
	CountryOrigin string  `json:"country_of_origin"`
	HSCode        string  `json:"hs_code,omitempty"`
}

// RateRequestResponse is the initial response from POST /rate (async).
type RateRequestResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"` // "pending", "complete", "error"
}

// RatesResponse represents the Freightcom rate quote response.
// GET /rate/{rate_id} endpoint
type RatesResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"` // "pending", "complete", "error"
	Rates     []Rate `json:"rates,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Rate represents a single shipping rate option.
type Rate struct {
	ID                string     `json:"id"`
	ServiceID         int        `json:"service_id"`
	CarrierCode       string     `json:"carrier_code"`
	CarrierName       string     `json:"carrier_name"`
	ServiceCode       string     `json:"service_code"`
	ServiceName       string     `json:"service_name"`
	BaseRate          float64    `json:"base_rate"`
	FuelSurcharge     float64    `json:"fuel_surcharge"`
	Surcharges        []Surcharge `json:"surcharges,omitempty"`
	Taxes             []Tax      `json:"taxes,omitempty"`
	TotalTax          float64    `json:"total_tax"`
	TotalPrice        float64    `json:"total_price"`
	Currency          string     `json:"currency"`
	TransitDays       int        `json:"transit_days"`
	TransitDaysMin    int        `json:"transit_days_min,omitempty"`
	TransitDaysMax    int        `json:"transit_days_max,omitempty"`
	EstimatedDelivery string     `json:"estimated_delivery,omitempty"`
	Guaranteed        bool       `json:"guaranteed"`
	ExpiresAt         string     `json:"expires_at"`
}

// Surcharge represents an additional charge.
type Surcharge struct {
	Code        string  `json:"code"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

// Tax represents a tax component.
type Tax struct {
	Code   string  `json:"code"`
	Rate   float64 `json:"rate"`
	Amount float64 `json:"amount"`
}

// ShipmentRequest represents a Freightcom shipment creation request.
// POST /shipment endpoint
type ShipmentRequest struct {
	UniqueID        string          `json:"unique_id"`         // Max 128 chars, prevents duplicates
	PaymentMethodID int             `json:"payment_method_id"` // From /finance/payment-methods
	ServiceID       int             `json:"service_id"`
	Details         ShippingDetails `json:"details"`
	Sender          Contact         `json:"sender"`
	Recipient       Contact         `json:"recipient"`
	Reference       string          `json:"reference,omitempty"`
	PONumber        string          `json:"po_number,omitempty"`
	Instructions    string          `json:"instructions,omitempty"`
	CustomsInvoice  *CustomsInvoice `json:"customs_invoice,omitempty"` // For international
	PickupDetails   *PickupDetails  `json:"pickup_details,omitempty"`
}

// Contact represents sender/recipient contact info.
type Contact struct {
	Name       string `json:"name"`
	Company    string `json:"company,omitempty"`
	Phone      string `json:"phone"`
	Email      string `json:"email,omitempty"`
	AttentionTo string `json:"attention_to,omitempty"`
}

// CustomsInvoice for international shipments.
type CustomsInvoice struct {
	InvoiceNumber string        `json:"invoice_number,omitempty"`
	InvoiceDate   string        `json:"invoice_date,omitempty"`
	Items         []CustomsItem `json:"items"`
}

// PickupDetails for scheduling pickup.
type PickupDetails struct {
	Date        string `json:"date"`         // YYYY-MM-DD
	ReadyTime   string `json:"ready_time"`   // HH:MM
	ClosingTime string `json:"closing_time"` // HH:MM
	Instructions string `json:"instructions,omitempty"`
}

// ShipmentResponse represents the Freightcom shipment creation response.
type ShipmentResponse struct {
	ID                string   `json:"id"`
	UniqueID          string   `json:"unique_id"`
	PreviouslyCreated bool     `json:"previously_created"`
	Status            string   `json:"status"`
	TrackingNumbers   []string `json:"tracking_numbers"`
	TrackingURL       string   `json:"tracking_url,omitempty"`
	CarrierCode       string   `json:"carrier_code"`
	ServiceName       string   `json:"service_name"`
	TotalCharged      float64  `json:"total_charged"`
	Currency          string   `json:"currency"`
	EstimatedDelivery string   `json:"estimated_delivery,omitempty"`
	Labels            []Label  `json:"labels,omitempty"`
}

// Label represents a shipping label.
type Label struct {
	Size   string `json:"size"`   // "4x6", "letter"
	Format string `json:"format"` // "pdf", "zpl", "png"
	URL    string `json:"url"`
}

// LabelResponse represents the Freightcom label response.
// Obtained from GET /shipment/{shipment_id}
type LabelResponse struct {
	ShipmentID string  `json:"shipment_id"`
	Labels     []Label `json:"labels"`
}

// CancelResponse represents the Freightcom cancellation response.
// DELETE /shipment/{shipment_id}
type CancelResponse struct {
	ShipmentID         string  `json:"shipment_id"`
	Status             string  `json:"status"`
	RefundAmount       float64 `json:"refund_amount,omitempty"`
	Currency           string  `json:"currency,omitempty"`
	ConfirmationNumber string  `json:"confirmation_number,omitempty"`
}

// TrackingResponse represents tracking information.
// GET /shipment/{shipment_id}/tracking-events
type TrackingResponse struct {
	ShipmentID     string          `json:"shipment_id"`
	TrackingNumber string          `json:"tracking_number"`
	Status         string          `json:"status"`
	Events         []TrackingEvent `json:"events"`
}

// TrackingEvent represents a single tracking event.
type TrackingEvent struct {
	Timestamp   string `json:"timestamp"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Status      string `json:"status"`
	Code        string `json:"code,omitempty"`
}

// APIError represents an error from the Freightcom API.
type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors,omitempty"` // Field-level errors
}

func (e *APIError) Error() string {
	return e.Code + ": " + e.Message
}
