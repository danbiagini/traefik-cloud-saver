package gcp

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(handler http.HandlerFunc) (*httptest.Server, *ComputeClient) {
	mux := http.NewServeMux()

	// Handle token endpoint to match token_uri in credentials
	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
	})

	// Handle compute endpoints
	mux.HandleFunc("/compute/", func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Verify correct API version and endpoint format
		if !strings.HasPrefix(r.URL.Path, "/compute/v1/projects/") {
			log.Printf("Expected URL path to start with /compute/v1/projects/, got %s", r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Add typical GCP response headers
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-GoogApis-Metadata", "service=compute")

		// Call the handler for all requests
		handler(w, r)
	})

	server := httptest.NewServer(mux)

	// Create token manager using the same server
	tokenManager, err := testTokenManager(server)
	if err != nil {
		log.Fatalf("Failed to create token manager: %v", err)
	}

	var baseURL = server.URL + "/compute/v1"
	client, err := NewComputeClient(&baseURL, tokenManager, WithTimeout(5*time.Second))
	if err != nil {
		log.Fatalf("Failed to create compute client: %v", err)
	}

	return server, client
}

func TestGetInstance(t *testing.T) {
	tests := []struct {
		name       string
		projectID  string
		zone       string
		instance   string
		mockResp   *Instance
		mockStatus int
		wantErr    bool
	}{
		{
			name:      "successful_get_running",
			projectID: "test-project",
			zone:      "test-zone",
			instance:  "test-instance",
			mockResp: &Instance{
				Name:   "test-instance",
				Status: "RUNNING",
			},
			mockStatus: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set up the test server
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.mockStatus)
				json.NewEncoder(w).Encode(test.mockResp)
			}
			server, client := setupTestServer(handler)

			// Call the function being tested
			got, err := client.GetInstance(context.Background(), test.projectID, test.zone, test.instance)

			// Check the result
			if (err != nil) != test.wantErr {
				t.Errorf("GetInstance() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if got != nil && got.Name != test.mockResp.Name {
				t.Errorf("GetInstance() = %v, want %v", got, test.mockResp)
			}

			// Clean up
			server.Close()
		})
	}
}

func TestComputeClient_StopInstance(t *testing.T) {
	tests := []struct {
		name      string
		responses map[string]struct {
			status int
			body   string
		}
		expectedError string
		timeout       time.Duration
	}{
		{
			name: "successful stop",
			responses: map[string]struct {
				status int
				body   string
			}{
				"instances/instance-1/stop": {
					status: http.StatusOK,
					body:   `{"name": "operation-123"}`,
				},
				"operations/operation-123": {
					status: http.StatusOK,
					body:   `{"status": "DONE"}`,
				},
				"instances/instance-1": {
					status: http.StatusOK,
					body:   `{"name": "instance-1", "status": "TERMINATED"}`,
				},
			},
			timeout: 2 * time.Second,
		},
		{
			name: "error during status check",
			responses: map[string]struct {
				status int
				body   string
			}{
				"instances/instance-1/stop": {
					status: http.StatusOK,
					body:   `{"name": "operation-123"}`,
				},
				"operations/operation-123": {
					status: http.StatusForbidden,
					body:   `{"error": {"message": "request failed with status 403"}}`,
				},
				"instances/instance-1": {
					status: http.StatusOK,
					body:   `{"name": "instance-1", "status": "RUNNING"}`,
				},
			},
			expectedError: "request failed with status 403",
			timeout:       1 * time.Second,
		},
		{
			name: "error response from stop operation",
			responses: map[string]struct {
				status int
				body   string
			}{
				"instances/instance-1/stop": {
					status: http.StatusOK,
					body:   `{"name": "operation-123"}`,
				},
				"operations/operation-123": {
					status: http.StatusOK,
					body:   `{"status": "DONE"}`,
				},
				"instances/instance-1": {
					status: http.StatusNotFound,
					body:   `{"error": {"code": 404, "message": "Instance not found"}}`,
				},
			},
			expectedError: "Instance not found",
			timeout:       1 * time.Second,
		},
		{
			name: "timeout while stopping",
			responses: map[string]struct {
				status int
				body   string
			}{
				"instances/instance-1/stop": {
					status: http.StatusOK,
					body:   `{"name": "operation-123"}`,
				},
				"operations/operation-123": {
					status: http.StatusOK,
					body:   `{"status": "RUNNING"}`,
				},
				"instances/instance-1": {
					status: http.StatusOK,
					body:   `{"name": "instance-1", "status": "STOPPING"}`,
				},
			},
			expectedError: "context deadline exceeded",
			timeout:       100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				parts := strings.Split(r.URL.Path, "/compute/v1/projects/test-project/zones/test-zone/")
				if len(parts) != 2 {
					t.Logf("Invalid path format: %s", r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				pathSuffix := parts[1]

				response, exists := tt.responses[pathSuffix]
				if !exists {
					t.Logf("No response configured for path: %s", pathSuffix)
					w.WriteHeader(http.StatusNotFound)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(response.status)
				w.Write([]byte(response.body))
			}

			server, client := setupTestServer(handler)
			defer server.Close()

			// Override the timeout for this specific test
			client.timeout = tt.timeout
			client.pollInterval = 100 * time.Millisecond
			// Call StopInstance
			op, err := client.StopInstance(context.Background(), "test-project", "test-zone", "instance-1")

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
					return
				}
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, op)
			assert.Equal(t, "DONE", op.Status)
		})
	}
}
