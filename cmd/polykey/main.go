package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spounge-ai/polykey/internal/app/grpc"
	"github.com/spounge-ai/polykey/internal/domain"
	infra_aws "github.com/spounge-ai/polykey/internal/infra/aws"
	infra_config "github.com/spounge-ai/polykey/internal/infra/config"
	"github.com/spounge-ai/polykey/internal/infra/persistence"

	dev_auth "github.com/spounge-ai/polykey/dev/auth"
)

func main() {
	cfg, err := infra_config.Load(os.Getenv("POLYKEY_CONFIG_PATH"))
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	var kmsAdapter domain.KMSService
	var authorizer domain.Authorizer
	var keyRepo domain.KeyRepository

	// For local development, we will now use the real AWS implementations.
	// The mock implementations can be used for pure unit testing.
	log.Println("Using AWS-backed implementations for local testing.")

	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.AWS.Region))
	if err != nil {
		log.Fatalf("failed to load aws config: %v", err)
	}

	kmsAdapter = infra_aws.NewKMSCachedAdapter(infra_aws.NewKMSAdapter(awsCfg), 5*time.Minute)
	authorizer = dev_auth.NewMockAuthorizer() // Using mock authorizer for now
	keyRepo, err = persistence.NewS3Storage(awsCfg, cfg.AWS.S3Bucket)
	if err != nil {
		log.Fatalf("failed to create s3 storage: %v", err)
	}

	srv, _, err := grpc.New(cfg, keyRepo, kmsAdapter, authorizer, nil) // nil for audit logger for now
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	if err := srv.RunBlocking(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
