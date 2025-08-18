package service

import (
	"context"
	"time"

	app_errors "github.com/spounge-ai/polykey/internal/errors"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *keyServiceImpl) ListKeys(ctx context.Context, req *pk.ListKeysRequest) (*pk.ListKeysResponse, error) {
	if req == nil {
		return nil, app_errors.ErrInvalidInput
	}

	var cursor *time.Time
	if req.PageToken != "" {
		t, err := time.Parse(time.RFC3339Nano, req.PageToken)
		if err != nil {
			return nil, app_errors.ErrInvalidInput
		}
		cursor = &t
	}

	limit := int(req.GetPageSize())
	if limit == 0 {
		limit = 100 // default page size
	}

	keys, err := s.keyRepo.ListKeys(ctx, cursor, limit)
	if err != nil {
		return nil, err // The error from the repository is a standard Go error.
	}

	metadataKeys := make([]*pk.KeyMetadata, len(keys))
	for i, key := range keys {
		metadataKeys[i] = key.Metadata
	}

	var nextPageToken string
	if len(keys) == limit {
		nextPageToken = keys[len(keys)-1].CreatedAt.Format(time.RFC3339Nano)
	}

	resp := &pk.ListKeysResponse{
		Keys:              metadataKeys,
		NextPageToken:     nextPageToken,
		ResponseTimestamp: timestamppb.Now(),
	}

	s.logger.InfoContext(ctx, "keys listed", "count", len(metadataKeys))
	return resp, nil
}
