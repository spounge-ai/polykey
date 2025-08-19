package testutil

import "time"

// Config holds the configuration for the test client.
type Config struct {
	ServerAddr       string
	SecretConfigPath string
	TLSConfigPath    string
	DefaultTimeout   time.Duration
}

// ClientSecretConfig holds the client ID and secret.
type ClientSecretConfig struct {
	ID     string `yaml:"id"`
	Secret string `yaml:"secret"`
}