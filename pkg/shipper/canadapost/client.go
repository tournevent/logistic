// Package canadapost provides integration with the Canada Post shipping API.
package canadapost

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const carrierName = "canadapost"

// Config holds Canada Post configuration.
type Config struct {
	APIKey    string
	AccountID string
	BaseURL   string
	UseMock   bool
}

// Client is the Canada Post shipper client.
type Client struct {
	config    Config
	apiClient APIClient
	logger    *otelzap.Logger
	tracer    trace.Tracer
}

// New creates a new Canada Post client.
func New(cfg Config, logger *otelzap.Logger, tracer trace.Tracer) *Client {
	var apiClient APIClient

	if cfg.UseMock {
		apiClient = NewMockAPIClient()
	} else {
		apiClient = NewHTTPAPIClient(HTTPAPIClientConfig{
			BaseURL:   cfg.BaseURL,
			APIKey:    cfg.APIKey,
			AccountID: cfg.AccountID,
			Timeout:   30 * time.Second,
		})
	}

	return &Client{
		config:    cfg,
		apiClient: apiClient,
		logger:    logger,
		tracer:    tracer,
	}
}

// NewWithAPIClient creates a new Canada Post client with a custom API client.
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

// GetQuote returns shipping quotes from Canada Post.
func (c *Client) GetQuote(ctx context.Context, req *shipper.QuoteRequest) (*shipper.QuoteResponse, error) {
	c.logger.Info("Getting Canada Post quotes",
		zap.String("origin_postal", req.Origin.PostalCode),
		zap.String("destination_postal", req.Destination.PostalCode),
		zap.Int("package_count", len(req.Packages)),
	)

	// Convert to API request
	apiReq := &RatesRequest{
		CustomerNumber: c.config.AccountID,
		OriginPostal:   req.Origin.PostalCode,
	}

	// Set destination
	if req.Destination.CountryCode == "" || req.Destination.CountryCode == "CA" {
		apiReq.Destination.Domestic = &DomesticDestination{
			PostalCode: req.Destination.PostalCode,
		}
	} else {
		apiReq.Destination.International = &InternationalDestination{
			CountryCode: req.Destination.CountryCode,
		}
	}

	// Use first package for dimensions/weight
	if len(req.Packages) > 0 {
		pkg := req.Packages[0]
		apiReq.Weight = pkg.Weight
		apiReq.Dimensions = Dimensions{
			Length: pkg.Length,
			Width:  pkg.Width,
			Height: pkg.Height,
		}
	}

	// Call API
	apiResp, err := c.apiClient.GetRates(ctx, apiReq)
	if err != nil {
		c.logger.Error("Canada Post API error", zap.Error(err))
		return nil, err
	}

	// Convert to shipper response
	return ratesResponseToShipper(apiResp), nil
}

// CreateOrder creates a shipment with Canada Post.
func (c *Client) CreateOrder(ctx context.Context, req *shipper.CreateOrderRequest) (*shipper.CreateOrderResponse, error) {
	c.logger.Info("Creating Canada Post order",
		zap.String("rate_id", req.RateID),
		zap.String("recipient", req.Recipient.Name),
	)

	// Extract service code from rate ID (e.g., "cp-rate-regular-xxx" -> DOM.RP)
	serviceCode := extractServiceCode(req.RateID)

	// Convert to API request
	apiReq := &ShipmentRequest{
		CustomerNumber:    c.config.AccountID,
		GroupID:           "default",
		RequestedShipping: ServiceCode{Code: serviceCode},
		Sender:            addressToAPI(req.SenderAddress),
		Destination:       addressToAPI(req.RecipientAddress),
	}

	if len(req.Packages) > 0 {
		pkg := req.Packages[0]
		apiReq.ParcelWeight = pkg.Weight
		apiReq.ParcelDimensions = Dimensions{
			Length: pkg.Length,
			Width:  pkg.Width,
			Height: pkg.Height,
		}
	}

	// Call API
	apiResp, err := c.apiClient.CreateShipment(ctx, apiReq)
	if err != nil {
		c.logger.Error("Canada Post API error", zap.Error(err))
		return nil, err
	}

	// Convert to shipper response
	return shipmentResponseToShipper(apiResp), nil
}

// GetLabel retrieves the shipping label from Canada Post.
func (c *Client) GetLabel(ctx context.Context, req *shipper.GetLabelRequest) (*shipper.GetLabelResponse, error) {
	c.logger.Info("Getting Canada Post label",
		zap.String("order_id", req.OrderID),
		zap.String("format", string(req.Format)),
	)

	format := "application/pdf"
	if req.Format == shipper.LabelZPL {
		format = "application/zpl"
	}

	// Call API
	apiResp, err := c.apiClient.GetLabel(ctx, req.OrderID, format)
	if err != nil {
		c.logger.Error("Canada Post API error", zap.Error(err))
		return nil, err
	}

	// Convert to shipper response
	return labelResponseToShipper(apiResp), nil
}

// CancelOrder cancels a shipment with Canada Post.
func (c *Client) CancelOrder(ctx context.Context, req *shipper.CancelOrderRequest) (*shipper.CancelOrderResponse, error) {
	c.logger.Info("Cancelling Canada Post order",
		zap.String("order_id", req.OrderID),
		zap.String("reason", req.Reason),
	)

	// Call API
	apiResp, err := c.apiClient.VoidShipment(ctx, req.OrderID)
	if err != nil {
		c.logger.Error("Canada Post API error", zap.Error(err))
		return nil, err
	}

	// Convert to shipper response
	return voidResponseToShipper(apiResp), nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func addressToAPI(addr shipper.Address) Address {
	return Address{
		Name:         addr.Name,
		Company:      addr.Company,
		AddressLine1: addr.Line1,
		AddressLine2: addr.Line2,
		City:         addr.City,
		Province:     addr.ProvinceCode,
		PostalCode:   addr.PostalCode,
		CountryCode:  addr.CountryCode,
		Phone:        addr.Phone,
		Email:        addr.Email,
	}
}

func ratesResponseToShipper(resp *RatesResponse) *shipper.QuoteResponse {
	rates := make([]shipper.RateOption, len(resp.Rates))
	expiresAt := time.Now().Add(30 * time.Minute)

	for i, r := range resp.Rates {
		var estimatedDelivery *time.Time
		if r.ExpectedDelivery != "" {
			if t, err := time.Parse("2006-01-02", r.ExpectedDelivery); err == nil {
				estimatedDelivery = &t
			}
		}

		rates[i] = shipper.RateOption{
			RateID:            generateRateID(r.ServiceCode),
			Carrier:           carrierName,
			ServiceCode:       r.ServiceCode,
			ServiceName:       r.ServiceName,
			ServiceType:       mapServiceType(r.ServiceCode),
			BaseRate:          shipper.Money{Amount: r.BaseRate, Currency: "CAD"},
			FuelSurcharge:     shipper.Money{Amount: r.FuelSurcharge, Currency: "CAD"},
			Taxes:             shipper.Money{Amount: r.Taxes, Currency: "CAD"},
			TotalPrice:        shipper.Money{Amount: r.TotalPrice, Currency: "CAD"},
			TransitDays:       r.ExpectedTransit,
			EstimatedDelivery: estimatedDelivery,
			ExpiresAt:         expiresAt,
			Guaranteed:        r.GuaranteedDelivery,
		}
	}

	return &shipper.QuoteResponse{
		QuoteID:   resp.QuoteID,
		Rates:     rates,
		ExpiresAt: expiresAt,
	}
}

func shipmentResponseToShipper(resp *ShipmentResponse) *shipper.CreateOrderResponse {
	var estimatedDelivery *time.Time
	if resp.ExpectedDelivery != "" {
		if t, err := time.Parse("2006-01-02", resp.ExpectedDelivery); err == nil {
			estimatedDelivery = &t
		}
	}

	// Find label and tracking URLs from links
	var labelURL, trackingURL string
	for _, link := range resp.Links {
		switch link.Rel {
		case "label":
			labelURL = link.Href
		case "tracking":
			trackingURL = link.Href
		}
	}

	return &shipper.CreateOrderResponse{
		OrderID:           resp.ShipmentID,
		TrackingNumber:    resp.TrackingPIN,
		TrackingURL:       trackingURL,
		Status:            mapStatus(resp.ShipmentStatus),
		Carrier:           carrierName,
		ServiceName:       resp.ServiceName,
		TotalCharged:      shipper.Money{Amount: resp.TotalCharged, Currency: "CAD"},
		EstimatedDelivery: estimatedDelivery,
		LabelURL:          labelURL,
	}
}

func labelResponseToShipper(resp *LabelResponse) *shipper.GetLabelResponse {
	format := shipper.LabelPDF
	if resp.Format == "application/zpl" {
		format = shipper.LabelZPL
	}

	// Encode binary data as base64 for the shipper response
	data := ""
	if len(resp.Data) > 0 {
		data = base64.StdEncoding.EncodeToString(resp.Data)
	}

	return &shipper.GetLabelResponse{
		OrderID: resp.ShipmentID,
		Label: shipper.Label{
			Format: format,
			Data:   data,
		},
	}
}

func voidResponseToShipper(resp *VoidResponse) *shipper.CancelOrderResponse {
	return &shipper.CancelOrderResponse{
		OrderID:            resp.ShipmentID,
		Status:             mapStatus(resp.Status),
		ConfirmationNumber: resp.ShipmentID + "-VOID",
	}
}

func generateRateID(serviceCode string) string {
	return "cp-" + serviceCode + "-" + time.Now().Format("20060102150405")
}

func extractServiceCode(rateID string) string {
	// Parse rate ID like "cp-DOM.RP-20231215120000" -> "DOM.RP"
	// For simplicity, default to Regular Parcel
	if len(rateID) > 3 {
		// Try to extract service code
		for _, code := range []string{"DOM.RP", "DOM.XP", "DOM.PC", "DOM.EP"} {
			if contains(rateID, code) {
				return code
			}
		}
	}
	return "DOM.RP"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func mapServiceType(code string) shipper.ServiceType {
	switch code {
	case "DOM.RP":
		return shipper.ServiceStandard
	case "DOM.XP":
		return shipper.ServiceExpress
	case "DOM.PC":
		return shipper.ServicePriority
	case "DOM.EP":
		return shipper.ServiceExpress
	default:
		return shipper.ServiceStandard
	}
}

func mapStatus(status string) shipper.ShipmentStatus {
	switch status {
	case "created", "transmitted":
		return shipper.StatusConfirmed
	case "voided":
		return shipper.StatusCancelled
	case "in_transit":
		return shipper.StatusInTransit
	case "delivered":
		return shipper.StatusDelivered
	default:
		return shipper.StatusPending
	}
}
