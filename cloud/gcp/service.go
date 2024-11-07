package gcp

import (
	"context"
	"fmt"
	"time"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	"github.com/danbiagini/traefik-cloud-saver/cloud"
)

type Service struct {
	computeService *compute.Service
	projectID      string
	zone          string
	minInstances  int32
	maxInstances  int32
}

func NewService(config *cloud.ServiceConfig) (cloud.Service, error) {
	gcpConfig, ok := config.Provider.(*Config)
	if !ok {
		return nil, fmt.Errorf("invalid provider config type for GCP")
	}

	if err := gcpConfig.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	computeService, err := compute.NewService(ctx, 
		option.WithCredentialsJSON([]byte(config.Credentials.Secret)))
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	return &Service{
		computeService: computeService,
		projectID:     gcpConfig.ProjectID,
		zone:         gcpConfig.Zone,
		minInstances: gcpConfig.MinInstances,
		maxInstances: gcpConfig.MaxInstances,
	}, nil
}

func (s *Service) Initialize(_ *cloud.ServiceConfig) error {
	// Validate required fields
	if s.projectID == "" {
		return fmt.Errorf("GCP project ID is required")
	}
	if s.zone == "" {
		return fmt.Errorf("GCP zone is required")
	}
	return nil
}

func (s *Service) ScaleDown(ctx context.Context, serviceName string) error {
	return fmt.Errorf("GCP ScaleDown implementation not yet available")
}

func (s *Service) ScaleUp(ctx context.Context, serviceName string) error {
	return fmt.Errorf("GCP ScaleUpimplementation not yet available")
}

func (s *Service) GetCurrentScale(ctx context.Context, serviceName string) (int32, error) {
	ig, err := s.computeService.InstanceGroups.Get(
		s.projectID, 
		s.zone, 
		serviceName,
	).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("failed to get instance group %s: %w", serviceName, err)
	}

	return int32(ig.Size), nil
}

// Helper method to wait for GCP operations to complete
func (s *Service) waitForOperation(ctx context.Context, operationName string) error {
	for {
		operation, err := s.computeService.ZoneOperations.Get(
			s.projectID, 
			s.zone, 
			operationName,
		).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to get operation status: %w", err)
		}

		if operation.Status == "DONE" {
			if operation.Error != nil {
				return fmt.Errorf("operation failed: %v", operation.Error)
			}
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second * 2):
			// Continue polling
		}
	}
} 