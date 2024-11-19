package common

import (
	"fmt"
	"log"
)

var (
	debugEnabled bool
)

// CredentialsConfig contains authentication details
type CredentialsConfig struct {
	Type   string `json:"type,omitempty"`
	Secret string `json:"secret,omitempty"` // Generic secret field
}

// CloudServiceConfig 
type CloudServiceConfig struct {
	Type         string             `json:"type"`
	Region       string             `json:"region,omitempty"`
	ResourceTags map[string]string  `json:"resourceTags,omitempty"`
	Credentials  *CredentialsConfig `json:"credentials,omitempty"`
	Endpoint     string             `json:"endpoint,omitempty"`
	// GCP specific fields
	ServiceAccount string `json:"serviceAccount,omitempty"`
	ProjectID      string `json:"projectID,omitempty"`
	Zone           string `json:"zone,omitempty"`

	// Mock-specific fields
	InitialScale map[string]int32 `json:"initialScale,omitempty"`
	FailAfter    int              `json:"failAfter,omitempty"`
	ResetAfter   string           `json:"resetAfter,omitempty"`
}

func SetDebug(enabled bool) {
	debugEnabled = enabled
}

// LogProvider is a simple helper for consistent cloud provider logging
func LogProvider(provider, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("[%s] %s", provider, msg)
}

func DebugLog(provider, format string, v ...interface{}) {
	if debugEnabled {
		msg := fmt.Sprintf("[DEBUG] %s", format)
		LogProvider(provider, msg, v...)
	}
}

func (c *CloudServiceConfig) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("type is required")
	}
	switch c.Type {
	case "gcp":
		if c.ProjectID == "" {
			return fmt.Errorf("projectID is required")
		}
		if c.Zone == "" {
			return fmt.Errorf("zone is required")
		}
	case "mock":
		if c.InitialScale == nil {
			return fmt.Errorf("initialScale is required")
		}
	default:
		return fmt.Errorf("invalid type: %s", c.Type)
	}
	return nil
}

func (c *CloudServiceConfig) GetType() string {
	return c.Type
}

func (c *CloudServiceConfig) GetRegion() string {
	return c.Region
}

func (c *CloudServiceConfig) GetResourceTags() map[string]string {
	return c.ResourceTags
}

func (c *CloudServiceConfig) GetEndpoint() string {
	return c.Endpoint
}
