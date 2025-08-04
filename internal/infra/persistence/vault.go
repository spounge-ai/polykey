package persistence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// S3Storage implements the KeyRepository using an S3 bucket.
type S3Storage struct {
	client     *s3.Client
	bucketName string
}

// NewS3Storage creates a new S3-backed KeyRepository.
func NewS3Storage(cfg aws.Config, bucketName string) (*S3Storage, error) {
	s3Client := s3.NewFromConfig(cfg)
	return &S3Storage{
		client:     s3Client,
		bucketName: bucketName,
	}, nil
}

// s3KeyObject represents the structure of the JSON object stored in S3.
type s3KeyObject struct {
	ID            string         `json:"id"`
	EncryptedDEK  []byte         `json:"encrypted_dek"`
	Metadata      *pk.KeyMetadata `json:"metadata"`
	Version       int32          `json:"version"`
	Status        pk.KeyStatus   `json:"status"`
	CreatedAt     int64          `json:"created_at"`
	UpdatedAt     int64          `json:"updated_at"`
}

func (s *S3Storage) GetKey(ctx context.Context, id string) (*domain.Key, error) {
	keyPath := fmt.Sprintf("keys/%s/latest.json", id)
	return s.getKeyFromPath(ctx, keyPath)
}

func (s *S3Storage) GetKeyByVersion(ctx context.Context, id string, version int32) (*domain.Key, error) {
	keyPath := fmt.Sprintf("keys/%s/v%d.json", id, version)
	return s.getKeyFromPath(ctx, keyPath)
}

func (s *S3Storage) CreateKey(ctx context.Context, key *domain.Key) error {
	keyObj := s3KeyObject{
		ID:           key.ID,
		EncryptedDEK: key.EncryptedDEK,
		Metadata:     key.Metadata,
		Version:      key.Version,
		Status:       pk.KeyStatus(pk.KeyStatus_value[string(key.Status)]),
		CreatedAt:    key.CreatedAt.Unix(),
		UpdatedAt:    key.UpdatedAt.Unix(),
	}

	data, err := json.Marshal(keyObj)
	if err != nil {
		return fmt.Errorf("failed to marshal key object: %w", err)
	}

	// Store the specific version
	versionPath := fmt.Sprintf("keys/%s/v%d.json", key.ID, key.Version)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucketName,
		Key:    &versionPath,
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to put versioned key object to S3: %w", err)
	}

	// Store as the latest version
	latestPath := fmt.Sprintf("keys/%s/latest.json", key.ID)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucketName,
		Key:    &latestPath,
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		// Attempt to roll back the versioned key if this fails
		if _, delErr := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucketName, Key: &versionPath}); delErr != nil {
			// Log the deletion error, but return the original error
			fmt.Printf("failed to roll back S3 object %s: %v", versionPath, delErr)
		}
		return fmt.Errorf("failed to put latest key object to S3: %w", err)
	}

	return nil
}

func (s *S3Storage) getKeyFromPath(ctx context.Context, path string) (*domain.Key, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucketName,
		Key:    &path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get key object from S3: %w", err)
	}
	defer func() {
		if err := output.Body.Close(); err != nil {
			fmt.Printf("failed to close S3 object body: %v", err)
		}
	}()

	var keyObj s3KeyObject
	if err := json.NewDecoder(output.Body).Decode(&keyObj); err != nil {
		return nil, fmt.Errorf("failed to decode key object from S3: %w", err)
	}

	return &domain.Key{
		ID:           keyObj.ID,
		EncryptedDEK: keyObj.EncryptedDEK,
		Metadata:     keyObj.Metadata,
		Version:      keyObj.Version,
		Status:       domain.KeyStatus(pk.KeyStatus_name[int32(keyObj.Status)]),
		CreatedAt:    time.Unix(keyObj.CreatedAt, 0),
		UpdatedAt:    time.Unix(keyObj.UpdatedAt, 0),
	}, nil
}

// --- Unimplemented Methods ---

func (s *S3Storage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	return nil, fmt.Errorf("ListKeys not implemented")
}

func (s *S3Storage) UpdateKeyMetadata(ctx context.Context, id string, metadata *pk.KeyMetadata) error {
    return fmt.Errorf("UpdateKeyMetadata not implemented")
}

func (s *S3Storage) RotateKey(ctx context.Context, id string, newEncryptedDEK []byte) (*domain.Key, error) {
	return nil, fmt.Errorf("RotateKey not implemented")
}

func (s *S3Storage) RevokeKey(ctx context.Context, id string) error {
	return fmt.Errorf("RevokeKey not implemented")
}

func (s *S3Storage) GetKeyVersions(ctx context.Context, id string) ([]*domain.Key, error) {
	return nil, fmt.Errorf("GetKeyVersions not implemented")
}

func (s *S3Storage) HealthCheck() error {
	// A simple health check could be a HeadBucket call
	_, err := s.client.HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: &s.bucketName,
	})
	if err != nil {
		return fmt.Errorf("S3 health check failed: %w", err)
	}
	return nil
}