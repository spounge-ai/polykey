package main

import (
	"bytes"
	"context"
	
	"log/slog"
	"os"
	"time"

	"github.com/spounge-ai/polykey/tests/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

	creds, err := credentials.NewClientTLSFromFile("certs/cert.pem", "")
	if err != nil {
		logger.Error("failed to load TLS credentials", "error", err)
		utils.PrintJestReport(logBuf.String())
		os.Exit(1)
	}

	conn, err := grpc.NewClient("localhost:"+port, grpc.WithTransportCredentials(creds))
	if err != nil {
		logger.Error("gRPC connection failed", "error", err)
		utils.PrintJestReport(logBuf.String())
		os.Exit(1)
	}
	logger.Info("gRPC connection established successfully")
	defer func() {
		if err := conn.Close(); err != nil {
			logger.Error("failed to close connection", "error", err)
		}
	}()

	c := pk.NewPolykeyServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	r, err := c.HealthCheck(ctx, &emptypb.Empty{})
	if err != nil {
		logger.Error("HealthCheck failed", "error", err)
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
	} else {
		logger.Info("CreateKey successful",
			"keyId", createKeyResp.GetMetadata().GetKeyId(),
			"keyType", createKeyResp.GetMetadata().GetKeyType().String(),
		)
	}

	newKeyId := createKeyResp.GetMetadata().GetKeyId()
	getKeyReq := &pk.GetKeyRequest{
		KeyId:            newKeyId,
		RequesterContext: &pk.RequesterContext{ClientIdentity: "admin"},
	}
	getKeyResp, err := c.GetKey(ctx, getKeyReq)
	if err != nil {
		logger.Error("GetKey failed", "error", err)
	} else {
		logger.Info("GetKey successful",
			"keyId", getKeyResp.GetMetadata().GetKeyId(),
		)
	}

	rotateKeyReq := &pk.RotateKeyRequest{
		KeyId:            newKeyId,
		RequesterContext: &pk.RequesterContext{ClientIdentity: "admin"},
	}
	rotateKeyResp, err := c.RotateKey(ctx, rotateKeyReq)
	if err != nil {
		logger.Error("RotateKey failed", "error", err)
	} else {
		logger.Info("RotateKey successful",
			"keyId", rotateKeyResp.GetKeyId(),
			"newVersion", rotateKeyResp.GetNewVersion(),
		)
	}

	listKeysResp, err := c.ListKeys(ctx, &pk.ListKeysRequest{RequesterContext: &pk.RequesterContext{ClientIdentity: "admin"}})
	if err != nil {
		logger.Error("ListKeys failed", "error", err)
	} else {
		logger.Info("ListKeys successful", "count", len(listKeysResp.GetKeys()))
	}

	if utils.PrintJestReport(logBuf.String()) {
		os.Exit(1)
	}
}
