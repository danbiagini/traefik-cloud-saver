package gcp

import (
	"context"
	"fmt"

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
)

// Service implementation
type Service struct {
	compute   ComputeClient
	projectID string
	zone      string
	region    string
	config    *common.CloudServiceConfig
}

func New(config *common.CloudServiceConfig) (*Service, error) {

	if config == nil {
		return nil, fmt.Errorf("config can't be nil for GCP")
	}

	if config.ProjectID == "" {
		return nil, fmt.Errorf("project ID is required for GCP")
	}

	if config.Zone == "" {
		return nil, fmt.Errorf("zone is required for GCP")
	}

	if config.Region == "" {
		return nil, fmt.Errorf("region is required for GCP")
	}

	if config.Credentials == nil || config.Credentials.Secret == "" || config.Credentials.Type != "token" {
		return nil, fmt.Errorf("token credentials are required for GCP")
	}

	compute, err := NewComputeClient(&config.Endpoint, &config.Credentials.Secret, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client: %w", err)
	}

	return &Service{
		compute:   *compute,
		projectID: config.ProjectID,
		zone:      config.Zone,
		region:    config.Region,
		config:    config,
	}, nil
}

func (s *Service) ScaleDown(ctx context.Context, instanceName string) error {
	// First check instance status

	instance, err := s.compute.GetInstance(ctx, s.projectID, s.zone, instanceName)
	if err != nil {
		return fmt.Errorf("failed to get instance %s: %w", instanceName, err)
	}

	// If instance is already stopped or stopping, return early
	if instance.Status == "TERMINATED" || instance.Status == "STOPPING" {
		return nil
	}

	_, err = s.compute.StopInstance(ctx, s.projectID, s.zone, instanceName)
	if err != nil {
		return fmt.Errorf("failed to stop instance %s: %w", instanceName, err)
	}

	return nil
}

func (s *Service) ScaleUp(ctx context.Context, instanceName string) error {
	return fmt.Errorf("scale up operation not implemented for GCP instances")
}

func (s *Service) GetCurrentScale(ctx context.Context, instanceName string) (int32, error) {
	instance, err := s.compute.GetInstance(ctx, s.projectID, s.zone, instanceName)
	if err != nil {
		return 0, fmt.Errorf("failed to get instance %s: %w", instanceName, err)
	}

	switch instance.Status {
	case "RUNNING", "PROVISIONING", "STAGING":
		return 1, nil
	case "TERMINATED", "SUSPENDED", "STOPPING":
		return 0, nil
	default:
		fmt.Printf("Instance %s is in transitional state: %s", instanceName, instance.Status)
		return 0, nil
	}
}
