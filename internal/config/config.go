// internal/config/config.go
package config

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// Config holds application configuration
type Config struct {
	ServerAddress string
	Timeout       time.Duration
	LogLevel      string
	Environment   string
}

// RuntimeEnvironment represents where the application is running
type RuntimeEnvironment int

const (
	RuntimeLocal RuntimeEnvironment = iota
	RuntimeDocker
	RuntimeKubernetes
	RuntimeContainerd
	RuntimePodman
)

func (r RuntimeEnvironment) String() string {
	switch r {
	case RuntimeLocal:
		return "local"
	case RuntimeDocker:
		return "docker"
	case RuntimeKubernetes:
		return "kubernetes"
	case RuntimeContainerd:
		return "containerd"
	case RuntimePodman:
		return "podman"
	default:
		return "unknown"
	}
}

// RuntimeDetector detects the current runtime environment
type RuntimeDetector struct{}

func NewRuntimeDetector() *RuntimeDetector {
	return &RuntimeDetector{}
}

func (rd *RuntimeDetector) DetectRuntime() RuntimeEnvironment {
	// Check Kubernetes first (most specific)
	if rd.isKubernetes() {
		return RuntimeKubernetes
	}

	// Check container runtimes
	if rd.isPodman() {
		return RuntimePodman
	}

	if rd.isContainerd() {
		return RuntimeContainerd
	}

	if rd.isDocker() {
		return RuntimeDocker
	}

	return RuntimeLocal
}

func (rd *RuntimeDetector) isKubernetes() bool {
	// Check for Kubernetes service account
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); err == nil {
		return true
	}

	// Check environment variables
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

func (rd *RuntimeDetector) isDocker() bool {
	// Check for .dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for docker
	return rd.checkCgroup("docker")
}

func (rd *RuntimeDetector) isContainerd() bool {
	return rd.checkCgroup("containerd")
}

func (rd *RuntimeDetector) isPodman() bool {
	// Check for podman-specific environment
	if os.Getenv("container") == "podman" {
		return true
	}

	return rd.checkCgroup("podman")
}

func (rd *RuntimeDetector) checkCgroup(runtime string) bool {
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}

	content := string(data)
	return strings.Contains(content, runtime) || strings.Contains(content, "/"+runtime+"/")
}

type ConfigLoader struct {
	Detector *RuntimeDetector // <-- Capitalize this field
}

func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{
		Detector: NewRuntimeDetector(), // <-- Update the assignment
	}
}

// ... (Load, loadFromFlags, loadFromEnv are unchanged)

func (cl *ConfigLoader) detectServerAddress() string {
	runtime := cl.Detector.DetectRuntime() // <-- Update this usage

	switch runtime {
	case RuntimeKubernetes:
		// In Kubernetes, use service name
		return "polykey-service:50051"
	case RuntimeDocker:
		// In Docker Compose, use service name
		return "polykey-server:50051"
	case RuntimeContainerd, RuntimePodman:
		// Similar to Docker
		return "polykey-server:50051"
	default:
		// Local development
		if cl.isDockerHostReachable() {
			return "localhost:50051"
		}
		return "localhost:50051"
	}
}

func (cl *ConfigLoader) Load() (*Config, error) {
	config := &Config{
		Timeout:   5 * time.Second,
		LogLevel:  "info",
		Environment: "development",
	}

	// Load from flags
	cl.loadFromFlags(config)

	// Load from environment variables (higher priority)
	cl.loadFromEnv(config)

	// Auto-detect server address if not set
	if config.ServerAddress == "" {
		config.ServerAddress = cl.detectServerAddress()
	}

	return config, nil
}

func (cl *ConfigLoader) loadFromFlags(config *Config) {
	flag.StringVar(&config.ServerAddress, "server", "", "gRPC server address")
	flag.DurationVar(&config.Timeout, "timeout", config.Timeout, "Connection timeout")
	flag.StringVar(&config.LogLevel, "log-level", config.LogLevel, "Log level")
	flag.StringVar(&config.Environment, "env", config.Environment, "Environment")
	flag.Parse()
}

func (cl *ConfigLoader) loadFromEnv(config *Config) {
	if addr := os.Getenv("POLYKEY_SERVER_ADDR"); addr != "" {
		config.ServerAddress = addr
	}

	if timeout := os.Getenv("POLYKEY_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			config.Timeout = d
		}
	}

	if level := os.Getenv("POLYKEY_LOG_LEVEL"); level != "" {
		config.LogLevel = level
	}

	if env := os.Getenv("POLYKEY_ENV"); env != "" {
		config.Environment = env
	}
}

func (cl *ConfigLoader) isDockerHostReachable() bool {
	// Test connection to common Docker host addresses
	addresses := []string{"host.docker.internal:50051", "localhost:50051"}
	
	for _, addr := range addresses {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

// NetworkTester helps test network connectivity
type NetworkTester struct{}

func NewNetworkTester() *NetworkTester {
	return &NetworkTester{}
}

func (nt *NetworkTester) TestConnection(ctx context.Context, address string) error {
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	defer conn.Close()
	return nil
}