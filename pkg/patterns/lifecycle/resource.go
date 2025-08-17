package lifecycle

import "context"

// HealthStatus represents the health of a component.
type HealthStatus struct {
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

// ManagedResource defines a component with a managed lifecycle.
// This interface provides a standard way to start, stop, and check the health of application components.
type ManagedResource interface {
	// Start initializes and starts the component. It should be idempotent.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the component, releasing any resources. It should be idempotent.
	Stop(ctx context.Context) error

	// Health returns the current health status of the component.
	Health(ctx context.Context) HealthStatus
}
