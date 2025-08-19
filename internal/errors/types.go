package errors

import "errors"

var (
	ErrKeyNotFound    = errors.New("key not found")
	ErrInvalidInput   = errors.New("invalid input")
	ErrKMSFailure     = errors.New("kms operation failed")
	ErrAuthentication = errors.New("authentication failed")
	ErrAuthorization  = errors.New("authorization failed")
	ErrConflict       = errors.New("resource conflict")
	ErrRateLimit      = errors.New("rate limit exceeded")
	ErrExternal       = errors.New("external service error")
	ErrKeyRotationLocked = errors.New("key rotation is locked")
	ErrKeyRevoked     = errors.New("key is revoked")
)
