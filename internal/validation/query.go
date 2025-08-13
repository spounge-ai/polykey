package validation

import (
	"fmt"
	"strings"
	"time"

	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

const (
	MaxQueryLimit     = 1000
	DefaultQueryLimit = 100
)

type QueryValidator struct{}

func NewQueryValidator() *QueryValidator {
	return &QueryValidator{}
}

func (qv *QueryValidator) ValidateListKeysRequest(req *pk.ListKeysRequest) error {
	pageSize := req.GetPageSize()
	if pageSize < 0 {
		return fmt.Errorf("page_size cannot be negative")
	} else if pageSize == 0 {
		req.PageSize = DefaultQueryLimit
	} else if pageSize > MaxQueryLimit {
		return fmt.Errorf("page_size %d exceeds maximum of %d", pageSize, MaxQueryLimit)
	}

	if token := req.GetPageToken(); token != "" {
		if !isValidPageToken(token) {
			return fmt.Errorf("invalid page token format")
		}
	}

	if filters := req.GetTagFilters(); len(filters) > 0 {
		for k, v := range filters {
			if strings.ContainsAny(k, ";'\"") || strings.ContainsAny(v, ";'\"") {
				return fmt.Errorf("invalid characters in tag filter")
			}
		}
	}

	if req.GetCreatedAfter() != nil && req.GetCreatedBefore() != nil {
		after := req.GetCreatedAfter().AsTime()
		before := req.GetCreatedBefore().AsTime()

		if after.After(before) {
			return fmt.Errorf("created_after must be before created_before")
		}

		if before.Sub(after) > 365*24*time.Hour {
			return fmt.Errorf("time range too wide (max 1 year)")
		}
	}

	return nil
}

func isValidPageToken(token string) bool {
	return len(token) < 256 && !strings.ContainsAny(token, ";'\"<>\"")
}
