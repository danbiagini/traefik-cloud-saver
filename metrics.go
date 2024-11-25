package traefik_cloud_saver

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
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

	common.DebugLog("traefik-cloud-saver", "Current counts: %v, Last counts: %v, Duration: %v", currentCounts, mc.lastCounts, duration)

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
			Total:       count,
			PerMin:      ratePerMin,
			Duration:    duration,
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
			common.LogProvider("traefik-cloud-saver", "[Error] closing response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics: %w", err)
	}

	// if the body is empty, lets log a warning and return an empty map
	if len(body) == 0 {
		common.LogProvider("traefik-cloud-saver", "[WARNING] Metrics response body is empty")
		return make(map[string]float64), nil
	}

	serviceCounts := make(map[string]float64)
	scanner := bufio.NewScanner(strings.NewReader(string(body)))

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "traefik_service_requests_total") {
			// Parse service name and count from the metric line.
			// Accumulate the count for each service if the response code is 200 or it has no response codes.
			// Example:
			// traefik_service_requests_total{service="servicename",method="GET",code="200"} 10
			// traefik_service_requests_total{service="servicename",method="POST",code="200"} 20
			// traefik_service_requests_total{service="servicename",method="GET",code="404"} 50
			// will be accumulated as:
			// serviceCounts["servicename"] = 30
			if service, count, ok := parseMetricLine(line); ok {
				serviceCounts[service] += count
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

		// Parse service name & response code
		if start := strings.Index(line, `service="`); start != -1 {
			start += len(`service="`)
			if end := strings.Index(line[start:], `"`); end != -1 {
				serviceName = line[start : start+end]

				// only return true count if the response code is 200 or it has no response codes
				if responseCode := strings.Index(line, `code="`); responseCode != -1 {
					code := line[responseCode+len(`code="`) : responseCode+len(`code="`)+3]
					if code != "200" && code != "" {
						return "", 0, false
					}
					return serviceName, count, true
				}
				// return true count if there is no response code
				return serviceName, count, true
			}
		}
	}

	return "", 0, false
}
