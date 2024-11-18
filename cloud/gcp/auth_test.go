package gcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenManager_GetToken(t *testing.T) {
	// Mock server to simulate Google's token endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}

		resp := TokenResponse{
			AccessToken: "test-token",
			ExpiresIn:   3600,
			TokenType:   "Bearer",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	tests := []struct {
		name    string
		setup   func(*TokenManager)
		want    string
		wantErr bool
	}{
		{
			name: "first token fetch",
			setup: func(tm *TokenManager) {
				// No setup needed - fresh manager
			},
			want:    "test-token",
			wantErr: false,
		},
		{
			name: "use cached token",
			setup: func(tm *TokenManager) {
				tm.currentToken = &TokenResponse{AccessToken: "cached-token"}
				tm.expiresAt = time.Now().Add(time.Hour)
			},
			want:    "cached-token",
			wantErr: false,
		},
		{
			name: "refresh expired token",
			setup: func(tm *TokenManager) {
				tm.currentToken = &TokenResponse{AccessToken: "expired-token"}
				tm.expiresAt = time.Now().Add(-time.Hour)
			},
			want:    "test-token",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := testTokenManager(server)
			if err != nil {
				t.Fatalf("NewTokenManager() error = %v", err)
			}
			if tt.setup != nil {
				tt.setup(tm)
			}

			got, err := tm.GetToken(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("TokenManager.GetToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TokenManager.GetToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenManager_Concurrent(t *testing.T) {
	// Mock server with artificial delay to test concurrent requests
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		time.Sleep(100 * time.Millisecond) // Artificial delay
		w.WriteHeader(http.StatusOK)
		resp := TokenResponse{
			AccessToken: "test-token",
			ExpiresIn:   3600,
			TokenType:   "Bearer",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Use the testhelpers credentials instead of defining new ones
	tm, err := testTokenManager(server)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}

	// Force token to be expired
	tm.currentToken = &TokenResponse{AccessToken: "expired-token"}
	tm.expiresAt = time.Now().Add(-time.Hour)

	// Launch multiple concurrent requests
	const numRequests = 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			token, err := tm.GetToken(context.Background())
			if err != nil {
				t.Errorf("Concurrent GetToken() error = %v", err)
			}
			if token != "test-token" {
				t.Errorf("Concurrent GetToken() = %v, want test-token", token)
			}
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}

	// Should only have made one request to the server
	if requestCount != 1 {
		t.Errorf("Expected 1 request to server, got %d", requestCount)
	}
}
