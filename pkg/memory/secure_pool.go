package memory

import (
	"sync"
)

// SecureDEKPool is a sync.Pool for managing DEKs with proper cleanup.
type SecureDEKPool struct {
	pool *sync.Pool
	size int
}

// NewSecureDEKPool creates a new SecureDEKPool.
func NewSecureDEKPool(size int) *SecureDEKPool {
	return &SecureDEKPool{
		size: size,
		pool: &sync.Pool{
			New: func() interface{} {
				b := make([]byte, size)
				return &b
			},
		},
	}
}

// Get gets a buffer from the pool.
func (p *SecureDEKPool) Get() []byte {
	return *p.pool.Get().(*[]byte)
}

// Put returns a buffer to the pool.
func (p *SecureDEKPool) Put(buf []byte) {
	SecureZeroBytes(buf) // Always zero before returning
	p.pool.Put(&buf)
}