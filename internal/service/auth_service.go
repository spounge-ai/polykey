package service

import (
	"context"
	"fmt"
	"time"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/infra/auth"
	"golang.org/x/crypto/bcrypt"
)

// AuthenticationResult is a domain-specific struct to hold the result of an authentication attempt.
// This decouples the service layer from the transport layer's protobuf types.
type AuthenticationResult struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int64
}

// AuthService defines the interface for the authentication business logic.
type AuthService interface {
	Authenticate(ctx context.Context, clientID, clientSecret string) (*AuthenticationResult, error)
}

type authService struct {
	clientStore  domain.ClientStore
	tokenManager *auth.TokenManager
	tokenTTL     time.Duration
}

// NewAuthService creates a new authentication service.
func NewAuthService(clientStore domain.ClientStore, tokenManager *auth.TokenManager, tokenTTL time.Duration) AuthService {
	return &authService{
		clientStore:  clientStore,
		tokenManager: tokenManager,
		tokenTTL:     tokenTTL,
	}
}

// Authenticate verifies client credentials and issues a JWT upon success.
func (s *authService) Authenticate(ctx context.Context, clientID, clientSecret string) (*AuthenticationResult, error) {
	client, err := s.clientStore.FindClientByID(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err) // Consider a more generic error type here
	}

	err = bcrypt.CompareHashAndPassword([]byte(client.HashedAPIKey), []byte(clientSecret))
	if err != nil {
		return nil, fmt.Errorf("authentication failed: invalid credentials")
	}

	accessToken, err := s.tokenManager.GenerateToken(client.ID, client.Permissions, s.tokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthenticationResult{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.tokenTTL.Seconds()),
	}, nil
}
