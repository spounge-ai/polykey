package auth

import (
	"context"
	"fmt"
	"os"

	"github.com/spounge-ai/polykey/internal/domain"
	"gopkg.in/yaml.v3"
)

var (
	ErrClientNotFound = fmt.Errorf("client not found")
	ErrInvalidConfig  = fmt.Errorf("invalid client configuration")
)

// clientConfig represents the YAML structure for client configuration
type clientConfig struct {
	Clients map[string]clientData `yaml:"clients"`
}

// clientData represents the YAML structure for individual client data
type clientData struct {
	HashedAPIKey string   `yaml:"hashed_api_key"`
	Permissions  []string `yaml:"permissions"`
	Description  string   `yaml:"description,omitempty"`
}

// FileClientStore implements the domain.ClientStore interface using a local YAML file.
// It holds an in-memory map of clients for fast O(1) lookups.
type FileClientStore struct {
	clients map[string]domain.Client
}

// NewFileClientStore creates and initializes a new FileClientStore from a given file path.
// It validates the configuration and pre-populates the in-memory store.
func NewFileClientStore(filePath string) (*FileClientStore, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read client config file %s: %w", filePath, err)
	}

	var config clientConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal client config: %w", err)
	}

	if len(config.Clients) == 0 {
		return nil, fmt.Errorf("%w: no clients defined", ErrInvalidConfig)
	}

	clients := make(map[string]domain.Client, len(config.Clients))
	for id, data := range config.Clients {
		if err := validateClientData(id, data); err != nil {
			return nil, fmt.Errorf("invalid client %s: %w", id, err)
		}

		clients[id] = domain.Client{
			ID:           id,
			HashedAPIKey: data.HashedAPIKey,
			Permissions:  data.Permissions,
		}
	}

	return &FileClientStore{clients: clients}, nil
}

// FindClientByID finds a client by its ID in the in-memory map.
// Returns ErrClientNotFound if the client doesn't exist.
func (s *FileClientStore) FindClientByID(ctx context.Context, clientID string) (*domain.Client, error) {
	client, exists := s.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("%w: client ID '%s'", ErrClientNotFound, clientID)
	}

	// Return a copy to prevent external modification
	return &domain.Client{
		ID:           client.ID,
		HashedAPIKey: client.HashedAPIKey,
		Permissions:  append([]string(nil), client.Permissions...),
	}, nil
}

// GetClientCount returns the number of configured clients.
// Useful for monitoring and health checks.
func (s *FileClientStore) GetClientCount() int {
	return len(s.clients)
}

// validateClientData validates individual client configuration
func validateClientData(id string, data clientData) error {
	if id == "" {
		return fmt.Errorf("client ID cannot be empty")
	}
	if data.HashedAPIKey == "" {
		return fmt.Errorf("hashed_api_key cannot be empty")
	}
	if len(data.Permissions) == 0 {
		return fmt.Errorf("permissions cannot be empty")
	}

	// Validate bcrypt hash format (starts with $2a$, $2b$, or $2y$)
	if len(data.HashedAPIKey) < 60 || (data.HashedAPIKey[:4] != "$2a$" &&
		data.HashedAPIKey[:4] != "$2b$" && data.HashedAPIKey[:4] != "$2y$") {
		return fmt.Errorf("hashed_api_key must be a valid bcrypt hash")
	}
	return nil
}
