package internal

// Unified constants for the JSON library
// These constants are used across multiple files to ensure consistency
// IMPORTANT: These are the single source of truth - do not duplicate in other packages

const (
	// Depth limits for various operations
	MaxDeepMergeDepth     = 100 // Maximum depth for deep merge operations
	MaxPathParseDepth     = 100 // Maximum depth for path parsing
	MaxNestingDepth       = 200 // Maximum JSON nesting depth for security validation
	MaxSensitiveDataDepth = 10  // Maximum depth for sensitive data detection

	// Path and cache limits
	MaxPathLength     = 5000 // Maximum path length for security (single source of truth)
	MaxCacheKeyLength = 1024 // Maximum cache key length to prevent memory issues

	// Array index sentinel values
	// These values are distinct from valid array indices to avoid confusion
	// ArrayIndexInvalid is returned when the index cannot be determined
	ArrayIndexInvalid = -999999 // Kept for backward compatibility

	// ============================================================================
	// POOL SIZE LIMITS - Single source of truth for all resource pools
	// These limits prevent memory bloat while maintaining good pool hit rates
	// ============================================================================

	// Buffer pool limits (for []byte)
	MinPoolBufferSize = 256   // Minimum capacity to pool (smaller buffers not worth pooling)
	MaxPoolBufferSize = 32768 // Maximum capacity to pool (32KB - larger buffers discarded)

	// Slice pool limits (for []any, []PathSegment, etc.)
	MinPoolSliceSize = 4   // Minimum slice capacity to pool
	MaxPoolSliceSize = 256 // Maximum slice capacity to pool

	// Map pool limits (for map[string]any)
	MinPoolMapSize = 4  // Minimum map size to pool
	MaxPoolMapSize = 64 // Maximum map size to pool

	// StringBuilder pool limits
	MinPoolBuilderCap = 256   // Minimum StringBuilder capacity to pool
	MaxPoolBuilderCap = 16384 // Maximum StringBuilder capacity to pool (16KB)
)
