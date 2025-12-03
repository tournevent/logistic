package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tournevent/logistic/internal/graphql"
	"github.com/tournevent/logistic/internal/graphql/generated"
	"github.com/tournevent/logistic/internal/telemetry"
	"github.com/tournevent/logistic/pkg/shipper"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

// Server is the HTTP server for the logistics service.
type Server struct {
	port     int
	registry *shipper.Registry
	logger   *otelzap.Logger
	metrics  *telemetry.Metrics
	resolver *graphql.Resolver
}

// Config holds server configuration.
type Config struct {
	Port int
}

// New creates a new server instance.
func New(cfg Config, registry *shipper.Registry, logger *otelzap.Logger) *Server {
	metrics := telemetry.NewMetrics()
	resolver := graphql.NewResolver(registry, logger, metrics)

	return &Server{
		port:     cfg.Port,
		registry: registry,
		logger:   logger,
		metrics:  metrics,
		resolver: resolver,
	}
}

// Run starts the HTTP server and blocks until context is cancelled.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Prometheus metrics
	mux.Handle("/metrics", promhttp.Handler())

	// GraphQL endpoint
	mux.HandleFunc("/graphql", s.handleGraphQL)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("Starting server", zap.Int("port", s.port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		s.logger.Info("Shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// GraphQL request/response types
type graphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   interface{}      `json:"data,omitempty"`
	Errors []graphQLError   `json:"errors,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

func (s *Server) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(graphQLResponse{
			Errors: []graphQLError{{Message: "Method not allowed, use POST"}},
		})
		return
	}

	var req graphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(graphQLResponse{
			Errors: []graphQLError{{Message: "Invalid JSON: " + err.Error()}},
		})
		return
	}

	ctx := r.Context()

	// Simple query router based on operation
	// In production, use gqlgen's generated handler
	var response interface{}
	var err error

	switch {
	case containsQuery(req.Query, "health"):
		health, _ := s.resolver.Query().Health(ctx)
		response = map[string]interface{}{"health": health}

	case containsQuery(req.Query, "carriers"):
		carriers, _ := s.resolver.Query().Carriers(ctx)
		response = map[string]interface{}{"carriers": carriers}

	case containsQuery(req.Query, "serviceTypes"):
		types, _ := s.resolver.Query().ServiceTypes(ctx)
		response = map[string]interface{}{"serviceTypes": types}

	case containsQuery(req.Query, "delivro_get_quote"):
		input, err := parseGetQuoteInput(req.Variables)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(graphQLResponse{
				Errors: []graphQLError{{Message: err.Error()}},
			})
			return
		}
		result, _ := s.resolver.Mutation().DelivroGetQuote(ctx, input)
		response = map[string]interface{}{"delivro_get_quote": result}

	case containsQuery(req.Query, "delivro_create_order"):
		input, err := parseCreateOrderInput(req.Variables)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(graphQLResponse{
				Errors: []graphQLError{{Message: err.Error()}},
			})
			return
		}
		result, _ := s.resolver.Mutation().DelivroCreateOrder(ctx, input)
		response = map[string]interface{}{"delivro_create_order": result}

	case containsQuery(req.Query, "delivro_get_label"):
		input, err := parseGetLabelInput(req.Variables)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(graphQLResponse{
				Errors: []graphQLError{{Message: err.Error()}},
			})
			return
		}
		result, _ := s.resolver.Mutation().DelivroGetLabel(ctx, input)
		response = map[string]interface{}{"delivro_get_label": result}

	case containsQuery(req.Query, "delivro_cancel_order"):
		input, err := parseCancelOrderInput(req.Variables)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(graphQLResponse{
				Errors: []graphQLError{{Message: err.Error()}},
			})
			return
		}
		result, _ := s.resolver.Mutation().DelivroCancelOrder(ctx, input)
		response = map[string]interface{}{"delivro_cancel_order": result}

	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(graphQLResponse{
			Errors: []graphQLError{{Message: "Unknown operation"}},
		})
		return
	}

	if err != nil {
		json.NewEncoder(w).Encode(graphQLResponse{
			Errors: []graphQLError{{Message: err.Error()}},
		})
		return
	}

	json.NewEncoder(w).Encode(graphQLResponse{Data: response})
}

func containsQuery(query, operation string) bool {
	return len(query) > 0 && (contains(query, operation) || contains(query, camelCase(operation)))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func camelCase(s string) string {
	// Simple conversion: delivro_get_quote -> delivroGetQuote
	result := make([]byte, 0, len(s))
	capitalizeNext := false
	for i := 0; i < len(s); i++ {
		if s[i] == '_' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext && s[i] >= 'a' && s[i] <= 'z' {
			result = append(result, s[i]-32)
			capitalizeNext = false
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}

// Input parsing helpers
func parseGetQuoteInput(vars map[string]interface{}) (generated.GetQuoteInput, error) {
	var input generated.GetQuoteInput
	inputData, ok := vars["input"].(map[string]interface{})
	if !ok {
		return input, fmt.Errorf("missing or invalid 'input' variable")
	}

	// Parse required fields
	input.ShipperID, _ = inputData["shipperId"].(string)

	if origin, ok := inputData["origin"].(map[string]interface{}); ok {
		input.Origin = parseAddressInputPtr(origin)
	}
	if dest, ok := inputData["destination"].(map[string]interface{}); ok {
		input.Destination = parseAddressInputPtr(dest)
	}
	if pkgs, ok := inputData["packages"].([]interface{}); ok {
		input.Packages = parsePackagesInput(pkgs)
	}

	return input, nil
}

func parseCreateOrderInput(vars map[string]interface{}) (generated.CreateOrderInput, error) {
	var input generated.CreateOrderInput
	inputData, ok := vars["input"].(map[string]interface{})
	if !ok {
		return input, fmt.Errorf("missing or invalid 'input' variable")
	}

	input.ShipperID, _ = inputData["shipperId"].(string)
	input.RateID, _ = inputData["rateId"].(string)

	if quoteID, ok := inputData["quoteId"].(string); ok {
		input.QuoteID = &quoteID
	}
	if sender, ok := inputData["sender"].(map[string]interface{}); ok {
		input.Sender = parseContactInputPtr(sender)
	}
	if senderAddr, ok := inputData["senderAddress"].(map[string]interface{}); ok {
		input.SenderAddress = parseAddressInputPtr(senderAddr)
	}
	if recipient, ok := inputData["recipient"].(map[string]interface{}); ok {
		input.Recipient = parseContactInputPtr(recipient)
	}
	if recipientAddr, ok := inputData["recipientAddress"].(map[string]interface{}); ok {
		input.RecipientAddress = parseAddressInputPtr(recipientAddr)
	}
	if pkgs, ok := inputData["packages"].([]interface{}); ok {
		input.Packages = parsePackagesInput(pkgs)
	}

	return input, nil
}

func parseGetLabelInput(vars map[string]interface{}) (generated.GetLabelInput, error) {
	var input generated.GetLabelInput
	inputData, ok := vars["input"].(map[string]interface{})
	if !ok {
		return input, fmt.Errorf("missing or invalid 'input' variable")
	}

	input.OrderID, _ = inputData["orderId"].(string)

	return input, nil
}

func parseCancelOrderInput(vars map[string]interface{}) (generated.CancelOrderInput, error) {
	var input generated.CancelOrderInput
	inputData, ok := vars["input"].(map[string]interface{})
	if !ok {
		return input, fmt.Errorf("missing or invalid 'input' variable")
	}

	input.OrderID, _ = inputData["orderId"].(string)
	if reason, ok := inputData["reason"].(string); ok {
		input.Reason = &reason
	}

	return input, nil
}

func parseAddressInputPtr(data map[string]interface{}) *generated.AddressInput {
	addr := &generated.AddressInput{}
	addr.Name, _ = data["name"].(string)
	addr.Line1, _ = data["line1"].(string)
	addr.City, _ = data["city"].(string)
	addr.ProvinceCode, _ = data["provinceCode"].(string)
	addr.PostalCode, _ = data["postalCode"].(string)
	addr.Phone, _ = data["phone"].(string)

	if company, ok := data["company"].(string); ok {
		addr.Company = &company
	}
	if line2, ok := data["line2"].(string); ok {
		addr.Line2 = &line2
	}
	if countryCode, ok := data["countryCode"].(string); ok {
		addr.CountryCode = &countryCode
	}
	if email, ok := data["email"].(string); ok {
		addr.Email = &email
	}

	return addr
}

func parseContactInputPtr(data map[string]interface{}) *generated.ContactInput {
	contact := &generated.ContactInput{}
	contact.Name, _ = data["name"].(string)
	contact.Phone, _ = data["phone"].(string)

	if company, ok := data["company"].(string); ok {
		contact.Company = &company
	}
	if email, ok := data["email"].(string); ok {
		contact.Email = &email
	}

	return contact
}

func parsePackagesInput(pkgs []interface{}) []*generated.PackageInput {
	result := make([]*generated.PackageInput, 0, len(pkgs))
	for _, p := range pkgs {
		if pkg, ok := p.(map[string]interface{}); ok {
			input := &generated.PackageInput{}
			input.Length, _ = pkg["length"].(string)
			input.Width, _ = pkg["width"].(string)
			input.Height, _ = pkg["height"].(string)
			input.Weight, _ = pkg["weight"].(string)
			result = append(result, input)
		}
	}
	return result
}
