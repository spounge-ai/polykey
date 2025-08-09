package service

import (
	"context"
	"fmt"

	
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *keyServiceImpl) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}

	keys, err := s.keyRepo.ListKeys(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list keys", "error", err)
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	metadataKeys := make([]*pk.KeyMetadata, len(keys))
	for i, key := range keys {
		metadataKeys[i] = key.Metadata
	}

	count := int32(len(metadataKeys))
	resp := &pk.ListKeysResponse{
		Keys:              metadataKeys,
		TotalCount:        count,
		FilteredCount:     count,
		ResponseTimestamp: timestamppb.Now(),
	}

	s.logger.InfoContext(ctx, "keys listed", "count", len(metadataKeys))
	return resp, nil
}
