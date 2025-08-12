package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/spounge-ai/polykey/internal/domain"
)

// Claims represents the JWT claims.

type Claims struct {
	UserID string         `json:"user_id"`
	Roles  []string       `json:"roles"`
	Tier   domain.KeyTier `json:"tier"`
	jwt.RegisteredClaims
}
