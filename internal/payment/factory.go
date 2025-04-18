package payment

import (
	"fmt"
)

// Factory creates payment providers
type Factory struct {
	providers map[string]Provider
}

// NewFactory creates a new payment provider factory
func NewFactory() *Factory {
	return &Factory{
		providers: make(map[string]Provider),
	}
}

// RegisterProvider adds a provider to the factory
func (f *Factory) RegisterProvider(name string, provider Provider) {
	f.providers[name] = provider
}

// GetProvider retrieves a provider by name
func (f *Factory) GetProvider(name string) (Provider, error) {
	provider, exists := f.providers[name]
	if !exists {
		return nil, fmt.Errorf("payment provider '%s' not found", name)
	}
	return provider, nil
}

// GetDefaultProvider returns the default payment provider
func (f *Factory) GetDefaultProvider() (Provider, error) {
	// You could make this configurable, but for now we'll just return
	// the first provider in the map, or an error if none exists
	if len(f.providers) == 0 {
		return nil, fmt.Errorf("no payment providers registered")
	}

	// Get the first provider (arbitrary)
	for _, provider := range f.providers {
		return provider, nil
	}

	return nil, fmt.Errorf("no payment providers available")
}
