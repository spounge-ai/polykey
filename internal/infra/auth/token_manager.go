package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spounge-ai/polykey/internal/domain"
)

// TokenManager manages JWT token generation and validation using RSA keys.
type TokenManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	tokenStore TokenStore
	auditLogger domain.AuditLogger
}

// NewTokenManager creates a new TokenManager from a PEM-encoded RSA private key.
func NewTokenManager(privateKeyPEM string, tokenStore TokenStore, auditLogger domain.AuditLogger) (*TokenManager, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
	}

	return &TokenManager{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		tokenStore: tokenStore,
		auditLogger: auditLogger,
	}, nil
}

// GenerateToken generates a new JWT token signed with RS256.
func (tm *TokenManager) GenerateToken(userID string, roles []string, expiration time.Duration) (string, error) {
	expirationTime := time.Now().Add(expiration)
	claims := &Claims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(tm.privateKey)
}

// ValidateToken validates a JWT token signed with RS256 and checks if it has been revoked.
func (tm *TokenManager) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	if tm.tokenStore.IsRevoked(ctx, claims.ID) {
		return nil, fmt.Errorf("token has been revoked")
	}

	return claims, nil
}

// Revoke adds a token to the revocation list.
// It parses the token to get its ID and expiration, then adds the ID to the token store
// with a TTL equal to the token's remaining validity.
func (tm *TokenManager) Revoke(ctx context.Context, tokenString string) error {
	// We use ParseUnverified because we don't need to validate the signature to revoke a token.
	// We only need the claims to get the token ID (jti) and expiration time.
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return fmt.Errorf("failed to parse token for revocation: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return fmt.Errorf("invalid claims type in token")
	}

	// If the token has no expiration, we cannot revoke it effectively.
	if claims.ExpiresAt == nil {
		return fmt.Errorf("cannot revoke token with no expiration")
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		// Token is already expired, no need to add to revocation list.
		return nil
	}

	tm.tokenStore.Revoke(ctx, claims.ID, ttl)
	tm.auditLogger.AuditLog(ctx, claims.UserID, "RevokeToken", claims.ID, "", true, nil)
	return nil
}
