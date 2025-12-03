package freightcom_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/tournevent/logistic/pkg/shipper/freightcom"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

func newTestClient(mockClient *freightcom.MockAPIClient) *freightcom.Client {
	logger := otelzap.New(zap.NewNop())
	return freightcom.NewWithAPIClient(
		freightcom.Config{},
		mockClient,
		logger,
		nil,
	)
}

func TestClient_GetQuote_Success(t *testing.T) {
	mockAPI := freightcom.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.QuoteRequest{
		Origin: shipper.Address{
			Name:         "Sender",
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V 1A1",
			CountryCode:  "CA",
		},
		Destination: shipper.Address{
			Name:         "Receiver",
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B 2W2",
			CountryCode:  "CA",
		},
		Packages: []shipper.Package{
			{Length: 10, Width: 10, Height: 10, Weight: 5},
		},
	}

	ctx := context.Background()
	resp, err := client.GetQuote(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.QuoteID)
	assert.Len(t, resp.Rates, 3) // Mock returns 3 rates
	assert.Equal(t, "freightcom", resp.Rates[0].Carrier)
}

func TestClient_GetQuote_APIError(t *testing.T) {
	mockAPI := freightcom.NewMockAPIClient()
	mockAPI.SimulateErrors = true

	client := newTestClient(mockAPI)

	req := &shipper.QuoteRequest{
		Origin:      shipper.Address{City: "Toronto"},
		Destination: shipper.Address{City: "Vancouver"},
	}

	ctx := context.Background()
	_, err := client.GetQuote(ctx, req)

	assert.Error(t, err)
}

func TestClient_GetQuote_CustomMock(t *testing.T) {
	mockAPI := freightcom.NewMockAPIClient()
	mockAPI.OnGetRates = func(ctx context.Context, req *freightcom.RatesRequest) (*freightcom.RatesResponse, error) {
		// Custom mock behavior for this test
		return &freightcom.RatesResponse{
			RequestID: "custom-quote-123",
			Status:    "complete",
			Rates: []freightcom.Rate{
				{
					ID:          "custom-rate-1",
					ServiceCode: "OVERNIGHT",
					ServiceName: "Overnight Express",
					TotalPrice:  99.99,
					Currency:    "CAD",
					TransitDays: 1,
					Guaranteed:  true,
				},
			},
		}, nil
	}

	client := newTestClient(mockAPI)

	req := &shipper.QuoteRequest{
		Origin:      shipper.Address{City: "Toronto"},
		Destination: shipper.Address{City: "Vancouver"},
	}

	ctx := context.Background()
	resp, err := client.GetQuote(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "custom-quote-123", resp.QuoteID)
	assert.Len(t, resp.Rates, 1)
	assert.Equal(t, "Overnight Express", resp.Rates[0].ServiceName)
	assert.Equal(t, shipper.ServiceOvernight, resp.Rates[0].ServiceType)
}

func TestClient_CreateOrder_Success(t *testing.T) {
	mockAPI := freightcom.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.CreateOrderRequest{
		RateID: "fc-rate-ground-123",
		Sender: shipper.Contact{Name: "John Doe", Phone: "416-555-1234"},
		SenderAddress: shipper.Address{
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V 1A1",
			CountryCode:  "CA",
		},
		Recipient: shipper.Contact{Name: "Jane Smith", Phone: "604-555-5678"},
		RecipientAddress: shipper.Address{
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B 2W2",
			CountryCode:  "CA",
		},
		Packages: []shipper.Package{
			{Length: 10, Width: 10, Height: 10, Weight: 5},
		},
	}

	ctx := context.Background()
	resp, err := client.CreateOrder(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.OrderID)
	assert.NotEmpty(t, resp.TrackingNumber)
	assert.Equal(t, shipper.StatusConfirmed, resp.Status)
	assert.Equal(t, "freightcom", resp.Carrier)
}

func TestClient_GetLabel_Success(t *testing.T) {
	mockAPI := freightcom.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.GetLabelRequest{
		OrderID: "fc-order-123",
		Format:  shipper.LabelPDF,
	}

	ctx := context.Background()
	resp, err := client.GetLabel(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "fc-order-123", resp.OrderID)
	assert.Equal(t, shipper.LabelPDF, resp.Label.Format)
	assert.NotEmpty(t, resp.Label.URL)
}

func TestClient_CancelOrder_Success(t *testing.T) {
	mockAPI := freightcom.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.CancelOrderRequest{
		OrderID: "fc-order-123",
		Reason:  "Customer requested cancellation",
	}

	ctx := context.Background()
	resp, err := client.CancelOrder(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "fc-order-123", resp.OrderID)
	assert.Equal(t, shipper.StatusCancelled, resp.Status)
	assert.NotNil(t, resp.RefundAmount)
	assert.NotEmpty(t, resp.ConfirmationNumber)
}

func TestClient_CancelOrder_CustomError(t *testing.T) {
	mockAPI := freightcom.NewMockAPIClient()
	mockAPI.OnCancelShipment = func(ctx context.Context, orderID string, reason string) (*freightcom.CancelResponse, error) {
		return nil, errors.New("shipment already delivered, cannot cancel")
	}

	client := newTestClient(mockAPI)

	req := &shipper.CancelOrderRequest{
		OrderID: "fc-order-123",
		Reason:  "Customer requested cancellation",
	}

	ctx := context.Background()
	_, err := client.CancelOrder(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel")
}

func TestClient_Name(t *testing.T) {
	mockAPI := freightcom.NewMockAPIClient()
	client := newTestClient(mockAPI)

	assert.Equal(t, "freightcom", client.Name())
}
