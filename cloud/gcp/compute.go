package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"
)

const computeBasePath = "https://compute.googleapis.com/compute/v1"

type ComputeClient struct {
	client       *http.Client
	baseURL      string
	authToken    string
	timeout      time.Duration
	pollInterval time.Duration
}

// Instance represents a GCP compute instance
type Instance struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Operation represents a GCP compute operation
type Operation struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  *struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	} `json:"error,omitempty"`
}

func NewComputeClient(baseURL *string, authToken *string, timeout *time.Duration) (*ComputeClient, error) {
	base := computeBasePath
	if baseURL != nil {
		base = *baseURL
	}

	t := 5 * time.Minute // default timeout
	if timeout != nil {
		t = *timeout
	}

	// Require explicit auth token
	if authToken == nil || *authToken == "" {
		return nil, fmt.Errorf("auth token is required")
	}
	auth := *authToken

	client := &http.Client{}

	return &ComputeClient{
		client:       client,
		baseURL:      base,
		authToken:    auth,
		timeout:      t,
		pollInterval: 10 * time.Second,
	}, nil
}

func (c *ComputeClient) doRequest(ctx context.Context, method, urlPath string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := fmt.Sprintf("%s/%s", c.baseURL, urlPath)

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *ComputeClient) GetInstance(ctx context.Context, projectID, zone, instance string) (*Instance, error) {
	urlPath := path.Join("projects", projectID, "zones", zone, "instances", instance)

	respBody, err := c.doRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, err
	}

	var result Instance
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance response: %w", err)
	}

	return &result, nil
}

// StopInstance stops the instance and waits for it to reach a terminal state
func (c *ComputeClient) StopInstance(ctx context.Context, projectID, zone, instance string) (*Operation, error) {
	urlPath := path.Join("projects", projectID, "zones", zone, "instances", instance, "stop")

	respBody, err := c.doRequest(ctx, http.MethodPost, urlPath, nil)
	if err != nil {
		return nil, err
	}

	var operation Operation
	if err := json.Unmarshal(respBody, &operation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation response: %w", err)
	}

	if operation.Error != nil && len(operation.Error.Errors) > 0 {
		return nil, fmt.Errorf("operation failed: %s", operation.Error.Errors[0].Message)
	}

	// Wait for the instance to stop
	timeout := time.After(c.timeout)
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for instance to stop")
		case <-ticker.C:
			instance, err := c.GetInstance(ctx, projectID, zone, instance)
			if err != nil {
				return nil, fmt.Errorf("failed to get instance status: %w", err)
			}
			if instance.Status == "TERMINATED" || instance.Status == "STOPPED" {
				operation.Status = instance.Status
				return &operation, nil
			}
		}
	}
}

func (c *ComputeClient) GetOperation(ctx context.Context, projectID, zone, operation string) (*Operation, error) {
	urlPath := path.Join("projects", projectID, "zones", zone, "operations", operation)

	respBody, err := c.doRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, err
	}

	var result Operation
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation response: %w", err)
	}

	return &result, nil
}
