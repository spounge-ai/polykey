package persistence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/spounge-ai/polykey/internal/domain"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

type S3Storage struct {
	client     *s3.Client
	bucketName string
	logger     *slog.Logger
}

func NewS3Storage(cfg aws.Config, bucketName string, logger *slog.Logger) (*S3Storage, error) {
	s3Client := s3.NewFromConfig(cfg)
	return &S3Storage{
		client:     s3Client,
		bucketName: bucketName,
		logger:     logger,
	}, nil
}

type s3KeyObject struct {
	ID            string          `json:"id"`
	EncryptedDEK  []byte          `json:"encrypted_dek"`
	Metadata      *pk.KeyMetadata `json:"metadata"`
	Version       int32           `json:"version"`
	Status        pk.KeyStatus    `json:"status"`
	CreatedAt     int64           `json:"created_at"`
	UpdatedAt     int64           `json:"updated_at"`
}

func (s *S3Storage) GetKey(ctx context.Context, id domain.KeyID) (*domain.Key, error) {
	keyPath := fmt.Sprintf("keys/%s/latest.json", id.String())
	return s.getKeyFromPath(ctx, keyPath)
}

func (s *S3Storage) GetKeyByVersion(ctx context.Context, id domain.KeyID, version int32) (*domain.Key, error) {
	keyPath := fmt.Sprintf("keys/%s/v%d.json", id.String(), version)
	return s.getKeyFromPath(ctx, keyPath)
}

func (s *S3Storage) GetKeyMetadata(ctx context.Context, id domain.KeyID) (*pk.KeyMetadata, error) {
	key, err := s.GetKey(ctx, id)
	if err != nil {
		return nil, err
	}
	return key.Metadata, nil
}

func (s *S3Storage) GetKeyMetadataByVersion(ctx context.Context, id domain.KeyID, version int32) (*pk.KeyMetadata, error) {
	key, err := s.GetKeyByVersion(ctx, id, version)
	if err != nil {
		return nil, err
	}
	return key.Metadata, nil
}

func (s *S3Storage) CreateKey(ctx context.Context, key *domain.Key) (*domain.Key, error) {
	err := s.putKey(ctx, key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (s *S3Storage) CreateKeys(ctx context.Context, keys []*domain.Key) error {
	for _, key := range keys {
		if err := s.putKey(ctx, key); err != nil {
			return err
		}
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
			s.logger.Error("failed to close S3 object body", "error", err)
		}
	}()

	var keyObj s3KeyObject
	if err := json.NewDecoder(output.Body).Decode(&keyObj); err != nil {
		return nil, fmt.Errorf("failed to decode key object from S3: %w", err)
	}

	id, err := domain.KeyIDFromString(keyObj.ID)
	if err != nil {
		return nil, err
	}

	return &domain.Key{
		ID:           id,
		EncryptedDEK: keyObj.EncryptedDEK,
		Metadata:     keyObj.Metadata,
		Version:      keyObj.Version,
		Status:       domain.KeyStatus(pk.KeyStatus_name[int32(keyObj.Status)]),
		CreatedAt:    time.Unix(keyObj.CreatedAt, 0),
		UpdatedAt:    time.Unix(keyObj.UpdatedAt, 0),
	}, nil
}

func (s *S3Storage) putKey(ctx context.Context, key *domain.Key) error {
	keyObj := s3KeyObject{
		ID:           key.ID.String(),
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

	versionPath := fmt.Sprintf("keys/%s/v%d.json", key.ID.String(), key.Version)
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucketName,
		Key:    &versionPath,
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to put versioned key object to S3: %w", err)
	}

	latestPath := fmt.Sprintf("keys/%s/latest.json", key.ID.String())
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucketName,
		Key:    &latestPath,
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		if _, delErr := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucketName, Key: &versionPath}); delErr != nil {
			s.logger.Error("failed to roll back S3 object", "path", versionPath, "error", delErr)
		}
		return fmt.Errorf("failed to put latest key object to S3: %w", err)
	}

	return nil
}

func (s *S3Storage) ListKeys(ctx context.Context) ([]*domain.Key, error) {
	prefix := "keys/"
	input := &s3.ListObjectsV2Input{
		Bucket:    &s.bucketName,
		Prefix:    &prefix,
		Delimiter: aws.String("/"),
	}

	var keys []*domain.Key
	paginator := s3.NewListObjectsV2Paginator(s.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects from S3: %w", err)
		}

		for _, obj := range page.CommonPrefixes {
			keyIDStr := strings.TrimSuffix(strings.TrimPrefix(*obj.Prefix, prefix), "/")
			keyID, err := domain.KeyIDFromString(keyIDStr)
			if err != nil {
				s.logger.Error("failed to parse key id while listing", "keyID", keyIDStr, "error", err)
				continue
			}
			key, err := s.GetKey(ctx, keyID)
			if err != nil {
				s.logger.Error("failed to get key while listing", "keyID", keyID, "error", err)
				continue
			}
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func (s *S3Storage) UpdateKeyMetadata(ctx context.Context, id domain.KeyID, metadata *pk.KeyMetadata) error {
	latestKey, err := s.GetKey(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get key for update: %w", err)
	}

	latestKey.Metadata = metadata
	latestKey.UpdatedAt = time.Now()

	return s.putKey(ctx, latestKey)
}

func (s *S3Storage) RotateKey(ctx context.Context, id domain.KeyID, newEncryptedDEK []byte) (*domain.Key, error) {
	latestKey, err := s.GetKey(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get key for rotation: %w", err)
	}

	newVersion := latestKey.Version + 1
	now := time.Now()

	rotatedKey := &domain.Key{
		ID:           id,
		EncryptedDEK: newEncryptedDEK,
		Metadata:     latestKey.Metadata,
		Version:      newVersion,
		Status:       domain.KeyStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.putKey(ctx, rotatedKey); err != nil {
		return nil, fmt.Errorf("failed to create new key version during rotation: %w", err)
	}

	return rotatedKey, nil
}

func (s *S3Storage) RevokeKey(ctx context.Context, id domain.KeyID) error {
	latestKey, err := s.GetKey(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get key for revocation: %w", err)
	}

	latestKey.Status = domain.KeyStatusRevoked
	latestKey.UpdatedAt = time.Now()

	return s.putKey(ctx, latestKey)
}

func (s *S3Storage) GetKeyVersions(ctx context.Context, id domain.KeyID) ([]*domain.Key, error) {
	prefix := fmt.Sprintf("keys/%s/v", id.String())
	input := &s3.ListObjectsV2Input{
		Bucket: &s.bucketName,
		Prefix: &prefix,
	}

	var versions []*domain.Key
	paginator := s3.NewListObjectsV2Paginator(s.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects from S3 for versions: %w", err)
		}

		for _, obj := range page.Contents {
			key, err := s.getKeyFromPath(ctx, *obj.Key)
			if err != nil {
				s.logger.Error("failed to get key version from path", "path", *obj.Key, "error", err)
				continue
			}
			versions = append(versions, key)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})

	return versions, nil
}

func (s *S3Storage) Exists(ctx context.Context, id domain.KeyID) (bool, error) {
	keyPath := fmt.Sprintf("keys/%s/latest.json", id.String())
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.bucketName,
		Key:    &keyPath,
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3Storage) HealthCheck() error {
	_, err := s.client.HeadBucket(context.Background(), &s3.HeadBucketInput{
		Bucket: &s.bucketName,
	})
	if err != nil {
		s.logger.Error("S3 health check failed", "error", err)
		return fmt.Errorf("S3 health check failed: %w", err)
	}
	return nil
}
