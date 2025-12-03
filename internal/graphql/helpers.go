package graphql

import (
	"fmt"
	"strings"

	"github.com/tournevent/logistic/internal/graphql/generated"
	"github.com/tournevent/logistic/pkg/shipper"
)

func addressInputToModel(input *generated.AddressInput) shipper.Address {
	if input == nil {
		return shipper.Address{}
	}
	addr := shipper.Address{
		Name:         input.Name,
		Line1:        input.Line1,
		City:         input.City,
		ProvinceCode: input.ProvinceCode,
		PostalCode:   input.PostalCode,
		Phone:        input.Phone,
	}
	if input.Company != nil {
		addr.Company = *input.Company
	}
	if input.Line2 != nil {
		addr.Line2 = *input.Line2
	}
	if input.CountryCode != nil {
		addr.CountryCode = *input.CountryCode
	} else {
		addr.CountryCode = "CA"
	}
	if input.Email != nil {
		addr.Email = *input.Email
	}
	if input.Instructions != nil {
		addr.Instructions = *input.Instructions
	}
	if input.IsResidential != nil {
		addr.IsResidential = *input.IsResidential
	}
	return addr
}

func contactInputToModel(input *generated.ContactInput) shipper.Contact {
	if input == nil {
		return shipper.Contact{}
	}
	contact := shipper.Contact{
		Name:  input.Name,
		Phone: input.Phone,
	}
	if input.Company != nil {
		contact.Company = *input.Company
	}
	if input.Email != nil {
		contact.Email = *input.Email
	}
	if input.TaxID != nil {
		contact.TaxID = *input.TaxID
	}
	return contact
}

func packagesInputToModel(inputs []*generated.PackageInput) []shipper.Package {
	packages := make([]shipper.Package, len(inputs))
	for i, input := range inputs {
		pkg := shipper.Package{
			Length: parseDecimal(input.Length),
			Width:  parseDecimal(input.Width),
			Height: parseDecimal(input.Height),
			Weight: parseDecimal(input.Weight),
		}
		if input.DimensionUnit != nil {
			pkg.DimensionUnit = dimensionUnitToModel(*input.DimensionUnit)
		} else {
			pkg.DimensionUnit = shipper.DimensionCM
		}
		if input.WeightUnit != nil {
			pkg.WeightUnit = weightUnitToModel(*input.WeightUnit)
		} else {
			pkg.WeightUnit = shipper.WeightKG
		}
		if input.PackageType != nil {
			pkg.PackageType = packageTypeToModel(*input.PackageType)
		} else {
			pkg.PackageType = shipper.PackageBox
		}
		if input.Description != nil {
			pkg.Description = *input.Description
		}
		if input.DeclaredValue != nil {
			pkg.DeclaredValue = parseDecimal(*input.DeclaredValue)
		}
		if input.Currency != nil {
			pkg.Currency = *input.Currency
		} else {
			pkg.Currency = "CAD"
		}
		packages[i] = pkg
	}
	return packages
}

func optionsInputToModel(input *generated.ShippingOptionsInput) shipper.ShippingOptions {
	opts := shipper.ShippingOptions{}
	if input.Carriers != nil {
		opts.Carriers = make([]string, len(input.Carriers))
		for i, c := range input.Carriers {
			opts.Carriers[i] = carrierEnumToName(c)
		}
	}
	if input.ServiceTypes != nil {
		opts.ServiceTypes = make([]shipper.ServiceType, len(input.ServiceTypes))
		for i, st := range input.ServiceTypes {
			opts.ServiceTypes[i] = serviceTypeToModel(st)
		}
	}
	if input.SignatureRequired != nil {
		opts.SignatureRequired = *input.SignatureRequired
	}
	if input.InsuranceRequired != nil {
		opts.InsuranceRequired = *input.InsuranceRequired
	}
	if input.SaturdayDelivery != nil {
		opts.SaturdayDelivery = *input.SaturdayDelivery
	}
	if input.ShipDate != nil {
		opts.ShipDate = input.ShipDate
	}
	return opts
}

func rateToGraphQL(rate *shipper.RateOption) *generated.RateOption {
	carrier := carrierNameToEnumValue(rate.Carrier)
	return &generated.RateOption{
		RateID:            rate.RateID,
		Carrier:           carrier,
		ServiceCode:       rate.ServiceCode,
		ServiceName:       rate.ServiceName,
		ServiceType:       serviceTypeToEnum(rate.ServiceType),
		BaseRate:          moneyToGraphQL(&rate.BaseRate),
		FuelSurcharge:     moneyToGraphQL(&rate.FuelSurcharge),
		Taxes:             moneyToGraphQL(&rate.Taxes),
		TotalPrice:        moneyToGraphQL(&rate.TotalPrice),
		TransitDays:       &rate.TransitDays,
		EstimatedDelivery: rate.EstimatedDelivery,
		ExpiresAt:         rate.ExpiresAt,
		SignatureRequired: &rate.SignatureRequired,
		Guaranteed:        &rate.Guaranteed,
	}
}

func moneyToGraphQL(m *shipper.Money) *generated.Money {
	if m == nil {
		return nil
	}
	return &generated.Money{
		Amount:   fmt.Sprintf("%.2f", m.Amount),
		Currency: m.Currency,
	}
}

func labelToGraphQL(l *shipper.Label) *generated.Label {
	if l == nil {
		return nil
	}
	format := labelFormatToEnum(l.Format)
	return &generated.Label{
		Format:    format,
		Data:      &l.Data,
		URL:       &l.URL,
		ExpiresAt: l.ExpiresAt,
	}
}

func errorsToGraphQL(errs []error) []*generated.Error {
	if len(errs) == 0 {
		return nil
	}
	result := make([]*generated.Error, len(errs))
	for i, err := range errs {
		result[i] = &generated.Error{
			Code:    "CARRIER_ERROR",
			Message: err.Error(),
		}
	}
	return result
}

func carrierEnumToName(c generated.Carrier) string {
	switch c {
	case generated.CarrierFreightcom:
		return "freightcom"
	case generated.CarrierCanadaPost:
		return "canadapost"
	case generated.CarrierPurolator:
		return "purolator"
	default:
		return strings.ToLower(string(c))
	}
}

func carrierNameToEnum(name string) *generated.Carrier {
	switch name {
	case "freightcom":
		c := generated.CarrierFreightcom
		return &c
	case "canadapost":
		c := generated.CarrierCanadaPost
		return &c
	case "purolator":
		c := generated.CarrierPurolator
		return &c
	default:
		return nil
	}
}

func carrierNameToEnumValue(name string) generated.Carrier {
	switch name {
	case "freightcom":
		return generated.CarrierFreightcom
	case "canadapost":
		return generated.CarrierCanadaPost
	case "purolator":
		return generated.CarrierPurolator
	default:
		return generated.CarrierFreightcom // default fallback
	}
}

func carrierFromRateID(rateID string) string {
	if strings.HasPrefix(rateID, "fc-") {
		return "freightcom"
	}
	if strings.HasPrefix(rateID, "cp-") {
		return "canadapost"
	}
	if strings.HasPrefix(rateID, "puro-") {
		return "purolator"
	}
	return "unknown"
}

func carrierFromOrderID(orderID string) string {
	if strings.HasPrefix(orderID, "fc-") {
		return "freightcom"
	}
	if strings.HasPrefix(orderID, "cp-") {
		return "canadapost"
	}
	if strings.HasPrefix(orderID, "puro-") {
		return "purolator"
	}
	return "unknown"
}

func statusToEnum(s shipper.ShipmentStatus) *generated.ShipmentStatus {
	var status generated.ShipmentStatus
	switch s {
	case shipper.StatusPending:
		status = generated.ShipmentStatusPending
	case shipper.StatusQuoted:
		status = generated.ShipmentStatusQuoted
	case shipper.StatusConfirmed:
		status = generated.ShipmentStatusConfirmed
	case shipper.StatusAssigned:
		status = generated.ShipmentStatusAssigned
	case shipper.StatusPickedUp:
		status = generated.ShipmentStatusPickedUp
	case shipper.StatusInTransit:
		status = generated.ShipmentStatusInTransit
	case shipper.StatusOutForDelivery:
		status = generated.ShipmentStatusOutForDelivery
	case shipper.StatusDelivered:
		status = generated.ShipmentStatusDelivered
	case shipper.StatusCancelled:
		status = generated.ShipmentStatusCancelled
	case shipper.StatusException:
		status = generated.ShipmentStatusException
	default:
		status = generated.ShipmentStatusPending
	}
	return &status
}

func serviceTypeToModel(st generated.ServiceType) shipper.ServiceType {
	switch st {
	case generated.ServiceTypeStandard:
		return shipper.ServiceStandard
	case generated.ServiceTypeExpress:
		return shipper.ServiceExpress
	case generated.ServiceTypePriority:
		return shipper.ServicePriority
	case generated.ServiceTypeOvernight:
		return shipper.ServiceOvernight
	case generated.ServiceTypeEconomy:
		return shipper.ServiceEconomy
	case generated.ServiceTypeFreight:
		return shipper.ServiceFreight
	default:
		return shipper.ServiceStandard
	}
}

func serviceTypeToEnum(st shipper.ServiceType) generated.ServiceType {
	switch st {
	case shipper.ServiceStandard:
		return generated.ServiceTypeStandard
	case shipper.ServiceExpress:
		return generated.ServiceTypeExpress
	case shipper.ServicePriority:
		return generated.ServiceTypePriority
	case shipper.ServiceOvernight:
		return generated.ServiceTypeOvernight
	case shipper.ServiceEconomy:
		return generated.ServiceTypeEconomy
	case shipper.ServiceFreight:
		return generated.ServiceTypeFreight
	default:
		return generated.ServiceTypeStandard
	}
}

func dimensionUnitToModel(du generated.DimensionUnit) shipper.DimensionUnit {
	switch du {
	case generated.DimensionUnitCm:
		return shipper.DimensionCM
	case generated.DimensionUnitIn:
		return shipper.DimensionIN
	default:
		return shipper.DimensionCM
	}
}

func weightUnitToModel(wu generated.WeightUnit) shipper.WeightUnit {
	switch wu {
	case generated.WeightUnitKg:
		return shipper.WeightKG
	case generated.WeightUnitLb:
		return shipper.WeightLB
	default:
		return shipper.WeightKG
	}
}

func packageTypeToModel(pt generated.PackageType) shipper.PackageType {
	switch pt {
	case generated.PackageTypeBox:
		return shipper.PackageBox
	case generated.PackageTypeEnvelope:
		return shipper.PackageEnvelope
	case generated.PackageTypeTube:
		return shipper.PackageTube
	case generated.PackageTypePallet:
		return shipper.PackagePallet
	case generated.PackageTypeCustom:
		return shipper.PackageCustom
	default:
		return shipper.PackageBox
	}
}

func labelFormatToModel(lf generated.LabelFormat) shipper.LabelFormat {
	switch lf {
	case generated.LabelFormatPDF:
		return shipper.LabelPDF
	case generated.LabelFormatPng:
		return shipper.LabelPNG
	case generated.LabelFormatZpl:
		return shipper.LabelZPL
	default:
		return shipper.LabelPDF
	}
}

func labelFormatToEnum(lf shipper.LabelFormat) generated.LabelFormat {
	switch lf {
	case shipper.LabelPDF:
		return generated.LabelFormatPDF
	case shipper.LabelPNG:
		return generated.LabelFormatPng
	case shipper.LabelZPL:
		return generated.LabelFormatZpl
	default:
		return generated.LabelFormatPDF
	}
}

func parseDecimal(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
