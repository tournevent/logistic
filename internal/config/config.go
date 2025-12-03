package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
	"go.opentelemetry.io/otel/attribute"
)

// Config holds all configuration for the service.
type Config struct {
	// Server
	Port     int    `envconfig:"PORT" default:"80"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"info"`

	// Freightcom
	FreightcomAPIKey  string `envconfig:"FREIGHTCOM_API_KEY"`
	FreightcomBaseURL string `envconfig:"FREIGHTCOM_BASE_URL" default:"https://api.freightcom.com/v1"`
	FreightcomEnabled bool   `envconfig:"FREIGHTCOM_ENABLED" default:"true"`
	FreightcomUseMock bool   `envconfig:"FREIGHTCOM_USE_MOCK" default:"false"`

	// Canada Post
	CanadaPostAPIKey    string `envconfig:"CANADAPOST_API_KEY"`
	CanadaPostAccountID string `envconfig:"CANADAPOST_ACCOUNT_ID"`
	CanadaPostBaseURL   string `envconfig:"CANADAPOST_BASE_URL" default:"https://soa-gw.canadapost.ca"`
	CanadaPostEnabled   bool   `envconfig:"CANADAPOST_ENABLED" default:"true"`
	CanadaPostUseMock   bool   `envconfig:"CANADAPOST_USE_MOCK" default:"false"`

	// Purolator
	PurolatorUsername string `envconfig:"PUROLATOR_USERNAME"`
	PurolatorPassword string `envconfig:"PUROLATOR_PASSWORD"`
	PurolatorWSDLURL  string `envconfig:"PUROLATOR_WSDL_URL" default:"https://webservices.purolator.com/EWS/V2/Shipping/ShippingService.asmx?wsdl"`
	PurolatorEnabled  bool   `envconfig:"PUROLATOR_ENABLED" default:"true"`
	PurolatorUseMock  bool   `envconfig:"PUROLATOR_USE_MOCK" default:"false"`

	// Telemetry
	OTELEnabled  bool   `envconfig:"OTEL_ENABLED" default:"true"`
	OTELEndpoint string `envconfig:"OTEL_ENDPOINT" default:"http://jaeger-collector.claude.svc.cluster.local:4318"`
	ServiceName  string `envconfig:"SERVICE_NAME" default:"delivro-logistic"`
	Version      string `envconfig:"SERVICE_VERSION" default:"0.0.1"`
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return &cfg, nil
}

// Attributes returns OpenTelemetry attributes for this configuration.
func (c *Config) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("service.name", c.ServiceName),
		attribute.String("service.version", c.Version),
		attribute.Bool("freightcom.enabled", c.FreightcomEnabled),
		attribute.Bool("canadapost.enabled", c.CanadaPostEnabled),
		attribute.Bool("purolator.enabled", c.PurolatorEnabled),
	}
}
