package purolator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"text/template"
	"time"
)

// SOAPAPIClient is the production implementation of APIClient using SOAP/WSDL.
type SOAPAPIClient struct {
	wsdlURL    string
	username   string
	password   string
	httpClient *http.Client
}

// SOAPAPIClientConfig holds configuration for the SOAP client.
type SOAPAPIClientConfig struct {
	WSDLURL  string
	Username string
	Password string
	Timeout  time.Duration
}

// NewSOAPAPIClient creates a new SOAP-based API client for production use.
func NewSOAPAPIClient(cfg SOAPAPIClientConfig) *SOAPAPIClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &SOAPAPIClient{
		wsdlURL:  cfg.WSDLURL,
		username: cfg.Username,
		password: cfg.Password,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetRates fetches shipping rates from the Purolator EstimatingService.
func (c *SOAPAPIClient) GetRates(ctx context.Context, req *RatesRequest) (*RatesResponse, error) {
	// Build SOAP envelope for GetFullEstimate request
	soapBody, err := c.buildRatesRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	endpoint := c.getEstimatingServiceEndpoint()
	resp, err := c.doSOAPRequest(ctx, endpoint, "GetFullEstimate", soapBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseSOAPError(resp)
	}

	return c.parseRatesResponse(resp.Body)
}

// CreateShipment creates a new shipment via the Purolator ShippingService.
func (c *SOAPAPIClient) CreateShipment(ctx context.Context, req *ShipmentRequest) (*ShipmentResponse, error) {
	soapBody, err := c.buildShipmentRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	endpoint := c.getShippingServiceEndpoint()
	resp, err := c.doSOAPRequest(ctx, endpoint, "CreateShipment", soapBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseSOAPError(resp)
	}

	return c.parseShipmentResponse(resp.Body)
}

// GetLabel retrieves the shipping label from Purolator ShippingDocumentsService.
func (c *SOAPAPIClient) GetLabel(ctx context.Context, shipmentPIN string, format string) (*LabelResponse, error) {
	soapBody, err := c.buildLabelRequest(shipmentPIN, format)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	endpoint := c.getDocumentsServiceEndpoint()
	resp, err := c.doSOAPRequest(ctx, endpoint, "GetDocuments", soapBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseSOAPError(resp)
	}

	return c.parseLabelResponse(resp.Body, shipmentPIN, format)
}

// VoidShipment cancels a shipment via the Purolator ShippingService.
func (c *SOAPAPIClient) VoidShipment(ctx context.Context, shipmentPIN string) (*VoidResponse, error) {
	soapBody, err := c.buildVoidRequest(shipmentPIN)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	endpoint := c.getShippingServiceEndpoint()
	resp, err := c.doSOAPRequest(ctx, endpoint, "VoidShipment", soapBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseSOAPError(resp)
	}

	return c.parseVoidResponse(resp.Body, shipmentPIN)
}

// GetTracking retrieves tracking info from the Purolator TrackingService.
func (c *SOAPAPIClient) GetTracking(ctx context.Context, trackingPIN string) (*TrackingResponse, error) {
	soapBody, err := c.buildTrackingRequest(trackingPIN)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	endpoint := c.getTrackingServiceEndpoint()
	resp, err := c.doSOAPRequest(ctx, endpoint, "TrackPackagesByPin", soapBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseSOAPError(resp)
	}

	return c.parseTrackingResponse(resp.Body, trackingPIN)
}

// ============================================================================
// SOAP Request Helpers
// ============================================================================

func (c *SOAPAPIClient) doSOAPRequest(ctx context.Context, endpoint, action string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Purolator uses Basic Auth
	auth := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", fmt.Sprintf("http://purolator.com/pws/service/v2/%s", action))

	return c.httpClient.Do(req)
}

func (c *SOAPAPIClient) getEstimatingServiceEndpoint() string {
	return c.wsdlURL + "/EWS/V2/Estimating/EstimatingService.asmx"
}

func (c *SOAPAPIClient) getShippingServiceEndpoint() string {
	return c.wsdlURL + "/EWS/V2/Shipping/ShippingService.asmx"
}

func (c *SOAPAPIClient) getDocumentsServiceEndpoint() string {
	return c.wsdlURL + "/EWS/V2/ShippingDocuments/ShippingDocumentsService.asmx"
}

func (c *SOAPAPIClient) getTrackingServiceEndpoint() string {
	return c.wsdlURL + "/PWS/V1/Tracking/TrackingService.asmx"
}

// ============================================================================
// SOAP Request Builders
// ============================================================================

const soapEnvelopeTemplate = `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:v2="http://purolator.com/pws/datatypes/v2">
  <soap:Header>
    <v2:RequestContext>
      <v2:Version>2.2</v2:Version>
      <v2:Language>en</v2:Language>
      <v2:GroupID>xxx</v2:GroupID>
      <v2:RequestReference>{{.RequestRef}}</v2:RequestReference>
    </v2:RequestContext>
  </soap:Header>
  <soap:Body>
    {{.Body}}
  </soap:Body>
</soap:Envelope>`

func (c *SOAPAPIClient) buildRatesRequest(req *RatesRequest) ([]byte, error) {
	bodyTmpl := `<v2:GetFullEstimateRequest>
      <v2:Shipment>
        <v2:SenderInformation>
          <v2:Address>
            <v2:PostalCode>{{.SenderPostalCode}}</v2:PostalCode>
            <v2:Country>CA</v2:Country>
          </v2:Address>
        </v2:SenderInformation>
        <v2:ReceiverInformation>
          <v2:Address>
            <v2:City>{{.ReceiverAddress.City}}</v2:City>
            <v2:Province>{{.ReceiverAddress.Province}}</v2:Province>
            <v2:PostalCode>{{.ReceiverAddress.PostalCode}}</v2:PostalCode>
            <v2:Country>{{.ReceiverAddress.Country}}</v2:Country>
          </v2:Address>
        </v2:ReceiverInformation>
        <v2:PackageInformation>
          <v2:TotalWeight>
            <v2:Value>{{.PackageInformation.TotalWeight.Value}}</v2:Value>
            <v2:WeightUnit>{{.PackageInformation.TotalWeight.Unit}}</v2:WeightUnit>
          </v2:TotalWeight>
          <v2:TotalPieces>{{.PackageInformation.TotalPieces}}</v2:TotalPieces>
        </v2:PackageInformation>
        <v2:PaymentInformation>
          <v2:PaymentType>Sender</v2:PaymentType>
          <v2:RegisteredAccountNumber>{{.BillingAccountNumber}}</v2:RegisteredAccountNumber>
        </v2:PaymentInformation>
      </v2:Shipment>
      <v2:ShowAlternativeServicesIndicator>true</v2:ShowAlternativeServicesIndicator>
    </v2:GetFullEstimateRequest>`

	return c.buildEnvelope(bodyTmpl, req)
}

func (c *SOAPAPIClient) buildShipmentRequest(req *ShipmentRequest) ([]byte, error) {
	bodyTmpl := `<v2:CreateShipmentRequest>
      <v2:Shipment>
        <v2:SenderInformation>
          <v2:Address>
            <v2:Name>{{.Sender.Address.Name}}</v2:Name>
            <v2:Company>{{.Sender.Address.Company}}</v2:Company>
            <v2:StreetNumber>{{.Sender.Address.StreetNumber}}</v2:StreetNumber>
            <v2:StreetName>{{.Sender.Address.StreetName}}</v2:StreetName>
            <v2:City>{{.Sender.Address.City}}</v2:City>
            <v2:Province>{{.Sender.Address.Province}}</v2:Province>
            <v2:PostalCode>{{.Sender.Address.PostalCode}}</v2:PostalCode>
            <v2:Country>{{.Sender.Address.Country}}</v2:Country>
            <v2:PhoneNumber>
              <v2:CountryCode>{{.Sender.Address.PhoneNumber.CountryCode}}</v2:CountryCode>
              <v2:AreaCode>{{.Sender.Address.PhoneNumber.AreaCode}}</v2:AreaCode>
              <v2:Phone>{{.Sender.Address.PhoneNumber.Phone}}</v2:Phone>
            </v2:PhoneNumber>
          </v2:Address>
        </v2:SenderInformation>
        <v2:ReceiverInformation>
          <v2:Address>
            <v2:Name>{{.Receiver.Address.Name}}</v2:Name>
            <v2:Company>{{.Receiver.Address.Company}}</v2:Company>
            <v2:StreetNumber>{{.Receiver.Address.StreetNumber}}</v2:StreetNumber>
            <v2:StreetName>{{.Receiver.Address.StreetName}}</v2:StreetName>
            <v2:City>{{.Receiver.Address.City}}</v2:City>
            <v2:Province>{{.Receiver.Address.Province}}</v2:Province>
            <v2:PostalCode>{{.Receiver.Address.PostalCode}}</v2:PostalCode>
            <v2:Country>{{.Receiver.Address.Country}}</v2:Country>
            <v2:PhoneNumber>
              <v2:CountryCode>{{.Receiver.Address.PhoneNumber.CountryCode}}</v2:CountryCode>
              <v2:AreaCode>{{.Receiver.Address.PhoneNumber.AreaCode}}</v2:AreaCode>
              <v2:Phone>{{.Receiver.Address.PhoneNumber.Phone}}</v2:Phone>
            </v2:PhoneNumber>
          </v2:Address>
        </v2:ReceiverInformation>
        <v2:PackageInformation>
          <v2:ServiceID>{{.ServiceCode}}</v2:ServiceID>
          <v2:TotalWeight>
            <v2:Value>{{.PackageInformation.TotalWeight.Value}}</v2:Value>
            <v2:WeightUnit>{{.PackageInformation.TotalWeight.Unit}}</v2:WeightUnit>
          </v2:TotalWeight>
          <v2:TotalPieces>{{.PackageInformation.TotalPieces}}</v2:TotalPieces>
        </v2:PackageInformation>
        <v2:PaymentInformation>
          <v2:PaymentType>Sender</v2:PaymentType>
          <v2:RegisteredAccountNumber>{{.BillingAccountNumber}}</v2:RegisteredAccountNumber>
        </v2:PaymentInformation>
      </v2:Shipment>
      <v2:PrinterType>{{.PrinterType}}</v2:PrinterType>
    </v2:CreateShipmentRequest>`

	return c.buildEnvelope(bodyTmpl, req)
}

func (c *SOAPAPIClient) buildLabelRequest(shipmentPIN, format string) ([]byte, error) {
	bodyTmpl := `<v2:GetDocumentsRequest>
      <v2:DocumentCriteria>
        <v2:PIN>
          <v2:Value>{{.ShipmentPIN}}</v2:Value>
        </v2:PIN>
      </v2:DocumentCriteria>
    </v2:GetDocumentsRequest>`

	data := struct {
		ShipmentPIN string
	}{ShipmentPIN: shipmentPIN}

	return c.buildEnvelope(bodyTmpl, data)
}

func (c *SOAPAPIClient) buildVoidRequest(shipmentPIN string) ([]byte, error) {
	bodyTmpl := `<v2:VoidShipmentRequest>
      <v2:PIN>
        <v2:Value>{{.ShipmentPIN}}</v2:Value>
      </v2:PIN>
    </v2:VoidShipmentRequest>`

	data := struct {
		ShipmentPIN string
	}{ShipmentPIN: shipmentPIN}

	return c.buildEnvelope(bodyTmpl, data)
}

func (c *SOAPAPIClient) buildTrackingRequest(trackingPIN string) ([]byte, error) {
	bodyTmpl := `<v1:TrackPackagesByPinRequest xmlns:v1="http://purolator.com/pws/datatypes/v1">
      <v1:PINs>
        <v1:PIN>
          <v1:Value>{{.TrackingPIN}}</v1:Value>
        </v1:PIN>
      </v1:PINs>
    </v1:TrackPackagesByPinRequest>`

	data := struct {
		TrackingPIN string
	}{TrackingPIN: trackingPIN}

	return c.buildEnvelope(bodyTmpl, data)
}

func (c *SOAPAPIClient) buildEnvelope(bodyTemplate string, data interface{}) ([]byte, error) {
	// Parse and execute body template
	bodyTmpl, err := template.New("body").Parse(bodyTemplate)
	if err != nil {
		return nil, err
	}

	var bodyBuf bytes.Buffer
	if err := bodyTmpl.Execute(&bodyBuf, data); err != nil {
		return nil, err
	}

	// Build envelope
	envTmpl, err := template.New("envelope").Parse(soapEnvelopeTemplate)
	if err != nil {
		return nil, err
	}

	envData := struct {
		RequestRef string
		Body       string
	}{
		RequestRef: fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Body:       bodyBuf.String(),
	}

	var envBuf bytes.Buffer
	if err := envTmpl.Execute(&envBuf, envData); err != nil {
		return nil, err
	}

	return envBuf.Bytes(), nil
}

// ============================================================================
// SOAP Response Parsers - XML Types
// ============================================================================

// soapEnvelope represents a SOAP envelope response
type soapEnvelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    soapBody `xml:"Body"`
}

type soapBody struct {
	Fault                   *soapFault                   `xml:"Fault,omitempty"`
	GetFullEstimateResponse *getFullEstimateResponse     `xml:"GetFullEstimateResponse,omitempty"`
	CreateShipmentResponse  *createShipmentResponse      `xml:"CreateShipmentResponse,omitempty"`
	GetDocumentsResponse    *getDocumentsResponse        `xml:"GetDocumentsResponse,omitempty"`
	VoidShipmentResponse    *voidShipmentResponse        `xml:"VoidShipmentResponse,omitempty"`
	TrackPackagesByPinResp  *trackPackagesByPinResponse  `xml:"TrackPackagesByPinResponse,omitempty"`
}

type soapFault struct {
	Code   string `xml:"faultcode"`
	String string `xml:"faultstring"`
}

// GetFullEstimate response types
type getFullEstimateResponse struct {
	ResponseInformation responseInfo      `xml:"ResponseInformation"`
	ShipmentEstimates   shipmentEstimates `xml:"ShipmentEstimates"`
}

type responseInfo struct {
	Errors   []responseError   `xml:"Errors>Error"`
	Messages []responseMessage `xml:"InformationalMessages>InformationalMessage"`
}

type responseError struct {
	Code        string `xml:"Code"`
	Description string `xml:"Description"`
}

type responseMessage struct {
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

type shipmentEstimates struct {
	ShipmentEstimate []shipmentEstimate `xml:"ShipmentEstimate"`
}

type shipmentEstimate struct {
	ServiceID            string          `xml:"ServiceID"`
	ShipmentDate         string          `xml:"ShipmentDate"`
	ExpectedDeliveryDate string          `xml:"ExpectedDeliveryDate"`
	EstimatedTransitDays int             `xml:"EstimatedTransitDays"`
	BasePrice            string          `xml:"BasePrice"`
	Surcharges           soapSurcharges  `xml:"Surcharges"`
	Taxes                soapTaxes       `xml:"Taxes"`
	TotalPrice           string          `xml:"TotalPrice"`
}

type soapSurcharges struct {
	Surcharge []soapSurcharge `xml:"Surcharge"`
}

type soapSurcharge struct {
	Amount      string `xml:"Amount"`
	Type        string `xml:"Type"`
	Description string `xml:"Description"`
}

type soapTaxes struct {
	Tax []soapTax `xml:"Tax"`
}

type soapTax struct {
	Amount      string `xml:"Amount"`
	Type        string `xml:"Type"`
	Description string `xml:"Description"`
}

// CreateShipment response types
type createShipmentResponse struct {
	ResponseInformation  responseInfo `xml:"ResponseInformation"`
	ShipmentPIN          soapPIN      `xml:"ShipmentPIN"`
	PiecePINs            piecePINs    `xml:"PiecePINs"`
	ExpectedDeliveryDate string       `xml:"ExpectedDeliveryDate"`
	TotalPrice           string       `xml:"TotalPrice"`
}

type soapPIN struct {
	Value string `xml:"Value"`
}

type piecePINs struct {
	PIN []soapPIN `xml:"PIN"`
}

// GetDocuments response types
type getDocumentsResponse struct {
	ResponseInformation responseInfo  `xml:"ResponseInformation"`
	Documents           soapDocuments `xml:"Documents"`
}

type soapDocuments struct {
	Document []soapDocument `xml:"Document"`
}

type soapDocument struct {
	PIN             soapPIN          `xml:"PIN"`
	DocumentDetails []documentDetail `xml:"DocumentDetails>DocumentDetail"`
}

type documentDetail struct {
	DocumentType   string `xml:"DocumentType"`
	DocumentStatus string `xml:"DocumentStatus"`
	URL            string `xml:"URL"`
	Data           string `xml:"Data"` // Base64 encoded
}

// VoidShipment response types
type voidShipmentResponse struct {
	ResponseInformation responseInfo `xml:"ResponseInformation"`
	ShipmentVoided      bool         `xml:"ShipmentVoided"`
}

// TrackPackagesByPin response types
type trackPackagesByPinResponse struct {
	ResponseInformation     responseInfo     `xml:"ResponseInformation"`
	TrackingInformationList trackingInfoList `xml:"TrackingInformationList"`
}

type trackingInfoList struct {
	TrackingInformation []trackingInfo `xml:"TrackingInformation"`
}

type trackingInfo struct {
	PIN   soapPIN   `xml:"PIN"`
	Scans soapScans `xml:"Scans"`
}

type soapScans struct {
	Scan []soapScan `xml:"Scan"`
}

type soapScan struct {
	ScanType    string    `xml:"ScanType"`
	ScanDate    string    `xml:"ScanDate"`
	ScanTime    string    `xml:"ScanTime"`
	Description string    `xml:"Description"`
	Depot       soapDepot `xml:"Depot"`
}

type soapDepot struct {
	Name    string      `xml:"Name"`
	Address soapAddress `xml:"Address"`
}

type soapAddress struct {
	City       string `xml:"City"`
	Province   string `xml:"Province"`
	Country    string `xml:"Country"`
	PostalCode string `xml:"PostalCode"`
}

// ============================================================================
// SOAP Response Parsing Functions
// ============================================================================

func (c *SOAPAPIClient) parseSOAPError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var env soapEnvelope
	if err := xml.Unmarshal(body, &env); err == nil && env.Body.Fault != nil {
		return &APIError{
			Code:        env.Body.Fault.Code,
			Description: env.Body.Fault.String,
		}
	}

	return &APIError{
		Code:        fmt.Sprintf("HTTP_%d", resp.StatusCode),
		Description: string(body),
	}
}

func (c *SOAPAPIClient) parseRatesResponse(body io.Reader) (*RatesResponse, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var env soapEnvelope
	if err := xml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if env.Body.Fault != nil {
		return nil, &APIError{
			Code:        env.Body.Fault.Code,
			Description: env.Body.Fault.String,
		}
	}

	if env.Body.GetFullEstimateResponse == nil {
		return nil, &APIError{
			Code:        "PARSE_ERROR",
			Description: "No rate estimates in response",
		}
	}

	resp := env.Body.GetFullEstimateResponse

	// Check for API errors in response
	if len(resp.ResponseInformation.Errors) > 0 {
		e := resp.ResponseInformation.Errors[0]
		return nil, &APIError{
			Code:        e.Code,
			Description: e.Description,
		}
	}

	rates := make([]ShipmentRate, len(resp.ShipmentEstimates.ShipmentEstimate))
	for i, est := range resp.ShipmentEstimates.ShipmentEstimate {
		// Calculate fuel surcharge from surcharges
		var fuelSurcharge float64
		for _, sc := range est.Surcharges.Surcharge {
			if sc.Type == "Fuel" || sc.Type == "FuelSurcharge" {
				fuelSurcharge = parseFloat(sc.Amount)
			}
		}

		// Calculate total taxes
		var taxes float64
		for _, tax := range est.Taxes.Tax {
			taxes += parseFloat(tax.Amount)
		}

		rates[i] = ShipmentRate{
			ServiceCode:          est.ServiceID,
			ServiceName:          mapServiceName(est.ServiceID),
			BasePrice:            parseFloat(est.BasePrice),
			FuelSurcharge:        fuelSurcharge,
			Taxes:                taxes,
			TotalPrice:           parseFloat(est.TotalPrice),
			ExpectedDeliveryDate: est.ExpectedDeliveryDate,
			EstimatedTransitDays: est.EstimatedTransitDays,
			GuaranteedDelivery:   isGuaranteedService(est.ServiceID),
		}
	}

	return &RatesResponse{
		QuoteID:       fmt.Sprintf("puro-quote-%d", time.Now().UnixNano()),
		ShipmentRates: rates,
	}, nil
}

func (c *SOAPAPIClient) parseShipmentResponse(body io.Reader) (*ShipmentResponse, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var env soapEnvelope
	if err := xml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if env.Body.Fault != nil {
		return nil, &APIError{
			Code:        env.Body.Fault.Code,
			Description: env.Body.Fault.String,
		}
	}

	if env.Body.CreateShipmentResponse == nil {
		return nil, &APIError{
			Code:        "PARSE_ERROR",
			Description: "No shipment data in response",
		}
	}

	resp := env.Body.CreateShipmentResponse

	// Check for API errors
	if len(resp.ResponseInformation.Errors) > 0 {
		e := resp.ResponseInformation.Errors[0]
		return nil, &APIError{
			Code:        e.Code,
			Description: e.Description,
		}
	}

	// Extract piece PINs
	piecePINs := make([]string, len(resp.PiecePINs.PIN))
	for i, pin := range resp.PiecePINs.PIN {
		piecePINs[i] = pin.Value
	}

	return &ShipmentResponse{
		ShipmentPIN:          resp.ShipmentPIN.Value,
		TrackingNumber:       resp.ShipmentPIN.Value,
		TotalPrice:           parseFloat(resp.TotalPrice),
		ExpectedDeliveryDate: resp.ExpectedDeliveryDate,
		PiecePINs:            piecePINs,
	}, nil
}

func (c *SOAPAPIClient) parseLabelResponse(body io.Reader, shipmentPIN, format string) (*LabelResponse, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var env soapEnvelope
	if err := xml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if env.Body.Fault != nil {
		return nil, &APIError{
			Code:        env.Body.Fault.Code,
			Description: env.Body.Fault.String,
		}
	}

	if env.Body.GetDocumentsResponse == nil {
		return nil, &APIError{
			Code:        "PARSE_ERROR",
			Description: "No document data in response",
		}
	}

	resp := env.Body.GetDocumentsResponse

	// Check for API errors
	if len(resp.ResponseInformation.Errors) > 0 {
		e := resp.ResponseInformation.Errors[0]
		return nil, &APIError{
			Code:        e.Code,
			Description: e.Description,
		}
	}

	// Find the label document
	for _, doc := range resp.Documents.Document {
		for _, detail := range doc.DocumentDetails {
			if detail.DocumentStatus == "Completed" {
				// Decode base64 label data
				labelData, err := base64.StdEncoding.DecodeString(detail.Data)
				if err != nil {
					return nil, fmt.Errorf("failed to decode label data: %w", err)
				}

				return &LabelResponse{
					ShipmentPIN: shipmentPIN,
					Format:      format,
					Data:        labelData,
				}, nil
			}
		}
	}

	return nil, &APIError{
		Code:        "LABEL_NOT_FOUND",
		Description: "No completed label found in response",
	}
}

func (c *SOAPAPIClient) parseVoidResponse(body io.Reader, shipmentPIN string) (*VoidResponse, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var env soapEnvelope
	if err := xml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if env.Body.Fault != nil {
		return nil, &APIError{
			Code:        env.Body.Fault.Code,
			Description: env.Body.Fault.String,
		}
	}

	if env.Body.VoidShipmentResponse == nil {
		return nil, &APIError{
			Code:        "PARSE_ERROR",
			Description: "No void response data",
		}
	}

	resp := env.Body.VoidShipmentResponse

	// Check for API errors
	if len(resp.ResponseInformation.Errors) > 0 {
		e := resp.ResponseInformation.Errors[0]
		return nil, &APIError{
			Code:        e.Code,
			Description: e.Description,
		}
	}

	status := "failed"
	message := "Failed to void shipment"
	if resp.ShipmentVoided {
		status = "voided"
		message = "Shipment successfully voided"
	}

	return &VoidResponse{
		ShipmentPIN: shipmentPIN,
		Status:      status,
		Message:     message,
	}, nil
}

func (c *SOAPAPIClient) parseTrackingResponse(body io.Reader, trackingPIN string) (*TrackingResponse, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var env soapEnvelope
	if err := xml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if env.Body.Fault != nil {
		return nil, &APIError{
			Code:        env.Body.Fault.Code,
			Description: env.Body.Fault.String,
		}
	}

	if env.Body.TrackPackagesByPinResp == nil {
		return nil, &APIError{
			Code:        "PARSE_ERROR",
			Description: "No tracking data in response",
		}
	}

	resp := env.Body.TrackPackagesByPinResp

	// Check for API errors
	if len(resp.ResponseInformation.Errors) > 0 {
		e := resp.ResponseInformation.Errors[0]
		return nil, &APIError{
			Code:        e.Code,
			Description: e.Description,
		}
	}

	// Find tracking info for our PIN
	for _, info := range resp.TrackingInformationList.TrackingInformation {
		if info.PIN.Value == trackingPIN {
			events := make([]TrackingEvent, len(info.Scans.Scan))
			latestStatus := ""

			for i, scan := range info.Scans.Scan {
				location := scan.Depot.Address.City
				if scan.Depot.Address.Province != "" {
					location += ", " + scan.Depot.Address.Province
				}

				events[i] = TrackingEvent{
					Timestamp:   scan.ScanDate + "T" + scan.ScanTime,
					Description: scan.Description,
					Location:    location,
					Type:        scan.ScanType,
				}
				if i == 0 {
					latestStatus = scan.ScanType
				}
			}

			return &TrackingResponse{
				TrackingPIN:    trackingPIN,
				Status:         latestStatus,
				DeliveryStatus: mapDeliveryStatus(latestStatus),
				Events:         events,
			}, nil
		}
	}

	return nil, &APIError{
		Code:        "TRACKING_NOT_FOUND",
		Description: "Tracking information not found for PIN",
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func mapServiceName(serviceID string) string {
	serviceNames := map[string]string{
		"PurolatorExpress":        "Purolator Express",
		"PurolatorExpress9AM":     "Purolator Express 9AM",
		"PurolatorExpress10:30AM": "Purolator Express 10:30AM",
		"PurolatorExpress12PM":    "Purolator Express 12PM",
		"PurolatorExpressEvening": "Purolator Express Evening",
		"PurolatorGround":         "Purolator Ground",
		"PurolatorGround9AM":      "Purolator Ground 9AM",
		"PurolatorGround10:30AM":  "Purolator Ground 10:30AM",
		"PurolatorExpressUS":      "Purolator Express U.S.",
		"PurolatorExpressUSPack":  "Purolator Express U.S. Pack",
		"PurolatorGroundUS":       "Purolator Ground U.S.",
	}
	if name, ok := serviceNames[serviceID]; ok {
		return name
	}
	return serviceID
}

func isGuaranteedService(serviceID string) bool {
	guaranteedServices := map[string]bool{
		"PurolatorExpress":        true,
		"PurolatorExpress9AM":     true,
		"PurolatorExpress10:30AM": true,
		"PurolatorExpress12PM":    true,
		"PurolatorExpressEvening": true,
		"PurolatorExpressUS":      true,
	}
	return guaranteedServices[serviceID]
}

func mapDeliveryStatus(scanType string) string {
	statusMap := map[string]string{
		"PickedUp":       "Picked Up",
		"InTransit":      "In Transit",
		"OutForDelivery": "Out for Delivery",
		"Delivered":      "Delivered",
		"Exception":      "Exception",
		"ReturnToSender": "Return to Sender",
	}
	if status, ok := statusMap[scanType]; ok {
		return status
	}
	return scanType
}

var _ APIClient = (*SOAPAPIClient)(nil)
