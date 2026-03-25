package json

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// Get retrieves a value from JSON at the specified path
func Get(jsonStr, path string, cfg ...*Config) (any, error) {
	return getDefaultProcessor().Get(jsonStr, path, cfg...)
}

// GetTyped retrieves a typed value from JSON at the specified path
func GetTyped[T any](jsonStr, path string, cfg ...*Config) (T, error) {
	return GetTypedWithProcessor[T](getDefaultProcessor(), jsonStr, path, cfg...)
}

// GetString retrieves a string value from JSON at the specified path
func GetString(jsonStr, path string, cfg ...*Config) (string, error) {
	return getDefaultProcessor().GetString(jsonStr, path, cfg...)
}

// GetInt retrieves an int value from JSON at the specified path
func GetInt(jsonStr, path string, cfg ...*Config) (int, error) {
	return getDefaultProcessor().GetInt(jsonStr, path, cfg...)
}

// GetFloat64 retrieves a float64 value from JSON at the specified path
func GetFloat64(jsonStr, path string, cfg ...*Config) (float64, error) {
	return getDefaultProcessor().GetFloat64(jsonStr, path, cfg...)
}

// GetBool retrieves a bool value from JSON at the specified path
func GetBool(jsonStr, path string, cfg ...*Config) (bool, error) {
	return getDefaultProcessor().GetBool(jsonStr, path, cfg...)
}

// GetArray retrieves an array value from JSON at the specified path
func GetArray(jsonStr, path string, cfg ...*Config) ([]any, error) {
	return getDefaultProcessor().GetArray(jsonStr, path, cfg...)
}

// GetObject retrieves an object value from JSON at the specified path
func GetObject(jsonStr, path string, cfg ...*Config) (map[string]any, error) {
	return getDefaultProcessor().GetObject(jsonStr, path, cfg...)
}

// GetWithDefault retrieves a value from JSON at the specified path with a default fallback
func GetWithDefault(jsonStr, path string, defaultValue any, cfg ...*Config) any {
	return getDefaultProcessor().GetWithDefault(jsonStr, path, defaultValue, cfg...)
}

// GetDefault retrieves a typed value from JSON at the specified path with a default fallback.
// This is the recommended generic function for getting values with defaults.
//
// Example:
//
//	name := json.GetDefault[string](data, "user.name", "unknown")
//	age := json.GetDefault[int](data, "user.age", 0)
func GetDefault[T any](jsonStr, path string, defaultValue T, cfg ...*Config) T {
	result, err := GetTyped[T](jsonStr, path, cfg...)
	if err != nil {
		return defaultValue
	}
	return result
}

// GetMultiple retrieves multiple values from JSON at the specified paths
func GetMultiple(jsonStr string, paths []string, cfg ...*Config) (map[string]any, error) {
	return getDefaultProcessor().GetMultiple(jsonStr, paths, cfg...)
}

// isZeroValue checks if a value is the zero value for its type
// Supports all standard numeric types, bool, string, slices, maps, and json.Number
func isZeroValue(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case int:
		return val == 0
	case int8:
		return val == 0
	case int16:
		return val == 0
	case int32:
		return val == 0
	case int64:
		return val == 0
	case uint:
		return val == 0
	case uint8:
		return val == 0
	case uint16:
		return val == 0
	case uint32:
		return val == 0
	case uint64:
		return val == 0
	case float32:
		return val == 0
	case float64:
		return val == 0
	case bool:
		return !val
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	case Number:
		// json.Number type - check if it represents zero
		n, err := val.Int64()
		return err == nil && n == 0
	default:
		return false
	}
}

// Set sets a value in JSON at the specified path
// Returns:
//   - On success: modified JSON string and nil error
//   - On failure: original unmodified JSON string and error information
func Set(jsonStr, path string, value any, cfg ...*Config) (string, error) {
	return getDefaultProcessor().Set(jsonStr, path, value, cfg...)
}

// SetMultiple sets multiple values using a map of path-value pairs
func SetMultiple(jsonStr string, updates map[string]any, cfg ...*Config) (string, error) {
	return getDefaultProcessor().SetMultiple(jsonStr, updates, cfg...)
}

// Delete deletes a value from JSON at the specified path
func Delete(jsonStr, path string, cfg ...*Config) (string, error) {
	return getDefaultProcessor().Delete(jsonStr, path, cfg...)
}

// Marshal returns the JSON encoding of v.
// This function is 100% compatible with encoding/json.Marshal.
func Marshal(v any) ([]byte, error) {
	return getDefaultProcessor().Marshal(v)
}

// Unmarshal parses the JSON-encoded data and stores the result in v.
// This function is 100% compatible with encoding/json.Unmarshal.
func Unmarshal(data []byte, v any) error {
	return getDefaultProcessor().Unmarshal(data, v)
}

// MarshalIndent is like Marshal but applies indentation to format the output.
// This function is 100% compatible with encoding/json.MarshalIndent.
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return getDefaultProcessor().MarshalIndent(v, prefix, indent)
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
	processor := getDefaultProcessor()
	result, err := processor.FormatPretty(string(src))
	if err != nil {
		return err
	}
	dst.WriteString(result)
	return nil
}

// HTMLEscape appends to dst the JSON-encoded src with <, >, &, U+2028, and U+2029 characters escaped.
// This function is 100% compatible with encoding/json.HTMLEscape.
func HTMLEscape(dst *bytes.Buffer, src []byte) {
	// Use standard library compatible HTML escaping
	result := htmlEscape(string(src))
	dst.WriteString(result)
}

// htmlEscape performs HTML escaping on JSON string
// Compatible with encoding/json: escapes <, >, &, U+2028, U+2029
func htmlEscape(s string) string {
	var buf bytes.Buffer
	buf.Grow(len(s))
	for _, r := range s {
		switch r {
		case '<':
			buf.WriteString("\\u003c")
		case '>':
			buf.WriteString("\\u003e")
		case '&':
			buf.WriteString("\\u0026")
		case '\u2028': // Line Separator - required for JSON-in-JS compatibility
			buf.WriteString("\\u2028")
		case '\u2029': // Paragraph Separator - required for JSON-in-JS compatibility
			buf.WriteString("\\u2029")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// CompactBuffer is an alias for Compact for buffer operations
func CompactBuffer(dst *bytes.Buffer, src []byte, cfg ...*Config) error {
	compacted, err := CompactString(string(src), cfg...)
	if err != nil {
		return err
	}
	dst.WriteString(compacted)
	return nil
}

// IndentBuffer is an alias for Indent for buffer operations
func IndentBuffer(dst *bytes.Buffer, src []byte, prefix, indent string, cfg ...*Config) error {
	result, err := FormatPretty(string(src), cfg...)
	if err != nil {
		return err
	}
	dst.WriteString(result)
	return nil
}

// HTMLEscapeBuffer is an alias for HTMLEscape for buffer operations
func HTMLEscapeBuffer(dst *bytes.Buffer, src []byte, cfg ...*Config) {
	result := htmlEscape(string(src))
	dst.WriteString(result)
}

// Encode converts any Go value to JSON string.
// For configuration options, use EncodeWithConfig.
func Encode(value any, cfg ...*Config) (string, error) {
	var c *Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	return getDefaultProcessor().EncodeWithConfig(value, c)
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
func EncodeWithConfig(value any, cfg *Config) (string, error) {
	return getDefaultProcessor().EncodeWithConfig(value, cfg)
}

// FormatPretty formats JSON string with pretty indentation.
func FormatPretty(jsonStr string, cfg ...*Config) (string, error) {
	return getDefaultProcessor().FormatPretty(jsonStr, cfg...)
}

// CompactString removes whitespace from JSON string.
// This is the recommended function name for consistency with Processor.Compact.
func CompactString(jsonStr string, cfg ...*Config) (string, error) {
	return getDefaultProcessor().Compact(jsonStr, cfg...)
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

// printData handles the core logic for Print and PrintPretty
func printData(data any, pretty bool) (string, error) {
	processor := getDefaultProcessor()

	switch v := data.(type) {
	case string:
		// Check if it's valid JSON - if so, format it directly
		if isValid, _ := processor.Valid(v); isValid {
			if pretty {
				return processor.FormatPretty(v)
			}
			return processor.Compact(v)
		}
		// Not valid JSON, encode as a normal string
		cfg := DefaultConfig()
		cfg.Pretty = pretty
		return EncodeWithConfig(v, cfg)

	case []byte:
		jsonStr := string(v)
		// Check if it's valid JSON - if so, format it directly
		if isValid, _ := processor.Valid(jsonStr); isValid {
			if pretty {
				return processor.FormatPretty(jsonStr)
			}
			return processor.Compact(jsonStr)
		}
		// Not valid JSON, encode as normal
		cfg := DefaultConfig()
		cfg.Pretty = pretty
		return EncodeWithConfig(v, cfg)

	default:
		// Encode other types normally
		cfg := DefaultConfig()
		cfg.Pretty = pretty
		return EncodeWithConfig(v, cfg)
	}
}

// Valid reports whether data is valid JSON.
// This function is 100% compatible with encoding/json.Valid.
func Valid(data []byte) bool {
	jsonStr := string(data)
	valid, err := getDefaultProcessor().Valid(jsonStr)
	return err == nil && valid
}

// ValidString reports whether the JSON string is valid.
// This is a convenience wrapper for Valid that accepts a string directly.
func ValidString(jsonStr string) bool {
	valid, err := getDefaultProcessor().Valid(jsonStr)
	return err == nil && valid
}

// ValidWithOptions reports whether the JSON string is valid with optional configuration.
// Returns both the validation result and any error that occurred during validation.
func ValidWithOptions(jsonStr string, cfg ...*Config) (bool, error) {
	return getDefaultProcessor().Valid(jsonStr, cfg...)
}

// ValidateSchema validates JSON data against a schema
func ValidateSchema(jsonStr string, schema *Schema, cfg ...*Config) ([]ValidationError, error) {
	return getDefaultProcessor().ValidateSchema(jsonStr, schema, cfg...)
}

// LoadFromFile loads JSON data from a file with optional configuration
// Uses the default processor with support for Config such as security validation
func LoadFromFile(filePath string, cfg ...*Config) (string, error) {
	return getDefaultProcessor().LoadFromFile(filePath, cfg...)
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
func UnmarshalFromFile(path string, v any, cfg ...*Config) error {
	return getDefaultProcessor().UnmarshalFromFile(path, v, cfg...)
}

// ProcessBatch processes multiple JSON operations in a single batch.
// This is more efficient than processing each operation individually.
func ProcessBatch(operations []BatchOperation, cfg ...*Config) ([]BatchResult, error) {
	return getDefaultProcessor().ProcessBatch(operations, cfg...)
}

// WarmupCache pre-warms the cache for frequently accessed paths.
// This can improve performance for subsequent operations on the same JSON.
func WarmupCache(jsonStr string, paths []string, cfg ...*Config) (*WarmupResult, error) {
	return getDefaultProcessor().WarmupCache(jsonStr, paths, cfg...)
}

// ClearCache clears the processor's internal cache.
func ClearCache() {
	getDefaultProcessor().ClearCache()
}

// GetStats returns statistics about the default processor.
func GetStats() Stats {
	return getDefaultProcessor().GetStats()
}

// GetHealthStatus returns the health status of the default processor.
func GetHealthStatus() HealthStatus {
	return getDefaultProcessor().GetHealthStatus()
}

// =============================================================================
// Unified API - Use these functions for common scenarios
// =============================================================================

// Parse parses a JSON string with optional configuration.
// This is the unified API that replaces Get for simple parsing use cases.
//
// Example:
//
//	// Simple parsing
//	data, err := json.Parse(jsonStr)
//
//	// With configuration
//	cfg := json.SecurityConfig()
//	data, err := json.Parse(jsonStr, cfg)
func Parse(jsonStr string, cfg ...*Config) (any, error) {
	if len(cfg) > 0 && cfg[0] != nil {
		return getProcessorWithConfig(cfg[0]).parse(jsonStr)
	}
	return getDefaultProcessor().parse(jsonStr)
}

// parse is the internal parsing method
func (p *Processor) parse(jsonStr string) (any, error) {
	return p.Get(jsonStr, "$")
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
func SaveToFile(filePath string, data any, cfg ...*Config) error {
	c := DefaultConfig()
	if len(cfg) > 0 && cfg[0] != nil {
		c = cfg[0]
	}
	return getProcessorWithConfig(c).SaveToFile(filePath, data, c)
}

// MarshalToFile marshals data to JSON and writes to a file.
// This is the unified API that replaces MarshalToFileWithOpts.
//
// Example:
//
//	err := json.MarshalToFile("data.json", myStruct, json.PrettyConfig())
func MarshalToFile(filePath string, data any, cfg ...*Config) error {
	c := DefaultConfig()
	if len(cfg) > 0 && cfg[0] != nil {
		c = cfg[0]
	}
	return getProcessorWithConfig(c).MarshalToFile(filePath, data, c)
}

// SaveToWriter writes JSON data to an io.Writer.
// This is the unified API that replaces SaveToWriterWithOpts.
//
// Example:
//
//	var buf bytes.Buffer
//	err := json.SaveToWriter(&buf, data, json.PrettyConfig())
func SaveToWriter(writer io.Writer, data any, cfg ...*Config) error {
	c := DefaultConfig()
	if len(cfg) > 0 && cfg[0] != nil {
		c = cfg[0]
	}
	return getProcessorWithConfig(c).SaveToWriter(writer, data, c)
}

// EncodeBatch encodes multiple key-value pairs as a JSON object.
// This is the unified API that replaces EncodeBatchWithOpts.
//
// Example:
//
//	result, err := json.EncodeBatch(map[string]any{"name": "Alice", "age": 30})
func EncodeBatch(pairs map[string]any, cfg ...*Config) (string, error) {
	c := DefaultConfig()
	if len(cfg) > 0 && cfg[0] != nil {
		c = cfg[0]
	}
	return getProcessorWithConfig(c).EncodeBatch(pairs, c)
}

// EncodeFields encodes specific fields from a struct or map.
// This is the unified API that replaces EncodeFieldsWithOpts.
//
// Example:
//
//	result, err := json.EncodeFields(user, []string{"name", "email"})
func EncodeFields(value any, fields []string, cfg ...*Config) (string, error) {
	c := DefaultConfig()
	if len(cfg) > 0 && cfg[0] != nil {
		c = cfg[0]
	}
	return getProcessorWithConfig(c).EncodeFields(value, fields, c)
}

// EncodeStream encodes multiple values as a JSON array.
// This is the unified API that replaces EncodeStreamWithOpts.
//
// Example:
//
//	result, err := json.EncodeStream([]any{1, 2, 3}, json.PrettyConfig())
func EncodeStream(values any, cfg ...*Config) (string, error) {
	c := DefaultConfig()
	if len(cfg) > 0 && cfg[0] != nil {
		c = cfg[0]
	}
	return getProcessorWithConfig(c).EncodeStream(values, c)
}

// getProcessorWithConfig returns a processor configured with the given config.
// For performance, this could be cached in the future.
func getProcessorWithConfig(cfg *Config) *Processor {
	if cfg == nil {
		return getDefaultProcessor()
	}
	return New(cfg)
}
