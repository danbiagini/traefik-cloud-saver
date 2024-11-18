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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		// Call the original handler
		handler(w, r)
	}))

	var authToken = "test-token"
	var baseURL = server.URL + "/compute/v1"
	client, err := NewComputeClient(&baseURL, &authToken, nil)
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
	// Setup test cases
	tests := []struct {
		name          string
		responses     []string // JSON responses from the server
		expectedError string
		timeout       time.Duration
	}{
		{
			name: "successful stop",
			responses: []string{
				`{"name": "operation-123", "status": "RUNNING"}`, // Stop operation response
				`{"name": "instance-1", "status": "STOPPING"}`,   // First status check
				`{"name": "instance-1", "status": "TERMINATED"}`, // Second status check
			},
			timeout: 1 * time.Second,
		},
		{
			name: "timeout while stopping",
			responses: []string{
				`{"name": "operation-123", "status": "RUNNING"}`,
				`{"name": "instance-1", "status": "STOPPING"}`,
				`{"name": "instance-1", "status": "STOPPING"}`,
			},
			expectedError: "timeout waiting for instance to stop",
			timeout:       100 * time.Millisecond,
		},
		{
			name: "error response from stop operation",
			responses: []string{
				`{"error": {"errors": [{"message": "Instance not found"}]}}`,
			},
			expectedError: "Instance not found",
			timeout:       1 * time.Second,
		},
		{
			name: "error during status check",
			responses: []string{
				`{"name": "operation-123", "status": "RUNNING"}`,
				`{
					"error": {
						"code": 403,
						"message": "The client does not have sufficient permission",
						"errors": [
							{
								"message": "The client does not have sufficient permission",
								"domain": "global",
								"reason": "forbidden"
							}
						],
						"status": "PERMISSION_DENIED"
					}
				}`,
			},
			expectedError: "failed to get instance status",
			timeout:       1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount int
			handler := func(w http.ResponseWriter, r *http.Request) {
				var response string
				if requestCount >= len(tt.responses) {
					response = tt.responses[len(tt.responses)-1]
				} else {
					response = tt.responses[requestCount]
				}
				// Check if this is an error response
				if strings.Contains(response, `"error"`) {
					var errorResp struct {
						Error struct {
							Code    int    `json:"code"`
							Message string `json:"message"`
						} `json:"error"`
					}
					if err := json.Unmarshal([]byte(response), &errorResp); err == nil && errorResp.Error.Code != 0 {
						w.WriteHeader(errorResp.Error.Code)
					} else {
						w.WriteHeader(http.StatusBadRequest)
					}
				}

				_, _ = w.Write([]byte(response))
				requestCount++
			}

			server, client := setupTestServer(handler)
			defer server.Close()

			// Override the timeout for this specific test
			client.timeout = tt.timeout
			client.pollInterval = 100 * time.Millisecond
			// Call StopInstance
			op, err := client.StopInstance(context.Background(), "test-project", "test-zone", "instance-1")

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, op)
			assert.Equal(t, "TERMINATED", op.Status)
		})
	}
}
