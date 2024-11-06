package traefik_cloud_saver

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"bufio"
)

// MetricsCollector handles all metrics-related operations
type MetricsCollector struct {
	client     *http.Client
	metricsURL string
	lastCounts map[string]float64
	lastTime   time.Time
}

type ServiceRate struct {
	ServiceName string
	Total       float64
	PerMin      float64
	Duration    time.Duration
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(url string) *MetricsCollector {
	return &MetricsCollector{
		client:     &http.Client{Timeout: 5 * time.Second},
		metricsURL: url,
		lastCounts: make(map[string]float64),
		lastTime:   time.Now(),
	}
}

// GetServiceRates fetches request rates for all services
func (mc *MetricsCollector) GetServiceRates() (map[string]*ServiceRate, error) {
	currentCounts, err := mc.fetchServiceRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch service metrics: %w", err)
	}
	
	now := time.Now()
	duration := now.Sub(mc.lastTime)
	rates := make(map[string]*ServiceRate)

	log.Printf("Current counts: %v, Last counts: %v, Duration: %v", currentCounts, mc.lastCounts, duration)

	for service, count := range currentCounts {
		var ratePerMin float64
		if len(mc.lastCounts) == 0 {
			// map is empty on first run - use total count divided by 1 minute as initial rate
			ratePerMin = count
		} else {
			lastCount := mc.lastCounts[service]
			requestDiff := count - lastCount
			if duration.Seconds() > 0 {
				ratePerMin = (requestDiff / duration.Seconds()) * 60
			}
		}

		rates[service] = &ServiceRate{
			ServiceName: service,
			Total:      count,
			PerMin:     ratePerMin,
			Duration:   duration,
		}
	}

	mc.lastCounts = currentCounts
	mc.lastTime = now

	return rates, nil
}

// fetchServiceRequests parses Prometheus metrics text format manually
func (mc *MetricsCollector) fetchServiceRequests() (map[string]float64, error) {
	resp, err := mc.client.Get(mc.metricsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metrics: %w", err)
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			log.Printf("failed to close response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics: %w", err)
	}

	// if the body is empty, lets log a warning and return an empty map
	if len(body) == 0 {
		log.Println("response body is empty")
		return make(map[string]float64), nil
	}

	serviceCounts := make(map[string]float64)
	scanner := bufio.NewScanner(strings.NewReader(string(body)))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "traefik_service_requests_total") {
			// Parse service name and count from the metric line
			// Format: traefik_service_requests_total{service="servicename"} 123
			if service, count, ok := parseMetricLine(line); ok {
				serviceCounts[service] = count
			}
		}
	}

	return serviceCounts, nil
}

// parseMetricLine extracts service name and count from a metric line
func parseMetricLine(line string) (string, float64, bool) {
	var serviceName string
	var count float64

	// Simple parsing of: traefik_service_requests_total{service="name"} 123
	if parts := strings.Split(line, " "); len(parts) == 2 {
		// Parse count
		_, err := fmt.Sscanf(parts[1], "%f", &count)
		if err != nil {
			return "", 0, false
		}

		// Parse service name
		if start := strings.Index(line, `service="`); start != -1 {
			start += len(`service="`)
			if end := strings.Index(line[start:], `"`); end != -1 {
				serviceName = line[start : start+end]
				return serviceName, count, true
			}
		}
	}

	return "", 0, false
} 