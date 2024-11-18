// *****************************************************
// ************** INTEGRATION TESTS ******************
// *****************************************************
// export INTEGRATION_TEST=true
// export GCP_PROJECT_ID=your-project-id
// export GCP_ZONE=your-zone
// export GCP_INSTANCE_NAME=your-instance-name
// go test -v ./test/compute_integration_test.go
// *****************************************************

package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
	"github.com/danbiagini/traefik-cloud-saver/cloud/gcp"
)

func skipIfNoIntegrationTest(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}
}

func TestIntegrationComputeClient(t *testing.T) {
	skipIfNoIntegrationTest(t)

	// Read configuration from environment variables
	credentialsPath := os.Getenv("GCP_CREDENTIALS_PATH")
	credentialsType := os.Getenv("GCP_CREDENTIALS_TYPE")
	if credentialsPath == "" {
		credentialsPath = filepath.Join(os.Getenv("HOME"), ".config", "gcloud", "application_default_credentials.json")
		credentialsType = "application_default"
	}

	zone := os.Getenv("GCP_ZONE")
	region := os.Getenv("GCP_REGION")
	projectID := os.Getenv("GCP_PROJECT_ID")
	instanceName := os.Getenv("GCP_INSTANCE_NAME")

	if zone == "" || instanceName == "" {
		t.Fatal("GCP_ZONE and GCP_INSTANCE_NAME environment variables must be set")
	}

	config := &common.CloudServiceConfig{
		Zone:      zone,
		Region:    region,
		ProjectID: projectID,
		Credentials: &common.CredentialsConfig{
			Secret: credentialsPath,
			Type:   credentialsType,
		},
	}
	s, err := gcp.New(config)
	if err != nil {
		t.Fatalf("Failed to create GCP service: %v", err)
	}

	ctx := context.Background()
	t.Run("get_instance_scale", func(t *testing.T) {
		count, err := s.GetCurrentScale(ctx, instanceName)
		if err != nil {
			t.Fatalf("Failed to get instance: %v", err)
		}
		t.Logf("Instance scale: %d", count)
	})

	t.Run("stop_and_start_instance", func(t *testing.T) {
		// Stop the instance
		err := s.ScaleDown(ctx, instanceName)
		if err != nil {
			t.Fatalf("Failed to stop instance: %v", err)
		}
	})
}
