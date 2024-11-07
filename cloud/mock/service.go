package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/danbiagini/traefik-cloud-saver/cloud"
)

// Service implements cloud.Service interface for testing
type Service struct {
	scale     map[string]int32
	mu        sync.RWMutex        // Protects scale map for concurrent access
	opCount   int
	failAfter int
	initError error
	scaleErr  error
	config    *cloud.ServiceConfig  // Add this if it's needed
}

// ServiceOption allows configuring the mock service for different test scenarios
type ServiceOption func(*Service)

// WithInitError configures the mock to return an error during initialization
func WithInitError(err error) ServiceOption {
	return func(p *Service) {
		p.initError = err
	}
}

// WithScaleError configures the mock to return an error during scaling operations
func WithScaleError(err error) ServiceOption {
	return func(p *Service) {
		p.scaleErr = err
	}
}

// New creates a new mock service
func New(_ *cloud.ServiceConfig, opts ...ServiceOption) (cloud.Service, error) {
	mockConfig, ok := config.Provider.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid provider config type for mock service")
	}

	s := &Service{
		scale:     make(map[string]int32),
		failAfter: mockConfig.FailAfter,
		config:    config,
	}

	// Initialize with any pre-configured scales
	if mockConfig.InitialScale != nil {
		for k, v := range mockConfig.InitialScale {
			s.scale[k] = v
		}
	}

	// Apply any configuration options
	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (p *Service) Initialize(_ *cloud.ServiceConfig) error {
	if p.initError != nil {
		return p.initError
	}
	return nil
}

func (p *Service) checkFailure() error {
	p.opCount++
	if p.failAfter > 0 && p.opCount > p.failAfter {
		return fmt.Errorf("mock service failed after %d operations", p.failAfter)
	}
	return nil
}

func (p *Service) ScaleDown(_ context.Context, serviceName string) error {
	if err := p.checkFailure(); err != nil {
		return err
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

func (p *Service) ScaleUp(_ context.Context, serviceName string) error {
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

func (p *Service) GetCurrentScale(_ context.Context, serviceName string) (int32, error) {
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
func (p *Service) SetScale(serviceName string, scale int32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scale[serviceName] = scale
}

// Reset clears all stored scales and errors
func (p *Service) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.scale = make(map[string]int32)
	p.initError = nil
	p.scaleErr = nil
}