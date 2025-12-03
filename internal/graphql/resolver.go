package graphql

import (
	"github.com/tournevent/logistic/internal/telemetry"
	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
)

// Resolver is the root resolver for the GraphQL schema.
// It holds dependencies needed by all resolvers.
type Resolver struct {
	Registry *shipper.Registry
	Logger   *otelzap.Logger
	Metrics  *telemetry.Metrics
}

// NewResolver creates a new resolver with the given dependencies.
func NewResolver(registry *shipper.Registry, logger *otelzap.Logger, metrics *telemetry.Metrics) *Resolver {
	return &Resolver{
		Registry: registry,
		Logger:   logger,
		Metrics:  metrics,
	}
}
