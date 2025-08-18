package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// clientSecretConfig matches the structure expected by the dev client
type clientSecretConfig struct {
	ID     string `yaml:"id"`
	Secret string `yaml:"secret"`
}

// serverClientData matches the structure expected by the FileClientStore
type serverClientData struct {
	HashedAPIKey string   `yaml:"hashed_api_key"`
	Permissions  []string `yaml:"permissions"`
	Description  string   `yaml:"description,omitempty"`
}

// serverConfig matches the top-level structure of the server-side client file
type serverConfig struct {
	Clients map[string]serverClientData `yaml:"clients"`
}

const serverConfigPath = "configs/dev_client/config.client.dev.yaml"
const clientSecretPath = "configs/dev_client/secret.dev.yaml"

func main() {
	if len(os.Args) < 3 || len(os.Args) > 4 {
		fmt.Fprintln(os.Stderr, "Usage: go run scripts/generate_client_config.go <client_id> <client_secret> [description]")
		os.Exit(1)
	}

	clientID := os.Args[1]
	clientSecret := os.Args[2]
	var description string
	if len(os.Args) == 4 {
		description = os.Args[3]
	}

	if err := generateServerConfig(clientID, clientSecret, description); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating server config: %v\n", err)
		os.Exit(1)
	}

	if err := generateClientSecret(clientID, clientSecret); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating client secret: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully generated configs for client '%s'.\n", clientID)
}

func generateServerConfig(clientID, clientSecret, description string) error {
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("generating bcrypt hash: %w", err)
	}

	newConfig := serverConfig{
		Clients: map[string]serverClientData{
			clientID: {
				HashedAPIKey: string(hashedSecret),
				Permissions: []string{"*"}, // Give dev client all permissions
				Description:  description,
			},
		},
	}

	yamlData, err := yaml.Marshal(&newConfig)
	if err != nil {
		return fmt.Errorf("marshalling YAML: %w", err)
	}

	return os.WriteFile(serverConfigPath, yamlData, 0644)
}


func generateClientSecret(clientID, clientSecret string) error {
	secret := clientSecretConfig{
		ID:     clientID,
		Secret: clientSecret,
	}

	yamlData, err := yaml.Marshal(&secret)
	if err != nil {
		return fmt.Errorf("marshalling YAML: %w", err)
	}

	return os.WriteFile(clientSecretPath, yamlData, 0644)
}