package internal

import (
	"sync"
)

// ============================================================================
// PERFORMANCE OPTIMIZATION POOLS
// These pools reduce heap allocations in hot paths
// ============================================================================

const (
	// Pool size thresholds
	smallSliceSize  = 8
	mediumSliceSize = 32
	largeSliceSize  = 128

	// Maximum capacity to pool (prevent memory bloat)
	maxSliceCap = 256
	// maxMapCap removed - maps are no longer pooled due to race condition risk
)

// ----------------------------------------------------------------------------
// RESULTS SLICE POOL - For recursive processing results
// PERFORMANCE: Reduces allocations in hot recursive paths
// ----------------------------------------------------------------------------

var (
	// smallResultsPool pools small []any slices (cap 8)
	smallResultsPool = sync.Pool{
		New: func() any {
			s := make([]any, 0, smallSliceSize)
			return &s
		},
	}

	// mediumResultsPool pools medium []any slices (cap 32)
	mediumResultsPool = sync.Pool{
		New: func() any {
			s := make([]any, 0, mediumSliceSize)
			return &s
		},
	}

	// largeResultsPool pools large []any slices (cap 128)
	largeResultsPool = sync.Pool{
		New: func() any {
			s := make([]any, 0, largeSliceSize)
			return &s
		},
	}
)

// GetResultsSlice retrieves a pooled []any slice with appropriate capacity
// SECURITY FIX: For hints larger than pool capacity, allocate directly
// This prevents capacity mismatch and reduces resize operations
func GetResultsSlice(hint int) *[]any {
	var s *[]any
	switch {
	case hint <= smallSliceSize:
		s = smallResultsPool.Get().(*[]any)
	case hint <= mediumSliceSize:
		s = mediumResultsPool.Get().(*[]any)
	case hint <= largeSliceSize:
		s = largeResultsPool.Get().(*[]any)
	default:
		// SECURITY FIX: For large hints, allocate directly without using pool
		// This ensures the slice has sufficient capacity
		newSlice := make([]any, 0, hint)
		return &newSlice
	}
	*s = (*s)[:0]
	return s
}

// PutResultsSlice returns a []any slice to the appropriate pool
func PutResultsSlice(s *[]any) {
	if s == nil {
		return
	}
	c := cap(*s)
	if c > maxSliceCap {
		return // Don't pool very large slices
	}
	*s = (*s)[:0]
	switch {
	case c <= smallSliceSize:
		smallResultsPool.Put(s)
	case c <= mediumSliceSize:
		mediumResultsPool.Put(s)
	case c <= largeSliceSize:
		largeResultsPool.Put(s)
	}
}

// ----------------------------------------------------------------------------
// ERROR SLICE POOL - For collecting errors in recursive processing
// ----------------------------------------------------------------------------

var errorSlicePool = sync.Pool{
	New: func() any {
		s := make([]error, 0, smallSliceSize)
		return &s
	},
}

// GetErrorSlice retrieves a pooled []error slice
func GetErrorSlice() *[]error {
	s := errorSlicePool.Get().(*[]error)
	*s = (*s)[:0]
	return s
}

// PutErrorSlice returns a []error slice to the pool
func PutErrorSlice(s *[]error) {
	if s == nil || cap(*s) > maxSliceCap {
		return
	}
	*s = (*s)[:0]
	errorSlicePool.Put(s)
}

// ----------------------------------------------------------------------------
// MAP POOL - For deep merge operations and intermediate results
// SECURITY FIX: Removed map pooling due to race condition risk
// Maps returned from pools could be shared between goroutines
// ----------------------------------------------------------------------------

// GetMergeMap creates a new map for merge operations
// SECURITY: No longer uses pooling to prevent race conditions
// PERFORMANCE: Slight increase in allocations, but eliminates data races
func GetMergeMap(hint int) map[string]any {
	if hint <= smallSliceSize {
		return make(map[string]any, smallSliceSize)
	}
	if hint <= mediumSliceSize {
		return make(map[string]any, mediumSliceSize)
	}
	return make(map[string]any, hint)
}

// PutMergeMap is a no-op for safety
// Maps are no longer pooled to prevent race conditions
// The map will be garbage collected naturally
func PutMergeMap(m map[string]any) {
	// No-op - let GC handle the map
	// This eliminates race conditions from pool reuse
}

// ----------------------------------------------------------------------------
// PATH SEGMENT SLICE POOL - For path parsing results
// ----------------------------------------------------------------------------

var (
	// smallPathPool pools small []PathSegment slices (cap 4)
	smallPathPool = sync.Pool{
		New: func() any {
			s := make([]PathSegment, 0, 4)
			return &s
		},
	}

	// mediumPathPool pools medium []PathSegment slices (cap 8)
	mediumPathPool = sync.Pool{
		New: func() any {
			s := make([]PathSegment, 0, 8)
			return &s
		},
	}

	// largePathPool pools large []PathSegment slices (cap 16)
	largePathPool = sync.Pool{
		New: func() any {
			s := make([]PathSegment, 0, 16)
			return &s
		},
	}
)

// GetPathSegmentSlice retrieves a pooled []PathSegment slice
func GetPathSegmentSlice(hint int) *[]PathSegment {
	var s *[]PathSegment
	switch {
	case hint <= 4:
		s = smallPathPool.Get().(*[]PathSegment)
	case hint <= 8:
		s = mediumPathPool.Get().(*[]PathSegment)
	default:
		s = largePathPool.Get().(*[]PathSegment)
	}
	*s = (*s)[:0]
	return s
}

// PutPathSegmentSlice returns a []PathSegment slice to the pool
func PutPathSegmentSlice(s *[]PathSegment) {
	if s == nil || cap(*s) > 32 {
		return
	}
	*s = (*s)[:0]
	switch {
	case cap(*s) <= 4:
		smallPathPool.Put(s)
	case cap(*s) <= 8:
		mediumPathPool.Put(s)
	default:
		largePathPool.Put(s)
	}
}

// ----------------------------------------------------------------------------
// STRING SET - For deduplication operations in merge
// SECURITY: Maps are no longer pooled due to race condition risk
// ----------------------------------------------------------------------------

// GetStringSet creates a new map[string]bool for deduplication
// SECURITY: No longer uses pooling to prevent race conditions
func GetStringSet() map[string]bool {
	return make(map[string]bool, mediumSliceSize)
}

// PutStringSet is a no-op for safety
// Maps are no longer pooled to prevent race conditions
func PutStringSet(s map[string]bool) {
	// No-op - let GC handle the map
}

// ----------------------------------------------------------------------------
// ANY SLICE POOL - For flattened results
// ----------------------------------------------------------------------------

var flattenedSlicePool = sync.Pool{
	New: func() any {
		s := make([]any, 0, mediumSliceSize)
		return &s
	},
}

// GetFlattenedSlice retrieves a pooled slice for flattening operations
func GetFlattenedSlice() *[]any {
	s := flattenedSlicePool.Get().(*[]any)
	*s = (*s)[:0]
	return s
}

// PutFlattenedSlice returns a slice used for flattening
func PutFlattenedSlice(s *[]any) {
	if s == nil || cap(*s) > maxSliceCap {
		return
	}
	*s = (*s)[:0]
	flattenedSlicePool.Put(s)
}
