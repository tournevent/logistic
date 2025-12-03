// Package purolator provides integration with the Purolator shipping API.
package purolator

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const carrierName = "purolator"

// Config holds Purolator configuration.
type Config struct {
	Username string
	Password string
	WSDLURL  string
	UseMock  bool
}

// Client is the Purolator API client.
type Client struct {
	config    Config
	apiClient APIClient
	logger    *otelzap.Logger
	tracer    trace.Tracer
}

// New creates a new Purolator client.
func New(cfg Config, logger *otelzap.Logger, tracer trace.Tracer) *Client {
	var apiClient APIClient

	if cfg.UseMock {
		apiClient = NewMockAPIClient()
	} else {
		apiClient = NewSOAPAPIClient(SOAPAPIClientConfig{
			WSDLURL:  cfg.WSDLURL,
			Username: cfg.Username,
			Password: cfg.Password,
			Timeout:  30 * time.Second,
		})
	}

	return &Client{
		config:    cfg,
		apiClient: apiClient,
		logger:    logger,
		tracer:    tracer,
	}
}

// NewWithAPIClient creates a new Purolator client with a custom API client.
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

// GetQuote returns shipping quotes from Purolator.
func (c *Client) GetQuote(ctx context.Context, req *shipper.QuoteRequest) (*shipper.QuoteResponse, error) {
	c.logger.Info("Getting Purolator quotes",
		zap.String("origin_postal", req.Origin.PostalCode),
		zap.String("destination_postal", req.Destination.PostalCode),
		zap.Int("package_count", len(req.Packages)),
	)

	// Convert to API request
	apiReq := &RatesRequest{
		SenderPostalCode: req.Origin.PostalCode,
		ReceiverAddress: Address{
			City:       req.Destination.City,
			Province:   req.Destination.ProvinceCode,
			PostalCode: req.Destination.PostalCode,
			Country:    req.Destination.CountryCode,
		},
	}

	// Set package information
	if len(req.Packages) > 0 {
		var totalWeight float64
		for _, pkg := range req.Packages {
			totalWeight += pkg.Weight
		}
		apiReq.PackageInformation = PackageInformation{
			TotalWeight: Weight{Value: totalWeight, Unit: "kg"},
			TotalPieces: len(req.Packages),
		}
	}

	// Call API
	apiResp, err := c.apiClient.GetRates(ctx, apiReq)
	if err != nil {
		c.logger.Error("Purolator API error", zap.Error(err))
		return nil, err
	}

	// Convert to shipper response
	return ratesResponseToShipper(apiResp), nil
}

// CreateOrder creates a shipment with Purolator.
func (c *Client) CreateOrder(ctx context.Context, req *shipper.CreateOrderRequest) (*shipper.CreateOrderResponse, error) {
	c.logger.Info("Creating Purolator order",
		zap.String("rate_id", req.RateID),
		zap.String("recipient", req.Recipient.Name),
	)

	// Extract service code from rate ID
	serviceCode := extractServiceCode(req.RateID)

	// Convert to API request
	apiReq := &ShipmentRequest{
		ServiceCode: serviceCode,
		Sender: Sender{
			Address: addressToAPI(req.SenderAddress),
		},
		Receiver: Receiver{
			Address: addressToAPI(req.RecipientAddress),
		},
		PrinterType: "Regular",
	}

	// Set package information
	if len(req.Packages) > 0 {
		var totalWeight float64
		for _, pkg := range req.Packages {
			totalWeight += pkg.Weight
		}
		apiReq.PackageInformation = PackageInformation{
			TotalWeight: Weight{Value: totalWeight, Unit: "kg"},
			TotalPieces: len(req.Packages),
		}
	}

	// Call API
	apiResp, err := c.apiClient.CreateShipment(ctx, apiReq)
	if err != nil {
		c.logger.Error("Purolator API error", zap.Error(err))
		return nil, err
	}

	// Convert to shipper response
	return shipmentResponseToShipper(apiResp), nil
}

// GetLabel retrieves the shipping label from Purolator.
func (c *Client) GetLabel(ctx context.Context, req *shipper.GetLabelRequest) (*shipper.GetLabelResponse, error) {
	c.logger.Info("Getting Purolator label",
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
		c.logger.Error("Purolator API error", zap.Error(err))
		return nil, err
	}

	// Convert to shipper response
	return labelResponseToShipper(apiResp), nil
}

// CancelOrder cancels a shipment with Purolator.
func (c *Client) CancelOrder(ctx context.Context, req *shipper.CancelOrderRequest) (*shipper.CancelOrderResponse, error) {
	c.logger.Info("Cancelling Purolator order",
		zap.String("order_id", req.OrderID),
		zap.String("reason", req.Reason),
	)

	// Call API
	apiResp, err := c.apiClient.VoidShipment(ctx, req.OrderID)
	if err != nil {
		c.logger.Error("Purolator API error", zap.Error(err))
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
		Name:          addr.Name,
		Company:       addr.Company,
		StreetAddress: addr.Line1,
		City:          addr.City,
		Province:      addr.ProvinceCode,
		PostalCode:    addr.PostalCode,
		Country:       addr.CountryCode,
	}
}

func ratesResponseToShipper(resp *RatesResponse) *shipper.QuoteResponse {
	rates := make([]shipper.RateOption, len(resp.ShipmentRates))
	expiresAt := time.Now().Add(30 * time.Minute)

	for i, r := range resp.ShipmentRates {
		var estimatedDelivery *time.Time
		if r.ExpectedDeliveryDate != "" {
			if t, err := time.Parse("2006-01-02", r.ExpectedDeliveryDate); err == nil {
				estimatedDelivery = &t
			}
		}

		rates[i] = shipper.RateOption{
			RateID:            generateRateID(r.ServiceCode),
			Carrier:           carrierName,
			ServiceCode:       r.ServiceCode,
			ServiceName:       r.ServiceName,
			ServiceType:       mapServiceType(r.ServiceCode),
			BaseRate:          shipper.Money{Amount: r.BasePrice, Currency: "CAD"},
			FuelSurcharge:     shipper.Money{Amount: r.FuelSurcharge, Currency: "CAD"},
			Taxes:             shipper.Money{Amount: r.Taxes, Currency: "CAD"},
			TotalPrice:        shipper.Money{Amount: r.TotalPrice, Currency: "CAD"},
			TransitDays:       r.EstimatedTransitDays,
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
	if resp.ExpectedDeliveryDate != "" {
		if t, err := time.Parse("2006-01-02", resp.ExpectedDeliveryDate); err == nil {
			estimatedDelivery = &t
		}
	}

	// Find label URL from document links
	var labelURL string
	for _, link := range resp.DocumentLinks {
		if link.Type == "Label" {
			labelURL = link.URL
			break
		}
	}

	return &shipper.CreateOrderResponse{
		OrderID:           resp.ShipmentPIN,
		TrackingNumber:    resp.TrackingNumber,
		TrackingURL:       "https://www.purolator.com/en/shipping/tracker?pin=" + resp.TrackingNumber,
		Status:            shipper.StatusConfirmed,
		Carrier:           carrierName,
		ServiceName:       "Purolator",
		TotalCharged:      shipper.Money{Amount: resp.TotalPrice, Currency: "CAD"},
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
		OrderID: resp.ShipmentPIN,
		Label: shipper.Label{
			Format: format,
			Data:   data,
		},
	}
}

func voidResponseToShipper(resp *VoidResponse) *shipper.CancelOrderResponse {
	return &shipper.CancelOrderResponse{
		OrderID:            resp.ShipmentPIN,
		Status:             shipper.StatusCancelled,
		ConfirmationNumber: resp.ShipmentPIN + "-VOID",
	}
}

func generateRateID(serviceCode string) string {
	return "puro-" + serviceCode + "-" + time.Now().Format("20060102150405")
}

func extractServiceCode(rateID string) string {
	// Parse rate ID like "puro-PurolatorGround-20231215120000" -> "PurolatorGround"
	// For simplicity, default to Ground
	for _, code := range []string{"PurolatorGround", "PurolatorExpress", "PurolatorExpress9AM", "PurolatorExpress10:30AM"} {
		if contains(rateID, code) {
			return code
		}
	}
	return "PurolatorGround"
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func mapServiceType(code string) shipper.ServiceType {
	switch code {
	case "PurolatorGround":
		return shipper.ServiceStandard
	case "PurolatorExpress":
		return shipper.ServiceExpress
	case "PurolatorExpress9AM", "PurolatorExpress10:30AM":
		return shipper.ServiceOvernight
	default:
		return shipper.ServiceStandard
	}
}
