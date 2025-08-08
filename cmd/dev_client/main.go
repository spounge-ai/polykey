package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spounge-ai/polykey/tests/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

func main() {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	port := os.Getenv("POLYKEY_GRPC_PORT")
	if port == "" {
		port = "50053"
	}
	logger.Info("Configuration loaded", "server", "localhost:"+port)

	conn, err := grpc.NewClient("localhost:"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("gRPC connection failed", "error", err)
		fmt.Println(logBuf.String())
		os.Exit(1)
	}
	logger.Info("gRPC connection established successfully")
	defer func() {
		if err := conn.Close(); err != nil {
			logger.Error("failed to close connection", "error", err)
		}
	}()

	c := pk.NewPolykeyServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	r, err := c.HealthCheck(ctx, &emptypb.Empty{})
	if err != nil {
		logger.Error("HealthCheck failed", "error", err)
		utils.PrintJestReport(logBuf.String())
		os.Exit(1)
	}
	logger.Info("HealthCheck successful", "status", r.GetStatus().String(), "version", r.GetServiceVersion())

	createKeyReq := &pk.CreateKeyRequest{
		KeyType:          pk.KeyType_KEY_TYPE_AES_256,
		RequesterContext: &pk.RequesterContext{ClientIdentity: "admin"},
		Description:      "A key created by the dev client",
		Tags:             map[string]string{"source": "dev_client"},
	}
	createKeyResp, err := c.CreateKey(ctx, createKeyReq)
	if err != nil {
		logger.Error("CreateKey failed", "error", err)
		utils.PrintJestReport(logBuf.String())
		os.Exit(1)
	}
	logger.Info("CreateKey successful",
		"keyId", createKeyResp.GetMetadata().GetKeyId(),
		"keyType", createKeyResp.GetMetadata().GetKeyType().String(),
		"plaintextKey", fmt.Sprintf("%x", createKeyResp.GetKeyMaterial().GetEncryptedKeyData()),
	)

	newKeyId := createKeyResp.GetMetadata().GetKeyId()
	getKeyReq := &pk.GetKeyRequest{
		KeyId:            newKeyId,
		RequesterContext: &pk.RequesterContext{ClientIdentity: "admin"},
	}
	getKeyResp, err := c.GetKey(ctx, getKeyReq)
	if err != nil {
		logger.Error("GetKey failed", "error", err)
		utils.PrintJestReport(logBuf.String())
		os.Exit(1)
	}
	logger.Info("GetKey successful",
		"keyId", getKeyResp.GetMetadata().GetKeyId(),
		"plaintextKey", fmt.Sprintf("%x", getKeyResp.GetKeyMaterial().GetEncryptedKeyData()),
	)

	utils.PrintJestReport(logBuf.String())
}