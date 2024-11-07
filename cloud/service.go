package cloud

import (
    "context"
    "fmt"
)

// Service interface defines operations that can be performed on cloud resources
type Service interface {
    Initialize(config *ServiceConfig) error
    ScaleDown(ctx context.Context, serviceName string) error
    ScaleUp(ctx context.Context, serviceName string) error
    GetCurrentScale(ctx context.Context, serviceName string) (int32, error)
}

// ServiceConfig contains common configuration elements
type ServiceConfig struct {
    Type         string            `json:"type,omitempty"`
    Region       string            `json:"region,omitempty"`
    Credentials  CredentialsConfig `json:"credentials,omitempty"`
    ResourceTags map[string]string `json:"resourceTags,omitempty"`
    Provider     ProviderConfig    `json:"provider,omitempty"`
}

// ProviderConfig interface allows for provider-specific configuration
type ProviderConfig interface {
    Validate() error
    GetType() string
}

// CredentialsConfig contains authentication details
type CredentialsConfig struct {
    Type   string `json:"type,omitempty"`
    Secret string `json:"secret,omitempty"` // Generic secret field
}

const (
    aws   = "aws"   // placeholder for future AWS implementation
    gcp   = "gcp"   // active GCP implementation
    azure = "azure" // placeholder for future Azure implementation
)

// NewService creates a new cloud service based on configuration
func NewService(config *ServiceConfig) (Service, error) {
    switch config.Type {
    case "aws":
        return nil, fmt.Errorf("AWS implementation not yet available")
    case gcp:
        return nil, fmt.Errorf("GCP implementation not yet available")
    case azure:
        return nil, fmt.Errorf("Azure implementation not yet available")
    default:
        return nil, fmt.Errorf("unknown cloud provider: %s", config.Type)
    }
}