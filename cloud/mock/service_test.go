package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
)

func TestMockProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("basic scaling operations", func(t *testing.T) {
		config := &common.CloudServiceConfig{
			Type: "mock",
		}

		provider, err := New(config)
		if err != nil {
			t.Fatalf("Failed to create mock provider: %v", err)
		}

		// Test initial scale up
		serviceName := "test-service"
		err = provider.ScaleUp(ctx, serviceName)
		if err != nil {
			t.Errorf("ScaleUp failed: %v", err)
		}

		scale, err := provider.GetCurrentScale(ctx, serviceName)
		if err != nil {
			t.Errorf("GetCurrentScale failed: %v", err)
		}
		if scale != 1 {
			t.Errorf("expected scale 1, got %d", scale)
		}

		// Test scale down
		err = provider.ScaleDown(ctx, serviceName)
		if err != nil {
			t.Errorf("ScaleDown failed: %v", err)
		}

		scale, err = provider.GetCurrentScale(ctx, serviceName)
		if err != nil {
			t.Errorf("GetCurrentScale failed: %v", err)
		}
		if scale != 0 {
			t.Errorf("expected scale 0, got %d", scale)
		}
	})

	t.Run("error simulation", func(t *testing.T) {
		expectedErr := errors.New("simulated error")
		config := &common.CloudServiceConfig{
			Type: "mock",
		}

		provider, err := New(config, WithScaleError(expectedErr))
		if err != nil {
			t.Fatal(err)
		}

		err = provider.ScaleUp(ctx, "test-service")
		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("concurrent access", func(t *testing.T) {
		config := &common.CloudServiceConfig{
			Type: "mock",
		}

		provider, err := New(config)
		if err != nil {
			t.Fatal(err)
		}

		serviceName := "concurrent-service"
		provider.SetScale(serviceName, 5) // Now using the concrete type

		// Run concurrent scale operations
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				_ = provider.ScaleUp(ctx, serviceName)
				_ = provider.ScaleDown(ctx, serviceName)
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Final scale should still be 5
		scale, err := provider.GetCurrentScale(ctx, serviceName)
		if err != nil {
			t.Errorf("GetCurrentScale failed: %v", err)
		}
		if scale != 5 {
			t.Errorf("expected scale 5, got %d", scale)
		}
	})
}
