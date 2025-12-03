package freightcom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPAPIClient is the production implementation of APIClient using HTTP.
type HTTPAPIClient struct {
	baseURL      string
	apiKey       string
	httpClient   *http.Client
	pollInterval time.Duration
	pollTimeout  time.Duration
}

// HTTPAPIClientConfig holds configuration for the HTTP client.
type HTTPAPIClientConfig struct {
	BaseURL      string
	APIKey       string
	Timeout      time.Duration
	PollInterval time.Duration // Interval between polling for async operations
	PollTimeout  time.Duration // Max time to wait for async operations
}

// NewHTTPAPIClient creates a new HTTP-based API client for production use.
func NewHTTPAPIClient(cfg HTTPAPIClientConfig) *HTTPAPIClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	pollInterval := cfg.PollInterval
	if pollInterval == 0 {
		pollInterval = 500 * time.Millisecond
	}

	pollTimeout := cfg.PollTimeout
	if pollTimeout == 0 {
		pollTimeout = 30 * time.Second
	}

	return &HTTPAPIClient{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		pollInterval: pollInterval,
		pollTimeout:  pollTimeout,
	}
}

// GetRates fetches shipping rates from the Freightcom API.
// This is an async operation: POST /rate returns a request_id,
// then we poll GET /rate/{request_id} until complete.
func (c *HTTPAPIClient) GetRates(ctx context.Context, req *RatesRequest) (*RatesResponse, error) {
	// Step 1: Submit rate request
	resp, err := c.doRequest(ctx, http.MethodPost, "/rate", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var rateReq RateRequestResponse
	if err := json.NewDecoder(resp.Body).Decode(&rateReq); err != nil {
		return nil, fmt.Errorf("failed to decode rate request response: %w", err)
	}

	// Step 2: Poll for results
	return c.pollRates(ctx, rateReq.RequestID)
}

// pollRates polls the rate endpoint until results are ready or timeout.
func (c *HTTPAPIClient) pollRates(ctx context.Context, requestID string) (*RatesResponse, error) {
	deadline := time.Now().Add(c.pollTimeout)
	path := fmt.Sprintf("/rate/%s", requestID)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return nil, &APIError{
				Code:    "TIMEOUT",
				Message: "Rate request timed out waiting for results",
			}
		}

		resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			err := c.parseError(resp)
			resp.Body.Close()
			return nil, err
		}

		var result RatesResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode rates response: %w", err)
		}
		resp.Body.Close()

		switch result.Status {
		case "complete":
			return &result, nil
		case "error":
			return nil, &APIError{
				Code:    "RATE_ERROR",
				Message: result.Error,
			}
		case "pending":
			// Continue polling
			time.Sleep(c.pollInterval)
		default:
			return nil, &APIError{
				Code:    "UNKNOWN_STATUS",
				Message: fmt.Sprintf("Unknown rate status: %s", result.Status),
			}
		}
	}
}

// CreateShipment creates a new shipment via the Freightcom API.
// POST /shipment - may return 202 Accepted for async processing.
func (c *HTTPAPIClient) CreateShipment(ctx context.Context, req *ShipmentRequest) (*ShipmentResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/shipment", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return nil, c.parseError(resp)
	}

	var result ShipmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode shipment response: %w", err)
	}

	// If status is pending, poll for completion
	if result.Status == "pending" || result.Status == "processing" {
		return c.pollShipment(ctx, result.ID)
	}

	return &result, nil
}

// pollShipment polls the shipment endpoint until it's ready.
func (c *HTTPAPIClient) pollShipment(ctx context.Context, shipmentID string) (*ShipmentResponse, error) {
	deadline := time.Now().Add(c.pollTimeout)
	path := fmt.Sprintf("/shipment/%s", shipmentID)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if time.Now().After(deadline) {
			return nil, &APIError{
				Code:    "TIMEOUT",
				Message: "Shipment creation timed out",
			}
		}

		resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			err := c.parseError(resp)
			resp.Body.Close()
			return nil, err
		}

		var result ShipmentResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode shipment response: %w", err)
		}
		resp.Body.Close()

		switch result.Status {
		case "booked", "confirmed", "complete":
			return &result, nil
		case "error", "failed":
			return nil, &APIError{
				Code:    "SHIPMENT_ERROR",
				Message: fmt.Sprintf("Shipment failed with status: %s", result.Status),
			}
		case "pending", "processing":
			time.Sleep(c.pollInterval)
		default:
			// Unknown status, return as-is
			return &result, nil
		}
	}
}

// GetLabel retrieves a shipping label from the Freightcom API.
// Labels are retrieved from the shipment details: GET /shipment/{shipment_id}
func (c *HTTPAPIClient) GetLabel(ctx context.Context, shipmentID string, format string) (*LabelResponse, error) {
	path := fmt.Sprintf("/shipment/%s", shipmentID)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var shipment ShipmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&shipment); err != nil {
		return nil, fmt.Errorf("failed to decode shipment response: %w", err)
	}

	// Filter labels by format if specified
	var labels []Label
	if format != "" {
		for _, label := range shipment.Labels {
			if label.Format == format {
				labels = append(labels, label)
			}
		}
	} else {
		labels = shipment.Labels
	}

	return &LabelResponse{
		ShipmentID: shipmentID,
		Labels:     labels,
	}, nil
}

// CancelShipment cancels a shipment via the Freightcom API.
// DELETE /shipment/{shipment_id}
func (c *HTTPAPIClient) CancelShipment(ctx context.Context, shipmentID string, reason string) (*CancelResponse, error) {
	path := fmt.Sprintf("/shipment/%s", shipmentID)

	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, c.parseError(resp)
	}

	// DELETE may return empty body on success
	if resp.StatusCode == http.StatusNoContent {
		return &CancelResponse{
			ShipmentID: shipmentID,
			Status:     "cancelled",
		}, nil
	}

	var result CancelResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// If decode fails but status was OK, return success
		return &CancelResponse{
			ShipmentID: shipmentID,
			Status:     "cancelled",
		}, nil
	}

	return &result, nil
}

// GetTracking retrieves tracking information from the Freightcom API.
// GET /shipment/{shipment_id}/tracking-events
func (c *HTTPAPIClient) GetTracking(ctx context.Context, shipmentID string) (*TrackingResponse, error) {
	path := fmt.Sprintf("/shipment/%s/tracking-events", shipmentID)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result TrackingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode tracking response: %w", err)
	}

	result.ShipmentID = shipmentID
	return &result, nil
}

// doRequest performs an HTTP request with proper headers and authentication.
func (c *HTTPAPIClient) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.apiKey) // Freightcom uses X-API-Key header
	req.Header.Set("User-Agent", "delivro-logistic/1.0")

	return c.httpClient.Do(req)
}

// parseError extracts error information from an HTTP response.
func (c *HTTPAPIClient) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Code != "" {
		return &apiErr
	}

	// Try to parse as a simple error message
	var simpleErr struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &simpleErr); err == nil {
		msg := simpleErr.Error
		if msg == "" {
			msg = simpleErr.Message
		}
		if msg != "" {
			return &APIError{
				Code:    fmt.Sprintf("HTTP_%d", resp.StatusCode),
				Message: msg,
			}
		}
	}

	return &APIError{
		Code:    fmt.Sprintf("HTTP_%d", resp.StatusCode),
		Message: string(body),
	}
}

// Ensure HTTPAPIClient implements APIClient interface
var _ APIClient = (*HTTPAPIClient)(nil)
