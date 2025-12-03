package main

import (
	"context"

	"github.com/tournevent/logistic/internal/config"
	"github.com/tournevent/logistic/internal/telemetry"
	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/tournevent/logistic/pkg/shipper/canadapost"
	"github.com/tournevent/logistic/pkg/shipper/freightcom"
	"github.com/tournevent/logistic/pkg/shipper/purolator"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/trace"
)

func loadConfig() (*config.Config, error) {
	return config.Load()
}

func initLogger(level string) (*otelzap.Logger, error) {
	return telemetry.NewLogger(level)
}

func initTracer(ctx context.Context, cfg *config.Config) (func(context.Context) error, error) {
	if !cfg.OTELEnabled {
		return func(context.Context) error { return nil }, nil
	}

	_, shutdown, err := telemetry.InitTracer(ctx, cfg.OTELEndpoint, cfg.ServiceName, cfg.Version)
	return shutdown, err
}

func initShipperRegistry(cfg *config.Config, logger *otelzap.Logger) *shipper.Registry {
	registry := shipper.NewRegistry()

	// Get tracer for carriers
	var tracer trace.Tracer
	// tracer would be initialized from otel.GetTracerProvider().Tracer(cfg.ServiceName)

	// Register enabled carriers
	if cfg.FreightcomEnabled {
		fc := freightcom.New(freightcom.Config{
			APIKey:  cfg.FreightcomAPIKey,
			BaseURL: cfg.FreightcomBaseURL,
			UseMock: cfg.FreightcomUseMock,
		}, logger, tracer)
		registry.Register(fc)
	}

	if cfg.CanadaPostEnabled {
		cp := canadapost.New(canadapost.Config{
			APIKey:    cfg.CanadaPostAPIKey,
			AccountID: cfg.CanadaPostAccountID,
			BaseURL:   cfg.CanadaPostBaseURL,
			UseMock:   cfg.CanadaPostUseMock,
		}, logger, tracer)
		registry.Register(cp)
	}

	if cfg.PurolatorEnabled {
		puro := purolator.New(purolator.Config{
			Username: cfg.PurolatorUsername,
			Password: cfg.PurolatorPassword,
			WSDLURL:  cfg.PurolatorWSDLURL,
			UseMock:  cfg.PurolatorUseMock,
		}, logger, tracer)
		registry.Register(puro)
	}

	return registry
}
