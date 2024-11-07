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
)

// Config the plugin configuration.
type Config struct {
	PollInterval string `json:"pollInterval,omitempty"`
	TrafficThreshold float64 `json:"trafficThreshold,omitempty"`
	WindowSize       string  `json:"windowSize,omitempty"`
	MetricsURL       string  `json:"metricsURL,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		PollInterval: 		"5s", // 5 * time.Second
		TrafficThreshold: 	1,
		WindowSize:       	"5m",
		MetricsURL:			"http://localhost:8080/metrics",
	}
}

// CloudSaver provider plugin to turn off cloud instances when traffic is below a threshold.
type Provider struct {
	name         string
	pollInterval time.Duration
	trafficThreshold float64
	metricsCollector *MetricsCollector
	cancel func()
}

// type TrafficStats struct {
// 	mutex sync.RWMutex
// 	requests []time.Time
// 	windowSize time.Duration
// }

// New creates a new Provider plugin.
func New(_ context.Context, config *Config, name string) (*Provider, error) {
	pi, err := time.ParseDuration(config.PollInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid poll interval: %w", err)
	}

	collector := NewMetricsCollector(config.MetricsURL)
	return &Provider{
		name:         name,
		pollInterval: pi,
		trafficThreshold: config.TrafficThreshold,
		metricsCollector: collector,
	}, nil
}

// Init the provider.
func (p *Provider) Init() error {
	if p.pollInterval <= 0 {
		return errors.New("poll interval must be greater than 0")
	}

	return nil
}

// Provide creates and send dynamic configuration.
func (p *Provider) Provide(cfgChan chan<- json.Marshaler) error {
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

func (p *Provider) loadConfiguration(ctx context.Context, cfgChan chan<- json.Marshaler) {
	ticker := time.NewTicker(p.pollInterval)
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
func (p *Provider) Stop() error {
	p.cancel()
	return nil
}

func (p *Provider) generateConfiguration() (*dynamic.JSONPayload, error) {
	// Get current service rates
	rates, err := p.metricsCollector.GetServiceRates()
	if err != nil {
		return nil, fmt.Errorf("failed to get service rates: %w", err)
	}

	// Log services below threshold
	for serviceName, rate := range rates {
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
