package cloud

import (
	"context"
	"fmt"
)

// Provider interface defines operations that can be performed on cloud resources
type Provider interface {
	Initialize(config *ProviderConfig) error
	ScaleDown(ctx context.Context, serviceName string) error
	ScaleUp(ctx context.Context, serviceName string) error
	GetCurrentScale(ctx context.Context, serviceName string) (int32, error)
}

// ProviderConfig contains authentication and configuration for cloud providers
type ProviderConfig struct {
	Provider     string            `json:"provider,omitempty"`
	Region       string            `json:"region,omitempty"`
	Credentials  CredentialsConfig `json:"credentials,omitempty"`
	ResourceTags map[string]string `json:"resourceTags,omitempty"`
}

// CredentialsConfig contains authentication details for cloud providers
type CredentialsConfig struct {
	Type            string `json:"type,omitempty"`
	AccessKeyID     string `json:"accessKeyId,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
	// ... other credential fields
}

// New creates a new cloud provider based on configuration
func New(config *ProviderConfig) (Provider, error) {
	switch config.Provider {
	case "aws":
		return aws.New(config)
	case "gcp":
		return gcp.New(config)
	case "azure":
		return azure.New(config)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", config.Provider)
	}
} 