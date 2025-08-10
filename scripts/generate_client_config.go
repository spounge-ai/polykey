package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// clientData matches the structure expected by the FileClientStore
type clientData struct {
	HashedAPIKey string   `yaml:"hashed_api_key"`
	Permissions  []string `yaml:"permissions"`
	Description  string   `yaml:"description,omitempty"`
}

// config matches the top-level structure of the YAML file
type config struct {
	Clients map[string]clientData `yaml:"clients"`
}

const filePath = "configs/dev_client/config.client.dev.yaml"

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

	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating bcrypt hash: %v\n", err)
		os.Exit(1)
	}

	newConfig := config{
		Clients: map[string]clientData{
			clientID: {
				HashedAPIKey: string(hashedSecret),
				Permissions: []string{ // A default set of permissions for a new client
					"keys:create",
					"keys:read",
					"keys:list",
					"keys:rotate",
					"keys:revoke",
					"keys:update",
				},
				Description: description,
			},
		},
	}

	yamlData, err := yaml.Marshal(&newConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshalling YAML: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to file %s: %v\n", filePath, err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully generated '%s' for client '%s'.\n", filePath, clientID)
}
