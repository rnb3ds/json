package internal

// Unified constants for the JSON library
// These constants are used across multiple files to ensure consistency
// IMPORTANT: These are the single source of truth - do not duplicate in other packages

const (
	// Depth limits for various operations
	MaxDeepMergeDepth      = 100 // Maximum depth for deep merge operations
	MaxPathParseDepth      = 100 // Maximum depth for path parsing
	MaxNestingDepth        = 200 // Maximum JSON nesting depth for security validation
	MaxSensitiveDataDepth  = 10  // Maximum depth for sensitive data detection

	// Path and cache limits
	MaxPathLength     = 5000 // Maximum path length for security (single source of truth)
	MaxCacheKeyLength = 1024 // Maximum cache key length to prevent memory issues

	// Array index sentinel values
	// These values are distinct from valid array indices to avoid confusion
	// ArrayIndexInvalid is returned when the index cannot be determined
	ArrayIndexInvalid = -999999 // Kept for backward compatibility
)
