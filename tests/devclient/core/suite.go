package core

// TestSuite defines the interface for a collection of related tests.
type TestSuite interface {
	Name() string
	Run(tc TestClient) error
}
