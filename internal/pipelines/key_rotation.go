package pipelines

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"

	"github.com/spounge-ai/polykey/internal/domain"
	"github.com/spounge-ai/polykey/internal/kms"
	"github.com/spounge-ai/polykey/pkg/memory"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// KeyRotationRequest holds the data for a key rotation request.
type KeyRotationRequest struct {
	KeyID              domain.KeyID
	KMSProvider        kms.KMSProvider
	DEKPool            *memory.BufferPool
	GracePeriodSeconds int32
	KeyType            pk.KeyType
}

// KeyRotationResult holds the result of a key rotation.
type KeyRotationResult struct {
	KeyID      domain.KeyID
	RotatedKey *domain.Key
	Error      error
	GracePeriodSeconds int32
}

// KeyRotationPipeline manages the concurrent processing of key rotations.
type KeyRotationPipeline struct {
	requests    chan KeyRotationRequest
	results     chan KeyRotationResult
	keyRepo     domain.KeyRepository
	logger      *slog.Logger
	workerCount int
}

// NewKeyRotationPipeline creates a new key rotation pipeline.
func NewKeyRotationPipeline(keyRepo domain.KeyRepository, logger *slog.Logger, workerCount, queueDepth int) *KeyRotationPipeline {
	return &KeyRotationPipeline{
		requests:    make(chan KeyRotationRequest, queueDepth),
		results:     make(chan KeyRotationResult, queueDepth),
		keyRepo:     keyRepo,
		logger:      logger,
		workerCount: workerCount,
	}
}

// Start begins the pipeline workers.
func (p *KeyRotationPipeline) Start(ctx context.Context) {
	for i := 0; i < p.workerCount; i++ {
		go p.worker(ctx)
	}
}

// Enqueue adds a new key rotation request to the pipeline.
// It returns false if the queue is full (non-blocking).
func (p *KeyRotationPipeline) Enqueue(req KeyRotationRequest) bool {
	select {
	case p.requests <- req:
		return true
	default:
		return false // Queue is full
	}
}

// Results returns the channel for reading rotation results.
func (p *KeyRotationPipeline) Results() <-chan KeyRotationResult {
	return p.results
}

// worker is a pipeline stage that processes key rotation requests.
func (p *KeyRotationPipeline) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-p.requests:
			rotatedKey, err := p.processRotation(ctx, req)
			result := KeyRotationResult{RotatedKey: rotatedKey, Error: err, KeyID: req.KeyID, GracePeriodSeconds: req.GracePeriodSeconds}

			// Send the result back
			select {
			case p.results <- result:
			case <-ctx.Done():
				// If the context is cancelled, don't block on sending the result.
				return
			}
		}
	}
}

// processRotation contains the actual logic for rotating a key.
func (p *KeyRotationPipeline) processRotation(ctx context.Context, req KeyRotationRequest) (*domain.Key, error) {
	currentKey, err := p.keyRepo.GetKey(ctx, req.KeyID)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to get current key for rotation", "keyId", req.KeyID, "error", err)
		return nil, fmt.Errorf("failed to get current key: %w", err)
	}

	newDEK := req.DEKPool.Get()
	defer req.DEKPool.Put(newDEK)

	if _, err := rand.Read(newDEK); err != nil {
		p.logger.ErrorContext(ctx, "failed to generate new DEK", "error", err)
		return nil, fmt.Errorf("failed to generate new DEK: %w", err)
	}

	encryptedNewDEK, err := req.KMSProvider.EncryptDEK(ctx, newDEK, currentKey)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to encrypt new DEK", "error", err)
		return nil, fmt.Errorf("failed to encrypt new DEK: %w", err)
	}

	rotatedKey, err := p.keyRepo.RotateKey(ctx, req.KeyID, encryptedNewDEK)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to rotate key in repository", "keyId", req.KeyID, "error", err)
		return nil, fmt.Errorf("failed to rotate key: %w", err)
	}

	p.logger.InfoContext(ctx, "key rotated via pipeline", "keyId", req.KeyID, "newVersion", rotatedKey.Version)
	return rotatedKey, nil
}
