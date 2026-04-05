package json

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cybergodev/json/internal"
)

// unifiedResourceManager consolidates all resource management for optimal performance
// sync.Pool is inherently thread-safe, no additional locks needed for pool operations
type unifiedResourceManager struct {
	stringBuilderPool *sync.Pool
	pathSegmentPool   *sync.Pool
	bufferPool        *sync.Pool
	optionsPool       *sync.Pool // Pool for ProcessorOptions to reduce allocations
	// PERFORMANCE: Added map pool for common map operations
	mapPool *sync.Pool

	allocatedBuilders int64
	allocatedSegments int64
	allocatedBuffers  int64
	allocatedOptions  int64
	allocatedMaps     int64
}

// drainPool removes potentially corrupted items from a sync.Pool
// This is called when pool corruption is detected to prevent repeated failures
// PERFORMANCE: Drains up to 8 items to balance recovery vs overhead
func drainPool(pool *sync.Pool) {
	for i := 0; i < 8; i++ {
		// Non-blocking drain - Get returns nil if pool is empty
		if pool.Get() == nil {
			break
		}
	}
}

// newUnifiedResourceManager creates a new unified resource manager
func newUnifiedResourceManager() *unifiedResourceManager {
	return &unifiedResourceManager{
		stringBuilderPool: &sync.Pool{
			New: func() any {
				sb := &strings.Builder{}
				sb.Grow(1024) // PERFORMANCE: Increased from 512 to reduce grow() calls
				return sb
			},
		},
		pathSegmentPool: &sync.Pool{
			New: func() any {
				return make([]internal.PathSegment, 0, 8)
			},
		},
		bufferPool: &sync.Pool{
			New: func() any {
				return make([]byte, 0, 1024)
			},
		},
		optionsPool: &sync.Pool{
			New: func() any {
				opts := DefaultConfig()
				return &opts
			},
		},
		mapPool: &sync.Pool{
			New: func() any {
				return make(map[string]any, 8)
			},
		},
	}
}

func (urm *unifiedResourceManager) GetStringBuilder() *strings.Builder {
	obj := urm.stringBuilderPool.Get()
	sb, ok := obj.(*strings.Builder)
	if !ok {
		// Pool corruption detected: type assertion failed
		// Log this rare event for debugging purposes
		slog.Debug("pool corruption detected: string builder type assertion failed, draining pool", "type", fmt.Sprintf("%T", obj))
		// RECOVERY: Drain corrupted pool to prevent repeated failures
		drainPool(urm.stringBuilderPool)
		// Fallback: create new builder if type assertion fails
		sb = &strings.Builder{}
		sb.Grow(1024) // PERFORMANCE: Match pool initial capacity
	}
	sb.Reset()
	atomic.AddInt64(&urm.allocatedBuilders, 1)
	return sb
}

func (urm *unifiedResourceManager) PutStringBuilder(sb *strings.Builder) {
	// Use centralized constants from internal package
	if sb == nil {
		return
	}

	c := sb.Cap()
	if c >= internal.MinPoolBuilderCap && c <= internal.MaxPoolBuilderCap {
		sb.Reset()
		urm.stringBuilderPool.Put(sb)
	}
	// oversized builders are discarded to prevent pool bloat

	// Decrement counter after processing (whether returned to pool or discarded)
	atomic.AddInt64(&urm.allocatedBuilders, -1)
}

func (urm *unifiedResourceManager) GetPathSegments() []internal.PathSegment {
	obj := urm.pathSegmentPool.Get()
	segments, ok := obj.([]internal.PathSegment)
	if !ok {
		// Pool corruption detected: type assertion failed
		// Log this rare event for debugging purposes
		slog.Debug("pool corruption detected: path segments type assertion failed, draining pool", "type", fmt.Sprintf("%T", obj))
		// RECOVERY: Drain corrupted pool to prevent repeated failures
		drainPool(urm.pathSegmentPool)
		// Fallback: create new slice if type assertion fails
		segments = make([]internal.PathSegment, 0, 8)
	}
	segments = segments[:0]
	atomic.AddInt64(&urm.allocatedSegments, 1)
	return segments
}

func (urm *unifiedResourceManager) PutPathSegments(segments []internal.PathSegment) {
	// Use centralized constants from internal package
	if segments == nil {
		return
	}

	if cap(segments) >= internal.MinPoolSliceSize && cap(segments) <= 32 {
		segments = segments[:0]
		urm.pathSegmentPool.Put(segments)
	}
	// oversized segments are discarded to prevent pool bloat

	// Decrement counter after processing (whether returned to pool or discarded)
	atomic.AddInt64(&urm.allocatedSegments, -1)
}

func (urm *unifiedResourceManager) GetBuffer() []byte {
	obj := urm.bufferPool.Get()
	buf, ok := obj.([]byte)
	if !ok {
		// Pool corruption detected: type assertion failed
		// Log this rare event for debugging purposes
		slog.Debug("pool corruption detected: buffer type assertion failed, draining pool", "type", fmt.Sprintf("%T", obj))
		// RECOVERY: Drain corrupted pool to prevent repeated failures
		drainPool(urm.bufferPool)
		// Fallback: create new buffer if type assertion fails
		buf = make([]byte, 0, 1024)
	}
	buf = buf[:0]
	atomic.AddInt64(&urm.allocatedBuffers, 1)
	return buf
}

func (urm *unifiedResourceManager) PutBuffer(buf []byte) {
	// Use centralized constants from internal package
	if buf == nil {
		return
	}

	if cap(buf) >= internal.MinPoolBufferSize && cap(buf) <= internal.MaxPoolBufferSize {
		buf = buf[:0]
		urm.bufferPool.Put(buf)
	}
	// oversized buffers are discarded to prevent pool bloat

	// Decrement counter after processing (whether returned to pool or discarded)
	atomic.AddInt64(&urm.allocatedBuffers, -1)
}

// GetOptions gets a Config from the pool
func (urm *unifiedResourceManager) GetOptions() *Config {
	obj := urm.optionsPool.Get()
	opts, ok := obj.(*Config)
	if !ok {
		// Pool corruption detected: type assertion failed
		// Log this rare event for debugging purposes
		slog.Debug("pool corruption detected: options type assertion failed", "type", fmt.Sprintf("%T", obj))
		// Fallback: create new options if type assertion fails
		cfg := DefaultConfig()
		opts = &cfg
	}
	// Reset to default configuration
	*opts = DefaultConfig()
	atomic.AddInt64(&urm.allocatedOptions, 1)
	return opts
}

// PutOptions returns a Config to the pool
func (urm *unifiedResourceManager) PutOptions(opts *Config) {
	if opts != nil {
		defer atomic.AddInt64(&urm.allocatedOptions, -1)
		// Clear context to prevent memory leaks
		opts.Context = nil
		urm.optionsPool.Put(opts)
	}
}

// GetMap gets a map from the pool
// PERFORMANCE: Reusable maps for JSON object operations
func (urm *unifiedResourceManager) GetMap() map[string]any {
	obj := urm.mapPool.Get()
	m, ok := obj.(map[string]any)
	if !ok {
		slog.Debug("pool corruption detected: map type assertion failed", "type", fmt.Sprintf("%T", obj))
		m = make(map[string]any, 8)
	}
	// PERFORMANCE: Use Go 1.21+ clear() for O(1) map clearing instead of O(n) loop
	clear(m)
	atomic.AddInt64(&urm.allocatedMaps, 1)
	return m
}

// PutMap returns a map to the pool
// RESOURCE FIX: Always decrement counter regardless of pooling decision
func (urm *unifiedResourceManager) PutMap(m map[string]any) {
	// Always decrement counter first to prevent counter leaks
	defer atomic.AddInt64(&urm.allocatedMaps, -1)

	if m == nil {
		return
	}
	// Only pool small to medium maps
	const maxMapSize = 64
	if len(m) <= maxMapSize {
		urm.mapPool.Put(m)
	}
}

func (urm *unifiedResourceManager) getStats() resourceManagerStats {
	return resourceManagerStats{
		allocatedBuilders: atomic.LoadInt64(&urm.allocatedBuilders),
		allocatedSegments: atomic.LoadInt64(&urm.allocatedSegments),
		allocatedBuffers:  atomic.LoadInt64(&urm.allocatedBuffers),
		allocatedOptions:  atomic.LoadInt64(&urm.allocatedOptions),
		allocatedMaps:     atomic.LoadInt64(&urm.allocatedMaps),
	}
}

type resourceManagerStats struct {
	allocatedBuilders int64
	allocatedSegments int64
	allocatedBuffers  int64
	allocatedOptions  int64
	allocatedMaps     int64
}

var (
	globalResourceManager     *unifiedResourceManager
	globalResourceManagerOnce sync.Once
)

func getGlobalResourceManager() *unifiedResourceManager {
	globalResourceManagerOnce.Do(func() {
		globalResourceManager = newUnifiedResourceManager()
	})
	return globalResourceManager
}
