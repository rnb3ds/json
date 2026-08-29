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
	// SECURITY: For hints larger than pool capacity, allocate directly — a
	// pooled cap-16 slice would immediately regrow and be dropped at Put
	// (cap > 32 is not pooled), churning allocations for deep paths.
	if hint > largeSliceSize {
		s := make([]PathSegment, 0, hint)
		return &s
	}
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
