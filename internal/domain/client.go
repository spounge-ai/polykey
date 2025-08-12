package domain

import "context"

// Client represents a registered API client and its permissions.
type Client struct {
	ID           string   `yaml:"id"`
	HashedAPIKey string   `yaml:"hashed_api_key"`
	Permissions  []string `yaml:"permissions"`
	Tier         KeyTier  `yaml:"tier"`
}

// ClientStore defines the interface for retrieving client credentials.
// This abstraction allows for easy future migration to a database.
type ClientStore interface {
	FindClientByID(ctx context.Context, clientID string) (*Client, error)
}
