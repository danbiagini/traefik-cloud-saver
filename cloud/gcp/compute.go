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

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
)

const computeBasePath = "https://compute.googleapis.com/compute/v1"

type ComputeClient struct {
	client       *http.Client
	baseURL      string
	tokenManager *TokenManager
	timeout      time.Duration
	pollInterval time.Duration
}

// Instance represents a GCP compute instance
type Instance struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ComputeClientOption func(*ComputeClient)

func WithTimeout(timeout time.Duration) ComputeClientOption {
	return func(c *ComputeClient) {
		c.timeout = timeout
	}
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

func NewComputeClient(baseURL *string, tokenManager *TokenManager, options ...ComputeClientOption) (*ComputeClient, error) {
	base := computeBasePath
	if baseURL != nil && *baseURL != "" {
		base = *baseURL
	}

	if tokenManager == nil {
		return nil, fmt.Errorf("token manager is required")
	}

	c := &ComputeClient{
		baseURL:      base,
		tokenManager: tokenManager,
		client:       &http.Client{},
		timeout:      5 * time.Minute,
		pollInterval: 10 * time.Second,
	}

	for _, option := range options {
		option(c)
	}

	return c, nil
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

	// Get token from token manager
	token, err := c.tokenManager.GetToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	common.DebugLog("traefik-cloud-saver", "Request: %s %s", req.Method, req.URL.Path)
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
		// Try to parse GCP error response
		var gcpError struct {
			Error struct {
				Message string `json:"message"`
				Errors  []struct {
					Message string `json:"message"`
				} `json:"errors"`
			} `json:"error"`
		}

		if err := json.Unmarshal(respBody, &gcpError); err == nil && gcpError.Error.Message != "" {
			return nil, fmt.Errorf("%s", gcpError.Error.Message)
		}

		// Fallback to simple error if can't parse GCP error format
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *ComputeClient) GetInstance(ctx context.Context, projectID, zone, instanceName string) (*Instance, error) {
	urlPath := path.Join("projects", projectID, "zones", zone, "instances", instanceName)

	resp, err := c.doRequest(ctx, http.MethodGet, urlPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	var result Instance
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance response: %w", err)
	}

	return &result, nil
}

// StopInstance stops the instance and waits for the operation to complete
func (c *ComputeClient) StopInstance(ctx context.Context, projectID, zone, instanceName string) (*Operation, error) {
	// First, make the stop request
	urlPath := path.Join("projects", projectID, "zones", zone, "instances", instanceName, "stop")
	respBody, err := c.doRequest(ctx, http.MethodPost, urlPath, nil)
	if err != nil {
		return nil, err
	}

	// Get the operation from the response
	var operation Operation
	if err := json.Unmarshal(respBody, &operation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation response: %w", err)
	}

	// Wait for the operation to complete using its name
	op, err := c.waitForOperation(ctx, projectID, zone, operation.Name)
	if err != nil {
		return nil, err
	}

	// Verify the instance state after the operation completes
	instance, err := c.GetInstance(ctx, projectID, zone, instanceName)
	if err != nil {
		return nil, err
	}

	if instance.Status != "TERMINATED" {
		return nil, fmt.Errorf("instance failed to stop: status is %s", instance.Status)
	}

	return op, nil
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

func (c *ComputeClient) waitForOperation(ctx context.Context, projectID, zone, operationName string) (*Operation, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for operation to complete: %w", ctx.Err())
		case <-ticker.C:
			urlPath := path.Join("projects", projectID, "zones", zone, "operations", operationName)

			respBody, err := c.doRequest(ctx, http.MethodGet, urlPath, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to get operation status: %w", err)
			}

			var operation Operation
			if err := json.Unmarshal(respBody, &operation); err != nil {
				return nil, fmt.Errorf("failed to decode operation response: %w", err)
			}

			if operation.Status == "DONE" {
				if operation.Error != nil {
					return nil, fmt.Errorf("operation failed: %v", operation.Error)
				}
				return &operation, nil
			}
		}
	}
}
