package json

import (
	"errors"
	"time"

	"github.com/cybergodev/json/internal"
)

// Configuration constants with optimized defaults for production workloads.
const (
	// Buffer and Pool Sizes - Optimized for production workloads
	DefaultBufferSize = 1024
	MaxPoolBufferSize = 32768 // 32KB max for better buffer reuse
	MinPoolBufferSize = 256   // 256B min for efficiency

	// Cache Sizes - Balanced for performance and memory
	DefaultCacheSize = 128

	// Operation Limits - Secure defaults with reasonable headroom
	DefaultMaxJSONSize     = 100 * 1024 * 1024 // 100MB
	DefaultMaxNestingDepth = 200
	DefaultMaxPathDepth    = 50
	DefaultMaxConcurrency  = 50

	// Internal operation limits
	DefaultMaxSecuritySize   = 10 * 1024 * 1024
	DefaultMaxObjectKeys     = 100000
	DefaultMaxArrayElements  = 100000
	DefaultMaxBatchSize      = 2000
	DefaultParallelThreshold = 10

	// Timing and Intervals - Optimized for responsiveness
	SlowOperationThreshold = 100 * time.Millisecond

	// Retry and Timeout - Production-ready settings
	DefaultOperationTimeout = 30 * time.Second

	// Processor lifecycle timeouts
	CloseOperationTimeout = 5 * time.Second // Timeout waiting for active operations during Close()
	SemaphoreDrainTimeout = 1 * time.Second // Timeout for draining concurrency semaphore

	// LargeStringHashThreshold is the byte threshold for using sampling-based hash.
	// Re-exported from internal package for public API access.
	LargeStringHashThreshold = internal.LargeStringHashThreshold

	// Path Validation - Secure but flexible
	// MaxPathLength is the maximum allowed path length for security.
	// Re-exported from internal package for public API access.
	MaxPathLength = internal.MaxPathLength

	// Cache TTL
	DefaultCacheTTL = 5 * time.Minute

	// Cache key constants
	// MaxCacheKeyLength is the maximum allowed cache key length.
	// Re-exported from internal package for public API access.
	MaxCacheKeyLength = internal.MaxCacheKeyLength

	// Validation constants
	ValidationBOMPrefix = "\uFEFF" // UTF-8 BOM prefix to detect and remove
)

// InvalidArrayIndex is a sentinel value indicating an invalid or out-of-bounds array index.
// Returned by array parsing functions when the index cannot be determined
// (e.g., invalid format, overflow, or empty string).
//
//	index := processor.ParseArrayIndex(str)
//	if index == InvalidArrayIndex {
//	    // Handle invalid index
//	}
const InvalidArrayIndex = internal.ArrayIndexInvalid

// clampInt64 clamps an int64 value to the specified range.
// If value is <= 0, it is set to min. If value > max, it is set to max.
func clampInt64(value *int64, min, max int64) {
	if *value <= 0 {
		*value = min
	} else if *value > max {
		*value = max
	}
}

// clampInt clamps an int value to the specified range.
// If value is <= 0, it is set to min. If value > max, it is set to max.
func clampInt(value *int, min, max int) {
	if *value <= 0 {
		*value = min
	} else if *value > max {
		*value = max
	}
}

// DefaultConfig returns the default configuration.
// Creates a new instance each time to allow modifications without affecting other callers.
func DefaultConfig() Config {
	return Config{
		// Cache Settings
		MaxCacheSize: DefaultCacheSize,
		CacheTTL:     DefaultCacheTTL,
		EnableCache:  true,
		CacheResults: true,

		// Size Limits
		MaxJSONSize:  DefaultMaxJSONSize,
		MaxPathDepth: DefaultMaxPathDepth,
		MaxBatchSize: DefaultMaxBatchSize,

		// Security Limits
		MaxNestingDepthSecurity:   DefaultMaxNestingDepth,
		MaxSecurityValidationSize: DefaultMaxSecuritySize,
		MaxObjectKeys:             DefaultMaxObjectKeys,
		MaxArrayElements:          DefaultMaxArrayElements,
		FullSecurityScan:          false,

		// Concurrency
		MaxConcurrency:    DefaultMaxConcurrency,
		ParallelThreshold: DefaultParallelThreshold,

		// Processing Options
		EnableValidation: true,
		StrictMode:       false,
		CreatePaths:      false,
		CleanupNulls:     false,
		CompactArrays:    false,
		ContinueOnError:  false,

		// Input/Output Options
		AllowComments:    false,
		PreserveNumbers:  false,
		ValidateInput:    true,
		ValidateFilePath: true,
		SkipValidation:   false,

		// Encoding Options
		Pretty:          false,
		Indent:          "  ",
		Prefix:          "",
		EscapeHTML:      true,
		SortKeys:        false,
		ValidateUTF8:    true,
		MaxDepth:        100,
		DisallowUnknown: false,
		FloatPrecision:  -1,
		FloatTruncate:   false,
		DisableEscaping: false,
		EscapeUnicode:   false,
		EscapeSlash:     false,
		EscapeNewlines:  true,
		EscapeTabs:      true,
		IncludeNulls:    true,
		CustomEscapes:   nil,

		// Observability
		EnableMetrics:     false,
		EnableHealthCheck: false,

		// Context
		Context: nil,
	}
}

// Clone creates a copy of the configuration.
// Performs a deep copy of reference types (maps, slices).
func (c Config) Clone() Config {
	clone := c

	// Deep copy CustomEscapes map
	if c.CustomEscapes != nil {
		clone.CustomEscapes = make(map[rune]string, len(c.CustomEscapes))
		for k, v := range c.CustomEscapes {
			clone.CustomEscapes[k] = v
		}
	}

	return clone
}

// Validate validates the configuration and applies corrections.
// This is the single source of truth for config validation.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config cannot be nil")
	}

	// Size and depth limits
	clampInt64(&c.MaxJSONSize, 1024*1024, 100*1024*1024)
	clampInt(&c.MaxPathDepth, 10, 200)
	clampInt(&c.MaxNestingDepthSecurity, 10, 200)
	clampInt(&c.MaxConcurrency, 1, 200)
	clampInt(&c.ParallelThreshold, 1, 50)

	// Security limits
	clampInt(&c.MaxObjectKeys, 100, 100000)
	clampInt(&c.MaxArrayElements, 100, 100000)
	clampInt64(&c.MaxSecurityValidationSize, 1024*1024, 100*1024*1024)

	// Cache settings
	if c.MaxCacheSize < 0 {
		c.MaxCacheSize = 0
		c.EnableCache = false
	} else if c.MaxCacheSize > 2000 {
		c.MaxCacheSize = 2000
	}

	if c.CacheTTL <= 0 {
		c.CacheTTL = DefaultCacheTTL
	}

	// Encoding options
	if c.MaxDepth < 0 || c.MaxDepth > 1000 {
		c.MaxDepth = 100
	}
	if c.FloatPrecision < -1 || c.FloatPrecision > 15 {
		c.FloatPrecision = -1
	}

	// Batch size limits
	clampInt(&c.MaxBatchSize, 10, 10000)

	return nil
}

// Config accessor methods.
// These methods implement the CacheConfig interface used by internal/cache.go
// and provide consistent API for testing and interface-based programming.

// Required by CacheConfig interface - do not remove.
func (c *Config) IsCacheEnabled() bool       { return c.EnableCache }
func (c *Config) GetMaxCacheSize() int       { return c.MaxCacheSize }
func (c *Config) GetCacheTTL() time.Duration { return c.CacheTTL }

// Convenience accessor methods for testing and interface-based usage.
// Note: For direct configuration access in application code, prefer field access:
//
//	cfg.MaxJSONSize instead of cfg.GetMaxJSONSize()
//	cfg.StrictMode instead of cfg.IsStrictMode()
//
// These methods are primarily for testing and future interface compatibility.
func (c *Config) GetMaxJSONSize() int64        { return c.MaxJSONSize }
func (c *Config) GetMaxPathDepth() int         { return c.MaxPathDepth }
func (c *Config) GetMaxConcurrency() int       { return c.MaxConcurrency }
func (c *Config) IsMetricsEnabled() bool       { return c.EnableMetrics }
func (c *Config) IsHealthCheckEnabled() bool   { return c.EnableHealthCheck }
func (c *Config) IsStrictMode() bool           { return c.StrictMode }
func (c *Config) IsCommentsAllowed() bool      { return c.AllowComments }
func (c *Config) ShouldPreserveNumbers() bool  { return c.PreserveNumbers }
func (c *Config) ShouldCreatePaths() bool      { return c.CreatePaths }
func (c *Config) ShouldCleanupNulls() bool     { return c.CleanupNulls }
func (c *Config) ShouldCompactArrays() bool    { return c.CompactArrays }
func (c *Config) ShouldValidateInput() bool    { return c.ValidateInput }
func (c *Config) GetMaxNestingDepth() int      { return c.MaxNestingDepthSecurity }
func (c *Config) ShouldValidateFilePath() bool { return c.ValidateFilePath }

// =============================================================================
// API Unification - Config presets for common scenarios
// =============================================================================

// SecurityConfig returns a configuration with enhanced security settings
// for processing untrusted input from external sources.
//
// This is the recommended configuration for:
//   - Public APIs and web services
//   - User-submitted data
//   - External webhooks
//   - Authentication endpoints
//   - Financial data processing
//
// Key characteristics:
//   - Full security scan enabled for all input
//   - Strict mode enabled for predictable parsing
//   - Conservative limits for untrusted payloads
//   - Caching enabled for repeated operations
//
// This function unifies HighSecurityConfig and WebAPIConfig into a single entry point.
func SecurityConfig() Config {
	config := DefaultConfig()
	// Security settings - conservative limits for untrusted input
	config.MaxNestingDepthSecurity = 30
	config.MaxSecurityValidationSize = 10 * 1024 * 1024 // 10MB
	config.MaxObjectKeys = 5000
	config.MaxArrayElements = 5000
	config.MaxJSONSize = 10 * 1024 * 1024 // 10MB max payload
	config.MaxPathDepth = 30
	// Security features enabled
	config.FullSecurityScan = true
	config.StrictMode = true
	config.EnableValidation = true
	// Performance settings
	config.EnableCache = true
	config.MaxCacheSize = 256
	config.CacheTTL = 3 * time.Minute
	return config
}

// =============================================================================
// Unified Config Presets - Use these for common scenarios
// =============================================================================

// PrettyConfig returns a Config for pretty-printed JSON output.
// This is the unified version that returns Config instead of EncodeConfig.
//
// Example:
//
//	result, err := json.Encode(data, json.PrettyConfig())
func PrettyConfig() Config {
	cfg := DefaultConfig()
	cfg.Pretty = true
	cfg.Indent = "  "
	return cfg
}
