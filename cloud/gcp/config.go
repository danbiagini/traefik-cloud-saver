package gcp

import (
	"fmt"

	"github.com/danbiagini/traefik-cloud-saver/cloud"
)

// Config implements cloud.ProviderConfig
type Config struct {
	Base      cloud.ServiceConfig
	ProjectID string `json:"projectId,omitempty"`
	Zone      string `json:"zone,omitempty"`
}

func (c *Config) Validate() error {
	if c.ProjectID == "" {
		return fmt.Errorf("projectId is required")
	}
	if c.Zone == "" {
		return fmt.Errorf("zone is required")
	}
	return nil
}

func (c *Config) GetType() string {
	return "gcp"
}
