package json

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cybergodev/json/internal"
)

// unifiedResourceManager consolidates all resource management for optimal performance
// REFACTORED: All pool operations delegate to internal/pools.go to avoid duplication
// This struct now only tracks allocation statistics for debugging purposes
type unifiedResourceManager struct {
	// Allocation counters for debugging/metrics
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
	return &unifiedResourceManager{}
}

// GetStringBuilder retrieves a pooled strings.Builder
// DELEGATED: Uses internal/pools.go to avoid duplication
func (urm *unifiedResourceManager) GetStringBuilder() *strings.Builder {
	return internal.GetStringBuilder()
}

// PutStringBuilder returns a strings.Builder to the pool
// DELEGATED: Uses internal/pools.go to avoid duplication
func (urm *unifiedResourceManager) PutStringBuilder(sb *strings.Builder) {
	internal.PutStringBuilder(sb)
}

// GetPathSegments retrieves a pooled path segment slice
// DELEGATED: Uses internal/pools.go to avoid duplication
func (urm *unifiedResourceManager) GetPathSegments() []internal.PathSegment {
	segments := internal.GetPathSegmentSlice(8)
	atomic.AddInt64(&urm.allocatedSegments, 1)
	return *segments
}

// PutPathSegments returns a path segment slice to the pool
// DELEGATED: Uses internal/pools.go to avoid duplication
func (urm *unifiedResourceManager) PutPathSegments(segments []internal.PathSegment) {
	if segments == nil {
		return
	}
	atomic.AddInt64(&urm.allocatedSegments, -1)
	internal.PutPathSegmentSlice(&segments)
}

// GetBuffer retrieves a pooled byte slice
// DELEGATED: Uses internal/pools.go to avoid duplication
func (urm *unifiedResourceManager) GetBuffer() []byte {
	buf := internal.GetByteSliceWithHint(1024)
	atomic.AddInt64(&urm.allocatedBuffers, 1)
	return *buf
}

// PutBuffer returns a byte slice to the pool
// DELEGATED: Uses internal/pools.go to avoid duplication
func (urm *unifiedResourceManager) PutBuffer(buf []byte) {
	if buf == nil {
		return
	}
	atomic.AddInt64(&urm.allocatedBuffers, -1)
	internal.PutByteSlice(&buf)
}

// GetMap retrieves a pooled map
// PERFORMANCE: Reusable maps for JSON object operations
func (urm *unifiedResourceManager) GetMap() map[string]any {
	m := internal.GetStreamingMap(8)
	atomic.AddInt64(&urm.allocatedMaps, 1)
	return m
}

// PutMap returns a map to the pool
// RESOURCE FIX: Always decrement counter regardless of pooling decision
func (urm *unifiedResourceManager) PutMap(m map[string]any) {
	// Always decrement counter first to prevent counter leaks
	atomic.AddInt64(&urm.allocatedMaps, -1)
	internal.PutStreamingMap(m)
}

func (urm *unifiedResourceManager) getStats() resourceManagerStats {
	return resourceManagerStats{
		allocatedSegments: atomic.LoadInt64(&urm.allocatedSegments),
		allocatedBuffers:  atomic.LoadInt64(&urm.allocatedBuffers),
		allocatedOptions:  atomic.LoadInt64(&urm.allocatedOptions),
		allocatedMaps:     atomic.LoadInt64(&urm.allocatedMaps),
	}
}

type resourceManagerStats struct {
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
