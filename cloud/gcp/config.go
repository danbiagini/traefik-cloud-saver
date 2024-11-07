package gcp

import (
	"fmt"
)

// Config implements cloud.ProviderConfig
type Config struct {
	ProjectID      string `json:"projectId,omitempty"`
	Zone           string `json:"zone,omitempty"`
	InstanceGroup  string `json:"instanceGroup,omitempty"`
	MinInstances   int32  `json:"minInstances,omitempty"`
	MaxInstances   int32  `json:"maxInstances,omitempty"`
}

func (c *Config) Validate() error {
	if c.ProjectID == "" {
		return fmt.Errorf("projectId is required")
	}
	if c.Zone == "" {
		return fmt.Errorf("zone is required")
	}
	if c.MinInstances < 0 {
		return fmt.Errorf("minInstances cannot be negative")
	}
	if c.MaxInstances < c.MinInstances {
		return fmt.Errorf("maxInstances cannot be less than minInstances")
	}
	return nil
}

func (c *Config) GetType() string {
	return "gcp"
} 