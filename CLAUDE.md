# Claude Code Working Notes

## Repository Purpose

Delivro Logistics Bridge - GraphQL service for multi-carrier shipping integration.
Serves as a Hasura Actions backend for shipping operations (quote, order, label, cancel).

## Supported Carriers

- **Freightcom** - REST/JSON API
- **Canada Post** - REST/XML API
- **Purolator** - SOAP/WSDL API

## Plugin Usage

### When to use plugins

- `/go:cmd-build` - Build Go binary
- `/go:cmd-test` - Run Go tests
- `/go:cmd-lint` - Lint Go code with golangci-lint
- `/go:cmd-tidy` - Update Go dependencies
- `/docker:cmd-lint` - Lint Dockerfile
- `/orchestrator:detect` - Auto-detect appropriate plugin

### Available plugins

- go, docker, github, markdown, orchestrator

## Development Workflow

**Build Process:**

1. Modify Go code
2. Run `/go:cmd-lint` and `/go:cmd-test`
3. Build binary: `/go:cmd-build .`
4. Build Docker image: `docker build -t delivro-logistic:test .`
5. Commit changes

## GraphQL Operations (Hasura Actions)

**Mutations:**

- `delivro_get_quote` - Get shipping rates from carriers
- `delivro_create_order` - Create shipment with selected rate
- `delivro_get_label` - Download shipping label
- `delivro_cancel_order` - Cancel shipment

## Project Structure

- `main.go` - Entry point with cobra CLI
- `internal/config/` - Configuration loading (envconfig)
- `internal/telemetry/` - Logging, tracing, metrics
- `internal/server/` - HTTP server setup
- `internal/graphql/` - GraphQL resolvers and generated code
- `pkg/shipper/` - Carrier abstraction layer
- `pkg/shipper/freightcom/` - Freightcom implementation
- `pkg/shipper/canadapost/` - Canada Post implementation
- `pkg/shipper/purolator/` - Purolator implementation
- `schema.graphql` - GraphQL schema
- `gqlgen.yml` - gqlgen configuration

## API Client Abstraction Pattern

Each carrier has a layered architecture:

```text
pkg/shipper/<carrier>/
├── api.go       # APIClient interface + request/response types
├── api_mock.go  # MockAPIClient for testing
├── api_http.go  # HTTPAPIClient (REST) or SOAPAPIClient (SOAP)
└── client.go    # Shipper interface implementation
```

**Key concepts:**

- `APIClient` interface defines carrier-specific API operations
- `MockAPIClient` returns realistic mock data, supports custom callbacks
- `HTTPAPIClient`/`SOAPAPIClient` makes real API calls
- `Client` implements `shipper.Shipper` interface, delegates to APIClient
- `UseMock` config flag switches between mock and real implementations

**Testing with mocks:**

```go
// Use mock by default
client := freightcom.New(freightcom.Config{UseMock: true}, logger, tracer)

// Or inject custom mock with callbacks
mock := &freightcom.MockAPIClient{
    OnGetRates: func(ctx context.Context, req *freightcom.RatesRequest) (*freightcom.RatesResponse, error) {
        return &freightcom.RatesResponse{...}, nil
    },
}
client := freightcom.NewWithAPIClient(cfg, mock, logger, tracer)
```

## Configuration

Environment variables:

- `PORT` - HTTP server port (default: 80)
- `LOG_LEVEL` - Logging level (default: info)
- `FREIGHTCOM_API_KEY` - Freightcom API key
- `FREIGHTCOM_BASE_URL` - Freightcom API base URL
- `FREIGHTCOM_ENABLED` - Enable Freightcom (default: true)
- `FREIGHTCOM_USE_MOCK` - Use mock API client (default: false)
- `CANADAPOST_API_KEY` - Canada Post API key
- `CANADAPOST_ACCOUNT_ID` - Canada Post account ID
- `CANADAPOST_BASE_URL` - Canada Post API base URL
- `CANADAPOST_ENABLED` - Enable Canada Post (default: true)
- `CANADAPOST_USE_MOCK` - Use mock API client (default: false)
- `PUROLATOR_USERNAME` - Purolator username
- `PUROLATOR_PASSWORD` - Purolator password
- `PUROLATOR_WSDL_URL` - Purolator WSDL base URL
- `PUROLATOR_ENABLED` - Enable Purolator (default: true)
- `PUROLATOR_USE_MOCK` - Use mock API client (default: false)
- `OTEL_ENABLED` - Enable OpenTelemetry (default: true)
- `OTEL_ENDPOINT` - OTLP endpoint for tracing

## Testing

- Unit tests: `/go:cmd-test ./...`
- Integration tests: Require carrier sandbox credentials
- Docker build: `docker build .`

## Deployment

1. Build and push Docker image to registry
2. Configure Hasura Actions to call this service
3. Deploy via ArgoCD
