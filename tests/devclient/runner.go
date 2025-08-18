package devclient

import (
	"fmt"

	"github.com/spounge-ai/polykey/tests/devclient/core"
	"github.com/spounge-ai/polykey/tests/devclient/suites"
)

func Run(tc *PolykeyTestClient) {
	fmt.Println("-- Running Dev Client Tests --")

	testSuites := []core.TestSuite{
		&suites.HappyPathSuite{},
		&suites.ErrorSuite{},
		&suites.BatchSuite{},
	}

	for _, s := range testSuites {
		fmt.Printf("-- Running Suite: %s ---\n", s.Name())
		if err := s.Run(tc); err != nil {
			tc.Logger().Error("test suite failed", "suite", s.Name(), "error", err)
		}
	}

	fmt.Println("-- Dev Client Tests Finished --")
}
