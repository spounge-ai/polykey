package main

import (
	"bytes"
	"log/slog"
	"os"
	"time"

	"github.com/spounge-ai/polykey/pkg/testutil"
	"github.com/spounge-ai/polykey/tests/devclient"
	"github.com/spounge-ai/polykey/tests/utils"
)


const (
	DefaultPort      = "50053"
	SecretConfigPath = "configs/dev_client/secret.dev.yaml"
	TLSConfigPath    = "configs/dev_client/tls.yaml"
	DefaultTimeout   = 30 * time.Second
)

func main() {
	logBuf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logBuf, nil))

	port := os.Getenv("POLYKEY_GRPC_PORT")
	if port == "" {
		port = DefaultPort
	}

	cfg := testutil.Config{
		ServerAddr:       "localhost:" + port,
		SecretConfigPath: SecretConfigPath,
		TLSConfigPath:    TLSConfigPath,
		DefaultTimeout:   DefaultTimeout,
	}

	testClient, err := testutil.New(cfg, logger)
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
