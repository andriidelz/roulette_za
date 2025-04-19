package payment

import (
	"fmt"
	"log"
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
	// Пытаемся получить oxapay
	provider, err := f.GetProvider("oxapay")
	if err == nil {
		log.Printf("[Factory] Using oxapay as default provider")
		return provider, nil
	}

	// Если oxapay не доступен, ищем первый доступный провайдер
	log.Printf("[Factory] Oxapay not available, looking for any provider")
	for name, provider := range f.providers {
		log.Printf("[Factory] Found provider: %s", name)
		return provider, nil
	}

	// Если нет ни одного провайдера
	return nil, fmt.Errorf("no payment providers registered")
}
