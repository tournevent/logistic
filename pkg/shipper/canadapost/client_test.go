package canadapost_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/tournevent/logistic/pkg/shipper/canadapost"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

func newTestClient(mockClient *canadapost.MockAPIClient) *canadapost.Client {
	logger := otelzap.New(zap.NewNop())
	return canadapost.NewWithAPIClient(
		canadapost.Config{AccountID: "test-account"},
		mockClient,
		logger,
		nil,
	)
}

func TestClient_GetQuote_Success(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.QuoteRequest{
		Origin: shipper.Address{
			Name:         "Sender",
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V1A1",
			CountryCode:  "CA",
		},
		Destination: shipper.Address{
			Name:         "Receiver",
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B2W2",
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
	assert.Equal(t, "canadapost", resp.Rates[0].Carrier)
}

func TestClient_GetQuote_APIError(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	mockAPI.SimulateErrors = true

	client := newTestClient(mockAPI)

	req := &shipper.QuoteRequest{
		Origin:      shipper.Address{City: "Toronto", PostalCode: "M5V1A1"},
		Destination: shipper.Address{City: "Vancouver", PostalCode: "V6B2W2"},
	}

	ctx := context.Background()
	_, err := client.GetQuote(ctx, req)

	assert.Error(t, err)
}

func TestClient_GetQuote_CustomMock(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	mockAPI.OnGetRates = func(ctx context.Context, req *canadapost.RatesRequest) (*canadapost.RatesResponse, error) {
		return &canadapost.RatesResponse{
			QuoteID: "custom-quote-cp-123",
			Rates: []canadapost.Rate{
				{
					ServiceCode:       "DOM.XP",
					ServiceName:       "Xpresspost",
					TotalPrice:        25.30,
					ExpectedTransit:   2,
					ExpectedDelivery:  "2024-01-15",
					GuaranteedDelivery: true,
				},
			},
		}, nil
	}

	client := newTestClient(mockAPI)

	req := &shipper.QuoteRequest{
		Origin:      shipper.Address{PostalCode: "M5V1A1"},
		Destination: shipper.Address{PostalCode: "V6B2W2"},
	}

	ctx := context.Background()
	resp, err := client.GetQuote(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "custom-quote-cp-123", resp.QuoteID)
	assert.Len(t, resp.Rates, 1)
	assert.Equal(t, "Xpresspost", resp.Rates[0].ServiceName)
	assert.Equal(t, shipper.ServiceExpress, resp.Rates[0].ServiceType)
}

func TestClient_GetQuote_International(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.QuoteRequest{
		Origin: shipper.Address{
			PostalCode:  "M5V1A1",
			CountryCode: "CA",
		},
		Destination: shipper.Address{
			City:        "New York",
			CountryCode: "US",
		},
		Packages: []shipper.Package{
			{Length: 10, Width: 10, Height: 10, Weight: 5},
		},
	}

	ctx := context.Background()
	resp, err := client.GetQuote(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Rates)
}

func TestClient_CreateOrder_Success(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.CreateOrderRequest{
		RateID: "cp-DOM.RP-20231215120000",
		Sender: shipper.Contact{Name: "John Doe", Phone: "416-555-1234"},
		SenderAddress: shipper.Address{
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V1A1",
			CountryCode:  "CA",
		},
		Recipient: shipper.Contact{Name: "Jane Smith", Phone: "604-555-5678"},
		RecipientAddress: shipper.Address{
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B2W2",
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
	assert.Equal(t, "canadapost", resp.Carrier)
}

func TestClient_CreateOrder_APIError(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	mockAPI.SimulateErrors = true

	client := newTestClient(mockAPI)

	req := &shipper.CreateOrderRequest{
		RateID:           "cp-DOM.RP-123",
		SenderAddress:    shipper.Address{City: "Toronto"},
		RecipientAddress: shipper.Address{City: "Vancouver"},
	}

	ctx := context.Background()
	_, err := client.CreateOrder(ctx, req)

	assert.Error(t, err)
}

func TestClient_GetLabel_Success(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.GetLabelRequest{
		OrderID: "cp-ship-12345678",
		Format:  shipper.LabelPDF,
	}

	ctx := context.Background()
	resp, err := client.GetLabel(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "cp-ship-12345678", resp.OrderID)
	assert.Equal(t, shipper.LabelPDF, resp.Label.Format)
	assert.NotEmpty(t, resp.Label.Data) // Base64 encoded
}

func TestClient_GetLabel_ZPLFormat(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	mockAPI.OnGetLabel = func(ctx context.Context, shipmentID string, format string) (*canadapost.LabelResponse, error) {
		return &canadapost.LabelResponse{
			ShipmentID: shipmentID,
			Format:     "application/zpl",
			Data:       []byte("^XA^FO50,50^A0N,50,50^FDZPL LABEL^FS^XZ"),
		}, nil
	}

	client := newTestClient(mockAPI)

	req := &shipper.GetLabelRequest{
		OrderID: "cp-ship-12345678",
		Format:  shipper.LabelZPL,
	}

	ctx := context.Background()
	resp, err := client.GetLabel(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, shipper.LabelZPL, resp.Label.Format)
}

func TestClient_GetLabel_APIError(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	mockAPI.SimulateErrors = true

	client := newTestClient(mockAPI)

	req := &shipper.GetLabelRequest{
		OrderID: "cp-ship-12345678",
	}

	ctx := context.Background()
	_, err := client.GetLabel(ctx, req)

	assert.Error(t, err)
}

func TestClient_CancelOrder_Success(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.CancelOrderRequest{
		OrderID: "cp-ship-12345678",
		Reason:  "Customer requested cancellation",
	}

	ctx := context.Background()
	resp, err := client.CancelOrder(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "cp-ship-12345678", resp.OrderID)
	assert.Equal(t, shipper.StatusCancelled, resp.Status)
	assert.NotEmpty(t, resp.ConfirmationNumber)
}

func TestClient_CancelOrder_APIError(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	mockAPI.SimulateErrors = true

	client := newTestClient(mockAPI)

	req := &shipper.CancelOrderRequest{
		OrderID: "cp-ship-12345678",
		Reason:  "Test cancellation",
	}

	ctx := context.Background()
	_, err := client.CancelOrder(ctx, req)

	assert.Error(t, err)
}

func TestClient_CancelOrder_CustomError(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	mockAPI.OnVoidShipment = func(ctx context.Context, shipmentID string) (*canadapost.VoidResponse, error) {
		return nil, errors.New("shipment already delivered, cannot cancel")
	}

	client := newTestClient(mockAPI)

	req := &shipper.CancelOrderRequest{
		OrderID: "cp-ship-12345678",
		Reason:  "Customer requested cancellation",
	}

	ctx := context.Background()
	_, err := client.CancelOrder(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel")
}

func TestClient_Name(t *testing.T) {
	mockAPI := canadapost.NewMockAPIClient()
	client := newTestClient(mockAPI)

	assert.Equal(t, "canadapost", client.Name())
}

func TestClient_New_WithMock(t *testing.T) {
	logger := otelzap.New(zap.NewNop())
	client := canadapost.New(
		canadapost.Config{
			UseMock:   true,
			AccountID: "test-account",
		},
		logger,
		nil,
	)

	assert.Equal(t, "canadapost", client.Name())

	// Test that mock works
	ctx := context.Background()
	resp, err := client.GetQuote(ctx, &shipper.QuoteRequest{
		Origin:      shipper.Address{PostalCode: "M5V1A1"},
		Destination: shipper.Address{PostalCode: "V6B2W2"},
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Rates)
}
