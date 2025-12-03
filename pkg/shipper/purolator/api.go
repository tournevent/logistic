package purolator

import (
	"context"
)

// APIClient defines the interface for Purolator API operations.
// This abstraction allows for mock implementations during testing
// and real SOAP implementations in production.
type APIClient interface {
	// GetRates fetches shipping rates from Purolator EstimatingService
	GetRates(ctx context.Context, req *RatesRequest) (*RatesResponse, error)

	// CreateShipment creates a new shipment via ShippingService
	CreateShipment(ctx context.Context, req *ShipmentRequest) (*ShipmentResponse, error)

	// GetLabel retrieves the shipping label via ShippingDocumentsService
	GetLabel(ctx context.Context, shipmentPIN string, format string) (*LabelResponse, error)

	// VoidShipment cancels an existing shipment via ShippingService
	VoidShipment(ctx context.Context, shipmentPIN string) (*VoidResponse, error)

	// GetTracking retrieves tracking information via TrackingService
	GetTracking(ctx context.Context, trackingPIN string) (*TrackingResponse, error)
}

// ============================================================================
// API Request/Response Types (match Purolator SOAP API structure)
// ============================================================================

// RatesRequest represents a Purolator rate quote request.
type RatesRequest struct {
	BillingAccountNumber string
	SenderPostalCode     string
	ReceiverAddress      Address
	PackageInformation   PackageInformation
}

// PackageInformation contains package details for rating.
type PackageInformation struct {
	TotalWeight    Weight
	TotalPieces    int
	PiecesInFormat []Piece
}

// Weight represents package weight.
type Weight struct {
	Value float64
	Unit  string // "lb" or "kg"
}

// Piece represents a single package piece.
type Piece struct {
	Weight     Weight
	Length     Dimension
	Width      Dimension
	Height     Dimension
	Quantity   int
}

// Dimension represents a package dimension.
type Dimension struct {
	Value float64
	Unit  string // "in" or "cm"
}

// Address represents a Purolator address.
type Address struct {
	Name          string
	Company       string
	StreetNumber  string
	StreetName    string
	StreetAddress string
	City          string
	Province      string
	PostalCode    string
	Country       string
	PhoneNumber   PhoneNumber
}

// PhoneNumber represents a phone number.
type PhoneNumber struct {
	CountryCode string
	AreaCode    string
	Phone       string
}

// RatesResponse represents the Purolator rate quote response.
type RatesResponse struct {
	QuoteID       string
	ShipmentRates []ShipmentRate
}

// ShipmentRate represents a single rate option.
type ShipmentRate struct {
	ServiceCode          string
	ServiceName          string
	BasePrice            float64
	FuelSurcharge        float64
	Taxes                float64
	TotalPrice           float64
	ExpectedDeliveryDate string
	EstimatedTransitDays int
	GuaranteedDelivery   bool
}

// ShipmentRequest represents a Purolator shipment creation request.
type ShipmentRequest struct {
	BillingAccountNumber string
	ServiceCode          string
	Sender               Sender
	Receiver             Receiver
	PackageInformation   PackageInformation
	PrinterType          string // "Thermal" or "Regular"
}

// Sender represents shipment sender information.
type Sender struct {
	Address Address
}

// Receiver represents shipment receiver information.
type Receiver struct {
	Address Address
}

// ShipmentResponse represents the Purolator shipment creation response.
type ShipmentResponse struct {
	ShipmentPIN          string
	TrackingNumber       string
	TotalPrice           float64
	ExpectedDeliveryDate string
	PiecePINs            []string
	DocumentLinks        []DocumentLink
}

// DocumentLink represents a link to a shipping document.
type DocumentLink struct {
	Type string // "Label", "CustomsInvoice", etc.
	URL  string
}

// LabelResponse represents the Purolator label response.
type LabelResponse struct {
	ShipmentPIN string
	Format      string
	Data        []byte // Raw label data (PDF, ZPL, etc.)
}

// VoidResponse represents the Purolator void shipment response.
type VoidResponse struct {
	ShipmentPIN string
	Status      string
	Message     string
}

// TrackingResponse represents tracking information.
type TrackingResponse struct {
	TrackingPIN    string
	Status         string
	DeliveryStatus string
	Events         []TrackingEvent
}

// TrackingEvent represents a single tracking event.
type TrackingEvent struct {
	Timestamp   string
	Description string
	Location    string
	Type        string
}

// APIError represents an error from the Purolator API.
type APIError struct {
	Code        string
	Description string
}

func (e *APIError) Error() string {
	return e.Code + ": " + e.Description
}
