package persistence

import (
	"github.com/spounge-ai/polykey/internal/domain"
)

type StorageProvider interface {
	domain.KeyRepository
}
