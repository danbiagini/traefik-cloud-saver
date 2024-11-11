package gcp

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/danbiagini/traefik-cloud-saver/cloud"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

type Service struct {
	computeService *compute.Service
	projectID      string
	zone           string
}

// NewService creates a new GCP compute service implementation.
// It requires a valid ServiceConfig with GCP-specific configuration
// and credentials for authentication.
func NewService(config *Config) (cloud.Service, error) {
	if config == nil {
		return nil, fmt.Errorf("invalid provider config type for GCP")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var options []option.ClientOption

	if config.Base.Credentials.Secret != "" {
		options = append(options, option.WithCredentialsJSON([]byte(config.Base.Credentials.Secret)))
	} else {
		log.Printf("No explicit credentials provided, using Application Default Credentials")
	}

	computeService, err := compute.NewService(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service: %w", err)
	}

	return &Service{
		computeService: computeService,
		projectID:      config.ProjectID,
		zone:           config.Zone,
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

func (s *Service) ScaleDown(ctx context.Context, instanceName string) error {
	// First check instance status
	instance, err := s.computeService.Instances.Get(
		s.projectID,
		s.zone,
		instanceName,
	).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to get instance %s: %w", instanceName, err)
	}

	// If instance is already stopped, return early
	if instance.Status == "TERMINATED" {
		return nil
	}

	// Stop the instance
	op, err := s.computeService.Instances.Stop(
		s.projectID,
		s.zone,
		instanceName,
	).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to stop instance %s: %w", instanceName, err)
	}

	// Wait for the operation to complete
	if err := s.waitForOperation(ctx, op.Name); err != nil {
		return fmt.Errorf("failed while waiting for instance to stop: %w", err)
	}

	return nil
}

func (s *Service) ScaleUp(ctx context.Context, instanceName string) error {
	return fmt.Errorf("scale up operation not implemented for GCP instances")
}

func (s *Service) GetCurrentScale(ctx context.Context, instanceName string) (int32, error) {
	instance, err := s.computeService.Instances.Get(
		s.projectID,
		s.zone,
		instanceName,
	).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("failed to get instance %s: %w", instanceName, err)
	}

	// Return 1 if the instance is running, 0 if it's stopped
	switch instance.Status {
	case "RUNNING":
		return 1, nil
	case "TERMINATED", "STOPPED":
		return 0, nil
	default:
		// For transitional states, return current state with a warning
		log.Printf("Instance %s is in transitional state: %s", instanceName, instance.Status)
		return 0, nil
	}
}

// Helper method to wait for GCP operations to complete
func (s *Service) waitForOperation(ctx context.Context, operationName string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			op, err := s.computeService.ZoneOperations.Get(
				s.projectID,
				s.zone,
				operationName,
			).Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("failed to get operation status: %w", err)
			}

			switch op.Status {
			case "DONE":
				if op.Error != nil {
					return fmt.Errorf("operation failed: %v", op.Error)
				}
				return nil
			case "PENDING", "RUNNING":
				continue
			default:
				return fmt.Errorf("unknown operation status: %s", op.Status)
			}
		}
	}
}
