package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	pkgvalidator "github.com/spounge-ai/polykey/pkg/validator"
)

const (
	MaxMetadataSize       = 64 * 1024 // 64KB
	MaxTagCount           = 50
	MaxTagKeyLen          = 128
	MaxTagValueLen        = 256
	MaxDescriptionLen     = 1024
	MaxAuthorizedContexts = 100
	MaxAccessPolicies     = 50
	MaxDataClassificationLen = 50
)

var allowedDataClassifications = map[string]bool{
	"public":       true,
	"confidential": true,
	"secret":       true,
}

type RequestValidator struct {
	validator    *validator.Validate
	uuidRegex    *regexp.Regexp
	tagKeyRegex  *regexp.Regexp
	contextRegex *regexp.Regexp
}

func NewRequestValidator() (*RequestValidator, error) {
	v := validator.New()

	if err := pkgvalidator.RegisterCustomValidators(v); err != nil {
		return nil, fmt.Errorf("failed to register custom validators: %w", err)
	}

	rv := &RequestValidator{
		validator:    v,
		uuidRegex:    regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`),
		tagKeyRegex:  regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,127}$`),
		contextRegex: regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,255}$`),
	}

	return rv, nil
}

func (rv *RequestValidator) ValidateCreateKeyRequest(ctx context.Context, req *pk.CreateKeyRequest) error {
	if err := rv.validateRequestSize(req); err != nil {
		return fmt.Errorf("request size validation failed: %w", err)
	}

	if req.GetKeyType() == pk.KeyType_KEY_TYPE_UNSPECIFIED {
		return fmt.Errorf("key type is required")
	}

	if len(req.GetDescription()) > MaxDescriptionLen {
		return fmt.Errorf("description exceeds maximum length of %d characters", MaxDescriptionLen)
	}

	if err := rv.validateTags(req.GetTags()); err != nil {
		return fmt.Errorf("tag validation failed: %w", err)
	}

	if err := rv.validateAuthorizedContexts(req.GetInitialAuthorizedContexts()); err != nil {
		return fmt.Errorf("authorized contexts validation failed: %w", err)
	}

	if err := rv.validateAccessPolicies(req.GetAccessPolicies()); err != nil {
		return fmt.Errorf("access policies validation failed: %w", err)
	}

	if err := rv.validateDataClassification(req.GetDataClassification()); err != nil {
		return fmt.Errorf("data classification validation failed: %w", err)
	}

	return nil
}

func (rv *RequestValidator) ValidateUpdateKeyMetadataRequest(ctx context.Context, req *pk.UpdateKeyMetadataRequest) error {
	if err := rv.validateRequestSize(req); err != nil {
		return fmt.Errorf("request size validation failed: %w", err)
	}

	if req.GetKeyId() == "" {
		return fmt.Errorf("key_id is required")
	}

	if len(req.GetDescription()) > MaxDescriptionLen {
		return fmt.Errorf("description exceeds maximum length of %d characters", MaxDescriptionLen)
	}

	if err := rv.validateTags(req.GetTagsToAdd()); err != nil {
		return fmt.Errorf("tags_to_add validation failed: %w", err)
	}

	if err := rv.validateAuthorizedContexts(req.GetContextsToAdd()); err != nil {
		return fmt.Errorf("contexts_to_add validation failed: %w", err)
	}

	if err := rv.validateAuthorizedContexts(req.GetContextsToRemove()); err != nil {
		return fmt.Errorf("contexts_to_remove validation failed: %w", err)
	}

	if err := rv.validateAccessPolicies(req.GetPoliciesToUpdate()); err != nil {
		return fmt.Errorf("policies_to_update validation failed: %w", err)
	}

	if err := rv.validateDataClassification(req.GetDataClassification()); err != nil {
		return fmt.Errorf("data classification validation failed: %w", err)
	}

	return nil
}

func (rv *RequestValidator) validateRequestSize(req interface{}) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to estimate request size: %w", err)
	}

	if len(data) > MaxMetadataSize {
		return fmt.Errorf("request size %d exceeds maximum of %d bytes", len(data), MaxMetadataSize)
	}

	return nil
}

func (rv *RequestValidator) validateTags(tags map[string]string) error {
	if len(tags) > MaxTagCount {
		return fmt.Errorf("tag count %d exceeds maximum of %d", len(tags), MaxTagCount)
	}

	for k, v := range tags {
		if len(k) > MaxTagKeyLen {
			return fmt.Errorf("tag key '%s' exceeds maximum length of %d", k, MaxTagKeyLen)
		}
		if len(v) > MaxTagValueLen {
			return fmt.Errorf("tag value for key '%s' exceeds maximum length of %d", k, MaxTagValueLen)
		}
		if !rv.tagKeyRegex.MatchString(k) {
			return fmt.Errorf("invalid tag key format: '%s'", k)
		}

		if strings.ContainsAny(v, "<>\"'") {
			return fmt.Errorf("tag value contains invalid characters")
		}
	}

	return nil
}

func (rv *RequestValidator) validateAuthorizedContexts(contexts []string) error {
	if len(contexts) > MaxAuthorizedContexts {
		return fmt.Errorf("authorized contexts count %d exceeds maximum of %d",
			len(contexts), MaxAuthorizedContexts)
	}

	seen := make(map[string]bool, len(contexts))
	for _, ctx := range contexts {
		if seen[ctx] {
			return fmt.Errorf("duplicate authorized context: %s", ctx)
		}
		seen[ctx] = true

		if !rv.contextRegex.MatchString(ctx) {
			return fmt.Errorf("invalid authorized context format: '%s'", ctx)
		}
	}

	return nil
}

func (rv *RequestValidator) validateAccessPolicies(policies map[string]string) error {
	if len(policies) > MaxAccessPolicies {
		return fmt.Errorf("access policies count %d exceeds maximum of %d",
			len(policies), MaxAccessPolicies)
	}

	for name, policy := range policies {
		if !rv.tagKeyRegex.MatchString(name) {
			return fmt.Errorf("invalid policy name format: '%s'", name)
		}

		var policyObj interface{}
		if err := json.Unmarshal([]byte(policy), &policyObj); err != nil {
			return fmt.Errorf("invalid policy JSON for '%s': %w", name, err)
		}
	}

	return nil
}

func (rv *RequestValidator) validateDataClassification(classification string) error {
	if classification == "" {
		return nil // Data classification is optional
	}

	if len(classification) > MaxDataClassificationLen {
		return fmt.Errorf("data classification exceeds maximum length of %d characters", MaxDataClassificationLen)
	}

	if !allowedDataClassifications[classification] {
		return fmt.Errorf("invalid data classification: '%s'", classification)
	}

	return nil
}

