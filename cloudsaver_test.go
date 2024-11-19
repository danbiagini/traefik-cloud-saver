package traefik_cloud_saver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/traefik/genconf/dynamic"
)

func TestNew(t *testing.T) {
	// Create a mock API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/http/routers" {
			// Return empty router configuration
			json.NewEncoder(w).Encode([]*TraefikRouter{})
		} else if r.URL.Path == "/metrics" {
			// Return empty Prometheus metrics
			w.Write([]byte("# Empty metrics for testing\n"))
		} else {
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if server == nil {
		t.Fatal("server is nil")
	}
	// expect an empty configuration
	expected := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers:           make(map[string]*dynamic.Router),
			Services:          make(map[string]*dynamic.Service),
			Middlewares:       make(map[string]*dynamic.Middleware),
			ServersTransports: make(map[string]*dynamic.ServersTransport),
		},
	}

	expectedJSON, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	config := CreateConfig()
	config.WindowSize = "1s"
	config.testMode = true

	provider, err := New(context.Background(), config, "test")
	if err != nil {
		t.Fatal(err)
	}
	provider.apiURL = server.URL + "/api"
	provider.metricsCollector.metricsURL = server.URL + "/metrics"

	t.Cleanup(func() {
		err = provider.Stop()
		if err != nil {
			t.Fatal(err)
		}
	})

	err = provider.Init()
	if err != nil {
		t.Fatal(err)
	}

	cfgChan := make(chan json.Marshaler)

	err = provider.Provide(cfgChan)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case data := <-cfgChan:
		dataJSON, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(expectedJSON, dataJSON) {
			t.Fatalf("got %s, want: %s", string(dataJSON), string(expectedJSON))
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for configuration")
	}

}

func TestRealWorldResponse(t *testing.T) {
	realWorldResponse := `
[
  {
    "entryPoints": [
      "traefik"
    ],
    "service": "api@internal",
    "rule": "PathPrefix(` + "`" + `/api` + "`" + `)",
    "priority": 2147483646,
    "status": "enabled",
    "using": [
      "traefik"
    ],
    "name": "api@internal",
    "provider": "internal"
  },
  {
    "entryPoints": [
      "web"
    ],
    "service": "whoami",
    "rule": "Host(` + "`" + `whoami.localhost` + "`" + `)",
    "status": "enabled",
    "using": [
      "web"
    ],
    "name": "whoami@docker",
    "provider": "docker"
  }
]`

	t.Run("real world response", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(realWorldResponse))
		}))
		defer server.Close()

		// Create CloudSaver instance
		config := CreateConfig()
		config.WindowSize = "1s"
		config.testMode = true

		saver, err := New(context.Background(), config, "test-routers-from-api")
		if err != nil {
			t.Fatal(err)
		}

		saver.apiURL = server.URL + "/api"

		// Call getRoutersFromAPI directly
		routers, err := saver.getRoutersFromAPI()

		// Check error
		if err != nil {
			t.Errorf("getRoutersFromAPI() error = %v", err)
			return
		}

		// If no error expected, verify response
		if len(routers) != 2 {
			t.Errorf("Expected %d routers, got %d", 2, len(routers))
		}

		expectedRouters := []*TraefikRouter{
			{
				Name:     "whoami@docker",
				Service:  "whoami",
				Provider: "docker",
				Status:   "enabled",
			},
			{
				Name:     "api@internal",
				Service:  "api@internal",
				Provider: "internal",
				Status:   "enabled",
			},
		}
		// let's validate that each router in the response is present in the actual routers
		for _, expectedResponse := range expectedRouters {
			// check that the router is present in the actual routers
			actualRouter, exists := routers[expectedResponse.Name]
			if !exists {
				t.Errorf("Expected router %s not found", expectedResponse.Name)
				continue
			}
			if actualRouter.Service != expectedResponse.Service {
				t.Errorf("Router %s: expected service %s, got %s",
					expectedResponse.Name, expectedResponse.Service, actualRouter.Service)
			}
		}
	})
}

func TestGetRoutersFromAPI(t *testing.T) {

	tests := []struct {
		name           string
		apiResponse    []*TraefikRouter
		expectedError  bool
		mockServerFunc func(http.ResponseWriter, *http.Request)
	}{
		{
			name: "valid routers response",
			apiResponse: []*TraefikRouter{
				{
					Name:     "router1@docker",
					Service:  "service1",
					Provider: "docker",
					Status:   "enabled",
				},
				{
					Name:     "router2@docker",
					Service:  "service2",
					Provider: "docker",
					Status:   "enabled",
				},
			},
			expectedError: false,
			mockServerFunc: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/http/routers" {
					t.Errorf("Expected path /api/http/routers, got %s", r.URL.Path)
				}
				json.NewEncoder(w).Encode([]*TraefikRouter{
					{
						Name:     "router1@docker",
						Service:  "service1",
						Provider: "docker",
						Status:   "enabled",
					},
					{
						Name:     "router2@docker",
						Service:  "service2",
						Provider: "docker",
						Status:   "enabled",
					},
				})
			},
		},
		{
			name:          "empty response",
			apiResponse:   []*TraefikRouter{},
			expectedError: false,
			mockServerFunc: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode([]*TraefikRouter{})
			},
		},
		{
			name:          "invalid response",
			expectedError: true,
			mockServerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("invalid json"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.mockServerFunc))
			defer server.Close()

			// Create CloudSaver instance
			config := CreateConfig()
			config.WindowSize = "1s"
			config.testMode = true

			saver, err := New(context.Background(), config, "test-routers-from-api")
			if err != nil {
				t.Fatal(err)
			}

			saver.apiURL = server.URL + "/api"

			// Call getRoutersFromAPI directly
			fmt.Println("Calling getRoutersFromAPI")
			routers, err := saver.getRoutersFromAPI()
			fmt.Println("getRoutersFromAPI returned", routers)

			// Check error
			if (err != nil) != tt.expectedError {
				t.Errorf("getRoutersFromAPI() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			// If no error expected, verify response
			if !tt.expectedError {
				if len(routers) != len(tt.apiResponse) {
					t.Errorf("Expected %d routers, got %d", len(tt.apiResponse), len(routers))
				}

				// let's validate that each router in the response is present in the actual routers
				for _, expectedResponse := range tt.apiResponse {
					// check that the router is present in the actual routers
					actualRouter, exists := routers[expectedResponse.Name]
					if !exists {
						t.Errorf("Expected router %s not found", expectedResponse.Name)
						continue
					}
					if actualRouter.Service != expectedResponse.Service {
						t.Errorf("Router %s: expected service %s, got %s",
							expectedResponse.Name, expectedResponse.Service, actualRouter.Service)
					}
				}
			}
		})
	}
}
