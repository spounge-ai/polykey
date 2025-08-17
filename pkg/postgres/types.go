package postgres

import "errors"

var (
	ErrKeyNotFound      = errors.New("key not found")
	ErrInvalidVersion   = errors.New("invalid version")
	ErrKeyAlreadyExists = errors.New("key already exists")
)
