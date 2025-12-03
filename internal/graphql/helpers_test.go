package graphql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tournevent/logistic/internal/graphql/generated"
	"github.com/tournevent/logistic/pkg/shipper"
)

func TestAddressInputToModel(t *testing.T) {
	company := "ACME Corp"
	line2 := "Suite 100"
	countryCode := "CA"
	email := "test@example.com"
	instructions := "Leave at door"
	isResidential := true

	input := &generated.AddressInput{
		Name:          "John Doe",
		Company:       &company,
		Line1:         "123 Main St",
		Line2:         &line2,
		City:          "Toronto",
		ProvinceCode:  "ON",
		PostalCode:    "M5V1A1",
		CountryCode:   &countryCode,
		Phone:         "416-555-1234",
		Email:         &email,
		Instructions:  &instructions,
		IsResidential: &isResidential,
	}

	result := addressInputToModel(input)

	assert.Equal(t, "John Doe", result.Name)
	assert.Equal(t, "ACME Corp", result.Company)
	assert.Equal(t, "123 Main St", result.Line1)
	assert.Equal(t, "Suite 100", result.Line2)
	assert.Equal(t, "Toronto", result.City)
	assert.Equal(t, "ON", result.ProvinceCode)
	assert.Equal(t, "M5V1A1", result.PostalCode)
	assert.Equal(t, "CA", result.CountryCode)
	assert.Equal(t, "416-555-1234", result.Phone)
	assert.Equal(t, "test@example.com", result.Email)
	assert.Equal(t, "Leave at door", result.Instructions)
	assert.True(t, result.IsResidential)
}

func TestAddressInputToModel_Nil(t *testing.T) {
	result := addressInputToModel(nil)
	assert.Equal(t, shipper.Address{}, result)
}

func TestAddressInputToModel_DefaultCountry(t *testing.T) {
	input := &generated.AddressInput{
		Name:         "John Doe",
		Line1:        "123 Main St",
		City:         "Toronto",
		ProvinceCode: "ON",
		PostalCode:   "M5V1A1",
		Phone:        "416-555-1234",
	}

	result := addressInputToModel(input)
	assert.Equal(t, "CA", result.CountryCode)
}

func TestContactInputToModel(t *testing.T) {
	company := "ACME Corp"
	email := "test@example.com"
	taxID := "123456789"

	input := &generated.ContactInput{
		Name:    "John Doe",
		Company: &company,
		Phone:   "416-555-1234",
		Email:   &email,
		TaxID:   &taxID,
	}

	result := contactInputToModel(input)

	assert.Equal(t, "John Doe", result.Name)
	assert.Equal(t, "ACME Corp", result.Company)
	assert.Equal(t, "416-555-1234", result.Phone)
	assert.Equal(t, "test@example.com", result.Email)
	assert.Equal(t, "123456789", result.TaxID)
}

func TestContactInputToModel_Nil(t *testing.T) {
	result := contactInputToModel(nil)
	assert.Equal(t, shipper.Contact{}, result)
}

func TestPackagesInputToModel(t *testing.T) {
	dimUnit := generated.DimensionUnitIn
	weightUnit := generated.WeightUnitLb
	pkgType := generated.PackageTypeEnvelope
	description := "Documents"
	declaredValue := "100.00"
	currency := "USD"

	inputs := []*generated.PackageInput{
		{
			Length:        "10",
			Width:         "8",
			Height:        "2",
			DimensionUnit: &dimUnit,
			Weight:        "1.5",
			WeightUnit:    &weightUnit,
			PackageType:   &pkgType,
			Description:   &description,
			DeclaredValue: &declaredValue,
			Currency:      &currency,
		},
	}

	result := packagesInputToModel(inputs)

	assert.Len(t, result, 1)
	assert.Equal(t, float64(10), result[0].Length)
	assert.Equal(t, float64(8), result[0].Width)
	assert.Equal(t, float64(2), result[0].Height)
	assert.Equal(t, shipper.DimensionIN, result[0].DimensionUnit)
	assert.Equal(t, 1.5, result[0].Weight)
	assert.Equal(t, shipper.WeightLB, result[0].WeightUnit)
	assert.Equal(t, shipper.PackageEnvelope, result[0].PackageType)
	assert.Equal(t, "Documents", result[0].Description)
	assert.Equal(t, float64(100), result[0].DeclaredValue)
	assert.Equal(t, "USD", result[0].Currency)
}

func TestPackagesInputToModel_Defaults(t *testing.T) {
	inputs := []*generated.PackageInput{
		{
			Length: "10",
			Width:  "10",
			Height: "10",
			Weight: "5",
		},
	}

	result := packagesInputToModel(inputs)

	assert.Len(t, result, 1)
	assert.Equal(t, shipper.DimensionCM, result[0].DimensionUnit)
	assert.Equal(t, shipper.WeightKG, result[0].WeightUnit)
	assert.Equal(t, shipper.PackageBox, result[0].PackageType)
	assert.Equal(t, "CAD", result[0].Currency)
}

func TestOptionsInputToModel(t *testing.T) {
	signatureRequired := true
	insuranceRequired := true
	saturdayDelivery := true
	shipDate := time.Now().Add(24 * time.Hour)

	input := &generated.ShippingOptionsInput{
		Carriers:          []generated.Carrier{generated.CarrierFreightcom, generated.CarrierCanadaPost},
		ServiceTypes:      []generated.ServiceType{generated.ServiceTypeExpress, generated.ServiceTypePriority},
		SignatureRequired: &signatureRequired,
		InsuranceRequired: &insuranceRequired,
		SaturdayDelivery:  &saturdayDelivery,
		ShipDate:          &shipDate,
	}

	result := optionsInputToModel(input)

	assert.Len(t, result.Carriers, 2)
	assert.Contains(t, result.Carriers, "freightcom")
	assert.Contains(t, result.Carriers, "canadapost")
	assert.Len(t, result.ServiceTypes, 2)
	assert.Contains(t, result.ServiceTypes, shipper.ServiceExpress)
	assert.Contains(t, result.ServiceTypes, shipper.ServicePriority)
	assert.True(t, result.SignatureRequired)
	assert.True(t, result.InsuranceRequired)
	assert.True(t, result.SaturdayDelivery)
	assert.Equal(t, &shipDate, result.ShipDate)
}

func TestCarrierEnumToName(t *testing.T) {
	tests := []struct {
		input    generated.Carrier
		expected string
	}{
		{generated.CarrierFreightcom, "freightcom"},
		{generated.CarrierCanadaPost, "canadapost"},
		{generated.CarrierPurolator, "purolator"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := carrierEnumToName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCarrierNameToEnum(t *testing.T) {
	tests := []struct {
		input    string
		expected *generated.Carrier
	}{
		{"freightcom", ptr(generated.CarrierFreightcom)},
		{"canadapost", ptr(generated.CarrierCanadaPost)},
		{"purolator", ptr(generated.CarrierPurolator)},
		{"unknown", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := carrierNameToEnum(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestCarrierFromRateID(t *testing.T) {
	tests := []struct {
		rateID   string
		expected string
	}{
		{"fc-rate-standard-123", "freightcom"},
		{"cp-DOM.RP-20231215", "canadapost"},
		{"puro-PurolatorGround-123", "purolator"},
		{"unknown-rate-123", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.rateID, func(t *testing.T) {
			result := carrierFromRateID(tt.rateID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCarrierFromOrderID(t *testing.T) {
	tests := []struct {
		orderID  string
		expected string
	}{
		{"fc-order-123", "freightcom"},
		{"cp-ship-12345678", "canadapost"},
		{"puro-ship-12345678", "purolator"},
		{"unknown-order-123", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.orderID, func(t *testing.T) {
			result := carrierFromOrderID(tt.orderID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatusToEnum(t *testing.T) {
	tests := []struct {
		input    shipper.ShipmentStatus
		expected generated.ShipmentStatus
	}{
		{shipper.StatusPending, generated.ShipmentStatusPending},
		{shipper.StatusQuoted, generated.ShipmentStatusQuoted},
		{shipper.StatusConfirmed, generated.ShipmentStatusConfirmed},
		{shipper.StatusAssigned, generated.ShipmentStatusAssigned},
		{shipper.StatusPickedUp, generated.ShipmentStatusPickedUp},
		{shipper.StatusInTransit, generated.ShipmentStatusInTransit},
		{shipper.StatusOutForDelivery, generated.ShipmentStatusOutForDelivery},
		{shipper.StatusDelivered, generated.ShipmentStatusDelivered},
		{shipper.StatusCancelled, generated.ShipmentStatusCancelled},
		{shipper.StatusException, generated.ShipmentStatusException},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := statusToEnum(tt.input)
			assert.Equal(t, tt.expected, *result)
		})
	}
}

func TestServiceTypeToModel(t *testing.T) {
	tests := []struct {
		input    generated.ServiceType
		expected shipper.ServiceType
	}{
		{generated.ServiceTypeStandard, shipper.ServiceStandard},
		{generated.ServiceTypeExpress, shipper.ServiceExpress},
		{generated.ServiceTypePriority, shipper.ServicePriority},
		{generated.ServiceTypeOvernight, shipper.ServiceOvernight},
		{generated.ServiceTypeEconomy, shipper.ServiceEconomy},
		{generated.ServiceTypeFreight, shipper.ServiceFreight},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := serviceTypeToModel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServiceTypeToEnum(t *testing.T) {
	tests := []struct {
		input    shipper.ServiceType
		expected generated.ServiceType
	}{
		{shipper.ServiceStandard, generated.ServiceTypeStandard},
		{shipper.ServiceExpress, generated.ServiceTypeExpress},
		{shipper.ServicePriority, generated.ServiceTypePriority},
		{shipper.ServiceOvernight, generated.ServiceTypeOvernight},
		{shipper.ServiceEconomy, generated.ServiceTypeEconomy},
		{shipper.ServiceFreight, generated.ServiceTypeFreight},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := serviceTypeToEnum(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLabelFormatToModel(t *testing.T) {
	tests := []struct {
		input    generated.LabelFormat
		expected shipper.LabelFormat
	}{
		{generated.LabelFormatPDF, shipper.LabelPDF},
		{generated.LabelFormatPng, shipper.LabelPNG},
		{generated.LabelFormatZpl, shipper.LabelZPL},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := labelFormatToModel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLabelFormatToEnum(t *testing.T) {
	tests := []struct {
		input    shipper.LabelFormat
		expected generated.LabelFormat
	}{
		{shipper.LabelPDF, generated.LabelFormatPDF},
		{shipper.LabelPNG, generated.LabelFormatPng},
		{shipper.LabelZPL, generated.LabelFormatZpl},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := labelFormatToEnum(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMoneyToGraphQL(t *testing.T) {
	money := &shipper.Money{Amount: 25.99, Currency: "CAD"}
	result := moneyToGraphQL(money)

	assert.Equal(t, "25.99", result.Amount)
	assert.Equal(t, "CAD", result.Currency)
}

func TestMoneyToGraphQL_Nil(t *testing.T) {
	result := moneyToGraphQL(nil)
	assert.Nil(t, result)
}

func TestRateToGraphQL(t *testing.T) {
	estimatedDelivery := time.Now().Add(48 * time.Hour)
	expiresAt := time.Now().Add(30 * time.Minute)
	transitDays := 2
	signatureRequired := false
	guaranteed := true

	rate := &shipper.RateOption{
		RateID:            "fc-rate-express-123",
		Carrier:           "freightcom",
		ServiceCode:       "EXPRESS",
		ServiceName:       "Express Shipping",
		ServiceType:       shipper.ServiceExpress,
		BaseRate:          shipper.Money{Amount: 20.00, Currency: "CAD"},
		FuelSurcharge:     shipper.Money{Amount: 2.50, Currency: "CAD"},
		Taxes:             shipper.Money{Amount: 2.93, Currency: "CAD"},
		TotalPrice:        shipper.Money{Amount: 25.43, Currency: "CAD"},
		TransitDays:       transitDays,
		EstimatedDelivery: &estimatedDelivery,
		ExpiresAt:         expiresAt,
		SignatureRequired: signatureRequired,
		Guaranteed:        guaranteed,
	}

	result := rateToGraphQL(rate)

	assert.Equal(t, "fc-rate-express-123", result.RateID)
	assert.Equal(t, generated.CarrierFreightcom, result.Carrier)
	assert.Equal(t, "EXPRESS", result.ServiceCode)
	assert.Equal(t, "Express Shipping", result.ServiceName)
	assert.Equal(t, generated.ServiceTypeExpress, result.ServiceType)
	assert.Equal(t, "20.00", result.BaseRate.Amount)
	assert.Equal(t, "2.50", result.FuelSurcharge.Amount)
	assert.Equal(t, "2.93", result.Taxes.Amount)
	assert.Equal(t, "25.43", result.TotalPrice.Amount)
	assert.Equal(t, &transitDays, result.TransitDays)
	assert.Equal(t, &estimatedDelivery, result.EstimatedDelivery)
	assert.Equal(t, expiresAt, result.ExpiresAt)
	assert.Equal(t, &signatureRequired, result.SignatureRequired)
	assert.Equal(t, &guaranteed, result.Guaranteed)
}

func TestLabelToGraphQL(t *testing.T) {
	expiresAt := time.Now().Add(24 * time.Hour)
	data := "base64encodeddata"
	url := "https://example.com/label.pdf"

	label := &shipper.Label{
		Format:    shipper.LabelPDF,
		Data:      data,
		URL:       url,
		ExpiresAt: &expiresAt,
	}

	result := labelToGraphQL(label)

	assert.Equal(t, generated.LabelFormatPDF, result.Format)
	assert.Equal(t, &data, result.Data)
	assert.Equal(t, &url, result.URL)
	assert.Equal(t, &expiresAt, result.ExpiresAt)
}

func TestLabelToGraphQL_Nil(t *testing.T) {
	result := labelToGraphQL(nil)
	assert.Nil(t, result)
}

func TestErrorsToGraphQL(t *testing.T) {
	errs := []error{
		shipper.ErrInvalidAddress,
		shipper.ErrServiceUnavailable,
	}

	result := errorsToGraphQL(errs)

	assert.Len(t, result, 2)
	assert.Equal(t, "CARRIER_ERROR", result[0].Code)
	assert.Equal(t, "CARRIER_ERROR", result[1].Code)
}

func TestErrorsToGraphQL_Empty(t *testing.T) {
	result := errorsToGraphQL(nil)
	assert.Nil(t, result)
}

func TestParseDecimal(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"10", 10.0},
		{"10.5", 10.5},
		{"0.99", 0.99},
		{"100.00", 100.0},
		{"", 0.0},
		{"invalid", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseDecimal(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
