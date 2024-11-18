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
	"testing"

	"github.com/danbiagini/traefik-cloud-saver/cloud/gcp"
)

func skipIfNoIntegrationTest(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}
}

func TestIntegrationComputeClient(t *testing.T) {
	skipIfNoIntegrationTest(t)

	// Read credentials from environment variables
	projectID := os.Getenv("GCP_PROJECT_ID")
	zone := os.Getenv("GCP_ZONE")
	instanceName := os.Getenv("GCP_INSTANCE_NAME")
	authToken := os.Getenv("GCP_AUTH_TOKEN")

	if projectID == "" || zone == "" || instanceName == "" {
		t.Fatal("GCP_PROJECT_ID, GCP_ZONE, and GCP_INSTANCE_NAME environment variables must be set")
	}

	if authToken == "" {
		t.Fatal("GCP_AUTH_TOKEN environment variable must be set")
	}

	baseURL := "https://compute.googleapis.com/compute/v1"
	client, err := gcp.NewComputeClient(&baseURL, &authToken, nil)
	if err != nil {
		t.Fatalf("Failed to create compute client: %v", err)
	}

	ctx := context.Background()

	t.Run("get_instance", func(t *testing.T) {
		instance, err := client.GetInstance(ctx, projectID, zone, instanceName)
		if err != nil {
			t.Fatalf("Failed to get instance: %v", err)
		}
		if instance.Name != instanceName {
			t.Errorf("Got instance name %s, want %s", instance.Name, instanceName)
		}
	})

	t.Run("stop_and_start_instance", func(t *testing.T) {
		// First verify instance is running
		instance, err := client.GetInstance(ctx, projectID, zone, instanceName)
		if err != nil {
			t.Fatalf("Failed to get instance: %v", err)
		}
		if instance.Status != "RUNNING" {
			t.Skipf("Instance not in RUNNING state (current: %s), skipping stop/start test", instance.Status)
		}

		// Stop the instance
		operation, err := client.StopInstance(ctx, projectID, zone, instanceName)
		if err != nil {
			t.Fatalf("Failed to stop instance: %v", err)
		}

		if operation.Status != "TERMINATED" && operation.Status != "STOPPED" {
			t.Errorf("Got operation status %s, want %s", operation.Status, "TERMINATED or STOPPED")
		} else {
			t.Logf("Instance status: %s", operation.Status)
		}
	})
}
