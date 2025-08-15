package errors

import "errors"

var (
	ErrKeyNotFound     = errors.New("key not found")
	ErrInvalidInput    = errors.New("invalid input")
	ErrKMSFailure      = errors.New("KMS operation failed")
	ErrAuthentication  = errors.New("authentication failed")
	ErrAuthorization   = errors.New("authorization failed")
	ErrConflict        = errors.New("resource conflict")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)