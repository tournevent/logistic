package graphql_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tournevent/logistic/internal/graphql"
	"github.com/tournevent/logistic/internal/graphql/generated"
	"github.com/tournevent/logistic/internal/telemetry"
	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/tournevent/logistic/pkg/shipper/mock"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

func newTestResolver() (*graphql.Resolver, *shipper.Registry) {
	registry := shipper.NewRegistry()
	registry.Register(mock.New("freightcom"))
	registry.Register(mock.New("canadapost"))
	registry.Register(mock.New("purolator"))

	logger := otelzap.New(zap.NewNop())
	metrics := telemetry.NewMetrics()

	resolver := graphql.NewResolver(registry, logger, metrics)
	return resolver, registry
}

func TestMutation_DelivroGetQuote_Success(t *testing.T) {
	resolver, _ := newTestResolver()
	mutation := resolver.Mutation()

	input := generated.GetQuoteInput{
		ShipperID: "shipper-123",
		Origin: &generated.AddressInput{
			Name:         "Sender",
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V1A1",
			Phone:        "416-555-1234",
		},
		Destination: &generated.AddressInput{
			Name:         "Receiver",
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B2W2",
			Phone:        "604-555-5678",
		},
		Packages: []*generated.PackageInput{
			{Length: "10", Width: "10", Height: "10", Weight: "5"},
		},
	}

	ctx := context.Background()
	resp, err := mutation.DelivroGetQuote(ctx, input)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.QuoteID)
	assert.NotEmpty(t, resp.Rates)
	assert.NotNil(t, resp.Metadata)
	assert.NotEmpty(t, resp.Metadata.RequestID)
}

func TestMutation_DelivroGetQuote_WithCarrierFilter(t *testing.T) {
	resolver, _ := newTestResolver()
	mutation := resolver.Mutation()

	input := generated.GetQuoteInput{
		ShipperID: "shipper-123",
		Origin: &generated.AddressInput{
			Name:         "Sender",
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V1A1",
			Phone:        "416-555-1234",
		},
		Destination: &generated.AddressInput{
			Name:         "Receiver",
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B2W2",
			Phone:        "604-555-5678",
		},
		Packages: []*generated.PackageInput{
			{Length: "10", Width: "10", Height: "10", Weight: "5"},
		},
		Options: &generated.ShippingOptionsInput{
			Carriers: []generated.Carrier{generated.CarrierFreightcom},
		},
	}

	ctx := context.Background()
	resp, err := mutation.DelivroGetQuote(ctx, input)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	// Should only have rates from freightcom
	for _, rate := range resp.Rates {
		assert.Equal(t, generated.CarrierFreightcom, rate.Carrier)
	}
}

func TestMutation_DelivroGetQuote_EmptyRegistry(t *testing.T) {
	registry := shipper.NewRegistry()
	logger := otelzap.New(zap.NewNop())
	metrics := telemetry.NewMetrics()
	resolver := graphql.NewResolver(registry, logger, metrics)
	mutation := resolver.Mutation()

	input := generated.GetQuoteInput{
		ShipperID: "shipper-123",
		Origin: &generated.AddressInput{
			Name:         "Sender",
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V1A1",
			Phone:        "416-555-1234",
		},
		Destination: &generated.AddressInput{
			Name:         "Receiver",
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B2W2",
			Phone:        "604-555-5678",
		},
		Packages: []*generated.PackageInput{
			{Length: "10", Width: "10", Height: "10", Weight: "5"},
		},
	}

	ctx := context.Background()
	resp, err := mutation.DelivroGetQuote(ctx, input)

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Empty(t, resp.Rates)
	assert.NotEmpty(t, resp.Errors)
}

func TestMutation_DelivroCreateOrder_Success(t *testing.T) {
	resolver, _ := newTestResolver()
	mutation := resolver.Mutation()

	input := generated.CreateOrderInput{
		ShipperID: "shipper-123",
		RateID:    "fc-rate-standard-123", // freightcom prefix
		Sender: &generated.ContactInput{
			Name:  "John Doe",
			Phone: "416-555-1234",
		},
		SenderAddress: &generated.AddressInput{
			Name:         "John Doe",
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V1A1",
			Phone:        "416-555-1234",
		},
		Recipient: &generated.ContactInput{
			Name:  "Jane Smith",
			Phone: "604-555-5678",
		},
		RecipientAddress: &generated.AddressInput{
			Name:         "Jane Smith",
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B2W2",
			Phone:        "604-555-5678",
		},
		Packages: []*generated.PackageInput{
			{Length: "10", Width: "10", Height: "10", Weight: "5"},
		},
	}

	ctx := context.Background()
	resp, err := mutation.DelivroCreateOrder(ctx, input)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.OrderID)
	assert.NotNil(t, resp.TrackingNumber)
	assert.NotNil(t, resp.Status)
	assert.Equal(t, generated.ShipmentStatusConfirmed, *resp.Status)
}

func TestMutation_DelivroCreateOrder_CanadaPost(t *testing.T) {
	resolver, _ := newTestResolver()
	mutation := resolver.Mutation()

	input := generated.CreateOrderInput{
		ShipperID: "shipper-123",
		RateID:    "cp-DOM.RP-123", // canadapost prefix
		Sender: &generated.ContactInput{
			Name:  "John Doe",
			Phone: "416-555-1234",
		},
		SenderAddress: &generated.AddressInput{
			Name:         "John Doe",
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V1A1",
			Phone:        "416-555-1234",
		},
		Recipient: &generated.ContactInput{
			Name:  "Jane Smith",
			Phone: "604-555-5678",
		},
		RecipientAddress: &generated.AddressInput{
			Name:         "Jane Smith",
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B2W2",
			Phone:        "604-555-5678",
		},
		Packages: []*generated.PackageInput{
			{Length: "10", Width: "10", Height: "10", Weight: "5"},
		},
	}

	ctx := context.Background()
	resp, err := mutation.DelivroCreateOrder(ctx, input)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, generated.CarrierCanadaPost, *resp.Carrier)
}

func TestMutation_DelivroCreateOrder_Purolator(t *testing.T) {
	resolver, _ := newTestResolver()
	mutation := resolver.Mutation()

	input := generated.CreateOrderInput{
		ShipperID: "shipper-123",
		RateID:    "puro-PurolatorGround-123", // purolator prefix
		Sender: &generated.ContactInput{
			Name:  "John Doe",
			Phone: "416-555-1234",
		},
		SenderAddress: &generated.AddressInput{
			Name:         "John Doe",
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V1A1",
			Phone:        "416-555-1234",
		},
		Recipient: &generated.ContactInput{
			Name:  "Jane Smith",
			Phone: "604-555-5678",
		},
		RecipientAddress: &generated.AddressInput{
			Name:         "Jane Smith",
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B2W2",
			Phone:        "604-555-5678",
		},
		Packages: []*generated.PackageInput{
			{Length: "10", Width: "10", Height: "10", Weight: "5"},
		},
	}

	ctx := context.Background()
	resp, err := mutation.DelivroCreateOrder(ctx, input)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, generated.CarrierPurolator, *resp.Carrier)
}

func TestMutation_DelivroCreateOrder_CarrierNotFound(t *testing.T) {
	resolver, _ := newTestResolver()
	mutation := resolver.Mutation()

	input := generated.CreateOrderInput{
		ShipperID: "shipper-123",
		RateID:    "unknown-rate-123", // unknown carrier prefix
		Sender: &generated.ContactInput{
			Name:  "John Doe",
			Phone: "416-555-1234",
		},
		SenderAddress: &generated.AddressInput{
			Name:         "John Doe",
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V1A1",
			Phone:        "416-555-1234",
		},
		Recipient: &generated.ContactInput{
			Name:  "Jane Smith",
			Phone: "604-555-5678",
		},
		RecipientAddress: &generated.AddressInput{
			Name:         "Jane Smith",
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B2W2",
			Phone:        "604-555-5678",
		},
		Packages: []*generated.PackageInput{
			{Length: "10", Width: "10", Height: "10", Weight: "5"},
		},
	}

	ctx := context.Background()
	resp, err := mutation.DelivroCreateOrder(ctx, input)

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotEmpty(t, resp.Errors)
	assert.Equal(t, "CARRIER_NOT_FOUND", resp.Errors[0].Code)
}

func TestMutation_DelivroGetLabel_Success(t *testing.T) {
	resolver, _ := newTestResolver()
	mutation := resolver.Mutation()

	input := generated.GetLabelInput{
		OrderID: "fc-order-123", // freightcom prefix
	}

	ctx := context.Background()
	resp, err := mutation.DelivroGetLabel(ctx, input)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.OrderID)
	assert.NotNil(t, resp.Label)
}

func TestMutation_DelivroGetLabel_WithFormat(t *testing.T) {
	resolver, _ := newTestResolver()
	mutation := resolver.Mutation()

	format := generated.LabelFormatZpl
	input := generated.GetLabelInput{
		OrderID: "fc-order-123",
		Format:  &format,
	}

	ctx := context.Background()
	resp, err := mutation.DelivroGetLabel(ctx, input)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Label)
}

func TestMutation_DelivroGetLabel_CarrierNotFound(t *testing.T) {
	resolver, _ := newTestResolver()
	mutation := resolver.Mutation()

	input := generated.GetLabelInput{
		OrderID: "unknown-order-123", // unknown carrier prefix
	}

	ctx := context.Background()
	resp, err := mutation.DelivroGetLabel(ctx, input)

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotEmpty(t, resp.Errors)
	assert.Equal(t, "CARRIER_NOT_FOUND", resp.Errors[0].Code)
}

func TestMutation_DelivroCancelOrder_Success(t *testing.T) {
	resolver, _ := newTestResolver()
	mutation := resolver.Mutation()

	reason := "Customer requested cancellation"
	input := generated.CancelOrderInput{
		OrderID: "fc-order-123", // freightcom prefix
		Reason:  &reason,
	}

	ctx := context.Background()
	resp, err := mutation.DelivroCancelOrder(ctx, input)

	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.OrderID)
	assert.NotNil(t, resp.Status)
	assert.Equal(t, generated.ShipmentStatusCancelled, *resp.Status)
	assert.NotNil(t, resp.ConfirmationNumber)
}

func TestMutation_DelivroCancelOrder_CarrierNotFound(t *testing.T) {
	resolver, _ := newTestResolver()
	mutation := resolver.Mutation()

	input := generated.CancelOrderInput{
		OrderID: "unknown-order-123", // unknown carrier prefix
	}

	ctx := context.Background()
	resp, err := mutation.DelivroCancelOrder(ctx, input)

	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.NotEmpty(t, resp.Errors)
	assert.Equal(t, "CARRIER_NOT_FOUND", resp.Errors[0].Code)
}

func TestQuery_Health(t *testing.T) {
	resolver, _ := newTestResolver()
	query := resolver.Query()

	ctx := context.Background()
	healthy, err := query.Health(ctx)

	require.NoError(t, err)
	assert.True(t, healthy)
}

func TestQuery_Carriers(t *testing.T) {
	resolver, _ := newTestResolver()
	query := resolver.Query()

	ctx := context.Background()
	carriers, err := query.Carriers(ctx)

	require.NoError(t, err)
	assert.Len(t, carriers, 3)
	assert.Contains(t, carriers, generated.CarrierFreightcom)
	assert.Contains(t, carriers, generated.CarrierCanadaPost)
	assert.Contains(t, carriers, generated.CarrierPurolator)
}

func TestQuery_ServiceTypes(t *testing.T) {
	resolver, _ := newTestResolver()
	query := resolver.Query()

	ctx := context.Background()
	serviceTypes, err := query.ServiceTypes(ctx)

	require.NoError(t, err)
	assert.Len(t, serviceTypes, 6)
	assert.Contains(t, serviceTypes, generated.ServiceTypeStandard)
	assert.Contains(t, serviceTypes, generated.ServiceTypeExpress)
	assert.Contains(t, serviceTypes, generated.ServiceTypePriority)
	assert.Contains(t, serviceTypes, generated.ServiceTypeOvernight)
	assert.Contains(t, serviceTypes, generated.ServiceTypeEconomy)
	assert.Contains(t, serviceTypes, generated.ServiceTypeFreight)
}

func TestResolver_NewResolver(t *testing.T) {
	registry := shipper.NewRegistry()
	logger := otelzap.New(zap.NewNop())
	metrics := telemetry.NewMetrics()

	resolver := graphql.NewResolver(registry, logger, metrics)

	assert.NotNil(t, resolver)
	assert.Equal(t, registry, resolver.Registry)
	assert.Equal(t, logger, resolver.Logger)
	assert.Equal(t, metrics, resolver.Metrics)
}
