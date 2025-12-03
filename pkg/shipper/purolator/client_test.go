package purolator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/tournevent/logistic/pkg/shipper/purolator"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

func newTestClient(mockClient *purolator.MockAPIClient) *purolator.Client {
	logger := otelzap.New(zap.NewNop())
	return purolator.NewWithAPIClient(
		purolator.Config{},
		mockClient,
		logger,
		nil,
	)
}

func TestClient_GetQuote_Success(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
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
	assert.Equal(t, "purolator", resp.Rates[0].Carrier)
}

func TestClient_GetQuote_APIError(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
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
	mockAPI := purolator.NewMockAPIClient()
	mockAPI.OnGetRates = func(ctx context.Context, req *purolator.RatesRequest) (*purolator.RatesResponse, error) {
		return &purolator.RatesResponse{
			QuoteID: "custom-puro-quote-123",
			ShipmentRates: []purolator.ShipmentRate{
				{
					ServiceCode:          "PurolatorExpress9AM",
					ServiceName:          "Purolator Express 9AM",
					TotalPrice:           56.95,
					EstimatedTransitDays: 1,
					ExpectedDeliveryDate: "2024-01-15",
					GuaranteedDelivery:   true,
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
	assert.Equal(t, "custom-puro-quote-123", resp.QuoteID)
	assert.Len(t, resp.Rates, 1)
	assert.Equal(t, "Purolator Express 9AM", resp.Rates[0].ServiceName)
	assert.Equal(t, shipper.ServiceOvernight, resp.Rates[0].ServiceType)
}

func TestClient_GetQuote_MultiplePackages(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.QuoteRequest{
		Origin:      shipper.Address{PostalCode: "M5V1A1"},
		Destination: shipper.Address{PostalCode: "V6B2W2"},
		Packages: []shipper.Package{
			{Weight: 5},
			{Weight: 3},
			{Weight: 2},
		},
	}

	ctx := context.Background()
	resp, err := client.GetQuote(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Rates)
}

func TestClient_CreateOrder_Success(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.CreateOrderRequest{
		RateID: "puro-PurolatorGround-20231215120000",
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
	assert.Equal(t, "purolator", resp.Carrier)
	assert.Contains(t, resp.TrackingURL, "purolator.com")
}

func TestClient_CreateOrder_APIError(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	mockAPI.SimulateErrors = true

	client := newTestClient(mockAPI)

	req := &shipper.CreateOrderRequest{
		RateID:           "puro-PurolatorGround-123",
		SenderAddress:    shipper.Address{City: "Toronto"},
		RecipientAddress: shipper.Address{City: "Vancouver"},
	}

	ctx := context.Background()
	_, err := client.CreateOrder(ctx, req)

	assert.Error(t, err)
}

func TestClient_CreateOrder_ExpressService(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.CreateOrderRequest{
		RateID: "puro-PurolatorExpress-20231215120000",
		SenderAddress: shipper.Address{
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V1A1",
		},
		RecipientAddress: shipper.Address{
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B2W2",
		},
		Packages: []shipper.Package{{Weight: 5}},
	}

	ctx := context.Background()
	resp, err := client.CreateOrder(ctx, req)

	require.NoError(t, err)
	assert.NotEmpty(t, resp.OrderID)
}

func TestClient_GetLabel_Success(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.GetLabelRequest{
		OrderID: "puro-ship-12345678",
		Format:  shipper.LabelPDF,
	}

	ctx := context.Background()
	resp, err := client.GetLabel(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "puro-ship-12345678", resp.OrderID)
	assert.Equal(t, shipper.LabelPDF, resp.Label.Format)
	assert.NotEmpty(t, resp.Label.Data) // Base64 encoded
}

func TestClient_GetLabel_ZPLFormat(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	mockAPI.OnGetLabel = func(ctx context.Context, shipmentPIN string, format string) (*purolator.LabelResponse, error) {
		return &purolator.LabelResponse{
			ShipmentPIN: shipmentPIN,
			Format:      "application/zpl",
			Data:        []byte("^XA^FO50,50^A0N,50,50^FDZPL LABEL^FS^XZ"),
		}, nil
	}

	client := newTestClient(mockAPI)

	req := &shipper.GetLabelRequest{
		OrderID: "puro-ship-12345678",
		Format:  shipper.LabelZPL,
	}

	ctx := context.Background()
	resp, err := client.GetLabel(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, shipper.LabelZPL, resp.Label.Format)
}

func TestClient_GetLabel_APIError(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	mockAPI.SimulateErrors = true

	client := newTestClient(mockAPI)

	req := &shipper.GetLabelRequest{
		OrderID: "puro-ship-12345678",
	}

	ctx := context.Background()
	_, err := client.GetLabel(ctx, req)

	assert.Error(t, err)
}

func TestClient_CancelOrder_Success(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.CancelOrderRequest{
		OrderID: "puro-ship-12345678",
		Reason:  "Customer requested cancellation",
	}

	ctx := context.Background()
	resp, err := client.CancelOrder(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "puro-ship-12345678", resp.OrderID)
	assert.Equal(t, shipper.StatusCancelled, resp.Status)
	assert.NotEmpty(t, resp.ConfirmationNumber)
}

func TestClient_CancelOrder_APIError(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	mockAPI.SimulateErrors = true

	client := newTestClient(mockAPI)

	req := &shipper.CancelOrderRequest{
		OrderID: "puro-ship-12345678",
		Reason:  "Test cancellation",
	}

	ctx := context.Background()
	_, err := client.CancelOrder(ctx, req)

	assert.Error(t, err)
}

func TestClient_CancelOrder_CustomError(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	mockAPI.OnVoidShipment = func(ctx context.Context, shipmentPIN string) (*purolator.VoidResponse, error) {
		return nil, errors.New("shipment already in transit, cannot cancel")
	}

	client := newTestClient(mockAPI)

	req := &shipper.CancelOrderRequest{
		OrderID: "puro-ship-12345678",
		Reason:  "Customer requested cancellation",
	}

	ctx := context.Background()
	_, err := client.CancelOrder(ctx, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel")
}

func TestClient_Name(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	client := newTestClient(mockAPI)

	assert.Equal(t, "purolator", client.Name())
}

func TestClient_New_WithMock(t *testing.T) {
	logger := otelzap.New(zap.NewNop())
	client := purolator.New(
		purolator.Config{
			UseMock: true,
		},
		logger,
		nil,
	)

	assert.Equal(t, "purolator", client.Name())

	// Test that mock works
	ctx := context.Background()
	resp, err := client.GetQuote(ctx, &shipper.QuoteRequest{
		Origin:      shipper.Address{PostalCode: "M5V1A1"},
		Destination: shipper.Address{PostalCode: "V6B2W2"},
	})

	require.NoError(t, err)
	assert.NotEmpty(t, resp.Rates)
}

func TestClient_ServiceTypeMapping(t *testing.T) {
	mockAPI := purolator.NewMockAPIClient()
	client := newTestClient(mockAPI)

	req := &shipper.QuoteRequest{
		Origin:      shipper.Address{PostalCode: "M5V1A1"},
		Destination: shipper.Address{PostalCode: "V6B2W2"},
	}

	ctx := context.Background()
	resp, err := client.GetQuote(ctx, req)

	require.NoError(t, err)

	// Check service type mappings
	serviceTypes := make(map[string]shipper.ServiceType)
	for _, rate := range resp.Rates {
		serviceTypes[rate.ServiceCode] = rate.ServiceType
	}

	assert.Equal(t, shipper.ServiceStandard, serviceTypes["PurolatorGround"])
	assert.Equal(t, shipper.ServiceExpress, serviceTypes["PurolatorExpress"])
	assert.Equal(t, shipper.ServiceOvernight, serviceTypes["PurolatorExpress9AM"])
}
