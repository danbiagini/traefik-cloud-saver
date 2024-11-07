package traefik_cloud_saver

import (
	"time"
	"github.com/danbiagini/traefik-cloud-saver/cloud"
)

// Config the plugin configuration.
type Config struct {
	TrafficThreshold float64            `json:"trafficThreshold,omitempty"`
	WindowSize       string             `json:"windowSize,omitempty"`
	MetricsURL       string             `json:"metricsURL,omitempty"`
	RouterFilter     *RouterFilter      `json:"routerFilter,omitempty"`
	CloudConfig      *cloud.ServiceConfig `json:"cloudService,omitempty"`
	testMode         bool
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		TrafficThreshold: 1,
		WindowSize:      "5m",
		MetricsURL:     "http://localhost:8080/metrics",
		RouterFilter:    nil,
		CloudConfig:     &cloud.ServiceConfig{
			ResourceTags: make(map[string]string),
		},
		testMode:        false,
	}
} 