package config

// AuthorizationConfig represents the authorization configuration.
type AuthorizationConfig struct {
	Roles map[string]RoleConfig `mapstructure:"roles"`
}

// RoleConfig represents the role configuration.
type RoleConfig struct {
	AllowedOperations []string `mapstructure:"allowed_operations"`
}
