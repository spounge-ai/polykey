package devclient

import "time"

const (
	DefaultPort      = "50053"
	SecretConfigPath = "configs/dev_client/secret.dev.yaml"
	TLSConfigPath    = "configs/dev_client/tls.yaml"
	DefaultTimeout   = 30 * time.Second
	AuthHeader       = "authorization"
	BearerPrefix     = "Bearer "
	InvalidToken     = "this-is-not-a-valid-token"
)
