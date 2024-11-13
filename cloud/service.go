package cloud

import (
	"context"
	"fmt"
	"log"

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
)

// Service interface defines operations that can be performed on cloud resources
type Service interface {
	ScaleDown(ctx context.Context, serviceName string) error
	ScaleUp(ctx context.Context, serviceName string) error
	GetCurrentScale(ctx context.Context, serviceName string) (int32, error)
}

const (
	aws    = "aws"   // placeholder for future AWS implementation
	gcp    = "gcp"   // active GCP implementation
	azure  = "azure" // placeholder for future Azure implementation
	mock_t = "mock"
)

// NewService creates a new cloud service based on configuration
func NewService(config *common.CloudServiceConfig) (Service, error) {
	switch config.Type {
	case "aws":
		return nil, fmt.Errorf("AWS implementation not yet available")
	case gcp:
		return nil, fmt.Errorf("GCP implementation not yet available")
	case azure:
		return nil, fmt.Errorf("AZURE implementation not yet available")
	// case mock_t:
	// return mock.New(config)
	default:
		return nil, fmt.Errorf("unknown cloud provider: %s", config.Type)
	}
}

// LogProvider is a simple helper for consistent cloud provider logging
func LogProvider(provider, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("[%s] %s", provider, msg)
}
