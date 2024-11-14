package gcp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

func setupMockService(t *testing.T, handler http.Handler) (*Service, *httptest.Server) {
	ts := httptest.NewServer(handler)

	computeService, err := compute.NewService(context.Background(),
		option.WithEndpoint(ts.URL),
		option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("Failed to create mock compute service: %v", err)
	}

	svc := &Service{
		computeService: computeService,
		projectID:      "test-project",
		zone:           "test-zone",
	}

	return svc, ts
}

func TestScaleDown(t *testing.T) {
	tests := []struct {
		name         string
		instanceName string
		setupMock    func(mux *http.ServeMux)
		wantErr      bool
	}{
		{
			name:         "successful_scale_down",
			instanceName: "test-instance",
			setupMock: func(mux *http.ServeMux) {
				// Mock GET instance status
				mux.HandleFunc("/projects/test-project/zones/test-zone/instances/test-instance", func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "GET" {
						w.Write([]byte(`{"status": "RUNNING", "name": "test-instance"}`))
					}
				})
				// Mock STOP instance
				mux.HandleFunc("/projects/test-project/zones/test-zone/instances/test-instance/stop", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(`{"name": "operation-123", "status": "DONE"}`))
				})
				// Mock operation status
				mux.HandleFunc("/projects/test-project/zones/test-zone/operations/operation-123", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(`{"status": "DONE"}`))
				})
			},
			wantErr: false,
		},
		{
			name:         "already_stopped",
			instanceName: "test-instance",
			setupMock: func(mux *http.ServeMux) {
				mux.HandleFunc("/projects/test-project/zones/test-zone/instances/test-instance", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(`{"status": "TERMINATED", "name": "test-instance"}`))
				})
			},
			wantErr: false,
		},
		{
			name:         "instance_not_found",
			instanceName: "nonexistent-instance",
			setupMock: func(mux *http.ServeMux) {
				mux.HandleFunc("/projects/test-project/zones/test-zone/instances/nonexistent-instance", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
					w.Write([]byte(`{"error": {"message": "Instance not found"}}`))
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			tt.setupMock(mux)

			svc, ts := setupMockService(t, mux)
			defer ts.Close()

			err := svc.ScaleDown(context.Background(), tt.instanceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ScaleDown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
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

			svc, ts := setupMockService(t, mux)
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

func TestScaleDownTimeout(t *testing.T) {
	mux := http.NewServeMux()

	// Mock an operation that never completes
	mux.HandleFunc("/projects/test-project/zones/test-zone/instances/test-instance", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Write([]byte(`{"status": "RUNNING", "name": "test-instance"}`))
		}
	})

	mux.HandleFunc("/projects/test-project/zones/test-zone/instances/test-instance/stop", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name": "operation-timeout", "status": "PENDING"}`))
	})

	mux.HandleFunc("/projects/test-project/zones/test-zone/operations/operation-timeout", func(w http.ResponseWriter, r *http.Request) {
		// Simulate a hanging operation by sleeping
		time.Sleep(2 * time.Second)
		w.Write([]byte(`{"status": "PENDING"}`))
	})

	svc, ts := setupMockService(t, mux)
	defer ts.Close()

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := svc.ScaleDown(ctx, "test-instance")
	if err == nil {
		t.Error("ScaleDown() should have timed out")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded error, got: %v", err)
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
				},
				ProjectID: "test-project",
				Zone:      "test-zone",
				Type:      "gcp",
			},
			wantErr: false,
		},
		{
			name: "missing project ID",
			config: &common.CloudServiceConfig{
				Credentials: &common.CredentialsConfig{
					Secret: mockCreds,
				},
				Zone: "test-zone",
				Type: "gcp",
			},
			wantErr:   true,
			errString: "projectID is required",
		},
		{
			name: "missing zone",
			config: &common.CloudServiceConfig{
				Credentials: &common.CredentialsConfig{
					Secret: mockCreds,
				},
				ProjectID: "test-project",
				Type:      "gcp",
			},
			wantErr:   true,
			errString: "zone is required",
		},
		{
			name:      "nil config",
			config:    nil,
			wantErr:   true,
			errString: "invalid provider config type for GCP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewService(tt.config)
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

			// Type assertion to access internal fields
			gcpSvc, ok := svc.(*Service)
			if !ok {
				t.Error("NewService() returned wrong type")
				return
			}

			// Verify service was properly initialized
			if gcpSvc.projectID == "" {
				t.Error("NewService() projectID not set")
			}
			if gcpSvc.zone == "" {
				t.Error("NewService() zone not set")
			}
			if gcpSvc.computeService == nil {
				t.Error("NewService() computeService not initialized")
			}
		})
	}
}
