package main

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/spounge-ai/polykey/tests/utils"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/emptypb"
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, err := c.HealthCheck(ctx, &emptypb.Empty{})
	if err != nil {
		logger.Error("HealthCheck failed", "error", err)
	} else {
		logger.Info("HealthCheck successful", "status", r.GetStatus().String(), "version", r.GetServiceVersion())
	}

	createKeyReq := &pk.CreateKeyRequest{
		KeyType:                   pk.KeyType_KEY_TYPE_AES_256,
		RequesterContext:          &pk.RequesterContext{ClientIdentity: "admin"},
		InitialAuthorizedContexts: []string{"admin"}, // Authorize the admin to use this key
		Description:               "A key created by the dev client",
		Tags:                      map[string]string{"source": "dev_client"},
	}
	createKeyResp, err := c.CreateKey(ctx, createKeyReq)
	if err != nil {
		logger.Error("CreateKey failed", "error", err)
	} else {
		logger.Info("CreateKey successful",
			"keyId", createKeyResp.GetMetadata().GetKeyId(),
			"keyType", createKeyResp.GetMetadata().GetKeyType().String(),
		)

		newKeyId := createKeyResp.GetMetadata().GetKeyId()
		
		// Store the original key before rotation
		getKeyReq := &pk.GetKeyRequest{
			KeyId:            newKeyId,
			RequesterContext: &pk.RequesterContext{ClientIdentity: "admin"},
		}
		originalKeyResp, err := c.GetKey(ctx, getKeyReq)
		if err != nil {
			logger.Error("GetKey (pre-rotation) failed", "error", err)
		} else {
			logger.Info("GetKey successful (pre-rotation)",
				"keyId", originalKeyResp.GetMetadata().GetKeyId(),
				"version", originalKeyResp.GetMetadata().GetVersion(),
			)

			// Rotate the key
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
					"previousVersion", rotateKeyResp.GetPreviousVersion(),
				)

				// Get the key after rotation to compare
				postRotateKeyResp, err := c.GetKey(ctx, getKeyReq)
				if err != nil {
					logger.Error("GetKey (post-rotation) failed", "error", err)
				} else {
					logger.Info("GetKey successful (post-rotation)",
						"keyId", postRotateKeyResp.GetMetadata().GetKeyId(),
						"version", postRotateKeyResp.GetMetadata().GetVersion(),
					)

					// Compare the keys before and after rotation
					compareKeys(logger, originalKeyResp, postRotateKeyResp)
				}
			}
		}
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

func compareKeys(logger *slog.Logger, originalKey, rotatedKey *pk.GetKeyResponse) {
	logger.Info("Starting key rotation validation")

	// Compare key IDs (should be the same)
	originalKeyId := originalKey.GetMetadata().GetKeyId()
	rotatedKeyId := rotatedKey.GetMetadata().GetKeyId()
	
	if originalKeyId == rotatedKeyId {
		logger.Info("Key ID preserved after rotation", "keyId", originalKeyId)
	} else {
		logger.Error("Key ID changed after rotation", 
			"originalKeyId", originalKeyId, 
			"rotatedKeyId", rotatedKeyId)
	}

	// Compare versions (should be different)
	originalVersion := originalKey.GetMetadata().GetVersion()
	rotatedVersion := rotatedKey.GetMetadata().GetVersion()
	
	if originalVersion != rotatedVersion && rotatedVersion == originalVersion+1 {
		logger.Info("Key version incremented correctly", 
			"originalVersion", originalVersion, 
			"rotatedVersion", rotatedVersion)
	} else {
		logger.Error("Key version not incremented properly", 
			"originalVersion", originalVersion, 
			"rotatedVersion", rotatedVersion)
	}

	// Compare key material (should be different)
	originalKeyMaterial := originalKey.GetKeyMaterial().GetEncryptedKeyData()
	rotatedKeyMaterial := rotatedKey.GetKeyMaterial().GetEncryptedKeyData()
	
	if !bytes.Equal(originalKeyMaterial, rotatedKeyMaterial) {
		logger.Info("Key material successfully rotated")
	} else {
		logger.Error("Key material unchanged after rotation")
	}

	// Compare key types (should be the same)
	originalKeyType := originalKey.GetMetadata().GetKeyType()
	rotatedKeyType := rotatedKey.GetMetadata().GetKeyType()
	
	if originalKeyType == rotatedKeyType {
		logger.Info("Key type preserved", "keyType", originalKeyType.String())
	} else {
		logger.Error("Key type changed unexpectedly", 
			"originalKeyType", originalKeyType.String(), 
			"rotatedKeyType", rotatedKeyType.String())
	}

	logger.Info("Key rotation validation completed")
}