package json

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"sync"

	"github.com/cybergodev/json/internal"
)

// Cache size limits for configProcessorCache
const (
	configProcessorCacheLimit    = 64 // Maximum cached processors
	configProcessorCacheEvictNum = 16 // Number to evict when limit reached
)

// Processor cache for config-based processor reuse
var (
	configProcessorCache   sync.Map   // map[uint64]*Processor
	configProcessorCacheMu sync.Mutex // Mutex for eviction serialization
)

// getProcessorOrFail returns the default processor or an error if unavailable.
// SAFETY: Use this for public APIs that return errors.
func getProcessorOrFail() (*Processor, error) {
	p := getDefaultProcessor()
	if p == nil {
		return nil, ErrInternalError
	}
	return p, nil
}

// hashConfig generates a cache key for Config for processor caching.
//
// ROBUSTNESS: Uses field-by-field hashing to include ALL Config fields,
// including Context (which is excluded from JSON serialization).
// This ensures accurate cache keys and prevents collisions.
//
// PERFORMANCE: For the common case of default configs, uses a fast path that
// compares against default config using reflect-lite comparison.
func hashConfig(cfg Config) uint64 {
	// Fast path: check if this is a default config (most common case)
	if isDefaultConfig(cfg) {
		return 1 // Reserved hash for default config
	}

	// Slow path: hash all fields explicitly
	// This is more reliable than JSON serialization which ignores Context
	return hashConfigFields(cfg)
}

// isDefaultConfig checks if the config matches the default configuration.
// Performs complete comparison including Context field (which JSON ignores).
// PERFORMANCE: Uses short-circuit evaluation for common mismatches first.
func isDefaultConfig(cfg Config) bool {
	// Fast checks for common non-default values
	// These are ordered by likelihood of being modified
	if cfg.Pretty ||
		cfg.StrictMode ||
		cfg.CreatePaths ||
		!cfg.EnableCache ||
		!cfg.EnableValidation ||
		cfg.Context != nil {
		return false
	}

	// Check all fields against default
	defaultCfg := DefaultConfig()
	return configFieldsEqual(cfg, defaultCfg)
}

// configFieldsEqual compares all fields of two Config structs.
// MAINTAINABILITY: Uses reflect.DeepEqual for comprehensive comparison
// that automatically handles new fields added to Config.
// PERFORMANCE: Context is handled specially since contexts aren't DeepEqual-comparable.
func configFieldsEqual(a, b Config) bool {
	// Handle Context specially since context.Context values aren't DeepEqual-comparable
	if !contextEqual(a.Context, b.Context) {
		return false
	}

	// Clear Context fields for DeepEqual comparison
	aCopy := a
	bCopy := b
	aCopy.Context = nil
	bCopy.Context = nil

	// Use DeepEqual for all other fields
	// This is maintainable and handles all field types including maps and slices
	return reflect.DeepEqual(aCopy, bCopy)
}

// contextEqual compares two contexts for equality.
// Uses standard interface comparison which compares both type and underlying value.
// For context.Context, this correctly identifies identical contexts.
func contextEqual(a, b context.Context) bool {
	return a == b
}

// hashConfigFields hashes all Config fields explicitly.
// More reliable than JSON serialization which ignores Context.
func hashConfigFields(cfg Config) uint64 {
	h := internal.FNVOffsetBasis

	// Cache settings
	h = internal.HashInt(h, cfg.MaxCacheSize)
	h = internal.HashInt64(h, int64(cfg.CacheTTL))
	h = internal.HashBool(h, cfg.EnableCache)
	h = internal.HashBool(h, cfg.CacheResults)

	// Size limits
	h = internal.HashInt64(h, cfg.MaxJSONSize)
	h = internal.HashInt(h, cfg.MaxPathDepth)
	h = internal.HashInt(h, cfg.MaxBatchSize)

	// Security limits
	h = internal.HashInt(h, cfg.MaxNestingDepthSecurity)
	h = internal.HashInt64(h, cfg.MaxSecurityValidationSize)
	h = internal.HashInt(h, cfg.MaxObjectKeys)
	h = internal.HashInt(h, cfg.MaxArrayElements)
	h = internal.HashBool(h, cfg.FullSecurityScan)

	// Concurrency
	h = internal.HashInt(h, cfg.MaxConcurrency)
	h = internal.HashInt(h, cfg.ParallelThreshold)

	// Processing options
	h = internal.HashBool(h, cfg.EnableValidation)
	h = internal.HashBool(h, cfg.StrictMode)
	h = internal.HashBool(h, cfg.CreatePaths)
	h = internal.HashBool(h, cfg.CleanupNulls)
	h = internal.HashBool(h, cfg.CompactArrays)
	h = internal.HashBool(h, cfg.ContinueOnError)

	// Input/Output options
	h = internal.HashBool(h, cfg.AllowComments)
	h = internal.HashBool(h, cfg.PreserveNumbers)
	h = internal.HashBool(h, cfg.ValidateInput)
	h = internal.HashBool(h, cfg.ValidateFilePath)
	h = internal.HashBool(h, cfg.SkipValidation)

	// Encoding options
	h = internal.HashBool(h, cfg.Pretty)
	h = internal.HashString(h, cfg.Indent)
	h = internal.HashString(h, cfg.Prefix)
	h = internal.HashBool(h, cfg.EscapeHTML)
	h = internal.HashBool(h, cfg.SortKeys)
	h = internal.HashBool(h, cfg.ValidateUTF8)
	h = internal.HashInt(h, cfg.MaxDepth)
	h = internal.HashBool(h, cfg.DisallowUnknown)
	h = internal.HashInt(h, cfg.FloatPrecision)
	h = internal.HashBool(h, cfg.FloatTruncate)
	h = internal.HashBool(h, cfg.DisableEscaping)
	h = internal.HashBool(h, cfg.EscapeUnicode)
	h = internal.HashBool(h, cfg.EscapeSlash)
	h = internal.HashBool(h, cfg.EscapeNewlines)
	h = internal.HashBool(h, cfg.EscapeTabs)
	h = internal.HashBool(h, cfg.IncludeNulls)

	// CustomEscapes
	h = hashCustomEscapes(h, cfg.CustomEscapes)

	// Observability
	h = internal.HashBool(h, cfg.EnableMetrics)
	h = internal.HashBool(h, cfg.EnableHealthCheck)

	// Context - hash based on nil/non-nil
	if cfg.Context != nil {
		h = internal.HashBool(h, true)
	}

	return h
}

// hashCustomEscapes hashes a CustomEscapes map
func hashCustomEscapes(h uint64, m map[rune]string) uint64 {
	if m == nil {
		return h
	}
	h = internal.HashInt(h, len(m))
	// DETERMINISM FIX: Sort keys to ensure consistent hash regardless of map iteration order
	keys := make([]rune, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, k := range keys {
		h = internal.HashInt(h, int(k))
		h = internal.HashString(h, m[k])
	}
	return h
}

// Get retrieves a value from JSON at the specified path
func Get(jsonStr, path string, cfg ...Config) (any, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return nil, err
	}
	return p.Get(jsonStr, path, cfg...)
}

// GetTyped retrieves a typed value from JSON at the specified path
func GetTyped[T any](jsonStr, path string, cfg ...Config) (T, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		var zero T
		return zero, err
	}
	return GetTypedWithProcessor[T](p, jsonStr, path, cfg...)
}

// GetString retrieves a string value from JSON at the specified path
func GetString(jsonStr, path string, cfg ...Config) (string, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return "", err
	}
	return p.GetString(jsonStr, path, cfg...)
}

// GetInt retrieves an int value from JSON at the specified path
func GetInt(jsonStr, path string, cfg ...Config) (int, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return 0, err
	}
	return p.GetInt(jsonStr, path, cfg...)
}

// GetFloat64 retrieves a float64 value from JSON at the specified path
func GetFloat64(jsonStr, path string, cfg ...Config) (float64, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return 0, err
	}
	return p.GetFloat64(jsonStr, path, cfg...)
}

// GetBool retrieves a bool value from JSON at the specified path
func GetBool(jsonStr, path string, cfg ...Config) (bool, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return false, err
	}
	return p.GetBool(jsonStr, path, cfg...)
}

// GetArray retrieves an array value from JSON at the specified path
func GetArray(jsonStr, path string, cfg ...Config) ([]any, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return nil, err
	}
	return p.GetArray(jsonStr, path, cfg...)
}

// GetObject retrieves an object value from JSON at the specified path
func GetObject(jsonStr, path string, cfg ...Config) (map[string]any, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return nil, err
	}
	return p.GetObject(jsonStr, path, cfg...)
}

// GetWithDefault retrieves a value from JSON at the specified path with a default fallback
func GetWithDefault(jsonStr, path string, defaultValue any, cfg ...Config) any {
	p := getDefaultProcessor()
	if p == nil {
		return defaultValue
	}
	return p.GetWithDefault(jsonStr, path, defaultValue, cfg...)
}

// GetDefault retrieves a typed value from JSON at the specified path with a default fallback.
// This is the recommended generic function for getting values with defaults.
//
// Example:
//
//	name := json.GetDefault[string](data, "user.name", "unknown")
//	age := json.GetDefault[int](data, "user.age", 0)
func GetDefault[T any](jsonStr, path string, defaultValue T, cfg ...Config) T {
	result, err := GetTyped[T](jsonStr, path, cfg...)
	if err != nil {
		return defaultValue
	}
	return result
}

// GetMultiple retrieves multiple values from JSON at the specified paths
func GetMultiple(jsonStr string, paths []string, cfg ...Config) (map[string]any, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return nil, err
	}
	return p.GetMultiple(jsonStr, paths, cfg...)
}

// Set sets a value in JSON at the specified path
// Returns:
//   - On success: modified JSON string and nil error
//   - On failure: original unmodified JSON string and error information
func Set(jsonStr, path string, value any, cfg ...Config) (string, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return jsonStr, err
	}
	return p.Set(jsonStr, path, value, cfg...)
}

// SetMultiple sets multiple values using a map of path-value pairs
func SetMultiple(jsonStr string, updates map[string]any, cfg ...Config) (string, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return jsonStr, err
	}
	return p.SetMultiple(jsonStr, updates, cfg...)
}

// Delete deletes a value from JSON at the specified path
func Delete(jsonStr, path string, cfg ...Config) (string, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return jsonStr, err
	}
	return p.Delete(jsonStr, path, cfg...)
}

// Marshal returns the JSON encoding of v.
// This function is 100% compatible with encoding/json.Marshal.
func Marshal(v any) ([]byte, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return nil, err
	}
	return p.Marshal(v)
}

// Unmarshal parses the JSON-encoded data and stores the result in v.
// This function is 100% compatible with encoding/json.Unmarshal.
func Unmarshal(data []byte, v any) error {
	p, err := getProcessorOrFail()
	if err != nil {
		return err
	}
	return p.Unmarshal(data, v)
}

// MarshalIndent is like Marshal but applies indentation to format the output.
// This function is 100% compatible with encoding/json.MarshalIndent.
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return nil, err
	}
	return p.MarshalIndent(v, prefix, indent)
}

// Compact appends to dst the JSON-encoded src with insignificant space characters elided.
// This function is 100% compatible with encoding/json.Compact.
func Compact(dst *bytes.Buffer, src []byte) error {
	compacted, err := CompactString(string(src))
	if err != nil {
		return err
	}
	dst.WriteString(compacted)
	return nil
}

// Indent appends to dst an indented form of the JSON-encoded src.
// This function is 100% compatible with encoding/json.Indent.
func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	p, err := getProcessorOrFail()
	if err != nil {
		return err
	}
	cfg := DefaultConfig()
	cfg.Pretty = true
	cfg.Prefix = prefix
	cfg.Indent = indent
	result, err := p.FormatPretty(string(src), cfg)
	if err != nil {
		return err
	}
	dst.WriteString(result)
	return nil
}

// HTMLEscape appends to dst the JSON-encoded src with <, >, &, U+2028, and U+2029 characters escaped.
// This function is 100% compatible with encoding/json.HTMLEscape.
func HTMLEscape(dst *bytes.Buffer, src []byte) {
	// Use shared implementation from internal package
	dst.WriteString(internal.HTMLEscape(string(src)))
}

// CompactBuffer is an alias for Compact for buffer operations
func CompactBuffer(dst *bytes.Buffer, src []byte, cfg ...Config) error {
	compacted, err := CompactString(string(src), cfg...)
	if err != nil {
		return err
	}
	dst.WriteString(compacted)
	return nil
}

// IndentBuffer appends to dst an indented form of the JSON-encoded src.
func IndentBuffer(dst *bytes.Buffer, src []byte, prefix, indent string, cfg ...Config) error {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c.Pretty = true
	c.Prefix = prefix
	c.Indent = indent
	result, err := FormatPretty(string(src), c)
	if err != nil {
		return err
	}
	dst.WriteString(result)
	return nil
}

// HTMLEscapeBuffer is an alias for HTMLEscape for buffer operations
func HTMLEscapeBuffer(dst *bytes.Buffer, src []byte, cfg ...Config) {
	// Use shared implementation - cfg is ignored for HTMLEscape (encoding/json compatible behavior)
	_ = cfg // cfg parameter kept for API consistency
	dst.WriteString(internal.HTMLEscape(string(src)))
}

// Encode converts any Go value to JSON string.
// For configuration options, use EncodeWithConfig.
func Encode(value any, cfg ...Config) (string, error) {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	p, err := getProcessorOrFail()
	if err != nil {
		return "", err
	}
	return p.EncodeWithConfig(value, c)
}

// EncodeWithConfig converts any Go value to JSON string using the unified Config.
// This is the recommended way to encode JSON with configuration.
//
// Example:
//
//	// Pretty output
//	result, err := json.EncodeWithConfig(data, json.PrettyConfig())
//
//	// Security-focused output
//	result, err := json.EncodeWithConfig(data, json.SecurityConfig())
//
//	// Custom configuration
//	cfg := json.DefaultConfig()
//	cfg.Pretty = true
//	cfg.SortKeys = true
//	result, err := json.EncodeWithConfig(data, cfg)
func EncodeWithConfig(value any, cfg Config) (string, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return "", err
	}
	return p.EncodeWithConfig(value, cfg)
}

// FormatPretty formats JSON string with pretty indentation.
func FormatPretty(jsonStr string, cfg ...Config) (string, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return "", err
	}
	return p.FormatPretty(jsonStr, cfg...)
}

// CompactString removes whitespace from JSON string.
// This is the recommended function name for consistency with Processor.Compact.
func CompactString(jsonStr string, cfg ...Config) (string, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return "", err
	}
	return p.Compact(jsonStr, cfg...)
}

// Print prints any Go value as JSON to stdout in compact format.
// Note: Writes errors to stderr. Use PrintE for error handling.
func Print(data any) {
	result, err := printData(data, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json.Print error: %v\n", err)
		return
	}
	fmt.Println(result)
}

// PrintPretty prints any Go value as formatted JSON to stdout.
// Note: Writes errors to stderr. Use PrintPrettyE for error handling.
func PrintPretty(data any) {
	result, err := printData(data, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "json.PrintPretty error: %v\n", err)
		return
	}
	fmt.Println(result)
}

// PrintE prints any Go value as JSON to stdout in compact format.
// Returns an error instead of writing to stderr, allowing callers to handle errors.
func PrintE(data any) error {
	result, err := printData(data, false)
	if err != nil {
		return fmt.Errorf("json.Print error: %w", err)
	}
	fmt.Println(result)
	return nil
}

// PrintPrettyE prints any Go value as formatted JSON to stdout.
// Returns an error instead of writing to stderr, allowing callers to handle errors.
func PrintPrettyE(data any) error {
	result, err := printData(data, true)
	if err != nil {
		return fmt.Errorf("json.PrintPretty error: %w", err)
	}
	fmt.Println(result)
	return nil
}

// formatJSONString formats a JSON string or encodes a non-JSON string.
func formatJSONString(jsonStr string, pretty bool, p *Processor) (string, error) {
	if isValid, _ := p.Valid(jsonStr); isValid {
		if pretty {
			return p.FormatPretty(jsonStr)
		}
		return p.Compact(jsonStr)
	}
	// Not valid JSON - encode as a string value
	cfg := DefaultConfig()
	cfg.Pretty = pretty
	return EncodeWithConfig(jsonStr, cfg)
}

// encodeValue encodes any Go value to JSON string.
func encodeValue(value any, pretty bool) (string, error) {
	cfg := DefaultConfig()
	cfg.Pretty = pretty
	return EncodeWithConfig(value, cfg)
}

// printData handles the core logic for Print and PrintPretty
func printData(data any, pretty bool) (string, error) {
	processor := getDefaultProcessor()
	if processor == nil {
		return "", ErrInternalError
	}

	switch v := data.(type) {
	case string:
		return formatJSONString(v, pretty, processor)
	case []byte:
		return formatJSONString(string(v), pretty, processor)
	default:
		return encodeValue(v, pretty)
	}
}

// Valid reports whether data is valid JSON.
// This function is 100% compatible with encoding/json.Valid.
func Valid(data []byte) bool {
	jsonStr := string(data)
	p := getDefaultProcessor()
	if p == nil {
		return false
	}
	valid, err := p.Valid(jsonStr)
	return err == nil && valid
}

// ValidString reports whether the JSON string is valid.
// This is a convenience wrapper for Valid that accepts a string directly.
func ValidString(jsonStr string) bool {
	p := getDefaultProcessor()
	if p == nil {
		return false
	}
	valid, err := p.Valid(jsonStr)
	return err == nil && valid
}

// ValidWithOptions reports whether the JSON string is valid with optional configuration.
// Returns both the validation result and any error that occurred during validation.
func ValidWithOptions(jsonStr string, cfg ...Config) (bool, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return false, err
	}
	return p.Valid(jsonStr, cfg...)
}

// ValidateSchema validates JSON data against a schema
func ValidateSchema(jsonStr string, schema *Schema, cfg ...Config) ([]ValidationError, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return nil, err
	}
	return p.ValidateSchema(jsonStr, schema, cfg...)
}

// LoadFromFile loads JSON data from a file with optional configuration
// Uses the default processor with support for Config such as security validation
func LoadFromFile(filePath string, cfg ...Config) (string, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return "", err
	}
	return p.LoadFromFile(filePath, cfg...)
}

// UnmarshalFromFile reads JSON from a file and unmarshals it into v.
// This is a convenience function that combines file reading and unmarshalling.
// Uses the default processor for security validation and decoding.
//
// Parameters:
//   - path: file path to read JSON from
//   - v: pointer to the target variable where JSON will be unmarshaled
//   - cfg: optional Config for security validation and processing
//
// Returns error if file reading fails or JSON cannot be unmarshaled.
func UnmarshalFromFile(path string, v any, cfg ...Config) error {
	p, err := getProcessorOrFail()
	if err != nil {
		return err
	}
	return p.UnmarshalFromFile(path, v, cfg...)
}

// ProcessBatch processes multiple JSON operations in a single batch.
// This is more efficient than processing each operation individually.
func ProcessBatch(operations []BatchOperation, cfg ...Config) ([]BatchResult, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return nil, err
	}
	return p.ProcessBatch(operations, cfg...)
}

// WarmupCache pre-warms the cache for frequently accessed paths.
// This can improve performance for subsequent operations on the same JSON.
func WarmupCache(jsonStr string, paths []string, cfg ...Config) (*WarmupResult, error) {
	p, err := getProcessorOrFail()
	if err != nil {
		return nil, err
	}
	return p.WarmupCache(jsonStr, paths, cfg...)
}

// ClearCache clears the processor's internal cache.
func ClearCache() {
	p := getDefaultProcessor()
	if p != nil {
		p.ClearCache()
	}
}

// GetStats returns statistics about the default processor.
func GetStats() Stats {
	p := getDefaultProcessor()
	if p == nil {
		return Stats{}
	}
	return p.GetStats()
}

// GetHealthStatus returns the health status of the default processor.
func GetHealthStatus() HealthStatus {
	p := getDefaultProcessor()
	if p == nil {
		return HealthStatus{Healthy: false}
	}
	return p.GetHealthStatus()
}

// =============================================================================
// Unified API - Use these functions for common scenarios
// =============================================================================

// Parse parses a JSON string and returns the root value.
// This is the recommended entry point for parsing JSON strings.
//
// Layer Architecture:
//   - Package-level (this function): Convenience wrapper that uses cached processors
//   - Processor-level: Full control with explicit lifecycle management
//
// Example:
//
//	// Simple parsing (uses default processor)
//	data, err := json.Parse(jsonStr)
//
//	// With configuration (uses config-cached processor)
//	cfg := json.SecurityConfig()
//	data, err := json.Parse(jsonStr, cfg)
//
// For repeated operations, consider creating a Processor directly:
//
//	p := json.MustNew()
//	defer p.Close()
//	data, err := p.Get(jsonStr, "$") // Equivalent to Parse
func Parse(jsonStr string, cfg ...Config) (any, error) {
	if len(cfg) > 0 {
		p, err := getProcessorWithConfig(cfg[0])
		if err != nil {
			return nil, err
		}
		return p.ParseRoot(jsonStr)
	}
	p, err := getProcessorOrFail()
	if err != nil {
		return nil, err
	}
	return p.ParseRoot(jsonStr)
}

// ParseRoot parses a JSON string and returns the root value.
// This is the processor-level method equivalent to Get(jsonStr, "$").
//
// Use this when you have a Processor instance with specific configuration
// or need to manage the processor lifecycle explicitly.
func (p *Processor) ParseRoot(jsonStr string) (any, error) {
	return p.Get(jsonStr, "$")
}

// parse is kept for backward compatibility; delegates to ParseRoot.
//
// Deprecated: Use ParseRoot for clarity.
// Migration: p.parse(jsonStr) -> p.ParseRoot(jsonStr)
// Will be removed in v2.0.0.
func (p *Processor) parse(jsonStr string) (any, error) {
	return p.ParseRoot(jsonStr)
}

// SaveToFile saves JSON data to a file with optional configuration.
// This is the unified API that replaces SaveToFileWithOpts.
//
// Example:
//
//	// Simple save
//	err := json.SaveToFile("data.json", data)
//
//	// With pretty printing
//	cfg := json.PrettyConfig()
//	err := json.SaveToFile("data.json", data, cfg)
func SaveToFile(filePath string, data any, cfg ...Config) error {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	p, err := getProcessorWithConfig(c)
	if err != nil {
		return err
	}
	return p.SaveToFile(filePath, data, c)
}

// MarshalToFile marshals data to JSON and writes to a file.
// This is the unified API that replaces MarshalToFileWithOpts.
//
// Example:
//
//	err := json.MarshalToFile("data.json", myStruct, json.PrettyConfig())
func MarshalToFile(filePath string, data any, cfg ...Config) error {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	p, err := getProcessorWithConfig(c)
	if err != nil {
		return err
	}
	return p.MarshalToFile(filePath, data, c)
}

// SaveToWriter writes JSON data to an io.Writer.
// This is the unified API that replaces SaveToWriterWithOpts.
//
// Example:
//
//	var buf bytes.Buffer
//	err := json.SaveToWriter(&buf, data, json.PrettyConfig())
func SaveToWriter(writer io.Writer, data any, cfg ...Config) error {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	p, err := getProcessorWithConfig(c)
	if err != nil {
		return err
	}
	return p.SaveToWriter(writer, data, c)
}

// EncodeBatch encodes multiple key-value pairs as a JSON object.
// This is the unified API that replaces EncodeBatchWithOpts.
//
// Example:
//
//	result, err := json.EncodeBatch(map[string]any{"name": "Alice", "age": 30})
func EncodeBatch(pairs map[string]any, cfg ...Config) (string, error) {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	p, err := getProcessorWithConfig(c)
	if err != nil {
		return "", err
	}
	return p.EncodeBatch(pairs, c)
}

// EncodeFields encodes specific fields from a struct or map.
// This is the unified API that replaces EncodeFieldsWithOpts.
//
// Example:
//
//	result, err := json.EncodeFields(user, []string{"name", "email"})
func EncodeFields(value any, fields []string, cfg ...Config) (string, error) {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	p, err := getProcessorWithConfig(c)
	if err != nil {
		return "", err
	}
	return p.EncodeFields(value, fields, c)
}

// EncodeStream encodes multiple values as a JSON array.
// This is the unified API that replaces EncodeStreamWithOpts.
//
// Example:
//
//	result, err := json.EncodeStream([]any{1, 2, 3}, json.PrettyConfig())
func EncodeStream(values any, cfg ...Config) (string, error) {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	p, err := getProcessorWithConfig(c)
	if err != nil {
		return "", err
	}
	return p.EncodeStream(values, c)
}

// getProcessorWithConfig returns a processor configured with the given config.
// Uses caching for identical configurations to improve performance.
// SECURITY: Implements cache size limit to prevent unbounded memory growth.
// RACE-FIX: Uses retry loop with maximum attempts to handle concurrent stale entry replacement.
func getProcessorWithConfig(cfg Config) (*Processor, error) {
	// Compute cache key from config
	cacheKey := hashConfig(cfg)

	// Fast path: check cache first
	if cached, ok := configProcessorCache.Load(cacheKey); ok {
		if p, ok := cached.(*Processor); ok && !p.IsClosed() {
			return p, nil
		}
		// Remove stale entry
		configProcessorCache.Delete(cacheKey)
	}

	// Slow path: create new processor
	p, err := New(cfg)
	if err != nil {
		return nil, err
	}

	// Try to store in cache
	if existing, loaded := configProcessorCache.LoadOrStore(cacheKey, p); loaded {
		// Another goroutine stored first
		if ep, ok := existing.(*Processor); ok && !ep.IsClosed() {
			// Theirs is valid, close ours and use theirs
			p.Close()
			return ep, nil
		}
		// Existing entry is stale; replace it with ours
		configProcessorCache.Store(cacheKey, p)
	}

	// Check cache size and evict if necessary
	maybeEvictConfigCache()

	return p, nil
}

// maybeEvictConfigCache checks if the cache exceeds the size limit and evicts if needed.
// Uses a mutex to serialize eviction; counts entries via Range to avoid counter drift.
// RACE SAFETY: Deletes from cache BEFORE closing to minimize the window where another
// goroutine could retrieve a processor being closed. Closes asynchronously to avoid
// blocking eviction on Close() timeout (5s). Any goroutine that retrieves a processor
// between our delete and their new creation will get a fresh processor, which is safe.
func maybeEvictConfigCache() {
	configProcessorCacheMu.Lock()
	defer configProcessorCacheMu.Unlock()

	var count int
	configProcessorCache.Range(func(_, _ any) bool {
		count++
		return true
	})

	if count < configProcessorCacheLimit {
		return
	}

	var keysToDelete []uint64
	var validCount int

	// Scan and remove closed/invalid processors
	configProcessorCache.Range(func(key, value any) bool {
		if p, ok := value.(*Processor); ok {
			if p.IsClosed() {
				keysToDelete = append(keysToDelete, key.(uint64))
			} else {
				validCount++
			}
		} else {
			keysToDelete = append(keysToDelete, key.(uint64))
		}
		return true
	})

	for _, key := range keysToDelete {
		configProcessorCache.Delete(key)
	}

	// If still over limit, evict entries (random eviction)
	// SECURITY: Delete from cache BEFORE closing to ensure no goroutine
	// can retrieve this processor while it's being closed.
	// PERFORMANCE: Close asynchronously to avoid blocking eviction on Close() timeout.
	if validCount >= configProcessorCacheLimit {
		var toClose []*Processor
		evicted := 0

		configProcessorCache.Range(func(key, value any) bool {
			if evicted >= configProcessorCacheEvictNum {
				return false
			}
			configProcessorCache.Delete(key)
			if p, ok := value.(*Processor); ok && !p.IsClosed() {
				toClose = append(toClose, p)
			}
			evicted++
			return true
		})

		// Close processors asynchronously to avoid blocking
		for _, p := range toClose {
			go p.Close()
		}
	}
}
