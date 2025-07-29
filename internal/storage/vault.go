package storage

import (
	"fmt"
	"log"

	"github.com/hashicorp/vault/api"
)

// VaultStorage implements the Storage interface for HashiCorp Vault.
type VaultStorage struct {
	client *api.Client
	address string
	token string
}

// NewVaultStorage creates a new VaultStorage instance.
func NewVaultStorage(address, token string) (Storage, error) {
	config := api.DefaultConfig()
	config.Address = address

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	client.SetToken(token)

	return &VaultStorage{
		client: client,
		address: address,
		token: token,
	}, nil
}

// Get retrieves data from Vault.
func (v *VaultStorage) Get(path string) (map[string]interface{}, error) {
	log.Printf("VaultStorage: Get called for path: %s", path)
	// Mock implementation
	if path == "secret/data/mock_key" {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"value": "mock_secret_value",
			},
		}, nil
	}
	return nil, fmt.Errorf("mock: secret not found at path: %s", path)
}

// Put writes data to Vault.
func (v *VaultStorage) Put(path string, data map[string]interface{}) error {
	log.Printf("VaultStorage: Put called for path: %s, data: %+v", path, data)
	// Mock implementation
	return nil
}

// Delete deletes data from Vault.
func (v *VaultStorage) Delete(path string) error {
	log.Printf("VaultStorage: Delete called for path: %s", path)
	// Mock implementation
	return nil
}

// List lists keys in Vault.
func (v *VaultStorage) List(path string) ([]string, error) {
	log.Printf("VaultStorage: List called for path: %s", path)
	// Mock implementation
	return []string{"mock_key_1", "mock_key_2"}, nil
}

// HealthCheck performs a health check on Vault.
func (v *VaultStorage) HealthCheck() error {
	log.Println("VaultStorage: HealthCheck called")
	// Mock implementation
	// In a real scenario, you would use v.client.Sys().Health() or similar
	return nil
}