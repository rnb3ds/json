package json

import (
	"errors"
	"time"

	"github.com/cybergodev/json/internal"
)

// ConfigInterface defines the interface for configuration objects
type ConfigInterface interface {
	IsCacheEnabled() bool
	GetMaxCacheSize() int
	GetCacheTTL() time.Duration
	GetMaxJSONSize() int64
	GetMaxPathDepth() int
	GetMaxConcurrency() int
	IsMetricsEnabled() bool
	IsStrictMode() bool
	IsCommentsAllowed() bool
	ShouldPreserveNumbers() bool
	ShouldCreatePaths() bool
	ShouldCleanupNulls() bool
	ShouldCompactArrays() bool
	ShouldValidateInput() bool
	GetMaxNestingDepth() int
}

// Configuration constants with optimized defaults for production workloads.
const (
	// Buffer and Pool Sizes - Optimized for production workloads
	DefaultBufferSize = 1024
	MaxPoolBufferSize = 32768 // 32KB max for better buffer reuse
	MinPoolBufferSize = 256   // 256B min for efficiency

	// Internal buffer/pool sizes (not exported)
	defaultPathSegmentCap    = 8
	maxPathSegmentCap        = 32 // Reduced from 128
	defaultStringBuilderSize = 256

	// Cache Sizes - Balanced for performance and memory
	DefaultCacheSize = 128

	// Internal cache sizes (not exported)
	maxCacheEntries      = 512
	cacheCleanupKeepSize = 256

	// Operation Limits - Secure defaults with reasonable headroom
	// InvalidArrayIndex is a sentinel value indicating an invalid or out-of-bounds array index.
	// This value is returned by array parsing functions when the index cannot be determined
	// (e.g., invalid format, overflow, or empty string).
	// IMPORTANT: Do not use this value as a valid array index. Always check if the returned
	// value equals InvalidArrayIndex before using it.
	// Example:
	//
	//	index := helper.ParseArrayIndex(str)
	//	if index == json.InvalidArrayIndex {
	//	    // Handle invalid index
	//	}
	InvalidArrayIndex      = -999999
	DefaultMaxJSONSize     = 100 * 1024 * 1024 // 100MB
	DefaultMaxNestingDepth = 200
	DefaultMaxPathDepth    = 50
	DefaultMaxConcurrency  = 50

	// Internal operation limits (not exported)
	defaultMaxSecuritySize   = 10 * 1024 * 1024
	defaultMaxObjectKeys     = 100000
	defaultMaxArrayElements  = 100000
	defaultMaxBatchSize      = 2000
	defaultParallelThreshold = 10

	// Timing and Intervals - Optimized for responsiveness
	// Internal timing constants (not exported)
	memoryPressureCheckInterval = 30 * time.Second
	poolResetInterval           = 60 * time.Second
	poolResetIntervalPressure   = 30 * time.Second
	cacheCleanupInterval        = 30 * time.Second
	deadlockCheckInterval       = 30 * time.Second
	deadlockThreshold           = 30 * time.Second
	SlowOperationThreshold      = 100 * time.Millisecond

	// Retry and Timeout - Production-ready settings
	// Internal retry constants (not exported)
	maxRetries              = 3
	baseRetryDelay          = 10 * time.Millisecond
	DefaultOperationTimeout = 30 * time.Second
	acquireSlotRetryDelay   = 1 * time.Millisecond

	// Processor lifecycle timeouts
	CloseOperationTimeout    = 5 * time.Second // Timeout waiting for active operations during Close()
	SemaphoreDrainTimeout    = 1 * time.Second // Timeout for draining concurrency semaphore
	LargeStringHashThreshold = 4096            // Byte threshold for using sampling-based hash

	// Path Validation - Secure but flexible
	// MaxPathLength is the maximum allowed path length for security.
	// Re-exported from internal package for public API access.
	MaxPathLength = internal.MaxPathLength

	// Internal path validation (not exported)
	maxSegmentLength = 1024

	// Cache TTL
	DefaultCacheTTL = 5 * time.Minute

	// Cache key constants - OPTIMIZED: Increased limits for better cache hit rate
	// Internal cache key constants (not exported)
	cacheKeyHashLength   = 32      // Length for cache key hash
	smallJSONCacheLimit  = 2048    // Limit for caching small JSON strings (fast path)
	mediumJSONCacheLimit = 51200   // Limit for caching medium JSON strings (50KB)
	largeJSONCacheLimit  = 1048576 // Limit for caching large JSON strings (1MB) - OPTIMIZED: increased for better performance
	estimatedKeyOverhead = 32      // Estimated overhead for cache key generation
	largeJSONKeyOverhead = 64      // Overhead for large JSON cache keys
	MaxCacheKeyLength    = 500     // Maximum allowed cache key length

	// Validation constants
	ValidationBOMPrefix = "\uFEFF" // UTF-8 BOM prefix to detect and remove
)

// Error codes for machine-readable error identification.
// Internal use only - users should use errors.Is() with error variables.
const (
	errCodeInvalidJSON       = "ERR_INVALID_JSON"
	errCodePathNotFound      = "ERR_PATH_NOT_FOUND"
	errCodeTypeMismatch      = "ERR_TYPE_MISMATCH"
	errCodeSizeLimit         = "ERR_SIZE_LIMIT"
	errCodeDepthLimit        = "ERR_DEPTH_LIMIT"
	errCodeSecurityViolation = "ERR_SECURITY_VIOLATION"
	errCodeOperationFailed   = "ERR_OPERATION_FAILED"
	errCodeTimeout           = "ERR_TIMEOUT"
	errCodeConcurrencyLimit  = "ERR_CONCURRENCY_LIMIT"
	errCodeProcessorClosed   = "ERR_PROCESSOR_CLOSED"
	errCodeRateLimit         = "ERR_RATE_LIMIT"
)

// DefaultConfig returns the default configuration.
// Creates a new instance each time to allow modifications without affecting other callers.
// PERFORMANCE NOTE: For read-only access in hot paths, cache the result.
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
		MaxBatchSize: defaultMaxBatchSize,

		// Security Limits
		MaxNestingDepthSecurity:   DefaultMaxNestingDepth,
		MaxSecurityValidationSize: defaultMaxSecuritySize,
		MaxObjectKeys:             defaultMaxObjectKeys,
		MaxArrayElements:          defaultMaxArrayElements,
		FullSecurityScan:          false,

		// Concurrency
		MaxConcurrency:    DefaultMaxConcurrency,
		ParallelThreshold: defaultParallelThreshold,

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

// validateConfig validates configuration values and applies corrections
func validateConfig(config *Config) error {
	if config == nil {
		return newOperationError("validate_config", "config cannot be nil", ErrOperationFailed)
	}

	if config.MaxCacheSize < 0 {
		return newOperationError("validate_config", "MaxCacheSize cannot be negative", ErrOperationFailed)
	}

	// Apply defaults for invalid values
	if config.MaxJSONSize <= 0 {
		config.MaxJSONSize = DefaultMaxJSONSize
	}
	if config.MaxPathDepth <= 0 {
		config.MaxPathDepth = DefaultMaxPathDepth
	}
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = DefaultMaxConcurrency
	}
	if config.MaxNestingDepthSecurity <= 0 {
		config.MaxNestingDepthSecurity = DefaultMaxNestingDepth
	}
	if config.MaxObjectKeys <= 0 {
		config.MaxObjectKeys = defaultMaxObjectKeys
	}
	if config.MaxArrayElements <= 0 {
		config.MaxArrayElements = defaultMaxArrayElements
	}

	return nil
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

// Validate validates the configuration and applies corrections
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config cannot be nil")
	}

	// Clamp int64 values
	clampInt64 := func(value *int64, min, max int64) {
		if *value <= 0 {
			*value = min
		} else if *value > max {
			*value = max
		}
	}

	// Clamp int values
	clampInt := func(value *int, min, max int) {
		if *value <= 0 {
			*value = min
		} else if *value > max {
			*value = max
		}
	}

	clampInt64(&c.MaxJSONSize, 1024*1024, 100*1024*1024)
	clampInt(&c.MaxPathDepth, 10, 200)
	clampInt(&c.MaxNestingDepthSecurity, 10, 200)
	clampInt(&c.MaxConcurrency, 1, 200)
	clampInt(&c.ParallelThreshold, 1, 50)

	if c.MaxCacheSize < 0 {
		c.MaxCacheSize = 0
		c.EnableCache = false
	} else if c.MaxCacheSize > 2000 {
		c.MaxCacheSize = 2000
	}

	if c.CacheTTL <= 0 {
		c.CacheTTL = DefaultCacheTTL
	}

	// Validate new encoding fields
	if c.MaxDepth < 0 || c.MaxDepth > 1000 {
		c.MaxDepth = 100
	}
	if c.FloatPrecision < -1 || c.FloatPrecision > 15 {
		c.FloatPrecision = -1
	}

	return nil
}

// ConfigInterface implementation methods
func (c *Config) IsCacheEnabled() bool         { return c.EnableCache }
func (c *Config) GetMaxCacheSize() int         { return c.MaxCacheSize }
func (c *Config) GetCacheTTL() time.Duration   { return c.CacheTTL }
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

// Additional accessor methods for new Config fields
func (c *Config) ShouldCacheResults() bool     { return c.CacheResults }
func (c *Config) ShouldContinueOnError() bool  { return c.ContinueOnError }
func (c *Config) ShouldSkipValidation() bool   { return c.SkipValidation }
func (c *Config) GetEncodingMaxDepth() int     { return c.MaxDepth }
func (c *Config) ShouldEscapeHTML() bool       { return c.EscapeHTML }
func (c *Config) ShouldPrettyPrint() bool      { return c.Pretty }
func (c *Config) GetIndent() string            { return c.Indent }
func (c *Config) GetPrefix() string            { return c.Prefix }
func (c *Config) ShouldSortKeys() bool         { return c.SortKeys }
func (c *Config) ShouldValidateUTF8() bool     { return c.ValidateUTF8 }
func (c *Config) ShouldIncludeNulls() bool     { return c.IncludeNulls }
func (c *Config) GetFloatPrecision() int       { return c.FloatPrecision }
func (c *Config) ShouldFullSecurityScan() bool { return c.FullSecurityScan }

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
