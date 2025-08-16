package config

// AuthorizationConfig represents the authorization configuration.
type AuthorizationConfig struct {
	Roles     map[string]RoleConfig `mapstructure:"roles"`
	ZeroTrust ZeroTrustConfig       `mapstructure:"zero_trust"`
}

// RoleConfig represents the role configuration.
type RoleConfig struct {
	AllowedOperations []string `mapstructure:"allowed_operations"`
}

// ZeroTrustConfig contains policies for zero-trust security.
type ZeroTrustConfig struct {
	EnforceMTLSIdentityMatch bool `mapstructure:"enforce_mtls_identity_match"`
}
