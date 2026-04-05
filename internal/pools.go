package internal

import (
	"strings"
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
)

// ----------------------------------------------------------------------------
// STRING BUILDER POOL - For string building operations
// PERFORMANCE: Reduces allocations in string concatenation
// ----------------------------------------------------------------------------

var stringBuilderPool = sync.Pool{
	New: func() any {
		sb := &strings.Builder{}
		sb.Grow(256)
		return sb
	},
}

// GetStringBuilder retrieves a pooled strings.Builder
func GetStringBuilder() *strings.Builder {
	sb := stringBuilderPool.Get().(*strings.Builder)
	sb.Reset()
	return sb
}

// PutStringBuilder returns a strings.Builder to the pool
func PutStringBuilder(sb *strings.Builder) {
	if sb == nil {
		return
	}
	// Don't pool very large builders
	if sb.Cap() > 16*1024 {
		return
	}
	sb.Reset()
	stringBuilderPool.Put(sb)
}

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
	if s == nil {
		return
	}
	c := cap(*s)
	if c > 32 {
		return // Don't pool very large slices
	}
	*s = (*s)[:0]
	switch {
	case c <= 4:
		smallPathPool.Put(s)
	case c <= 8:
		mediumPathPool.Put(s)
	default:
		largePathPool.Put(s)
	}
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

// ----------------------------------------------------------------------------
// STREAMING SLICE POOL - For streaming JSON operations
// PERFORMANCE: Reduces allocations in streaming scenarios
// ----------------------------------------------------------------------------

var streamingSlicePool = sync.Pool{
	New: func() any {
		s := make([]any, 0, mediumSliceSize)
		return &s
	},
}

// GetStreamingSlice retrieves a pooled []any slice for streaming
func GetStreamingSlice(hint int) *[]any {
	if hint <= mediumSliceSize {
		s := streamingSlicePool.Get().(*[]any)
		*s = (*s)[:0]
		return s
	}
	// For large hints, allocate directly
	newSlice := make([]any, 0, hint)
	return &newSlice
}

// PutStreamingSlice returns a []any slice to the streaming pool
func PutStreamingSlice(s *[]any) {
	if s == nil || cap(*s) > maxSliceCap {
		return
	}
	*s = (*s)[:0]
	streamingSlicePool.Put(s)
}

// ----------------------------------------------------------------------------
// MAP POOL - For JSON object decoding
// PERFORMANCE: Reduces allocations when decoding JSON objects
// ----------------------------------------------------------------------------

var (
	// smallMapPool pools small map[string]any (cap 8)
	smallMapPool = sync.Pool{
		New: func() any {
			return make(map[string]any, smallSliceSize)
		},
	}

	// mediumMapPool pools medium map[string]any (cap 32)
	mediumMapPool = sync.Pool{
		New: func() any {
			return make(map[string]any, mediumSliceSize)
		},
	}

	// largeMapPool pools large map[string]any (cap 128)
	largeMapPool = sync.Pool{
		New: func() any {
			return make(map[string]any, largeSliceSize)
		},
	}
)

// GetStreamingMap retrieves a pooled map[string]any
func GetStreamingMap(hint int) map[string]any {
	switch {
	case hint <= smallSliceSize:
		m := smallMapPool.Get().(map[string]any)
		// Clear the map
		for k := range m {
			delete(m, k)
		}
		return m
	case hint <= mediumSliceSize:
		m := mediumMapPool.Get().(map[string]any)
		for k := range m {
			delete(m, k)
		}
		return m
	case hint <= largeSliceSize:
		m := largeMapPool.Get().(map[string]any)
		for k := range m {
			delete(m, k)
		}
		return m
	default:
		return make(map[string]any, hint)
	}
}

// PutStreamingMap returns a map[string]any to the pool
// Note: Uses original size before clearing since maps don't have capacity
func PutStreamingMap(m map[string]any) {
	if m == nil {
		return
	}
	// Capture original size BEFORE clearing - critical for correct pool selection
	originalSize := len(m)

	// Clear the map
	for k := range m {
		delete(m, k)
	}

	// Use original size for pool selection to ensure correct bucket
	switch {
	case originalSize <= smallSliceSize:
		smallMapPool.Put(m)
	case originalSize <= mediumSliceSize:
		mediumMapPool.Put(m)
	case originalSize <= largeSliceSize:
		largeMapPool.Put(m)
	// Maps larger than largeSliceSize are discarded to prevent pool bloat
	}
}
