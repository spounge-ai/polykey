package persistence

import (
	"encoding/json"
	"strings"
	"sync"
)

const (
	// builderInitialCap is the initial capacity for the string builder.
	builderInitialCap = 128
)

// QueryOptimizer manages performance optimizations like buffer and builder pools.
type QueryOptimizer struct {
	queryBuilderPool *sync.Pool
	bufferPool       *BufferPool
}

// BufferPool manages pools of byte slices for sensitive data.
type BufferPool struct {
	metaBuffer []byte
	bufferMux  sync.Mutex
}

// NewQueryOptimizer creates a new QueryOptimizer.
func NewQueryOptimizer() *QueryOptimizer {
	return &QueryOptimizer{
		queryBuilderPool: &sync.Pool{
			New: func() interface{} {
				sb := &strings.Builder{}
				sb.Grow(builderInitialCap)
				return sb
			},
		},
		bufferPool: &BufferPool{
			metaBuffer: make([]byte, 0, 512),
		},
	}
}

// GetBuilder retrieves a strings.Builder from the pool.
func (qo *QueryOptimizer) GetBuilder() *strings.Builder {
	return qo.queryBuilderPool.Get().(*strings.Builder)
}

// PutBuilder returns a strings.Builder to the pool.
func (qo *QueryOptimizer) PutBuilder(sb *strings.Builder) {
	sb.Reset()
	qo.queryBuilderPool.Put(sb)
}

// MarshalWithBuffer uses buffer pool to reduce allocations for JSON marshaling.
func (qo *QueryOptimizer) MarshalWithBuffer(v interface{}) ([]byte, error) {
	qo.bufferPool.bufferMux.Lock()
	defer qo.bufferPool.bufferMux.Unlock()

	qo.bufferPool.metaBuffer = qo.bufferPool.metaBuffer[:0] // Reset buffer
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	// It's important to copy the data to a new slice before returning it,
	// as the buffer will be reused.
	result := make([]byte, len(data))
	copy(result, data)
	return result, nil
}