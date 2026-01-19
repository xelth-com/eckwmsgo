package delivery

import (
	"fmt"
	"sync"
)

// Registry manages all registered delivery providers
// This follows the Odoo pattern of having a central registry for modules
type Registry struct {
	mu        sync.RWMutex
	providers map[string]ProviderInterface
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]ProviderInterface),
	}
}

// Register registers a new delivery provider
func (r *Registry) Register(provider ProviderInterface) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	code := provider.Code()
	if code == "" {
		return fmt.Errorf("provider code cannot be empty")
	}

	if _, exists := r.providers[code]; exists {
		return fmt.Errorf("provider %s is already registered", code)
	}

	r.providers[code] = provider
	return nil
}

// Get returns a provider by its code
func (r *Registry) Get(code string) (ProviderInterface, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[code]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", code)
	}

	return provider, nil
}

// List returns all registered providers
func (r *Registry) List() []ProviderInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]ProviderInterface, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}

	return providers
}

// Has checks if a provider is registered
func (r *Registry) Has(code string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.providers[code]
	return exists
}

// Global registry instance
var globalRegistry = NewRegistry()

// GetGlobalRegistry returns the global provider registry
func GetGlobalRegistry() *Registry {
	return globalRegistry
}
