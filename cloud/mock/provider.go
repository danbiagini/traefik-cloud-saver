package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/your-repo/traefik-cloud-saver/cloud"
)

// Provider implements cloud.Provider interface for testing
type Provider struct {
	scale     map[string]int32
	mu        sync.RWMutex        // Protects scale map for concurrent access
	initError error               // Used to simulate initialization errors
	scaleErr  error              // Used to simulate scaling errors
}

// ProviderOption allows configuring the mock provider for different test scenarios
type ProviderOption func(*Provider)

// WithInitError configures the mock to return an error during initialization
func WithInitError(err error) ProviderOption {
	return func(p *Provider) {
		p.initError = err
	}
}

// WithScaleError configures the mock to return an error during scaling operations
func WithScaleError(err error) ProviderOption {
	return func(p *Provider) {
		p.scaleErr = err
	}
}

// New creates a new mock provider
func New(_ *cloud.ProviderConfig, opts ...ProviderOption) (cloud.Provider, error) {
	p := &Provider{
		scale: make(map[string]int32),
	}

	// Apply any configuration options
	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}

func (p *Provider) Initialize(_ *cloud.ProviderConfig) error {
	if p.initError != nil {
		return p.initError
	}
	return nil
}

func (p *Provider) ScaleDown(_ context.Context, serviceName string) error {
	if p.scaleErr != nil {
		return p.scaleErr
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	current, exists := p.scale[serviceName]
	if !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}

	if current <= 0 {
		return fmt.Errorf("service %s already at minimum scale", serviceName)
	}

	p.scale[serviceName] = current - 1
	return nil
}

func (p *Provider) ScaleUp(_ context.Context, serviceName string) error {
	if p.scaleErr != nil {
		return p.scaleErr
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	current, exists := p.scale[serviceName]
	if !exists {
		// Initialize with scale of 1 if service doesn't exist
		p.scale[serviceName] = 1
		return nil
	}

	p.scale[serviceName] = current + 1
	return nil
}

func (p *Provider) GetCurrentScale(_ context.Context, serviceName string) (int32, error) {
	if p.scaleErr != nil {
		return 0, p.scaleErr
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	scale, exists := p.scale[serviceName]
	if !exists {
		return 0, fmt.Errorf("service %s not found", serviceName)
	}

	return scale, nil
}

// Test helper methods

// SetScale allows tests to preset the scale of a service
func (p *Provider) SetScale(serviceName string, scale int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scale[serviceName] = scale
}

// Reset clears all stored scales and errors
func (p *Provider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scale = make(map[string]int32)
	p.initError = nil
	p.scaleErr = nil
}