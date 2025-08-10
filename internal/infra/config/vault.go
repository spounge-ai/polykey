package config

// VaultConfig represents the Vault configuration.
type VaultConfig struct {
	Address string `mapstructure:"address" validate:"required,url"`
	Token   string `mapstructure:"token"   validate:"required"`
}
