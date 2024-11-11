// Package traefik_cloud_saver contains functionality to turn off cloud instances when traffic is below a thresh.  "Turn the lights off when the room is empty"
package traefik_cloud_saver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/danbiagini/traefik-cloud-saver/cloud"
	"github.com/traefik/genconf/dynamic"
)

// RouterFilter defines criteria for selecting which routers to monitor
type RouterFilter struct {
	Names []string `json:"names,omitempty"` // e.g., ["my-api-router", "web-router"]
}

// CloudSaver provider plugin to turn off cloud instances when traffic is below a threshold.
type CloudSaver struct {
	name             string
	trafficThreshold float64
	windowSize       time.Duration
	routerFilter     *RouterFilter
	metricsCollector *MetricsCollector
	cloudService     cloud.Service
	testMode         bool
	cancel           func()
	apiURL           string
	debug            bool
}

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
		name:             name,
		windowSize:       windowSize,
		trafficThreshold: config.TrafficThreshold,
		routerFilter:     config.RouterFilter,
		metricsCollector: collector,
		testMode:         config.testMode,
		apiURL:           config.APIURL,
		debug:            config.Debug,
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

// Update TraefikRouter struct to match all fields from the API response
type TraefikRouter struct {
	Name        string   `json:"name"`
	Rule        string   `json:"rule"`
	Service     string   `json:"service"`
	Provider    string   `json:"provider"`
	Status      string   `json:"status"`
	EntryPoints []string `json:"entryPoints"`
	Using       []string `json:"using"`
	Priority    int      `json:"priority,omitempty"`
	Middlewares []string `json:"middlewares,omitempty"`
}

// Add method to get routers from Traefik API
func (p *CloudSaver) getRoutersFromAPI() (map[string]*TraefikRouter, error) {
	resp, err := http.Get(p.apiURL + "/http/routers")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch routers: %w", err)
	}
	defer resp.Body.Close()

	var routerSlice []TraefikRouter
	if err := json.NewDecoder(resp.Body).Decode(&routerSlice); err != nil {
		return nil, fmt.Errorf("failed to decode routers: %w", err)
	}

	// Convert slice to map
	routerMap := make(map[string]*TraefikRouter)
	for i := range routerSlice {
		router := routerSlice[i] // Create a copy to avoid pointer to loop variable
		routerMap[router.Name] = &router
	}
	return routerMap, nil
}

func (p *CloudSaver) generateConfiguration() (*dynamic.JSONPayload, error) {
	// Get current service rates
	rates, err := p.metricsCollector.GetServiceRates()
	if err != nil {
		return nil, fmt.Errorf("failed to get service rates: %w", err)
	}

	// Get router configurations
	routers, err := p.getRoutersFromAPI()
	if err != nil {
		return nil, fmt.Errorf("failed to get routers: %w", err)
	}

	// Create service to router mapping
	serviceToRouter := make(map[string]string)
	for _, router := range routers {
		if router.Service != "" {
			serviceToRouter[router.Service] = router.Name
		}
	}

	// Filter and log services below threshold
	for serviceName, rate := range rates {
		routerName := serviceToRouter[serviceName]
		if routerName == "" {
			if p.debug {
				log.Printf("Skipping service %s - no matching router found", serviceName)
			}
			continue
		}

		if !p.shouldMonitorRouter(routerName) {
			if p.debug {
				log.Printf("Skipping router %s - not in filter list", routerName)
			}
			continue
		}

		if rate.PerMin < p.trafficThreshold {
			log.Printf("LOW TRAFFIC ALERT: Service %s (router %s) is below threshold (%.2f < %.2f req/min)",
				serviceName, routerName, rate.PerMin, p.trafficThreshold)
			// TODO: Add cloud scaling logic here
		}
	}

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

// shouldMonitorRouter checks if a router should be monitored based on filter criteria
func (p *CloudSaver) shouldMonitorRouter(routerName string) bool {
	if p.routerFilter == nil || len(p.routerFilter.Names) == 0 {
		return true // monitor all routers if no filter specified
	}

	// Check if router name matches any in the Names filter
	// TODO: This is a linear search, could be optimized, but we don't expect this list to be long
	for _, name := range p.routerFilter.Names {
		if name == routerName {
			return true
		}
	}
	return false
}
