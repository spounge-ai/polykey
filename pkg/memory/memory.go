package memory

import (
	"crypto/subtle"
	"runtime"
	"sync"
)

// SecureZeroBytes securely wipes the contents of a byte slice.
// It iterates through the slice, setting each byte to zero.
// runtime.KeepAlive is used to ensure the compiler does not optimize away
// the clearing of the buffer before it is returned to a pool.
func SecureZeroBytes(b []byte) {
	// Using subtle.ConstantTimeCompare to prevent the compiler from optimizing
	// away the memory-clearing loop.
	zeros := make([]byte, len(b))
	subtle.ConstantTimeCompare(b, zeros)
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}

// BufferPool is a pool of byte slices for sensitive data.
// It uses a sync.Pool to manage buffers and ensures they are securely zeroed
// before being returned to the pool.
// TODO: Add memory pressure monitoring and adaptive sizing.
// TODO: Implement worker pools for concurrent memory operations.
type BufferPool struct {
	pool *sync.Pool
}

// NewBufferPool creates a new pool for buffers of a given size.
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		pool: &sync.Pool{
			New: func() interface{} {
				// Allocate a new buffer and return a pointer to it.
				b := make([]byte, size)
				return &b
			},
		},
	}
}

// Get retrieves a buffer from the pool.
func (p *BufferPool) Get() []byte {
	return *p.pool.Get().(*[]byte)
}

// Put securely zeroes a buffer and returns it to the pool.
func (p *BufferPool) Put(b []byte) {
	SecureZeroBytes(b)
	p.pool.Put(&b)
}
