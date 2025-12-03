package shipper

import (
	"time"
)

// ShipmentStatus represents the normalized status of a shipment.
type ShipmentStatus string

const (
	StatusPending        ShipmentStatus = "pending"
	StatusQuoted         ShipmentStatus = "quoted"
	StatusConfirmed      ShipmentStatus = "confirmed"
	StatusAssigned       ShipmentStatus = "assigned"
	StatusPickedUp       ShipmentStatus = "picked_up"
	StatusInTransit      ShipmentStatus = "in_transit"
	StatusOutForDelivery ShipmentStatus = "out_for_delivery"
	StatusDelivered      ShipmentStatus = "delivered"
	StatusCancelled      ShipmentStatus = "cancelled"
	StatusException      ShipmentStatus = "exception"
)

// ServiceType represents the shipping service type.
type ServiceType string

const (
	ServiceStandard  ServiceType = "standard"
	ServiceExpress   ServiceType = "express"
	ServicePriority  ServiceType = "priority"
	ServiceOvernight ServiceType = "overnight"
	ServiceEconomy   ServiceType = "economy"
	ServiceFreight   ServiceType = "freight"
)

// PackageType represents the type of package.
type PackageType string

const (
	PackageBox      PackageType = "box"
	PackageEnvelope PackageType = "envelope"
	PackageTube     PackageType = "tube"
	PackagePallet   PackageType = "pallet"
	PackageCustom   PackageType = "custom"
)

// WeightUnit represents weight measurement unit.
type WeightUnit string

const (
	WeightKG WeightUnit = "kg"
	WeightLB WeightUnit = "lb"
)

// DimensionUnit represents dimension measurement unit.
type DimensionUnit string

const (
	DimensionCM DimensionUnit = "cm"
	DimensionIN DimensionUnit = "in"
)

// LabelFormat represents the format of shipping labels.
type LabelFormat string

const (
	LabelPDF LabelFormat = "pdf"
	LabelPNG LabelFormat = "png"
	LabelZPL LabelFormat = "zpl"
)

// Address represents a shipping address.
type Address struct {
	Name          string
	Company       string
	Line1         string
	Line2         string
	City          string
	ProvinceCode  string // e.g., "ON", "QC", "BC"
	PostalCode    string
	CountryCode   string // ISO 3166-1 alpha-2, e.g., "CA", "US"
	Phone         string
	Email         string
	Instructions  string
	IsResidential bool
}

// Contact represents sender or recipient contact info.
type Contact struct {
	Name    string
	Company string
	Phone   string
	Email   string
	TaxID   string // For customs (international)
}

// Package represents a package to be shipped.
type Package struct {
	ID            string
	Length        float64
	Width         float64
	Height        float64
	DimensionUnit DimensionUnit
	Weight        float64
	WeightUnit    WeightUnit
	PackageType   PackageType
	Description   string
	DeclaredValue float64
	Currency      string
}

// Money represents a monetary amount.
type Money struct {
	Amount   float64
	Currency string
}

// RateOption represents a shipping rate option from a carrier.
type RateOption struct {
	RateID            string
	Carrier           string
	ServiceCode       string
	ServiceName       string
	ServiceType       ServiceType
	BaseRate          Money
	FuelSurcharge     Money
	Taxes             Money
	TotalPrice        Money
	TransitDays       int
	EstimatedDelivery *time.Time
	ExpiresAt         time.Time
	SignatureRequired bool
	Guaranteed        bool
}

// TrackingEvent represents a tracking event.
type TrackingEvent struct {
	Timestamp   time.Time
	Description string
	Location    string
	Status      ShipmentStatus
	CarrierCode string
}

// Label represents a shipping label.
type Label struct {
	Format    LabelFormat
	Data      string // Base64 encoded if inline
	URL       string // URL if hosted
	ExpiresAt *time.Time
}

// ShippingOptions represents shipping preferences.
type ShippingOptions struct {
	Carriers          []string // Empty = all carriers
	ServiceTypes      []ServiceType
	SignatureRequired bool
	InsuranceRequired bool
	SaturdayDelivery  bool
	ShipDate          *time.Time
}

// ============================================================================
// Request/Response Types
// ============================================================================

// QuoteRequest is the request for getting shipping quotes.
type QuoteRequest struct {
	ShipperID   string
	Origin      Address
	Destination Address
	Packages    []Package
	Options     ShippingOptions
}

// QuoteResponse is the response from getting shipping quotes.
type QuoteResponse struct {
	QuoteID   string
	Rates     []RateOption
	ExpiresAt time.Time
}

// CreateOrderRequest is the request for creating a shipping order.
type CreateOrderRequest struct {
	ShipperID        string
	QuoteID          string // From GetQuote response
	RateID           string // Selected rate
	Sender           Contact
	SenderAddress    Address
	Recipient        Contact
	RecipientAddress Address
	Packages         []Package
	Reference        string
	PONumber         string
	Instructions     string
}

// CreateOrderResponse is the response from creating a shipping order.
type CreateOrderResponse struct {
	OrderID           string
	TrackingNumber    string
	TrackingURL       string
	Status            ShipmentStatus
	Carrier           string
	ServiceName       string
	TotalCharged      Money
	EstimatedDelivery *time.Time
	LabelURL          string
}

// GetLabelRequest is the request for getting a shipping label.
type GetLabelRequest struct {
	OrderID string
	Format  LabelFormat
}

// GetLabelResponse is the response from getting a shipping label.
type GetLabelResponse struct {
	OrderID          string
	Label            Label
	AdditionalLabels []Label // For multi-package shipments
}

// CancelOrderRequest is the request for cancelling an order.
type CancelOrderRequest struct {
	OrderID string
	Reason  string
}

// CancelOrderResponse is the response from cancelling an order.
type CancelOrderResponse struct {
	OrderID            string
	Status             ShipmentStatus
	RefundAmount       *Money
	ConfirmationNumber string
}
