package gcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
)

func setupMockService(handler http.Handler) (*Service, *httptest.Server) {
	ts := httptest.NewServer(handler)

	fmt.Printf("Test server URL: %s\n", ts.URL)

	timeout := 5 * time.Second
	authToken := "test-token"
	compute, err := NewComputeClient(&ts.URL, &authToken, &timeout)
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
				mux.HandleFunc("/projects/test-project/zones/test-zone/instances/test-instance", func(w http.ResponseWriter, r *http.Request) {
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
				mux.HandleFunc("/projects/test-project/zones/test-zone/instances/test-instance", func(w http.ResponseWriter, r *http.Request) {
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
				mux.HandleFunc("/projects/test-project/zones/test-zone/instances/test-instance", func(w http.ResponseWriter, r *http.Request) {
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
			tt.setupMock(mux)

			svc, ts := setupMockService(mux)
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
	mockCreds := `{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "mock-key-id",
		"private_key": "-----BEGIN PRIVATE KEY-----\nmock-key\n-----END PRIVATE KEY-----\n",
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"client_id": "123456789",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test@test-project.iam.gserviceaccount.com"
	}`

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
					Secret: mockCreds,
					Type:   "token",
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
					Secret: mockCreds,
					Type:   "token",
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
					Secret: mockCreds,
					Type:   "token",
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
