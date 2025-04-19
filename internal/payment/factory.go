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

// GetDefaultProvider returns the default payment provider based on the provided name
// If that provider is not available, returns the first available provider
func (f *Factory) GetDefaultProvider(defaultName string) (Provider, error) {
	// Пытаемся получить провайдер по имени из конфигурации
	if defaultName != "" {
		provider, err := f.GetProvider(defaultName)
		if err == nil {
			return provider, nil
		}
	}

	// Если не нашли или произошла ошибка, ищем первый доступный
	for _, provider := range f.providers {
		return provider, nil
	}

	return nil, fmt.Errorf("no payment providers registered")
}
