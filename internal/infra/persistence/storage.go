package storage

// Storage defines the interface for key storage operations.
type Storage interface {
	Get(path string) (map[string]interface{}, error)
	Put(path string, data map[string]interface{}) error
	Delete(path string) error
	List(path string) ([]string, error)
	HealthCheck() error
}
