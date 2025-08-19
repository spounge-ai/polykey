package devclient

import (
	"fmt"

	"github.com/spounge-ai/polykey/pkg/testutil"
	"github.com/spounge-ai/polykey/tests/devclient/core"
	"github.com/spounge-ai/polykey/tests/devclient/suites"
)

func Run(tc *testutil.Client) {
	fmt.Println("-- Running Dev Client Tests --")

	// The TestClient interface is now implemented by the testutil.Client.
	// We can cast it directly if needed, but the suites should accept the interface.
	var testClient core.TestClient = tc

	testSuites := []core.TestSuite{
		&suites.HappyPathSuite{},
		&suites.ErrorSuite{},
		&suites.BatchSuite{},
	}

	for _, s := range testSuites {
		fmt.Printf("-- Running Suite: %s ---\n", s.Name())
		if err := s.Run(testClient); err != nil {
			tc.Logger().Error("test suite failed", "suite", s.Name(), "error", err)
		}
	}

	fmt.Println("-- Dev Client Tests Finished --")
}
