package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/danbiagini/traefik-cloud-saver/cloud/common"
)

const (
	tokenEndpoint = "https://oauth2.googleapis.com/token"
	scope         = "https://www.googleapis.com/auth/compute"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type Credentials struct {
	Type         string `json:"type"`
	ClientEmail  string `json:"client_email"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	TokenURL     string `json:"token_uri"`
	ProjectID    string `json:"project_id"`
}

type TokenManager struct {
	credentials  *Credentials
	currentToken *TokenResponse
	expiresAt    time.Time
	mu           sync.Mutex
	client       *http.Client
	signer       *common.JWTSigner
}

func NewTokenManager(credentials *Credentials) (*TokenManager, error) {
	signer, err := common.NewJWTSigner(credentials.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT signer: %w", err)
	}

	return &TokenManager{
		credentials: credentials,
		client:      &http.Client{},
		signer:      signer,
	}, nil
}

func (tm *TokenManager) GetToken(ctx context.Context) (string, error) {
	// Check if current token is valid
	if tm.currentToken != nil && time.Now().Before(tm.expiresAt) {
		return tm.currentToken.AccessToken, nil
	}

	// Need to refresh
	token, err := tm.fetchToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch token: %w", err)
	}
	return token, nil
}

func (tm *TokenManager) fetchToken(ctx context.Context) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Double-check after acquiring lock
	if tm.currentToken != nil && time.Now().Before(tm.expiresAt) {
		return tm.currentToken.AccessToken, nil
	}

	now := time.Now()
	claims := map[string]interface{}{
		"iss":   tm.credentials.ClientEmail,
		"scope": scope,
		"aud":   tm.credentials.TokenURL,
		"exp":   now.Add(time.Hour).Unix(),
		"iat":   now.Unix(),
	}

	jwt, err := tm.signer.SignClaims(claims)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	data := strings.NewReader(fmt.Sprintf(
		"grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&assertion=%s",
		jwt,
	))

	req, err := http.NewRequestWithContext(ctx, "POST", tm.credentials.TokenURL, data)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := tm.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("received empty access token")
	}

	tm.currentToken = &tokenResp
	tm.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return tokenResp.AccessToken, nil
}
