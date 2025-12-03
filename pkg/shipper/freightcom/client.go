// Package freightcom provides integration with the Freightcom shipping API.
package freightcom

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const carrierName = "freightcom"

// Config holds Freightcom configuration.
type Config struct {
	APIKey          string
	BaseURL         string
	PaymentMethodID int  // Required for creating shipments
	UseMock         bool // When true, uses mock API client
}

// Client is the Freightcom shipper client.
// It implements the shipper.Shipper interface and delegates
// API calls to the underlying APIClient (mock or HTTP).
type Client struct {
	config    Config
	apiClient APIClient
	logger    *otelzap.Logger
	tracer    trace.Tracer
}

// New creates a new Freightcom client.
// If cfg.UseMock is true, it uses a mock API client for testing.
// Otherwise, it uses the real HTTP API client.
func New(cfg Config, logger *otelzap.Logger, tracer trace.Tracer) *Client {
	var apiClient APIClient

	if cfg.UseMock {
		apiClient = NewMockAPIClient()
	} else {
		apiClient = NewHTTPAPIClient(HTTPAPIClientConfig{
			BaseURL: cfg.BaseURL,
			APIKey:  cfg.APIKey,
			Timeout: 30 * time.Second,
		})
	}

	return &Client{
		config:    cfg,
		apiClient: apiClient,
		logger:    logger,
		tracer:    tracer,
	}
}

// NewWithAPIClient creates a new Freightcom client with a custom API client.
// This is useful for injecting mock clients in tests.
func NewWithAPIClient(cfg Config, apiClient APIClient, logger *otelzap.Logger, tracer trace.Tracer) *Client {
	return &Client{
		config:    cfg,
		apiClient: apiClient,
		logger:    logger,
		tracer:    tracer,
	}
}

// Name returns the carrier name.
func (c *Client) Name() string {
	return carrierName
}

// GetQuote returns shipping quotes from Freightcom.
func (c *Client) GetQuote(ctx context.Context, req *shipper.QuoteRequest) (*shipper.QuoteResponse, error) {
	c.logger.Info("Getting Freightcom quotes",
		zap.String("origin_city", req.Origin.City),
		zap.String("destination_city", req.Destination.City),
		zap.Int("package_count", len(req.Packages)),
	)

	// Convert to API request
	apiReq := &RatesRequest{
		Details: ShippingDetails{
			Origin:      addressToLocation(req.Origin),
			Destination: addressToLocation(req.Destination),
			Packaging: PackagingInfo{
				Type:     "package",
				Packages: packagesToAPI(req.Packages),
			},
		},
	}

	// Call API
	apiResp, err := c.apiClient.GetRates(ctx, apiReq)
	if err != nil {
		c.logger.Error("Freightcom API error", zap.Error(err))
		return nil, err
	}

	// Convert to shipper response
	return ratesResponseToShipper(apiResp), nil
}

// CreateOrder creates a shipment with Freightcom.
func (c *Client) CreateOrder(ctx context.Context, req *shipper.CreateOrderRequest) (*shipper.CreateOrderResponse, error) {
	c.logger.Info("Creating Freightcom order",
		zap.String("rate_id", req.RateID),
		zap.String("recipient", req.Recipient.Name),
	)

	// Extract service ID from rate - in production this would come from the rate selection
	serviceID := extractServiceID(req.RateID)

	// Generate unique ID for idempotency
	uniqueID := req.Reference
	if uniqueID == "" {
		uniqueID = uuid.New().String()
	}

	// Convert to API request
	apiReq := &ShipmentRequest{
		UniqueID:        uniqueID,
		PaymentMethodID: c.config.PaymentMethodID,
		ServiceID:       serviceID,
		Details: ShippingDetails{
			Origin:      addressToLocation(req.SenderAddress),
			Destination: addressToLocation(req.RecipientAddress),
			Packaging: PackagingInfo{
				Type:     "package",
				Packages: packagesToAPI(req.Packages),
			},
		},
		Sender:       contactToAPI(req.Sender),
		Recipient:    contactToAPI(req.Recipient),
		Reference:    req.Reference,
		PONumber:     req.PONumber,
		Instructions: req.Instructions,
	}

	// Call API
	apiResp, err := c.apiClient.CreateShipment(ctx, apiReq)
	if err != nil {
		c.logger.Error("Freightcom API error", zap.Error(err))
		return nil, err
	}

	// Convert to shipper response
	return shipmentResponseToShipper(apiResp), nil
}

// GetLabel retrieves the shipping label from Freightcom.
func (c *Client) GetLabel(ctx context.Context, req *shipper.GetLabelRequest) (*shipper.GetLabelResponse, error) {
	c.logger.Info("Getting Freightcom label",
		zap.String("order_id", req.OrderID),
		zap.String("format", string(req.Format)),
	)

	format := string(req.Format)
	if format == "" {
		format = "pdf"
	}

	// Call API
	apiResp, err := c.apiClient.GetLabel(ctx, req.OrderID, format)
	if err != nil {
		c.logger.Error("Freightcom API error", zap.Error(err))
		return nil, err
	}

	// Convert to shipper response
	return labelResponseToShipper(apiResp), nil
}

// CancelOrder cancels a shipment with Freightcom.
func (c *Client) CancelOrder(ctx context.Context, req *shipper.CancelOrderRequest) (*shipper.CancelOrderResponse, error) {
	c.logger.Info("Cancelling Freightcom order",
		zap.String("order_id", req.OrderID),
		zap.String("reason", req.Reason),
	)

	// Call API
	apiResp, err := c.apiClient.CancelShipment(ctx, req.OrderID, req.Reason)
	if err != nil {
		c.logger.Error("Freightcom API error", zap.Error(err))
		return nil, err
	}

	// Convert to shipper response
	return cancelResponseToShipper(apiResp), nil
}

// ============================================================================
// Conversion helpers: Shipper models -> API models
// ============================================================================

func addressToLocation(addr shipper.Address) Location {
	return Location{
		Name:        addr.Name,
		Company:     addr.Company,
		Address1:    addr.Line1,
		Address2:    addr.Line2,
		City:        addr.City,
		Province:    addr.ProvinceCode,
		PostalCode:  addr.PostalCode,
		Country:     addr.CountryCode,
		Phone:       addr.Phone,
		Email:       addr.Email,
		Residential: addr.IsResidential,
	}
}

func contactToAPI(c shipper.Contact) Contact {
	return Contact{
		Name:    c.Name,
		Company: c.Company,
		Phone:   c.Phone,
		Email:   c.Email,
	}
}

func packagesToAPI(pkgs []shipper.Package) []Package {
	result := make([]Package, len(pkgs))
	for i, p := range pkgs {
		result[i] = Package{
			Length:   p.Length,
			Width:    p.Width,
			Height:   p.Height,
			Weight:   p.Weight,
			Quantity: 1,
		}
	}
	return result
}

// ============================================================================
// Conversion helpers: API models -> Shipper models
// ============================================================================

func ratesResponseToShipper(resp *RatesResponse) *shipper.QuoteResponse {
	rates := make([]shipper.RateOption, len(resp.Rates))
	for i, r := range resp.Rates {
		expiresAt, _ := time.Parse(time.RFC3339, r.ExpiresAt)
		var estimatedDelivery *time.Time
		if r.EstimatedDelivery != "" {
			if t, err := time.Parse("2006-01-02", r.EstimatedDelivery); err == nil {
				estimatedDelivery = &t
			}
		}

		rates[i] = shipper.RateOption{
			RateID:            r.ID,
			Carrier:           carrierName,
			ServiceCode:       r.ServiceCode,
			ServiceName:       r.ServiceName,
			ServiceType:       mapServiceType(r.ServiceCode),
			BaseRate:          shipper.Money{Amount: r.BaseRate, Currency: r.Currency},
			FuelSurcharge:     shipper.Money{Amount: r.FuelSurcharge, Currency: r.Currency},
			Taxes:             shipper.Money{Amount: r.TotalTax, Currency: r.Currency},
			TotalPrice:        shipper.Money{Amount: r.TotalPrice, Currency: r.Currency},
			TransitDays:       r.TransitDays,
			EstimatedDelivery: estimatedDelivery,
			ExpiresAt:         expiresAt,
			Guaranteed:        r.Guaranteed,
		}
	}

	var expiresAt time.Time
	if len(rates) > 0 {
		expiresAt = rates[0].ExpiresAt
	}

	return &shipper.QuoteResponse{
		QuoteID:   resp.RequestID,
		Rates:     rates,
		ExpiresAt: expiresAt,
	}
}

func shipmentResponseToShipper(resp *ShipmentResponse) *shipper.CreateOrderResponse {
	var estimatedDelivery *time.Time
	if resp.EstimatedDelivery != "" {
		if t, err := time.Parse("2006-01-02", resp.EstimatedDelivery); err == nil {
			estimatedDelivery = &t
		}
	}

	// Get first tracking number if available
	trackingNumber := ""
	if len(resp.TrackingNumbers) > 0 {
		trackingNumber = resp.TrackingNumbers[0]
	}

	// Get first label URL if available
	labelURL := ""
	if len(resp.Labels) > 0 {
		labelURL = resp.Labels[0].URL
	}

	return &shipper.CreateOrderResponse{
		OrderID:           resp.ID,
		TrackingNumber:    trackingNumber,
		TrackingURL:       resp.TrackingURL,
		Status:            mapStatus(resp.Status),
		Carrier:           carrierName,
		ServiceName:       resp.ServiceName,
		TotalCharged:      shipper.Money{Amount: resp.TotalCharged, Currency: resp.Currency},
		EstimatedDelivery: estimatedDelivery,
		LabelURL:          labelURL,
	}
}

func labelResponseToShipper(resp *LabelResponse) *shipper.GetLabelResponse {
	// Get first label if available
	var label shipper.Label
	if len(resp.Labels) > 0 {
		l := resp.Labels[0]
		label = shipper.Label{
			Format: mapLabelFormat(l.Format),
			URL:    l.URL,
		}
	}

	return &shipper.GetLabelResponse{
		OrderID: resp.ShipmentID,
		Label:   label,
	}
}

func cancelResponseToShipper(resp *CancelResponse) *shipper.CancelOrderResponse {
	var refundAmount *shipper.Money
	if resp.RefundAmount > 0 {
		refundAmount = &shipper.Money{Amount: resp.RefundAmount, Currency: resp.Currency}
	}

	return &shipper.CancelOrderResponse{
		OrderID:            resp.ShipmentID,
		Status:             mapStatus(resp.Status),
		RefundAmount:       refundAmount,
		ConfirmationNumber: resp.ConfirmationNumber,
	}
}

// ============================================================================
// Mapping helpers
// ============================================================================

func extractServiceID(rateID string) int {
	// In production, this would parse the rate ID to get the service ID
	// For now, default to a common service ID
	return 101
}

func mapServiceType(code string) shipper.ServiceType {
	switch code {
	case "GROUND", "STANDARD", "FEDEX_GROUND", "UPS_GROUND":
		return shipper.ServiceStandard
	case "EXPRESS", "FEDEX_EXPRESS_SAVER", "UPS_EXPRESS_SAVER":
		return shipper.ServiceExpress
	case "PRIORITY", "FEDEX_PRIORITY_OVERNIGHT", "UPS_NEXT_DAY_AIR":
		return shipper.ServicePriority
	case "OVERNIGHT", "FEDEX_STANDARD_OVERNIGHT":
		return shipper.ServiceOvernight
	case "ECONOMY", "FEDEX_ECONOMY":
		return shipper.ServiceEconomy
	case "FREIGHT", "LTL":
		return shipper.ServiceFreight
	default:
		return shipper.ServiceStandard
	}
}

func mapStatus(status string) shipper.ShipmentStatus {
	switch status {
	case "pending", "processing":
		return shipper.StatusPending
	case "quoted":
		return shipper.StatusQuoted
	case "confirmed", "booked", "complete":
		return shipper.StatusConfirmed
	case "assigned":
		return shipper.StatusAssigned
	case "picked_up":
		return shipper.StatusPickedUp
	case "in_transit":
		return shipper.StatusInTransit
	case "out_for_delivery":
		return shipper.StatusOutForDelivery
	case "delivered":
		return shipper.StatusDelivered
	case "cancelled":
		return shipper.StatusCancelled
	case "exception", "error", "failed":
		return shipper.StatusException
	default:
		return shipper.StatusPending
	}
}

func mapLabelFormat(format string) shipper.LabelFormat {
	switch format {
	case "pdf", "PDF":
		return shipper.LabelPDF
	case "png", "PNG":
		return shipper.LabelPNG
	case "zpl", "ZPL":
		return shipper.LabelZPL
	default:
		return shipper.LabelPDF
	}
}
