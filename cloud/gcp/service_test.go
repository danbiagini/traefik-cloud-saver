package gcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
)

func setupMockService(handler http.Handler) (*Service, *httptest.Server) {
	ts := httptest.NewServer(handler)
	fmt.Printf("Test server URL: %s\n", ts.URL)

	// Use testTokenManager from testhelpers_test.go
	tokenManager, err := testTokenManager(ts)
	if err != nil {
		log.Fatalf("Failed to create token manager: %v", err)
	}

	baseURL := ts.URL + "/compute/v1"
	compute, err := NewComputeClient(&baseURL, tokenManager, WithTimeout(5*time.Second))
	if err != nil {
		log.Fatalf("Failed to create compute client: %v", err)
	}

	svc := &Service{
		compute:   *compute,
		projectID: "test-project",
		zone:      "test-zone",
	}

	return svc, ts
}

func TestGetCurrentScale(t *testing.T) {
	tests := []struct {
		name         string
		instanceName string
		setupMock    func(mux *http.ServeMux)
		want         int32
		wantErr      bool
	}{
		{
			name:         "running_instance",
			instanceName: "test-instance",
			setupMock: func(mux *http.ServeMux) {
				mux.HandleFunc("/compute/v1/projects/test-project/zones/test-zone/instances/test-instance", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"status": "RUNNING", "name": "test-instance"}`))
				})
			},
			want:    1,
			wantErr: false,
		},
		{
			name:         "stopped_instance",
			instanceName: "test-instance",
			setupMock: func(mux *http.ServeMux) {
				mux.HandleFunc("/compute/v1/projects/test-project/zones/test-zone/instances/test-instance", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"status": "TERMINATED", "name": "test-instance"}`))
				})
			},
			want:    0,
			wantErr: false,
		},
		{
			name:         "transitional_state",
			instanceName: "test-instance",
			setupMock: func(mux *http.ServeMux) {
				mux.HandleFunc("/compute/v1/projects/test-project/zones/test-zone/instances/test-instance", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"status": "STOPPING", "name": "test-instance"}`))
				})
			},
			want:    0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()

			// Keep the token endpoint at /token
			mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
			})

			// Add the test case's handlers
			tt.setupMock(mux)

			// Create service with the correct token URL
			svc, ts := setupMockService(mux)
			// Update the token URL to include the path
			svc.compute.tokenManager.credentials.TokenURL = ts.URL + "/token"
			defer ts.Close()

			got, err := svc.GetCurrentScale(context.Background(), tt.instanceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentScale() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetCurrentScale() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScaleUp(t *testing.T) {
	svc := &Service{}
	err := svc.ScaleUp(context.Background(), "test-instance")
	if err == nil {
		t.Error("ScaleUp() should return error as it's not implemented")
	}
}

func TestNewService(t *testing.T) {
	// Create temporary credentials files
	tmpFile, err := testCredentialsFile()
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}
	defer os.Remove(tmpFile)

	tmpFileNoProjectID, err := testCredentialsFileNoProjectID()
	if err != nil {
		t.Fatalf("Failed to create credentials file: %v", err)
	}
	defer os.Remove(tmpFileNoProjectID)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	tests := []struct {
		name      string
		config    *common.CloudServiceConfig
		wantErr   bool
		errString string
	}{
		{
			name: "valid config with credentials",
			config: &common.CloudServiceConfig{
				Credentials: &common.CredentialsConfig{
					Secret: tmpFile, // Use the temp file path instead of the JSON string
					Type:   "service_account",
				},
				ProjectID: "test-project",
				Zone:      "test-zone",
				Region:    "test-region",
				Type:      "gcp",
				Endpoint:  ts.URL,
			},
			wantErr: false,
		},
		{
			name: "missing project ID",
			config: &common.CloudServiceConfig{
				Credentials: &common.CredentialsConfig{
					Secret: tmpFileNoProjectID,
					Type:   "service_account",
				},
				Zone:   "test-zone",
				Region: "test-region",
				Type:   "gcp",
			},
			wantErr:   true,
			errString: "project ID is required for GCP",
		},
		{
			name: "missing zone",
			config: &common.CloudServiceConfig{
				Credentials: &common.CredentialsConfig{
					Secret: tmpFile,
					Type:   "service_account",
				},
				ProjectID: "test-project",
				Region:    "test-region",
				Type:      "gcp",
			},
			wantErr:   true,
			errString: "zone is required for GCP",
		},
		{
			name:      "nil config",
			config:    nil,
			wantErr:   true,
			errString: "config can't be nil for GCP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := New(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewService() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errString != "" && err.Error() != tt.errString {
					t.Errorf("NewService() error = %v, wantErr %v", err, tt.errString)
				}
				return
			}
			if err != nil {
				t.Errorf("NewService() unexpected error = %v", err)
				return
			}
			if svc == nil {
				t.Error("NewService() returned nil service")
				return
			}

			// Verify service was properly initialized
			if svc.projectID == "" {
				t.Error("NewService() projectID not set")
			}
			if svc.zone == "" {
				t.Error("NewService() zone not set")
			}

		})
	}
}
