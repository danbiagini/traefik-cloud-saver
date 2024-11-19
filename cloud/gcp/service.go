package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

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

// loadServiceAccountCredentials loads credentials from a service account JSON file
func loadServiceAccountCredentials(path string) (*Credentials, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account file: %w", err)
	}

	var serviceAccount struct {
		Type         string `json:"type"`
		ClientEmail  string `json:"client_email"`
		PrivateKey   string `json:"private_key"`
		PrivateKeyID string `json:"private_key_id"`
		ProjectID    string `json:"project_id"`
	}

	if err := json.Unmarshal(data, &serviceAccount); err != nil {
		return nil, fmt.Errorf("failed to parse service account JSON: %w", err)
	}

	return &Credentials{
		Type:        serviceAccount.Type,
		ClientEmail: serviceAccount.ClientEmail,
		PrivateKey:  serviceAccount.PrivateKey,
		TokenURL:    "https://oauth2.googleapis.com/token",
		ProjectID:   serviceAccount.ProjectID,
	}, nil
}

func New(config *common.CloudServiceConfig) (*Service, error) {
	if config == nil {
		return nil, fmt.Errorf("config can't be nil for GCP")
	}

	if config.Zone == "" {
		return nil, fmt.Errorf("zone is required for GCP")
	}

	if config.Region == "" {
		return nil, fmt.Errorf("region is required for GCP")
	}

	if config.Credentials == nil || config.Credentials.Secret == "" {
		return nil, fmt.Errorf("credentials are required for GCP")
	}

	var creds *Credentials
	var err error
	if config.Credentials.Type == "service_account" || config.Credentials.Type == "" {
		// Load credentials from service account JSON file
		creds, err = loadServiceAccountCredentials(config.Credentials.Secret)
		if err != nil {
			return nil, fmt.Errorf("failed to load service account credentials: %w", err)
		}
	} else if config.Credentials.Type == "token" {
		// Use token directly as the private key, this is used for testing, it won't work in production
		creds = &Credentials{
			PrivateKey: config.Credentials.Secret,
		}
	} else {
		return nil, fmt.Errorf("unsupported credentials type: %s", config.Credentials.Type)
	}

	// Use ProjectID from service account if not specified in config
	projectID := config.ProjectID
	if projectID == "" {
		if creds.ProjectID == "" {
			return nil, fmt.Errorf("project ID is required for GCP")
		}
		projectID = creds.ProjectID
	}

	// Create token manager
	tokenManager, err := NewTokenManager(creds)
	if err != nil {
		return nil, fmt.Errorf("failed to create token manager: %w", err)
	}

	// Create compute client with token manager
	compute, err := NewComputeClient(&config.Endpoint, tokenManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute client: %w", err)
	}

	if compute == nil {
		return nil, fmt.Errorf("compute client is nil")
	}

	return &Service{
		compute:   *compute,
		projectID: projectID,
		zone:      config.Zone,
		region:    config.Region,
		config:    config,
	}, nil
}

func (s *Service) ScaleDown(ctx context.Context, instanceName string) error {
	// First check instance status

	common.LogProvider("traefik-cloud-saver", "ScaleDown for instance %s", instanceName)
	if s == nil {
		return fmt.Errorf("service is nil")
	}

	instance, err := s.compute.GetInstance(ctx, s.projectID, s.zone, instanceName)
	if err != nil {
		return fmt.Errorf("failed to get instance %s: %w", instanceName, err)
	}
	if instance == nil {
		return fmt.Errorf("received nil instance from GetInstance for %s", instanceName)
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
