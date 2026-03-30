package json

import (
	"errors"
	"reflect"
	"time"

	"github.com/cybergodev/json/internal"
)

// Configuration constants with optimized defaults for production workloads.
const (
	// Buffer and Pool Sizes - Optimized for production workloads (internal)
	defaultBufferSize = 1024
	maxPoolBufferSize = 32768 // 32KB max for better buffer reuse
	minPoolBufferSize = 256   // 256B min for efficiency

	// Cache Sizes - Balanced for performance and memory (internal)
	defaultCacheSize = 128

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

	// Timing and Intervals - Optimized for responsiveness (internal)
	slowOperationThreshold = 100 * time.Millisecond

	// Retry and Timeout - Production-ready settings (internal)
	defaultOperationTimeout = 30 * time.Second

	// Processor lifecycle timeouts (internal)
	closeOperationTimeout = 5 * time.Second // Timeout waiting for active operations during Close()
	semaphoreDrainTimeout = 1 * time.Second // Timeout for draining concurrency semaphore

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

	// Validation constants (internal)
	validationBOMPrefix = "\uFEFF" // UTF-8 BOM prefix to detect and remove
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

// DefaultConfig returns the default configuration.
// Creates a new instance each time to allow modifications without affecting other callers.
func DefaultConfig() Config {
	return Config{
		// Cache Settings
		MaxCacheSize: defaultCacheSize,
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
// Returns a pointer to avoid unnecessary copying of the large Config struct.
//
// NOTE: Interface fields (CustomEncoder, CustomPathParser, Context) are shallow-copied
// as they typically contain stateless or singleton implementations.
// CustomTypeEncoders, CustomValidators, AdditionalDangerousPatterns, and Hooks are
// deep-copied as they may be modified independently.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	clone := *c

	// Deep copy CustomEscapes map
	if c.CustomEscapes != nil {
		clone.CustomEscapes = make(map[rune]string, len(c.CustomEscapes))
		for k, v := range c.CustomEscapes {
			clone.CustomEscapes[k] = v
		}
	}

	// Deep copy CustomTypeEncoders map
	if c.CustomTypeEncoders != nil {
		clone.CustomTypeEncoders = make(map[reflect.Type]TypeEncoder, len(c.CustomTypeEncoders))
		for k, v := range c.CustomTypeEncoders {
			clone.CustomTypeEncoders[k] = v
		}
	}

	// Deep copy CustomValidators slice
	if c.CustomValidators != nil {
		clone.CustomValidators = make([]Validator, len(c.CustomValidators))
		copy(clone.CustomValidators, c.CustomValidators)
	}

	// Deep copy AdditionalDangerousPatterns slice
	if c.AdditionalDangerousPatterns != nil {
		clone.AdditionalDangerousPatterns = make([]DangerousPattern, len(c.AdditionalDangerousPatterns))
		copy(clone.AdditionalDangerousPatterns, c.AdditionalDangerousPatterns)
	}

	// Deep copy Hooks slice
	if c.Hooks != nil {
		clone.Hooks = make([]Hook, len(c.Hooks))
		copy(clone.Hooks, c.Hooks)
	}

	return &clone
}

// Validate validates the configuration and applies corrections.
// This is the single source of truth for config validation.
// DRY FIX: Delegates to ValidateWithWarnings to avoid code duplication
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config cannot be nil")
	}

	// Delegate to ValidateWithWarnings, ignoring warnings for silent validation
	// This ensures both functions use the same validation logic
	_ = c.ValidateWithWarnings()
	return nil
}

// ConfigWarning represents a configuration modification made during validation.
type ConfigWarning struct {
	Field    string // The field that was modified
	OldValue any    // The original value (may be nil for invalid values)
	NewValue any    // The corrected value
	Reason   string // Why the modification was made
}

// ValidateWithWarnings validates the configuration and returns warnings for any
// modifications made. This is useful for debugging configuration issues or
// informing users about automatic adjustments.
//
// Example:
//
//	cfg := json.DefaultConfig()
//	cfg.MaxJSONSize = -1 // Invalid value
//	warnings := cfg.ValidateWithWarnings()
//	for _, w := range warnings {
//	    fmt.Printf("%s: %s\n", w.Field, w.Reason)
//	}
func (c *Config) ValidateWithWarnings() []ConfigWarning {
	if c == nil {
		return []ConfigWarning{{Field: "Config", Reason: "config cannot be nil"}}
	}

	var warnings []ConfigWarning

	// Helper to record clamped int64 values
	checkInt64Clamp := func(ptr *int64, min, max int64, fieldName string) {
		original := *ptr
		if original <= 0 {
			*ptr = min
			warnings = append(warnings, ConfigWarning{
				Field:    fieldName,
				OldValue: original,
				NewValue: min,
				Reason:   "value was invalid, set to minimum",
			})
		} else if original > max {
			*ptr = max
			warnings = append(warnings, ConfigWarning{
				Field:    fieldName,
				OldValue: original,
				NewValue: max,
				Reason:   "value exceeded maximum",
			})
		}
	}

	// Helper to record clamped int values
	checkIntClamp := func(ptr *int, min, max int, fieldName string) {
		original := *ptr
		if original <= 0 {
			*ptr = min
			warnings = append(warnings, ConfigWarning{
				Field:    fieldName,
				OldValue: original,
				NewValue: min,
				Reason:   "value was invalid, set to minimum",
			})
		} else if original > max {
			*ptr = max
			warnings = append(warnings, ConfigWarning{
				Field:    fieldName,
				OldValue: original,
				NewValue: max,
				Reason:   "value exceeded maximum",
			})
		}
	}

	// Size and depth limits
	checkInt64Clamp(&c.MaxJSONSize, 1024*1024, 100*1024*1024, "MaxJSONSize")
	checkIntClamp(&c.MaxPathDepth, 10, 200, "MaxPathDepth")
	checkIntClamp(&c.MaxNestingDepthSecurity, 10, 200, "MaxNestingDepthSecurity")
	checkIntClamp(&c.MaxConcurrency, 1, 200, "MaxConcurrency")
	checkIntClamp(&c.ParallelThreshold, 1, 50, "ParallelThreshold")

	// Security limits
	checkIntClamp(&c.MaxObjectKeys, 100, 100000, "MaxObjectKeys")
	checkIntClamp(&c.MaxArrayElements, 100, 100000, "MaxArrayElements")
	checkInt64Clamp(&c.MaxSecurityValidationSize, 1024*1024, 100*1024*1024, "MaxSecurityValidationSize")

	// Cache settings
	if c.MaxCacheSize < 0 {
		warnings = append(warnings, ConfigWarning{
			Field:    "MaxCacheSize",
			OldValue: c.MaxCacheSize,
			NewValue: 0,
			Reason:   "negative cache size is invalid, disabled caching",
		})
		c.MaxCacheSize = 0
		c.EnableCache = false
	} else if c.MaxCacheSize > 2000 {
		warnings = append(warnings, ConfigWarning{
			Field:    "MaxCacheSize",
			OldValue: c.MaxCacheSize,
			NewValue: 2000,
			Reason:   "cache size exceeded maximum",
		})
		c.MaxCacheSize = 2000
	}

	if c.CacheTTL <= 0 {
		warnings = append(warnings, ConfigWarning{
			Field:    "CacheTTL",
			OldValue: c.CacheTTL,
			NewValue: DefaultCacheTTL,
			Reason:   "invalid TTL, set to default",
		})
		c.CacheTTL = DefaultCacheTTL
	}

	// Encoding options
	if c.MaxDepth < 0 || c.MaxDepth > 1000 {
		warnings = append(warnings, ConfigWarning{
			Field:    "MaxDepth",
			OldValue: c.MaxDepth,
			NewValue: 100,
			Reason:   "value out of valid range [0, 1000]",
		})
		c.MaxDepth = 100
	}
	if c.FloatPrecision < -1 || c.FloatPrecision > 15 {
		warnings = append(warnings, ConfigWarning{
			Field:    "FloatPrecision",
			OldValue: c.FloatPrecision,
			NewValue: -1,
			Reason:   "value out of valid range [-1, 15]",
		})
		c.FloatPrecision = -1
	}

	// Batch size limits
	checkIntClamp(&c.MaxBatchSize, 10, 10000, "MaxBatchSize")

	return warnings
}

// Config accessor methods.
// These methods implement interfaces (CacheConfig, EncoderConfig) and provide
// consistent API for testing and interface-based programming.

// Required by CacheConfig interface (internal/cache.go) - do not remove.
func (c *Config) IsCacheEnabled() bool       { return c.EnableCache }
func (c *Config) GetMaxCacheSize() int       { return c.MaxCacheSize }
func (c *Config) GetCacheTTL() time.Duration { return c.CacheTTL }

// Convenience accessor methods for testing and interface-based usage.
// Rationale: These methods enable mock-based testing and potential future
// interface abstraction. In application code, direct field access is preferred:
//
//	cfg.MaxJSONSize instead of cfg.GetMaxJSONSize()
//	cfg.StrictMode instead of cfg.IsStrictMode()
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

// Required by EncoderConfig interface (interfaces.go) for custom encoders.
// These methods provide read-only access to encoding configuration.
func (c *Config) IsHTMLEscapeEnabled() bool      { return c.EscapeHTML }
func (c *Config) IsPrettyEnabled() bool          { return c.Pretty }
func (c *Config) GetIndent() string              { return c.Indent }
func (c *Config) GetPrefix() string              { return c.Prefix }
func (c *Config) IsSortKeysEnabled() bool        { return c.SortKeys }
func (c *Config) GetFloatPrecision() int         { return c.FloatPrecision }
func (c *Config) IsTruncateFloatEnabled() bool   { return c.FloatTruncate }
func (c *Config) GetMaxDepth() int               { return c.MaxDepth }
func (c *Config) ShouldIncludeNulls() bool       { return c.IncludeNulls }
func (c *Config) ShouldValidateUTF8() bool       { return c.ValidateUTF8 }
func (c *Config) IsDisallowUnknownEnabled() bool { return c.DisallowUnknown }
func (c *Config) ShouldEscapeUnicode() bool      { return c.EscapeUnicode }
func (c *Config) ShouldEscapeSlash() bool        { return c.EscapeSlash }
func (c *Config) ShouldEscapeNewlines() bool     { return c.EscapeNewlines }
func (c *Config) ShouldEscapeTabs() bool         { return c.EscapeTabs }

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
