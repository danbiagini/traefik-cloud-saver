package traefik_cloud_saver

import (
	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
)

// Config the plugin configuration.
type Config struct {
	TrafficThreshold float64                    `json:"trafficThreshold,omitempty"`
	WindowSize       string                     `json:"windowSize,omitempty"`
	MetricsURL       string                     `json:"metricsURL,omitempty"`
	RouterFilter     *RouterFilter              `json:"routerFilter,omitempty"`
	CloudConfig      *common.CloudServiceConfig `json:"cloudConfig,omitempty"`
	APIURL           string                     `json:"apiURL,omitempty"`
	Debug            bool                       `json:"debug,omitempty"`
	testMode         bool
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		TrafficThreshold: 1,
		WindowSize:       "5m",
		MetricsURL:       "http://localhost:8080/metrics",
		RouterFilter:     nil,
		CloudConfig: &common.CloudServiceConfig{
			Type: "mock",
		},
		testMode: false,
		APIURL:   "http://localhost:8080/api/",
		Debug:    false,
	}
}
