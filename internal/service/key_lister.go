package service

import (
	"context"

	app_errors "github.com/spounge-ai/polykey/internal/errors"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *keyServiceImpl) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	if req == nil {
		return nil, app_errors.ErrInvalidInput
	}

	keys, err := s.keyRepo.ListKeys(ctx)
	if err != nil {
		return nil, err // The error from the repository is a standard Go error.
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
