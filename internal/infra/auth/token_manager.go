package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenManager manages JWT token generation and validation.
type TokenManager struct {
	secretKey string
}

// NewTokenManager creates a new TokenManager.
func NewTokenManager(secretKey string) *TokenManager {
	return &TokenManager{secretKey: secretKey}
}

// GenerateToken generates a new JWT token.
func (tm *TokenManager) GenerateToken(userID string, roles []string, expiration time.Duration) (string, error) {
	expirationTime := time.Now().Add(expiration)
	claims := &Claims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(tm.secretKey))
}

// ValidateToken validates a JWT token.
func (tm *TokenManager) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(tm.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}
