package wiring

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/spounge-ai/polykey/internal/infra/config"
)

// ConfigureTLS creates a new tls.Config from the given TLS configuration.
// It handles loading the server certificate, client CA, and setting the client auth policy.
func ConfigureTLS(cfg config.TLS) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	serverCert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server TLS key pair: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		MinVersion:   tls.VersionTLS12,
	}

	if cfg.ClientCAFile != "" {
		caCert, err := os.ReadFile(cfg.ClientCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read client CA file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to add client CA certificate")
		}
		tlsConfig.ClientCAs = caCertPool
	}

	switch cfg.ClientAuth {
	case "RequestClientCert":
		tlsConfig.ClientAuth = tls.RequestClientCert
	case "RequireAnyClientCert":
		tlsConfig.ClientAuth = tls.RequireAnyClientCert
	case "VerifyClientCertIfGiven":
		tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
	case "RequireAndVerifyClientCert":
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	case "NoClientCert", "":
		tlsConfig.ClientAuth = tls.NoClientCert
	default:
		return nil, fmt.Errorf("unsupported client_auth type: %s", cfg.ClientAuth)
	}

	return tlsConfig, nil
}
