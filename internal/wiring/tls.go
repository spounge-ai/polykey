package wiring

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/spounge-ai/polykey/internal/infra/config"
	"gopkg.in/yaml.v3"
)

// ClientTLSConfig holds the paths for client-side TLS assets.
// This is loaded from a separate client-specific config file.
type ClientTLSConfig struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
}

// ConfigureTLS creates a new tls.Config from the given TLS configuration.
// It handles loading the server certificate, client CA, and setting the client auth policy.
func ConfigureTLS(cfg config.TLS) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	serverCert, err := tls.X509KeyPair([]byte(cfg.CertFile), []byte(cfg.KeyFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load server TLS key pair: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		MinVersion:   tls.VersionTLS12,
	}

	if cfg.ClientCAFile != "" {
		caCert := []byte(cfg.ClientCAFile)
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

// ConfigureClientTLS creates a new tls.Config for a gRPC client.
// It loads the client's certificate, key, and the server's CA certificate.
func ConfigureClientTLS(path string) (*tls.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read client TLS config %s: %w", path, err)
	}

	var tlsFileCfg ClientTLSConfig
	if err := yaml.Unmarshal(data, &tlsFileCfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal client TLS config: %w", err)
	}

	clientCert, err := tls.LoadX509KeyPair(tlsFileCfg.CertFile, tlsFileCfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client key pair: %w", err)
	}

	caCert, err := os.ReadFile(tlsFileCfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read server CA file: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add server CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
