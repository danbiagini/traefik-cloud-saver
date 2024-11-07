// Package traefik_cloud_saver contains functionality to turn off cloud instances when traffic is below a thresh.  "Turn the lights off when the room is empty"
package traefik_cloud_saver

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"
	"fmt"

	"github.com/traefik/genconf/dynamic"
	"github.com/danbiagini/traefik-cloud-saver/cloud"
)

// Config the plugin configuration.
type Config struct {
	TrafficThreshold float64           `json:"trafficThreshold,omitempty"`
	WindowSize      string            `json:"windowSize,omitempty"`
	MetricsURL      string            `json:"metricsURL,omitempty"`
	RouterFilter    *RouterFilter     `json:"routerFilter,omitempty"`
	testMode        bool              // unexported, internal UT flag
}

// RouterFilter defines criteria for selecting which routers to monitor
type RouterFilter struct {
	Labels map[string]string `json:"labels,omitempty"` // e.g., {"environment": "prod", "monitored": "true"}
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		TrafficThreshold: 1,
		WindowSize:      "5m",
		MetricsURL:     "http://localhost:8080/metrics",
		RouterFilter:    nil, // if nil, monitor all routers
		testMode:        false,
	}
}

// CloudSaver provider plugin to turn off cloud instances when traffic is below a threshold.
type Provider struct {
	name            string
	trafficThreshold float64
	windowSize      time.Duration
	routerFilter    *RouterFilter
	metricsCollector *MetricsCollector
	cloudService     cloud.Service
	testMode         bool
	cancel           func()
}

// type TrafficStats struct {
// 	mutex sync.RWMutex
// 	requests []time.Time
// 	windowSize time.Duration
// }

// New creates a new Provider plugin.
func New(_ context.Context, config *Config, name string) (*CloudSaver, error) {
	windowSize, err := time.ParseDuration(config.WindowSize)
	if err != nil {
		return nil, fmt.Errorf("invalid window size: %w", err)
	}

	// Basic configuration parsing validation
	if windowSize < time.Minute && !config.testMode {
		return nil, fmt.Errorf("window size must be at least 1 minute, got %v", windowSize)
	}

	collector := NewMetricsCollector(config.MetricsURL)
	return &CloudSaver{
		name:            name,
		windowSize:      windowSize,
		trafficThreshold: config.TrafficThreshold,
		routerFilter:    config.RouterFilter,
		metricsCollector: collector,
		testMode:         config.testMode,
	}, nil
}

// Init the provider.
func (p *CloudSaver) Init() error {
	// Runtime validation - ensures the plugin is in a valid state to start
	if p.windowSize < time.Minute && !p.testMode {
		return errors.New("window size must be at least 1 minute")
	}

	if p.trafficThreshold < 0 {
		return errors.New("traffic threshold must be non-negative")
	}

	// Could add other runtime checks here, like:
	// - Can we connect to the metrics URL?
	// - Do we have necessary permissions?
	// etc.

	return nil
}


// Provide creates and send dynamic configuration.
func (p *CloudSaver) Provide(cfgChan chan<- json.Marshaler) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Print(err)
			}
		}()

		p.loadConfiguration(ctx, cfgChan)
	}()

	return nil
}

func (p *CloudSaver) loadConfiguration(ctx context.Context, cfgChan chan<- json.Marshaler) {
	ticker := time.NewTicker(p.windowSize)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			configuration, err := p.generateConfiguration()
			if err != nil {
				log.Printf("ERROR: Failed to generate configuration: %v", err)
				continue
			}

			cfgChan <- configuration

		case <-ctx.Done():
			return
		}
	}
}

// Stop to stop the provider and the related go routines.
func (p *CloudSaver) Stop() error {
	p.cancel()
	return nil
}

func (p *CloudSaver) generateConfiguration() (*dynamic.JSONPayload, error) {
	// Get current service rates
	rates, err := p.metricsCollector.GetServiceRates()
	if err != nil {
		return nil, fmt.Errorf("failed to get service rates: %w", err)
	}

	// Filter and log services below threshold
	for serviceName, rate := range rates {
		// Check if this service's router matches our filter
		router := p.getRouterForService(serviceName)
		if router == nil || !p.shouldMonitorRouter(router) {
			continue
		}

		if rate.PerMin < p.trafficThreshold {
			log.Printf("LOW TRAFFIC ALERT: Service %s is below threshold (%.2f < %.2f req/min)",
				serviceName, rate.PerMin, p.trafficThreshold)
		}
	}

	// Return unchanged configuration
	return &dynamic.JSONPayload{
		Configuration: &dynamic.Configuration{
			HTTP: &dynamic.HTTPConfiguration{
				Routers:     make(map[string]*dynamic.Router),
				Services:    make(map[string]*dynamic.Service),
				Middlewares: make(map[string]*dynamic.Middleware),
			},
		},
	}, nil
}

// Get the router for a given service
func (p *CloudSaver) getRouterForService(serviceName string) *dynamic.Router {
	// TODO: Implement router lookup logic here.  Need to design a filtering mechanism.
	return nil
}

// Add a helper method to check if a router should be monitored
func (p *CloudSaver) shouldMonitorRouter(router *dynamic.Router) bool {
	if p.routerFilter == nil {
		return true // monitor all routers if no filter specified
	}

	// TODO: Implement router filter logic here.  Need to design a filtering mechanism.
	return true
}
