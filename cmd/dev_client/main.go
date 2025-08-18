package main

import (
	"bytes"
	"log/slog"
	"os"

	"github.com/spounge-ai/polykey/tests/devclient"
	"github.com/spounge-ai/polykey/tests/utils"
)

func main() {
	logBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuf, nil))

	testClient, err := devclient.NewPolykeyTestClient(logger)
	if err != nil {
		utils.PrintJestReport(logBuf.String())
		os.Exit(1)
	}
	defer testClient.Close()

	devclient.Run(testClient)

	if utils.PrintJestReport(logBuf.String()) {
		os.Exit(1)
	}
}
