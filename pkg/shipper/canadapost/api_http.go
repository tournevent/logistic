package canadapost

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPAPIClient is the production implementation of APIClient using HTTP/XML.
type HTTPAPIClient struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	accountID  string
	httpClient *http.Client
}

// HTTPAPIClientConfig holds configuration for the HTTP client.
type HTTPAPIClientConfig struct {
	BaseURL   string
	APIKey    string
	APISecret string // Password for Basic Auth
	AccountID string
	Timeout   time.Duration
}

// NewHTTPAPIClient creates a new HTTP-based API client for production use.
func NewHTTPAPIClient(cfg HTTPAPIClientConfig) *HTTPAPIClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &HTTPAPIClient{
		baseURL:   cfg.BaseURL,
		apiKey:    cfg.APIKey,
		apiSecret: cfg.APISecret,
		accountID: cfg.AccountID,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ============================================================================
// XML Request/Response structures for Canada Post API
// ============================================================================

// mailingScenario is the XML structure for rate requests
type mailingScenario struct {
	XMLName          xml.Name `xml:"mailing-scenario"`
	Xmlns            string   `xml:"xmlns,attr"`
	CustomerNumber   string   `xml:"customer-number,omitempty"`
	ContractID       string   `xml:"contract-id,omitempty"`
	ParcelCharacter  parcelCharacteristics `xml:"parcel-characteristics"`
	OriginPostalCode string   `xml:"origin-postal-code"`
	Destination      xmlDestination `xml:"destination"`
}

type parcelCharacteristics struct {
	Weight     float64        `xml:"weight"`
	Dimensions *xmlDimensions `xml:"dimensions,omitempty"`
}

type xmlDimensions struct {
	Length float64 `xml:"length"`
	Width  float64 `xml:"width"`
	Height float64 `xml:"height"`
}

type xmlDestination struct {
	Domestic      *xmlDomestic      `xml:"domestic,omitempty"`
	UnitedStates  *xmlUnitedStates  `xml:"united-states,omitempty"`
	International *xmlInternational `xml:"international,omitempty"`
}

type xmlDomestic struct {
	PostalCode string `xml:"postal-code"`
}

type xmlUnitedStates struct {
	ZipCode string `xml:"zip-code"`
}

type xmlInternational struct {
	CountryCode string `xml:"country-code"`
}

// priceQuotes is the XML response structure for rates
type priceQuotes struct {
	XMLName    xml.Name     `xml:"price-quotes"`
	PriceQuote []priceQuote `xml:"price-quote"`
}

type priceQuote struct {
	ServiceCode     string          `xml:"service-code"`
	ServiceLink     serviceLink     `xml:"service-link"`
	PriceDetails    priceDetails    `xml:"price-details"`
	ServiceStandard serviceStandard `xml:"service-standard"`
}

type serviceLink struct {
	ServiceName string `xml:"service-name"`
	Href        string `xml:"href,attr"`
}

type priceDetails struct {
	Base        float64      `xml:"base"`
	Taxes       priceTaxes   `xml:"taxes"`
	Due         float64      `xml:"due"`
	Adjustments adjustments  `xml:"adjustments"`
}

type priceTaxes struct {
	GST float64 `xml:"gst"`
	PST float64 `xml:"pst"`
	HST float64 `xml:"hst"`
}

type adjustments struct {
	Adjustment []adjustment `xml:"adjustment"`
}

type adjustment struct {
	AdjustmentCode string  `xml:"adjustment-code"`
	AdjustmentCost float64 `xml:"adjustment-cost"`
}

type serviceStandard struct {
	AMDelivery            bool   `xml:"am-delivery"`
	GuaranteedDelivery    bool   `xml:"guaranteed-delivery"`
	ExpectedTransitTime   int    `xml:"expected-transit-time"`
	ExpectedDeliveryDate  string `xml:"expected-delivery-date"`
}

// shipmentInfo is the XML structure for shipment requests
type shipmentInfo struct {
	XMLName            xml.Name               `xml:"shipment"`
	Xmlns              string                 `xml:"xmlns,attr"`
	GroupID            string                 `xml:"group-id,omitempty"`
	CpcPickupIndicator bool                   `xml:"cpc-pickup-indicator"`
	RequestedShipping  requestedShipping      `xml:"requested-shipping-point,omitempty"`
	DeliverySpec       deliverySpec           `xml:"delivery-spec"`
}

type requestedShipping struct {
	PostalCode string `xml:"postal-code"`
}

type deliverySpec struct {
	ServiceCode       string           `xml:"service-code"`
	Sender            xmlSenderInfo    `xml:"sender"`
	Destination       xmlDestinationInfo `xml:"destination"`
	ParcelCharacter   parcelCharacteristics `xml:"parcel-characteristics"`
	PrintPreferences  printPreferences `xml:"print-preferences,omitempty"`
}

type xmlSenderInfo struct {
	Name         string        `xml:"name"`
	Company      string        `xml:"company,omitempty"`
	ContactPhone string        `xml:"contact-phone"`
	AddressDetails xmlAddressDetails `xml:"address-details"`
}

type xmlDestinationInfo struct {
	Name         string        `xml:"name"`
	Company      string        `xml:"company,omitempty"`
	AddressDetails xmlAddressDetails `xml:"address-details"`
}

type xmlAddressDetails struct {
	AddressLine1 string `xml:"address-line-1"`
	AddressLine2 string `xml:"address-line-2,omitempty"`
	City         string `xml:"city"`
	ProvState    string `xml:"prov-state"`
	PostalZipCode string `xml:"postal-zip-code"`
	CountryCode  string `xml:"country-code"`
}

type printPreferences struct {
	OutputFormat     string `xml:"output-format"` // "4x6", "8.5x11"
	Encoding         string `xml:"encoding"`      // "PDF", "ZPL"
}

// shipmentInfoResponse is the XML response for shipment creation
type shipmentInfoResponse struct {
	XMLName      xml.Name `xml:"shipment-info"`
	ShipmentID   string   `xml:"shipment-id"`
	ShipmentStatus string `xml:"shipment-status"`
	TrackingPIN  string   `xml:"tracking-pin"`
	Links        xmlLinks `xml:"links"`
}

type xmlLinks struct {
	Link []xmlLink `xml:"link"`
}

type xmlLink struct {
	Rel       string `xml:"rel,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

// trackingSummary is the XML response for tracking
type trackingSummary struct {
	XMLName           xml.Name `xml:"tracking-summary"`
	PINSummary        pinSummary `xml:"pin-summary"`
}

type pinSummary struct {
	PIN                string `xml:"pin"`
	OriginPostalID     string `xml:"origin-postal-id"`
	DestinationPostalID string `xml:"destination-postal-id"`
	EventDescription   string `xml:"event-description"`
	EventDateTime      string `xml:"event-date-time"`
	EventType          string `xml:"event-type"`
	EventLocation      string `xml:"event-location"`
}

// messages is the XML error response structure
type messages struct {
	XMLName xml.Name `xml:"messages"`
	Message []message `xml:"message"`
}

type message struct {
	Code        string `xml:"code"`
	Description string `xml:"description"`
}

// ============================================================================
// API Implementation
// ============================================================================

// GetRates fetches shipping rates from the Canada Post API.
func (c *HTTPAPIClient) GetRates(ctx context.Context, req *RatesRequest) (*RatesResponse, error) {
	// Build XML request
	scenario := mailingScenario{
		Xmlns:            "http://www.canadapost.ca/ws/ship/rate-v4",
		CustomerNumber:   req.CustomerNumber,
		OriginPostalCode: normalizePostalCode(req.OriginPostal),
		ParcelCharacter: parcelCharacteristics{
			Weight: req.Weight,
		},
	}

	if req.Dimensions.Length > 0 {
		scenario.ParcelCharacter.Dimensions = &xmlDimensions{
			Length: req.Dimensions.Length,
			Width:  req.Dimensions.Width,
			Height: req.Dimensions.Height,
		}
	}

	// Set destination
	if req.Destination.Domestic != nil {
		scenario.Destination.Domestic = &xmlDomestic{
			PostalCode: normalizePostalCode(req.Destination.Domestic.PostalCode),
		}
	} else if req.Destination.International != nil {
		if req.Destination.International.CountryCode == "US" {
			scenario.Destination.UnitedStates = &xmlUnitedStates{
				ZipCode: req.Destination.International.CountryCode,
			}
		} else {
			scenario.Destination.International = &xmlInternational{
				CountryCode: req.Destination.International.CountryCode,
			}
		}
	}

	xmlBody, err := xml.Marshal(scenario)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	path := "/rs/ship/price"
	resp, err := c.doRequest(ctx, http.MethodPost, path, "application/vnd.cpc.ship.rate-v4+xml", xmlBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	// Parse response
	var quotes priceQuotes
	if err := xml.NewDecoder(resp.Body).Decode(&quotes); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our response type
	return c.convertRatesResponse(&quotes), nil
}

func (c *HTTPAPIClient) convertRatesResponse(quotes *priceQuotes) *RatesResponse {
	rates := make([]Rate, len(quotes.PriceQuote))
	for i, q := range quotes.PriceQuote {
		// Calculate fuel surcharge from adjustments
		var fuelSurcharge float64
		for _, adj := range q.PriceDetails.Adjustments.Adjustment {
			if adj.AdjustmentCode == "FUELSC" {
				fuelSurcharge = adj.AdjustmentCost
				break
			}
		}

		// Calculate total taxes
		taxes := q.PriceDetails.Taxes.GST + q.PriceDetails.Taxes.PST + q.PriceDetails.Taxes.HST

		rates[i] = Rate{
			ServiceCode:        q.ServiceCode,
			ServiceName:        q.ServiceLink.ServiceName,
			BaseRate:           q.PriceDetails.Base,
			FuelSurcharge:      fuelSurcharge,
			Taxes:              taxes,
			TotalPrice:         q.PriceDetails.Due,
			ExpectedTransit:    q.ServiceStandard.ExpectedTransitTime,
			ExpectedDelivery:   q.ServiceStandard.ExpectedDeliveryDate,
			GuaranteedDelivery: q.ServiceStandard.GuaranteedDelivery,
		}
	}

	return &RatesResponse{
		QuoteID: fmt.Sprintf("cp-quote-%d", time.Now().UnixNano()),
		Rates:   rates,
	}
}

// CreateShipment creates a new shipment via the Canada Post API.
func (c *HTTPAPIClient) CreateShipment(ctx context.Context, req *ShipmentRequest) (*ShipmentResponse, error) {
	shipment := shipmentInfo{
		Xmlns:              "http://www.canadapost.ca/ws/shipment-v8",
		GroupID:            req.GroupID,
		CpcPickupIndicator: true,
		DeliverySpec: deliverySpec{
			ServiceCode: req.RequestedShipping.Code,
			Sender: xmlSenderInfo{
				Name:         req.Sender.Name,
				Company:      req.Sender.Company,
				ContactPhone: req.Sender.Phone,
				AddressDetails: xmlAddressDetails{
					AddressLine1:  req.Sender.AddressLine1,
					AddressLine2:  req.Sender.AddressLine2,
					City:          req.Sender.City,
					ProvState:     req.Sender.Province,
					PostalZipCode: normalizePostalCode(req.Sender.PostalCode),
					CountryCode:   req.Sender.CountryCode,
				},
			},
			Destination: xmlDestinationInfo{
				Name:    req.Destination.Name,
				Company: req.Destination.Company,
				AddressDetails: xmlAddressDetails{
					AddressLine1:  req.Destination.AddressLine1,
					AddressLine2:  req.Destination.AddressLine2,
					City:          req.Destination.City,
					ProvState:     req.Destination.Province,
					PostalZipCode: normalizePostalCode(req.Destination.PostalCode),
					CountryCode:   req.Destination.CountryCode,
				},
			},
			ParcelCharacter: parcelCharacteristics{
				Weight: req.ParcelWeight,
			},
			PrintPreferences: printPreferences{
				OutputFormat: "4x6",
				Encoding:     "PDF",
			},
		},
	}

	if req.ParcelDimensions.Length > 0 {
		shipment.DeliverySpec.ParcelCharacter.Dimensions = &xmlDimensions{
			Length: req.ParcelDimensions.Length,
			Width:  req.ParcelDimensions.Width,
			Height: req.ParcelDimensions.Height,
		}
	}

	xmlBody, err := xml.Marshal(shipment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	path := fmt.Sprintf("/rs/%s/%s/shipment", c.accountID, req.GroupID)
	resp, err := c.doRequest(ctx, http.MethodPost, path, "application/vnd.cpc.shipment-v8+xml", xmlBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var shipmentResp shipmentInfoResponse
	if err := xml.NewDecoder(resp.Body).Decode(&shipmentResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.convertShipmentResponse(&shipmentResp), nil
}

func (c *HTTPAPIClient) convertShipmentResponse(resp *shipmentInfoResponse) *ShipmentResponse {
	links := make([]Link, len(resp.Links.Link))
	for i, l := range resp.Links.Link {
		links[i] = Link(l)
	}

	return &ShipmentResponse{
		ShipmentID:     resp.ShipmentID,
		TrackingPIN:    resp.TrackingPIN,
		ShipmentStatus: resp.ShipmentStatus,
		Links:          links,
	}
}

// GetLabel retrieves a shipping label from the Canada Post API.
func (c *HTTPAPIClient) GetLabel(ctx context.Context, shipmentID string, format string) (*LabelResponse, error) {
	if format == "" {
		format = "application/pdf"
	}

	path := fmt.Sprintf("/rs/%s/artifact/%s", c.accountID, shipmentID)
	resp, err := c.doRequestWithAccept(ctx, http.MethodGet, path, format, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read label data: %w", err)
	}

	return &LabelResponse{
		ShipmentID: shipmentID,
		Format:     format,
		Data:       data,
	}, nil
}

// VoidShipment voids a shipment via the Canada Post API.
func (c *HTTPAPIClient) VoidShipment(ctx context.Context, shipmentID string) (*VoidResponse, error) {
	path := fmt.Sprintf("/rs/%s/shipment/%s", c.accountID, shipmentID)
	resp, err := c.doRequest(ctx, http.MethodDelete, path, "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, c.parseError(resp)
	}

	return &VoidResponse{
		ShipmentID: shipmentID,
		Status:     "voided",
	}, nil
}

// GetTracking retrieves tracking information from the Canada Post API.
func (c *HTTPAPIClient) GetTracking(ctx context.Context, trackingNumber string) (*TrackingResponse, error) {
	path := fmt.Sprintf("/vis/track/pin/%s/summary", trackingNumber)
	resp, err := c.doRequest(ctx, http.MethodGet, path, "application/vnd.cpc.track-v2+xml", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var summary trackingSummary
	if err := xml.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &TrackingResponse{
		TrackingPIN: summary.PINSummary.PIN,
		Status:      summary.PINSummary.EventType,
		Events: []TrackingEvent{
			{
				Timestamp:   summary.PINSummary.EventDateTime,
				Description: summary.PINSummary.EventDescription,
				Location:    summary.PINSummary.EventLocation,
				Type:        summary.PINSummary.EventType,
			},
		},
	}, nil
}

// ============================================================================
// HTTP Helpers
// ============================================================================

func (c *HTTPAPIClient) doRequest(ctx context.Context, method, path, contentType string, body []byte) (*http.Response, error) {
	return c.doRequestWithAccept(ctx, method, path, contentType, body)
}

func (c *HTTPAPIClient) doRequestWithAccept(ctx context.Context, method, path, accept string, body []byte) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Canada Post uses Basic Auth with API key:secret
	credentials := c.apiKey
	if c.apiSecret != "" {
		credentials = c.apiKey + ":" + c.apiSecret
	}
	auth := base64.StdEncoding.EncodeToString([]byte(credentials))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept-Language", "en-CA")

	if body != nil && accept != "" {
		req.Header.Set("Content-Type", accept)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	return c.httpClient.Do(req)
}

func (c *HTTPAPIClient) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	// Try to parse as XML error
	var msgs messages
	if err := xml.Unmarshal(body, &msgs); err == nil && len(msgs.Message) > 0 {
		return &APIError{
			Code:        msgs.Message[0].Code,
			Description: msgs.Message[0].Description,
		}
	}

	return &APIError{
		Code:        fmt.Sprintf("HTTP_%d", resp.StatusCode),
		Description: string(body),
	}
}

// normalizePostalCode removes spaces from postal codes
func normalizePostalCode(pc string) string {
	return strings.ReplaceAll(strings.ToUpper(pc), " ", "")
}

var _ APIClient = (*HTTPAPIClient)(nil)
