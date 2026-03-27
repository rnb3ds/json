package json

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/cybergodev/json/internal"
)

// Config holds all configuration for the JSON processor.
// Start with DefaultConfig() and modify as needed.
//
// Example:
//
//	cfg := json.DefaultConfig()
//	cfg.CreatePaths = true
//	cfg.Pretty = true
//	result, err := json.Set(data, "path", value, cfg)
type Config struct {
	// ===== Cache Settings =====
	MaxCacheSize int           `json:"max_cache_size"`
	CacheTTL     time.Duration `json:"cache_ttl"`
	EnableCache  bool          `json:"enable_cache"`
	CacheResults bool          `json:"cache_results"` // Per-operation caching

	// ===== Size Limits =====
	MaxJSONSize  int64 `json:"max_json_size"`
	MaxPathDepth int   `json:"max_path_depth"`
	MaxBatchSize int   `json:"max_batch_size"`

	// ===== Security Limits =====
	MaxNestingDepthSecurity   int   `json:"max_nesting_depth"`
	MaxSecurityValidationSize int64 `json:"max_security_validation_size"`
	MaxObjectKeys             int   `json:"max_object_keys"`
	MaxArrayElements          int   `json:"max_array_elements"`
	// FullSecurityScan enables full (non-sampling) security validation for all JSON input.
	//
	// When false (default): Large JSON (>4KB) uses optimized sampling with:
	//   - 16KB beginning section scan
	//   - 8KB end section scan
	//   - 15-30 distributed middle samples with 512-byte overlap
	//   - Critical patterns (__proto__, constructor, prototype) always fully scanned
	//   - Suspicious character density triggers automatic full scan
	//
	// When true: All JSON is fully scanned regardless of size.
	//
	// SECURITY RECOMMENDATION: Enable FullSecurityScan when:
	//   - Processing untrusted input from external sources
	//   - Handling sensitive data (authentication, financial, personal)
	//   - Building public-facing APIs or web services
	//   - Compliance requirements mandate full content inspection
	//
	// PERFORMANCE NOTE: Full scanning adds ~10-30% overhead for JSON >100KB.
	// For trusted internal services with large JSON payloads, sampling mode is acceptable.
	FullSecurityScan bool `json:"full_security_scan"`

	// ===== Concurrency =====
	MaxConcurrency    int `json:"max_concurrency"`
	ParallelThreshold int `json:"parallel_threshold"`

	// ===== Processing Options =====
	EnableValidation bool `json:"enable_validation"`
	StrictMode       bool `json:"strict_mode"`
	CreatePaths      bool `json:"create_paths"`
	CleanupNulls     bool `json:"cleanup_nulls"`
	CompactArrays    bool `json:"compact_arrays"`
	ContinueOnError  bool `json:"continue_on_error"` // Continue on batch errors

	// ===== Input/Output Options =====
	AllowComments    bool `json:"allow_comments"`
	PreserveNumbers  bool `json:"preserve_numbers"`
	ValidateInput    bool `json:"validate_input"`
	ValidateFilePath bool `json:"validate_file_path"`
	SkipValidation   bool `json:"skip_validation"` // Skip validation for trusted input

	// ===== Encoding Options =====
	Pretty          bool            `json:"pretty"`
	Indent          string          `json:"indent"`
	Prefix          string          `json:"prefix"`
	EscapeHTML      bool            `json:"escape_html"`
	SortKeys        bool            `json:"sort_keys"`
	ValidateUTF8    bool            `json:"validate_utf8"`
	MaxDepth        int             `json:"max_depth"`
	DisallowUnknown bool            `json:"disallow_unknown"`
	FloatPrecision  int             `json:"float_precision"`
	FloatTruncate   bool            `json:"float_truncate"`
	DisableEscaping bool            `json:"disable_escaping"`
	EscapeUnicode   bool            `json:"escape_unicode"`
	EscapeSlash     bool            `json:"escape_slash"`
	EscapeNewlines  bool            `json:"escape_newlines"`
	EscapeTabs      bool            `json:"escape_tabs"`
	IncludeNulls    bool            `json:"include_nulls"`
	CustomEscapes   map[rune]string `json:"custom_escapes,omitempty"`

	// ===== Observability =====
	EnableMetrics     bool `json:"enable_metrics"`
	EnableHealthCheck bool `json:"enable_health_check"`

	// ===== Context =====
	Context context.Context `json:"-"` // Operation context
}

// GetSecurityLimits returns a summary of current security limits
func (c *Config) GetSecurityLimits() map[string]any {
	return map[string]any{
		"max_nesting_depth":            c.MaxNestingDepthSecurity,
		"max_security_validation_size": c.MaxSecurityValidationSize,
		"max_object_keys":              c.MaxObjectKeys,
		"max_array_elements":           c.MaxArrayElements,
		"max_json_size":                c.MaxJSONSize,
		"max_path_depth":               c.MaxPathDepth,
	}
}

// ParsedJSON represents a pre-parsed JSON document that can be reused for multiple operations.
// This is a performance optimization for scenarios where the same JSON is queried multiple times.
// OPTIMIZED: Pre-parsing avoids repeated JSON parsing overhead for repeated queries.
type ParsedJSON struct {
	data      any
	hash      uint64
	jsonLen   int
	processor *Processor
}

// Data returns the underlying parsed data
func (p *ParsedJSON) Data() any {
	return p.data
}

// Stats provides processor performance statistics
type Stats struct {
	CacheSize        int64         `json:"cache_size"`
	CacheMemory      int64         `json:"cache_memory"`
	MaxCacheSize     int           `json:"max_cache_size"`
	HitCount         int64         `json:"hit_count"`
	MissCount        int64         `json:"miss_count"`
	HitRatio         float64       `json:"hit_ratio"`
	CacheTTL         time.Duration `json:"cache_ttl"`
	CacheEnabled     bool          `json:"cache_enabled"`
	IsClosed         bool          `json:"is_closed"`
	MemoryEfficiency float64       `json:"memory_efficiency"`
	OperationCount   int64         `json:"operation_count"`
	ErrorCount       int64         `json:"error_count"`
}

// HealthStatus represents the health status of the processor
type HealthStatus struct {
	Timestamp time.Time              `json:"timestamp"`
	Healthy   bool                   `json:"healthy"`
	Checks    map[string]CheckResult `json:"checks"`
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	Healthy bool   `json:"healthy"`
	Message string `json:"message"`
}

// WarmupResult represents the result of a cache warmup operation
type WarmupResult struct {
	TotalPaths  int      `json:"total_paths"`
	Successful  int      `json:"successful"`
	Failed      int      `json:"failed"`
	SuccessRate float64  `json:"success_rate"`
	FailedPaths []string `json:"failed_paths,omitempty"`
}

// BatchOperation represents a single operation in a batch
type BatchOperation struct {
	Type    string `json:"type"`
	JSONStr string `json:"json_str"`
	Path    string `json:"path"`
	Value   any    `json:"value"`
	ID      string `json:"id"`
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	ID     string `json:"id"`
	Result any    `json:"result"`
	Error  error  `json:"error"`
}

// Marshaler is the interface implemented by types that
// can marshal themselves into valid JSON.
type Marshaler interface {
	MarshalJSON() ([]byte, error)
}

// Unmarshaler is the interface implemented by types
// that can unmarshal a JSON description of themselves.
// The input can be assumed to be a valid encoding of
// a JSON value. UnmarshalJSON must copy the JSON data
// if it wishes to retain the data after returning.
//
// By convention, to approximate the behavior of Unmarshal itself,
// Unmarshalers implement UnmarshalJSON([]byte("null")) as a no-op.
type Unmarshaler interface {
	UnmarshalJSON([]byte) error
}

// TextMarshaler is the interface implemented by an object that can
// marshal itself into a textual form.
//
// MarshalText encodes the receiver into UTF-8-encoded text and returns the result.
type TextMarshaler interface {
	MarshalText() (text []byte, err error)
}

// TextUnmarshaler is the interface implemented by an object that can
// unmarshal a textual representation of itself.
//
// UnmarshalText must be able to decode the form generated by MarshalText.
// UnmarshalText must copy the text if it wishes to retain the text
// after returning.
type TextUnmarshaler interface {
	UnmarshalText(text []byte) error
}

// InvalidUnmarshalError describes an invalid argument passed to Unmarshal.
// (The argument to Unmarshal must be a non-nil pointer.)
type InvalidUnmarshalError struct {
	Type reflect.Type
}

func (e *InvalidUnmarshalError) Error() string {
	if e.Type == nil {
		return "json: Unmarshal(nil)"
	}

	if e.Type.Kind() != reflect.Ptr {
		return "json: Unmarshal(non-pointer " + e.Type.String() + ")"
	}
	return "json: Unmarshal(nil " + e.Type.String() + ")"
}

// SyntaxError is a description of a JSON syntax error.
// Unmarshal will return a SyntaxError if the JSON can't be parsed.
type SyntaxError struct {
	msg    string // description of error
	Offset int64  // error occurred after reading Offset bytes
}

func (e *SyntaxError) Error() string { return e.msg }

// UnmarshalTypeError describes a JSON value that was
// not appropriate for a value of a specific Go type.
type UnmarshalTypeError struct {
	Value  string       // description of JSON value - "bool", "array", "number -5"
	Type   reflect.Type // type of Go value it could not be assigned to
	Offset int64        // error occurred after reading Offset bytes
	Struct string       // name of the root type containing the field
	Field  string       // the full path from root node to the value
	Err    error        // may be nil
}

func (e *UnmarshalTypeError) Error() string {
	if e.Struct != "" || e.Field != "" {
		return "json: cannot unmarshal " + e.Value + " into Go struct field " + e.Struct + "." + e.Field + " of type " + e.Type.String()
	}
	return "json: cannot unmarshal " + e.Value + " into Go value of type " + e.Type.String()
}

func (e *UnmarshalTypeError) Unwrap() error {
	return e.Err
}

// UnsupportedTypeError is returned by Marshal when attempting
// to encode an unsupported value type.
type UnsupportedTypeError struct {
	Type reflect.Type
}

func (e *UnsupportedTypeError) Error() string {
	return "json: unsupported type: " + e.Type.String()
}

// UnsupportedValueError is returned by Marshal when attempting
// to encode an unsupported value.
type UnsupportedValueError struct {
	Value reflect.Value
	Str   string
}

func (e *UnsupportedValueError) Error() string {
	return "json: unsupported value: " + e.Str
}

// MarshalerError represents an error from calling a MarshalJSON or MarshalText method.
type MarshalerError struct {
	Type       reflect.Type
	Err        error
	sourceFunc string
}

func (e *MarshalerError) Error() string {
	srcFunc := e.sourceFunc
	if srcFunc == "" {
		srcFunc = "MarshalJSON"
	}
	return "json: error calling " + srcFunc + " for type " + e.Type.String() + ": " + e.Err.Error()
}

func (e *MarshalerError) Unwrap() error { return e.Err }

// deletedMarker is a special sentinel value used to mark array elements
// for deletion. It is compared by pointer identity (using ==).
//
// SECURITY: This is an unexported struct pointer to prevent external modification.
// The zero-size struct{}{} is used because we only need unique pointer identity.
//
// Naming Convention: Uses camelCase (unexported) to prevent external packages from
// accessing this internal implementation detail. The "Marker" suffix indicates this
// is a sentinel value for marking items, not a data container.
//
// IMPORTANT: Do not reassign this variable. Use IsDeletedMarker() for comparisons.
var deletedMarker = &struct{}{} // deleted marker - empty struct for pointer identity

// IsDeletedMarker checks if a value is the deleted marker sentinel.
// This is the recommended way to check for deleted markers instead of direct comparison.
func IsDeletedMarker(v any) bool {
	return v == deletedMarker
}

// propertyAccessResult represents the result of a property access operation
type propertyAccessResult struct {
	value  any
	exists bool
}

// ============================================================================
// INTERNAL ERROR TYPES
// These are unexported types used internally for control flow during operations.
// They should not be used directly by external code.
// ============================================================================

// rootDataTypeConversionError signals that root data type conversion is needed
type rootDataTypeConversionError struct {
	requiredType string
	requiredSize int
	currentType  string
}

func (e *rootDataTypeConversionError) Error() string {
	return fmt.Sprintf("root data type conversion required: from %s to %s (size: %d)",
		e.currentType, e.requiredType, e.requiredSize)
}

// arrayExtensionError signals that array extension is needed
type arrayExtensionError struct {
	currentLength  int
	requiredLength int
	targetIndex    int
	value          any
	message        string
}

func (e *arrayExtensionError) Error() string {
	if e.message != "" {
		return e.message
	}
	return fmt.Sprintf("array extension required: current length %d, required length %d for index %d",
		e.currentLength, e.requiredLength, e.targetIndex)
}

// pathSegment represents a parsed path segment with its type and value.
// This is an internal type alias - users should not need to access path segments directly.
type pathSegment = internal.PathSegment

// pathInfo contains parsed path information.
// This is an internal type used for path parsing.
type pathInfo struct {
	segments     []pathSegment
	isPointer    bool
	originalPath string
}

// PathSegment is an alias for internal path segment type.
// Deprecated: This is an internal implementation detail. Do not use.
// Will be removed in v2.0.0.
type PathSegment = internal.PathSegment

// PathInfo contains parsed path information.
// Deprecated: This is an internal implementation detail. Do not use.
// Will be removed in v2.0.0.
type PathInfo = pathInfo

// resourceMonitor provides resource monitoring and leak detection
type resourceMonitor struct {
	allocatedBytes    int64
	freedBytes        int64
	peakMemoryUsage   int64
	poolHits          int64
	poolMisses        int64
	poolEvictions     int64
	maxGoroutines     int64
	currentGoroutines int64
	lastLeakCheck     int64
	leakCheckInterval int64
	avgResponseTime   int64
	totalOperations   int64
}

// resourceStats represents resource usage statistics
type resourceStats struct {
	AllocatedBytes    int64         `json:"allocated_bytes"`
	FreedBytes        int64         `json:"freed_bytes"`
	PeakMemoryUsage   int64         `json:"peak_memory_usage"`
	PoolHits          int64         `json:"pool_hits"`
	PoolMisses        int64         `json:"pool_misses"`
	PoolEvictions     int64         `json:"pool_evictions"`
	MaxGoroutines     int64         `json:"max_goroutines"`
	CurrentGoroutines int64         `json:"current_goroutines"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	TotalOperations   int64         `json:"total_operations"`
}

// newResourceMonitor creates a new resource monitor
func newResourceMonitor() *resourceMonitor {
	return &resourceMonitor{
		leakCheckInterval: 300, // 5 minutes
		lastLeakCheck:     time.Now().Unix(),
	}
}

// RecordAllocation records an allocation of the specified size.
// Note: the peak memory calculation uses a snapshot of allocated/freed counters,
// so it is approximate under high concurrency. This is acceptable for monitoring.
func (rm *resourceMonitor) RecordAllocation(bytes int64) {
	allocated := atomic.AddInt64(&rm.allocatedBytes, bytes)
	freed := atomic.LoadInt64(&rm.freedBytes)
	current := allocated - freed
	for {
		peak := atomic.LoadInt64(&rm.peakMemoryUsage)
		if current <= peak || atomic.CompareAndSwapInt64(&rm.peakMemoryUsage, peak, current) {
			break
		}
	}
}

// RecordDeallocation records a deallocation of the specified size
func (rm *resourceMonitor) RecordDeallocation(bytes int64) {
	atomic.AddInt64(&rm.freedBytes, bytes)
}

// RecordPoolHit records a pool cache hit
func (rm *resourceMonitor) RecordPoolHit() {
	atomic.AddInt64(&rm.poolHits, 1)
}

// RecordPoolMiss records a pool cache miss
func (rm *resourceMonitor) RecordPoolMiss() {
	atomic.AddInt64(&rm.poolMisses, 1)
}

// RecordPoolEviction records a pool eviction
func (rm *resourceMonitor) RecordPoolEviction() {
	atomic.AddInt64(&rm.poolEvictions, 1)
}

// RecordOperation records an operation with its duration
func (rm *resourceMonitor) RecordOperation(duration time.Duration) {
	atomic.AddInt64(&rm.totalOperations, 1)

	newTime := duration.Nanoseconds()
	for {
		oldAvg := atomic.LoadInt64(&rm.avgResponseTime)
		newAvg := oldAvg + (newTime-oldAvg)/10
		if atomic.CompareAndSwapInt64(&rm.avgResponseTime, oldAvg, newAvg) {
			break
		}
	}
}

// CheckForLeaks checks for potential resource leaks
func (rm *resourceMonitor) CheckForLeaks() []string {
	for {
		now := time.Now().Unix()
		lastCheck := atomic.LoadInt64(&rm.lastLeakCheck)

		if now-lastCheck < rm.leakCheckInterval {
			return nil
		}

		if atomic.CompareAndSwapInt64(&rm.lastLeakCheck, lastCheck, now) {
			break
		}
	}

	var issues []string

	allocated := atomic.LoadInt64(&rm.allocatedBytes)
	freed := atomic.LoadInt64(&rm.freedBytes)
	netMemory := allocated - freed

	if netMemory > 100*1024*1024 {
		issues = append(issues, "High memory usage detected")
	}

	currentGoroutines := int64(runtime.NumGoroutine())
	atomic.StoreInt64(&rm.currentGoroutines, currentGoroutines)

	maxGoroutines := atomic.LoadInt64(&rm.maxGoroutines)
	if currentGoroutines > maxGoroutines {
		atomic.StoreInt64(&rm.maxGoroutines, currentGoroutines)
	}

	if currentGoroutines > 1000 {
		issues = append(issues, "High goroutine count detected")
	}

	hits := atomic.LoadInt64(&rm.poolHits)
	misses := atomic.LoadInt64(&rm.poolMisses)

	if hits+misses > 1000 && hits < misses {
		issues = append(issues, "Poor pool cache efficiency")
	}

	return issues
}

// getStats returns current resource statistics
func (rm *resourceMonitor) getStats() resourceStats {
	return resourceStats{
		AllocatedBytes:    atomic.LoadInt64(&rm.allocatedBytes),
		FreedBytes:        atomic.LoadInt64(&rm.freedBytes),
		PeakMemoryUsage:   atomic.LoadInt64(&rm.peakMemoryUsage),
		PoolHits:          atomic.LoadInt64(&rm.poolHits),
		PoolMisses:        atomic.LoadInt64(&rm.poolMisses),
		PoolEvictions:     atomic.LoadInt64(&rm.poolEvictions),
		MaxGoroutines:     atomic.LoadInt64(&rm.maxGoroutines),
		CurrentGoroutines: atomic.LoadInt64(&rm.currentGoroutines),
		AvgResponseTime:   time.Duration(atomic.LoadInt64(&rm.avgResponseTime)),
		TotalOperations:   atomic.LoadInt64(&rm.totalOperations),
	}
}

// Reset resets all resource statistics
func (rm *resourceMonitor) Reset() {
	atomic.StoreInt64(&rm.allocatedBytes, 0)
	atomic.StoreInt64(&rm.freedBytes, 0)
	atomic.StoreInt64(&rm.peakMemoryUsage, 0)
	atomic.StoreInt64(&rm.poolHits, 0)
	atomic.StoreInt64(&rm.poolMisses, 0)
	atomic.StoreInt64(&rm.poolEvictions, 0)
	atomic.StoreInt64(&rm.maxGoroutines, 0)
	atomic.StoreInt64(&rm.currentGoroutines, 0)
	atomic.StoreInt64(&rm.avgResponseTime, 0)
	atomic.StoreInt64(&rm.totalOperations, 0)
	atomic.StoreInt64(&rm.lastLeakCheck, time.Now().Unix())
}

// GetDeallocationRatio returns the ratio of freed bytes to allocated bytes as a percentage.
// Values > 100% indicate the same memory was reused multiple times via pooling.
func (rm *resourceMonitor) GetDeallocationRatio() float64 {
	allocated := atomic.LoadInt64(&rm.allocatedBytes)
	freed := atomic.LoadInt64(&rm.freedBytes)

	if allocated == 0 {
		return 100.0
	}

	return float64(freed) / float64(allocated) * 100.0
}

// GetMemoryEfficiency returns the deallocation ratio.
//
// Deprecated: Use GetDeallocationRatio for clarity on what this metric represents.
// Migration: rm.GetMemoryEfficiency() -> rm.GetDeallocationRatio()
// Will be removed in v2.0.0.
func (rm *resourceMonitor) GetMemoryEfficiency() float64 {
	return rm.GetDeallocationRatio()
}

// GetPoolEfficiency returns the pool efficiency percentage
// This is the hit ratio of the pool cache
func (rm *resourceMonitor) GetPoolEfficiency() float64 {
	return rm.GetPoolHitRatio()
}

// GetPoolHitRatio returns the pool cache hit ratio as a percentage.
// This is an alias for GetPoolEfficiency with consistent naming.
func (rm *resourceMonitor) GetPoolHitRatio() float64 {
	hits := atomic.LoadInt64(&rm.poolHits)
	misses := atomic.LoadInt64(&rm.poolMisses)
	total := hits + misses

	if total == 0 {
		return 100.0
	}

	return float64(hits) / float64(total) * 100.0
}

// Schema represents a JSON schema for validation
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Required             []string           `json:"required,omitempty"`
	MinLength            int                `json:"minLength,omitempty"`
	MaxLength            int                `json:"maxLength,omitempty"`
	Minimum              float64            `json:"minimum,omitempty"`
	Maximum              float64            `json:"maximum,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
	Format               string             `json:"format,omitempty"`
	AdditionalProperties bool               `json:"additionalProperties,omitempty"`
	MinItems             int                `json:"minItems,omitempty"`
	MaxItems             int                `json:"maxItems,omitempty"`
	UniqueItems          bool               `json:"uniqueItems,omitempty"`
	Enum                 []any              `json:"enum,omitempty"`
	Const                any                `json:"const,omitempty"`
	MultipleOf           float64            `json:"multipleOf,omitempty"`
	ExclusiveMinimum     bool               `json:"exclusiveMinimum,omitempty"`
	ExclusiveMaximum     bool               `json:"exclusiveMaximum,omitempty"`
	Title                string             `json:"title,omitempty"`
	Description          string             `json:"description,omitempty"`
	Default              any                `json:"default,omitempty"`
	Examples             []any              `json:"examples,omitempty"`

	// Internal flags
	hasMinLength bool
	hasMaxLength bool
	hasMinimum   bool
	hasMaximum   bool
	hasMinItems  bool
	hasMaxItems  bool
}

// ValidationError represents a schema validation error
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (ve *ValidationError) Error() string {
	if ve.Path != "" {
		return fmt.Sprintf("validation error at path '%s': %s", ve.Path, ve.Message)
	}
	return fmt.Sprintf("validation error: %s", ve.Message)
}

// TypeSafeResult represents a type-safe operation result
type TypeSafeResult[T any] struct {
	Value  T
	Exists bool
	Error  error
}

// Ok returns true if the result is valid (no error and exists)
func (r TypeSafeResult[T]) Ok() bool {
	return r.Error == nil && r.Exists
}

// Unwrap returns the value or zero value if there's an error
// For panic behavior, use UnwrapOrPanic instead
func (r TypeSafeResult[T]) Unwrap() T {
	if r.Error != nil {
		var zero T
		return zero
	}
	return r.Value
}

// UnwrapOrPanic returns the value or panics if there's an error
// Use this only when you're certain the operation succeeded
func (r TypeSafeResult[T]) UnwrapOrPanic() T {
	if r.Error != nil {
		panic(fmt.Sprintf("unwrap called on result with error: %v", r.Error))
	}
	return r.Value
}

// UnwrapOr returns the value or the provided default if there's an error or value doesn't exist
func (r TypeSafeResult[T]) UnwrapOr(defaultValue T) T {
	if r.Error != nil || !r.Exists {
		return defaultValue
	}
	return r.Value
}

// TypeSafeAccessResult represents the result of a type-safe access operation
type TypeSafeAccessResult struct {
	Value  any
	Exists bool
	Type   string
}

// AsString safely converts the result to string.
// Returns ErrTypeMismatch if the value is not a string type.
// Use AsStringConverted() for explicit type conversion with formatting.
func (r TypeSafeAccessResult) AsString() (string, error) {
	if !r.Exists {
		return "", ErrPathNotFound
	}
	if str, ok := r.Value.(string); ok {
		return str, nil
	}
	// SECURITY: Return error for non-string types instead of silent conversion
	return "", fmt.Errorf("cannot convert %T to string: type mismatch", r.Value)
}

// AsStringConverted converts the result to string using fmt.Sprintf formatting.
// Use this when you explicitly want string representation of any type.
// For strict type checking, use AsString() instead.
func (r TypeSafeAccessResult) AsStringConverted() (string, error) {
	if !r.Exists {
		return "", ErrPathNotFound
	}
	if str, ok := r.Value.(string); ok {
		return str, nil
	}
	// Explicit conversion requested - use formatting
	return fmt.Sprintf("%v", r.Value), nil
}

// AsInt safely converts the result to int with overflow and precision checks.
// Unlike ConvertToInt, this method is stricter and does NOT convert bool to int.
// Use ConvertToInt directly if you need more permissive conversion.
func (r TypeSafeAccessResult) AsInt() (int, error) {
	if !r.Exists {
		return 0, ErrPathNotFound
	}

	// Strict type check: bool should not convert to int
	switch r.Value.(type) {
	case bool:
		return 0, fmt.Errorf("cannot convert bool to int")
	}

	result, ok := ConvertToInt(r.Value)
	if !ok {
		return 0, fmt.Errorf("cannot convert %T to int", r.Value)
	}
	return result, nil
}

// AsFloat64 safely converts the result to float64 with precision checks.
// Unlike ConvertToFloat64, this method is stricter and does NOT convert bool to float64.
// Use ConvertToFloat64 directly if you need more permissive conversion.
func (r TypeSafeAccessResult) AsFloat64() (float64, error) {
	if !r.Exists {
		return 0, ErrPathNotFound
	}

	// Strict type check: bool should not convert to float64
	switch r.Value.(type) {
	case bool:
		return 0, fmt.Errorf("cannot convert bool to float64")
	}

	result, ok := ConvertToFloat64(r.Value)
	if !ok {
		return 0, fmt.Errorf("cannot convert %T to float64", r.Value)
	}
	return result, nil
}

// AsBool safely converts the result to bool.
// Unlike ConvertToBool, this method is stricter and only accepts bool and string types.
// Use ConvertToBool directly if you need more permissive conversion (e.g., int to bool).
func (r TypeSafeAccessResult) AsBool() (bool, error) {
	if !r.Exists {
		return false, ErrPathNotFound
	}

	switch v := r.Value.(type) {
	case bool:
		return v, nil
	case string:
		result, err := strconv.ParseBool(v)
		if err != nil {
			return false, fmt.Errorf("cannot convert %q to bool: %w", v, err)
		}
		return result, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", r.Value)
	}
}

// DefaultSchema returns a default schema configuration
func DefaultSchema() *Schema {
	return &Schema{
		Type:                 "",
		Properties:           make(map[string]*Schema),
		Items:                nil,
		Required:             []string{},
		MinLength:            0,
		MaxLength:            0,
		Minimum:              0,
		Maximum:              0,
		Pattern:              "",
		Format:               "",
		AdditionalProperties: true,
		MinItems:             0,
		MaxItems:             0,
	}
}

// SchemaConfig provides configuration options for creating a Schema.
// This follows the Config pattern as required by the design guidelines.
type SchemaConfig struct {
	Type                 string
	Properties           map[string]*Schema
	Items                *Schema
	Required             []string
	MinLength            *int
	MaxLength            *int
	Minimum              *float64
	Maximum              *float64
	Pattern              string
	Format               string
	AdditionalProperties *bool
	MinItems             *int
	MaxItems             *int
	UniqueItems          bool
	Enum                 []any
	Const                any
	MultipleOf           *float64
	ExclusiveMinimum     *bool
	ExclusiveMaximum     *bool
	Title                string
	Description          string
	Default              any
	Examples             []any
}

// NewSchemaWithConfig creates a new Schema with the provided configuration.
// This is the recommended way to create configured Schema instances.
func NewSchemaWithConfig(cfg SchemaConfig) *Schema {
	s := &Schema{
		Type:        cfg.Type,
		Properties:  cfg.Properties,
		Items:       cfg.Items,
		Required:    cfg.Required,
		Pattern:     cfg.Pattern,
		Format:      cfg.Format,
		UniqueItems: cfg.UniqueItems,
		Enum:        cfg.Enum,
		Const:       cfg.Const,
		Title:       cfg.Title,
		Description: cfg.Description,
		Default:     cfg.Default,
		Examples:    cfg.Examples,
	}

	if cfg.Properties == nil {
		s.Properties = make(map[string]*Schema)
	}
	if cfg.Required == nil {
		s.Required = []string{}
	}

	// Set optional fields with their has* flags
	if cfg.MinLength != nil {
		s.MinLength = *cfg.MinLength
		s.hasMinLength = true
	}
	if cfg.MaxLength != nil {
		s.MaxLength = *cfg.MaxLength
		s.hasMaxLength = true
	}
	if cfg.Minimum != nil {
		s.Minimum = *cfg.Minimum
		s.hasMinimum = true
	}
	if cfg.Maximum != nil {
		s.Maximum = *cfg.Maximum
		s.hasMaximum = true
	}
	if cfg.AdditionalProperties != nil {
		s.AdditionalProperties = *cfg.AdditionalProperties
	} else {
		s.AdditionalProperties = true
	}
	if cfg.MinItems != nil {
		s.MinItems = *cfg.MinItems
		s.hasMinItems = true
	}
	if cfg.MaxItems != nil {
		s.MaxItems = *cfg.MaxItems
		s.hasMaxItems = true
	}
	if cfg.MultipleOf != nil {
		s.MultipleOf = *cfg.MultipleOf
	}
	if cfg.ExclusiveMinimum != nil {
		s.ExclusiveMinimum = *cfg.ExclusiveMinimum
	}
	if cfg.ExclusiveMaximum != nil {
		s.ExclusiveMaximum = *cfg.ExclusiveMaximum
	}

	return s
}

// HasMinLength returns true if MinLength constraint is explicitly set
func (s *Schema) HasMinLength() bool {
	return s.hasMinLength
}

// HasMaxLength returns true if MaxLength constraint is explicitly set
func (s *Schema) HasMaxLength() bool {
	return s.hasMaxLength
}

// HasMinimum returns true if Minimum constraint is explicitly set
func (s *Schema) HasMinimum() bool {
	return s.hasMinimum
}

// HasMaximum returns true if Maximum constraint is explicitly set
func (s *Schema) HasMaximum() bool {
	return s.hasMaximum
}

// HasMinItems returns true if MinItems constraint is explicitly set
func (s *Schema) HasMinItems() bool {
	return s.hasMinItems
}

// HasMaxItems returns true if MaxItems constraint is explicitly set
func (s *Schema) HasMaxItems() bool {
	return s.hasMaxItems
}

// ============================================================================
// MERGE CONFIGURATION
// Configuration types for JSON merge operations with union/intersection/difference modes.
// ============================================================================

// MergeMode defines the merge strategy for combining JSON objects and arrays.
// This is a type alias to internal.MergeMode to ensure consistency across the codebase.
type MergeMode = internal.MergeMode

// Merge mode constants - re-exported from internal package for public API
const (
	// MergeUnion performs union merge - combines all keys/elements (default)
	// For objects: all keys from both, conflicts resolved by override value
	// For arrays: all elements from both with deduplication
	MergeUnion = internal.MergeUnion

	// MergeIntersection performs intersection merge - only common keys/elements
	// For objects: only keys present in both, values from override
	// For arrays: only elements present in both arrays
	MergeIntersection = internal.MergeIntersection

	// MergeDifference performs difference merge - keys/elements only in base
	// For objects: keys in base but not in override
	// For arrays: elements in base but not in override
	MergeDifference = internal.MergeDifference
)
