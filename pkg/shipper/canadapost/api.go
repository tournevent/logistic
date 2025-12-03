package canadapost

import (
	"context"
)

// APIClient defines the interface for Canada Post API operations.
// This abstraction allows for mock implementations during testing
// and real implementations in production.
type APIClient interface {
	// GetRates fetches shipping rates from Canada Post API
	GetRates(ctx context.Context, req *RatesRequest) (*RatesResponse, error)

	// CreateShipment creates a new shipment order
	CreateShipment(ctx context.Context, req *ShipmentRequest) (*ShipmentResponse, error)

	// GetLabel retrieves the shipping label (artifact) for a shipment
	GetLabel(ctx context.Context, shipmentID string, format string) (*LabelResponse, error)

	// VoidShipment voids/cancels an existing shipment
	VoidShipment(ctx context.Context, shipmentID string) (*VoidResponse, error)

	// GetTracking retrieves tracking information
	GetTracking(ctx context.Context, trackingNumber string) (*TrackingResponse, error)
}

// ============================================================================
// API Request/Response Types (match Canada Post REST/XML API structure)
// ============================================================================

// RatesRequest represents a Canada Post rate quote request.
type RatesRequest struct {
	CustomerNumber string     `xml:"customer-number"`
	ParcelType     string     `xml:"parcel-characteristics>parcel-type"`
	Weight         float64    `xml:"parcel-characteristics>weight"`
	Dimensions     Dimensions `xml:"parcel-characteristics>dimensions,omitempty"`
	OriginPostal   string     `xml:"origin-postal-code"`
	Destination    Destination
}

// Dimensions represents package dimensions.
type Dimensions struct {
	Length float64 `xml:"length"`
	Width  float64 `xml:"width"`
	Height float64 `xml:"height"`
}

// Destination represents shipping destination.
type Destination struct {
	Domestic      *DomesticDestination      `xml:"domestic,omitempty"`
	International *InternationalDestination `xml:"international,omitempty"`
}

// DomesticDestination for Canadian addresses.
type DomesticDestination struct {
	PostalCode string `xml:"postal-code"`
}

// InternationalDestination for non-Canadian addresses.
type InternationalDestination struct {
	CountryCode string `xml:"country-code"`
}

// RatesResponse represents the Canada Post rate quote response.
type RatesResponse struct {
	QuoteID string
	Rates   []Rate
}

// Rate represents a single shipping rate option.
type Rate struct {
	ServiceCode       string  `xml:"service-code"`
	ServiceName       string  `xml:"service-name"`
	BaseRate          float64 `xml:"price-details>base"`
	FuelSurcharge     float64 `xml:"price-details>adjustments>adjustment[adjustment-code='FUELSC']/adjustment-cost"`
	Taxes             float64 `xml:"price-details>taxes>gst+price-details>taxes>pst+price-details>taxes>hst"`
	TotalPrice        float64 `xml:"price-details>due"`
	ExpectedTransit   int     `xml:"service-standard>expected-transit-time"`
	ExpectedDelivery  string  `xml:"service-standard>expected-delivery-date"`
	GuaranteedDelivery bool   `xml:"service-standard>guaranteed-delivery"`
}

// ShipmentRequest represents a Canada Post shipment creation request.
type ShipmentRequest struct {
	CustomerNumber    string
	GroupID           string
	RequestedShipping ServiceCode
	Sender            Address
	Destination       Address
	ParcelWeight      float64
	ParcelDimensions  Dimensions
	Options           []Option
}

// ServiceCode represents the shipping service.
type ServiceCode struct {
	Code string
}

// Address represents a Canada Post address.
type Address struct {
	Name         string
	Company      string
	AddressLine1 string
	AddressLine2 string
	City         string
	Province     string
	PostalCode   string
	CountryCode  string
	Phone        string
	Email        string
}

// Option represents shipping options.
type Option struct {
	Code  string
	Value string
}

// ShipmentResponse represents the Canada Post shipment creation response.
type ShipmentResponse struct {
	ShipmentID        string
	TrackingPIN       string
	Links             []Link
	ShipmentStatus    string
	ServiceName       string
	TotalCharged      float64
	ExpectedDelivery  string
}

// Link represents a hypermedia link in the response.
type Link struct {
	Rel         string
	Href        string
	MediaType   string
}

// LabelResponse represents the Canada Post label (artifact) response.
type LabelResponse struct {
	ShipmentID string
	Format     string
	Data       []byte // Raw label data (PDF, ZPL, etc.)
}

// VoidResponse represents the Canada Post void shipment response.
type VoidResponse struct {
	ShipmentID string
	Status     string
}

// TrackingResponse represents tracking information.
type TrackingResponse struct {
	TrackingPIN string
	Status      string
	Events      []TrackingEvent
}

// TrackingEvent represents a single tracking event.
type TrackingEvent struct {
	Timestamp   string
	Description string
	Location    string
	Type        string
}

// APIError represents an error from the Canada Post API.
type APIError struct {
	Code        string
	Description string
}

func (e *APIError) Error() string {
	return e.Code + ": " + e.Description
}
