package cloud

import (
	"context"
	"fmt"
	"log"

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
	"github.com/danbiagini/traefik-cloud-saver/cloud/gcp"
	"github.com/danbiagini/traefik-cloud-saver/cloud/mock"
)

// Service interface defines operations that can be performed on cloud resources
type Service interface {
	ScaleDown(ctx context.Context, serviceName string) error
	ScaleUp(ctx context.Context, serviceName string) error
	GetCurrentScale(ctx context.Context, serviceName string) (int32, error)
}

const (
	aws_t   = "aws"   // placeholder for future AWS implementation
	gcp_t   = "gcp"   // active GCP implementation
	azure_t = "azure" // placeholder for future Azure implementation
	mock_t  = "mock"
)

// NewService creates a new cloud service based on configuration
func NewService(config *common.CloudServiceConfig) (Service, error) {
	switch config.Type {
	case aws_t:
		return nil, fmt.Errorf("AWS implementation not yet available")
	case gcp_t:
		svc, err := gcp.New(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCP cloud service: %w", err)
		}
		return svc, nil
	case azure_t:
		return nil, fmt.Errorf("AZURE implementation not yet available")
	case mock_t:
		svc, err := mock.New(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create mock cloud service: %w", err)
		}
		return svc, nil
	default:
		return nil, fmt.Errorf("unknown cloud provider: %s", config.Type)
	}
}

// LogProvider is a simple helper for consistent cloud provider logging
func LogProvider(provider, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("[%s] %s", provider, msg)
}
