package shipper

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Registry manages registered shipping carriers.
type Registry struct {
	shippers map[string]Shipper
	mu       sync.RWMutex
}

// NewRegistry creates a new shipper registry.
func NewRegistry() *Registry {
	return &Registry{
		shippers: make(map[string]Shipper),
	}
}

// Register adds a shipper to the registry.
func (r *Registry) Register(s Shipper) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.shippers[s.Name()] = s
}

// Get returns a shipper by name.
func (r *Registry) Get(name string) (Shipper, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.shippers[name]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrCarrierNotFound, name)
}

// All returns all registered shippers.
func (r *Registry) All() []Shipper {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Shipper, 0, len(r.shippers))
	for _, s := range r.shippers {
		result = append(result, s)
	}
	return result
}

// Names returns the names of all registered shippers.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.shippers))
	for name := range r.shippers {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered shippers.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.shippers)
}

// GetAllQuotes fetches quotes from all registered carriers in parallel.
// Errors from individual carriers are logged but don't fail the entire request.
func (r *Registry) GetAllQuotes(ctx context.Context, req *QuoteRequest) ([]*QuoteResponse, []error) {
	shippers := r.All()
	if len(shippers) == 0 {
		return nil, []error{ErrCarrierNotFound}
	}

	results := make([]*QuoteResponse, 0, len(shippers))
	errs := make([]error, 0)
	mu := &sync.Mutex{}

	g, ctx := errgroup.WithContext(ctx)

	for _, s := range shippers {
		s := s // capture loop variable
		g.Go(func() error {
			resp, err := s.GetQuote(ctx, req)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", s.Name(), err))
				return nil // Don't fail the group, continue with other carriers
			}
			results = append(results, resp)
			return nil
		})
	}

	g.Wait()
	return results, errs
}

// GetQuotesFromCarriers fetches quotes from specific carriers.
func (r *Registry) GetQuotesFromCarriers(ctx context.Context, req *QuoteRequest, carriers []string) ([]*QuoteResponse, []error) {
	if len(carriers) == 0 {
		return r.GetAllQuotes(ctx, req)
	}

	results := make([]*QuoteResponse, 0, len(carriers))
	errs := make([]error, 0)
	mu := &sync.Mutex{}

	g, ctx := errgroup.WithContext(ctx)

	for _, name := range carriers {
		name := name // capture loop variable
		g.Go(func() error {
			s, err := r.Get(name)
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return nil
			}

			resp, err := s.GetQuote(ctx, req)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
				return nil
			}
			results = append(results, resp)
			return nil
		})
	}

	g.Wait()
	return results, errs
}
