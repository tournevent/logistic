package shipper_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/tournevent/logistic/pkg/shipper/mock"
)

func TestRegistry_Register(t *testing.T) {
	registry := shipper.NewRegistry()

	mockShipper := mock.New("test-shipper")
	registry.Register(mockShipper)

	got, err := registry.Get("test-shipper")
	require.NoError(t, err, "shipper should be registered")
	assert.Equal(t, "test-shipper", got.Name())
}

func TestRegistry_Register_Override(t *testing.T) {
	registry := shipper.NewRegistry()

	// Register first shipper
	registry.Register(mock.New("test-shipper"))
	assert.Equal(t, 1, registry.Count())

	// Register again with same name should override
	registry.Register(mock.New("test-shipper"))
	assert.Equal(t, 1, registry.Count())
}

func TestRegistry_Get_NotFound(t *testing.T) {
	registry := shipper.NewRegistry()

	_, err := registry.Get("nonexistent")
	assert.Error(t, err, "should return error for unregistered shipper")
	assert.True(t, errors.Is(err, shipper.ErrCarrierNotFound))
}

func TestRegistry_All(t *testing.T) {
	registry := shipper.NewRegistry()

	registry.Register(mock.New("shipper-a"))
	registry.Register(mock.New("shipper-b"))
	registry.Register(mock.New("shipper-c"))

	all := registry.All()
	assert.Len(t, all, 3)
}

func TestRegistry_Names(t *testing.T) {
	registry := shipper.NewRegistry()

	registry.Register(mock.New("freightcom"))
	registry.Register(mock.New("canadapost"))
	registry.Register(mock.New("purolator"))

	names := registry.Names()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "freightcom")
	assert.Contains(t, names, "canadapost")
	assert.Contains(t, names, "purolator")
}

func TestRegistry_Count(t *testing.T) {
	registry := shipper.NewRegistry()
	assert.Equal(t, 0, registry.Count())

	registry.Register(mock.New("shipper-a"))
	assert.Equal(t, 1, registry.Count())

	registry.Register(mock.New("shipper-b"))
	assert.Equal(t, 2, registry.Count())
}

func TestRegistry_GetAllQuotes(t *testing.T) {
	registry := shipper.NewRegistry()

	registry.Register(mock.New("freightcom"))
	registry.Register(mock.New("canadapost"))

	req := &shipper.QuoteRequest{
		Origin: shipper.Address{
			Name:         "Sender",
			Line1:        "123 Main St",
			City:         "Toronto",
			ProvinceCode: "ON",
			PostalCode:   "M5V 1A1",
			CountryCode:  "CA",
			Phone:        "416-555-1234",
		},
		Destination: shipper.Address{
			Name:         "Receiver",
			Line1:        "456 Oak Ave",
			City:         "Vancouver",
			ProvinceCode: "BC",
			PostalCode:   "V6B 2W2",
			CountryCode:  "CA",
			Phone:        "604-555-5678",
		},
		Packages: []shipper.Package{
			{
				Length: 10,
				Width:  10,
				Height: 10,
				Weight: 5,
			},
		},
	}

	ctx := context.Background()
	results, errs := registry.GetAllQuotes(ctx, req)

	assert.Empty(t, errs, "should have no errors from mock shippers")
	assert.Len(t, results, 2, "should have results from both shippers")

	for _, result := range results {
		assert.NotEmpty(t, result.QuoteID)
		assert.NotEmpty(t, result.Rates)
	}
}

func TestRegistry_GetAllQuotes_Empty(t *testing.T) {
	registry := shipper.NewRegistry()

	req := &shipper.QuoteRequest{
		Origin: shipper.Address{
			Name: "Test",
		},
	}

	ctx := context.Background()
	results, errs := registry.GetAllQuotes(ctx, req)

	assert.Empty(t, results, "should return empty results for empty registry")
	assert.NotEmpty(t, errs, "should return error for empty registry")
}

func TestRegistry_GetQuotesFromCarriers_Success(t *testing.T) {
	registry := shipper.NewRegistry()

	registry.Register(mock.New("freightcom"))
	registry.Register(mock.New("canadapost"))
	registry.Register(mock.New("purolator"))

	req := &shipper.QuoteRequest{
		Origin:      shipper.Address{PostalCode: "M5V 1A1"},
		Destination: shipper.Address{PostalCode: "V6B 2W2"},
		Packages:    []shipper.Package{{Weight: 5}},
	}

	ctx := context.Background()
	// Only request quotes from 2 carriers
	results, errs := registry.GetQuotesFromCarriers(ctx, req, []string{"freightcom", "purolator"})

	assert.Empty(t, errs)
	assert.Len(t, results, 2)
}

func TestRegistry_GetQuotesFromCarriers_EmptyCarriers(t *testing.T) {
	registry := shipper.NewRegistry()

	registry.Register(mock.New("freightcom"))
	registry.Register(mock.New("canadapost"))

	req := &shipper.QuoteRequest{
		Origin:      shipper.Address{PostalCode: "M5V 1A1"},
		Destination: shipper.Address{PostalCode: "V6B 2W2"},
	}

	ctx := context.Background()
	// Empty carriers list should get all quotes
	results, errs := registry.GetQuotesFromCarriers(ctx, req, []string{})

	assert.Empty(t, errs)
	assert.Len(t, results, 2, "should get quotes from all carriers when empty list")
}

func TestRegistry_GetQuotesFromCarriers_NotFound(t *testing.T) {
	registry := shipper.NewRegistry()

	registry.Register(mock.New("freightcom"))

	req := &shipper.QuoteRequest{
		Origin:      shipper.Address{PostalCode: "M5V 1A1"},
		Destination: shipper.Address{PostalCode: "V6B 2W2"},
	}

	ctx := context.Background()
	results, errs := registry.GetQuotesFromCarriers(ctx, req, []string{"nonexistent"})

	assert.Len(t, results, 0)
	assert.Len(t, errs, 1)
	assert.True(t, errors.Is(errs[0], shipper.ErrCarrierNotFound))
}
