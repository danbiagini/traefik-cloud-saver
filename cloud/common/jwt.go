package common

import (
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSigner handles JWT token creation and signing
type JWTSigner struct {
	privateKey *rsa.PrivateKey
	method     *jwt.SigningMethodRSA
}

// NewJWTSigner creates a new JWT signer from a PEM-encoded private key
func NewJWTSigner(privateKeyPEM string) (*JWTSigner, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &JWTSigner{
		privateKey: privateKey,
		method:     jwt.SigningMethodRS256,
	}, nil
}

// SignClaims creates and signs a JWT with the provided claims
func (s *JWTSigner) SignClaims(claims map[string]interface{}) (string, error) {
	token := jwt.New(s.method)
	token.Claims = jwt.MapClaims(claims)

	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
