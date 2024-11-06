package traefik_cloud_saver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseMetricLine(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantService   string
		wantCount     float64
		wantSucceeded bool
	}{
		{
			name:          "valid metric line",
			input:         `traefik_service_requests_total{service="my-service"} 123`,
			wantService:   "my-service",
			wantCount:     123,
			wantSucceeded: true,
		},
		{
			name:          "invalid format",
			input:         "invalid metric line",
			wantService:   "",
			wantCount:     0,
			wantSucceeded: false,
		},
		// Add more test cases here
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, count, ok := parseMetricLine(tt.input)
			if ok != tt.wantSucceeded {
				t.Errorf("parseMetricLine() succeeded = %v, want %v", ok, tt.wantSucceeded)
			}
			if service != tt.wantService {
				t.Errorf("parseMetricLine() service = %v, want %v", service, tt.wantService)
			}
			if count != tt.wantCount {
				t.Errorf("parseMetricLine() count = %v, want %v", count, tt.wantCount)
			}
		})
	}
}

func TestGetServiceRates(t *testing.T) {
	// Create a test server that returns mock metrics
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`
traefik_service_requests_total{service="service1"} 100
traefik_service_requests_total{service="service2"} 200
`))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Create a metrics collector with our test server
	mc := NewMetricsCollector(server.URL)

	// First call to establish baseline
	_, err := mc.GetServiceRates()
	if err != nil {
		t.Fatalf("First GetServiceRates() failed: %v", err)
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Second call to get rates
	rates, err := mc.GetServiceRates()
	if err != nil {
		t.Fatalf("Second GetServiceRates() failed: %v", err)
	}

	// Check results
	if len(rates) != 2 {
		t.Errorf("Expected 2 services, got %d", len(rates))
	}

	// Check if service1 exists
	if rate, exists := rates["service1"]; !exists {
		t.Error("service1 not found in rates")
	} else {
		if rate.Total != 100 {
			t.Errorf("service1 total = %v, want 100", rate.Total)
		}
	}
}

func TestFetchServiceRequests(t *testing.T) {
	// Test with empty response
	t.Run("empty response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		mc := NewMetricsCollector(server.URL)
		counts, err := mc.fetchServiceRequests()
		if err != nil {
			t.Errorf("fetchServiceRequests() error = %v", err)
		}
		if len(counts) != 0 {
			t.Errorf("Expected empty map, got %d entries", len(counts))
		}
	})

	// Test with valid metrics
	t.Run("valid metrics", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`
traefik_service_requests_total{service="service1"} 100
traefik_service_requests_total{service="service2"} 200
`))
			if err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer server.Close()

		mc := NewMetricsCollector(server.URL)
		counts, err := mc.fetchServiceRequests()
		if err != nil {
			t.Errorf("fetchServiceRequests() error = %v", err)
		}
		if len(counts) != 2 {
			t.Errorf("Expected 2 entries, got %d", len(counts))
		}
		if counts["service1"] != 100 {
			t.Errorf("service1 count = %v, want 100", counts["service1"])
		}
	})
} 