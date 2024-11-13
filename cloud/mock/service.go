package mock

import (
	"context"
	"fmt"
	"sync"

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
)

// Service implements cloud.Service interface for testing
type Service struct {
	scale     map[string]int32
	mu        sync.RWMutex // Protects scale map for concurrent access
	opCount   int
	failAfter int
	initError error
	scaleErr  error
	config    *common.CloudServiceConfig
}

// ServiceOption allows configuring the mock service for different test scenarios
type ServiceOption func(*Service)

// WithInitError configures the mock to return an error during initialization
func WithInitError(err error) ServiceOption {
	return func(s *Service) {
		s.initError = err
	}
}

// WithScaleError configures the mock to return an error during scaling operations
func WithScaleError(err error) ServiceOption {
	return func(s *Service) {
		s.scaleErr = err
	}
}

// New creates a new mock service
func New(config *common.CloudServiceConfig, opts ...ServiceOption) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	s := &Service{
		scale:     make(map[string]int32),
		failAfter: config.FailAfter,
		config:    config,
	}

	// Initialize with any pre-configured scales
	if config.InitialScale != nil {
		for k, v := range config.InitialScale {
			s.scale[k] = v
		}
	}

	// Apply any configuration options
	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (s *Service) Initialize(_ *common.CloudServiceConfig) error {
	if s.initError != nil {
		return s.initError
	}
	return nil
}

func (s *Service) checkFailure() error {
	s.opCount++
	if s.failAfter > 0 && s.opCount > s.failAfter {
		return fmt.Errorf("mock service failed after %d operations", s.failAfter)
	}
	return nil
}

func (s *Service) ScaleDown(_ context.Context, serviceName string) error {
	if err := s.checkFailure(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current, exists := s.scale[serviceName]
	if !exists {
		return fmt.Errorf("service %s not found", serviceName)
	}

	if current <= 0 {
		return fmt.Errorf("service %s already at minimum scale", serviceName)
	}

	s.scale[serviceName] = current - 1
	return nil
}

func (s *Service) ScaleUp(_ context.Context, serviceName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	common.LogProvider("mock", "scaling up service '%s' (current scale: %d)",
		serviceName, s.scale[serviceName])

	if s.scaleErr != nil {
		common.LogProvider("mock", "error scaling up: %v", s.scaleErr)
		return s.scaleErr
	}

	s.scale[serviceName]++
	return nil
}

func (s *Service) GetCurrentScale(_ context.Context, serviceName string) (int32, error) {
	if s.scaleErr != nil {
		return 0, s.scaleErr
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	scale, exists := s.scale[serviceName]
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
