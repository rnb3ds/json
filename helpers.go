package json

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/cybergodev/json/internal"
)

// ============================================================================
// TYPE CONVERSION HELPERS
// Internal helpers to reduce code duplication while maintaining
// zero-allocation type switches for performance-critical paths.
// ============================================================================

// int64Result holds the result of integer conversion to avoid multiple returns
type int64Result struct {
	value int64
	ok    bool
}

// convertToInt64Core is the internal core function for integer conversion.
// Uses a single type switch with all integer types to minimize branching.
func convertToInt64Core(value any) int64Result {
	switch v := value.(type) {
	case int:
		return int64Result{int64(v), true}
	case int8:
		return int64Result{int64(v), true}
	case int16:
		return int64Result{int64(v), true}
	case int32:
		return int64Result{int64(v), true}
	case int64:
		return int64Result{v, true}
	case uint:
		u64 := uint64(v)
		if u64 > uint64(math.MaxInt64) {
			return int64Result{0, false}
		}
		return int64Result{int64(u64), true}
	case uint8:
		return int64Result{int64(v), true}
	case uint16:
		return int64Result{int64(v), true}
	case uint32:
		return int64Result{int64(v), true}
	case uint64:
		if v <= 9223372036854775807 {
			return int64Result{int64(v), true}
		}
		return int64Result{0, false}
	}
	return int64Result{0, false}
}

// ConvertToInt converts any value to int with comprehensive type support.
// Delegates to internal core function to reduce code duplication.
// MAINTENANCE: Keep type switch cases in sync with ConvertToInt64, ConvertToUint64, ConvertToFloat64.
func ConvertToInt(value any) (int, bool) {
	// Fast path: use core integer conversion
	if result := convertToInt64Core(value); result.ok {
		if result.value >= -2147483648 && result.value <= 2147483647 {
			return int(result.value), true
		}
		return 0, false
	}

	// Handle non-integer types
	switch v := value.(type) {
	case float32:
		if v == float32(int(v)) && v >= -2147483648 && v <= 2147483647 {
			return int(v), true
		}
	case float64:
		if v == float64(int(v)) && v >= -2147483648 && v <= 2147483647 {
			return int(v), true
		}
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i, true
		}
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case json.Number:
		if i, err := v.Int64(); err == nil && i >= -2147483648 && i <= 2147483647 {
			return int(i), true
		}
	}
	return 0, false
}

// ConvertToInt64 converts any value to int64.
// Delegates to internal core function to reduce code duplication.
// MAINTENANCE: Keep type switch cases in sync with ConvertToInt, ConvertToUint64, ConvertToFloat64.
func ConvertToInt64(value any) (int64, bool) {
	// Fast path: use core integer conversion
	if result := convertToInt64Core(value); result.ok {
		return result.value, true
	}

	// Handle non-integer types
	switch v := value.(type) {
	case float32:
		if v == float32(int64(v)) {
			return int64(v), true
		}
	case float64:
		if v == float64(int64(v)) {
			return int64(v), true
		}
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i, true
		}
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, true
		}
	}
	return 0, false
}

// ConvertToUint64 converts any value to uint64.
// Delegates to internal core function to reduce code duplication.
// MAINTENANCE: Keep type switch cases in sync with ConvertToInt, ConvertToInt64, ConvertToFloat64.
func ConvertToUint64(value any) (uint64, bool) {
	// Special case: uint64 needs direct handling for values > int64 max
	switch v := value.(type) {
	case uint64:
		return v, true
	case uint:
		return uint64(v), true
	}

	// Fast path: use core integer conversion for other signed types
	if result := convertToInt64Core(value); result.ok && result.value >= 0 {
		return uint64(result.value), true
	}

	// Handle non-integer types
	switch v := value.(type) {
	case float32:
		if v >= 0 && v == float32(uint64(v)) {
			return uint64(v), true
		}
	case float64:
		if v >= 0 && v == float64(uint64(v)) {
			return uint64(v), true
		}
	case string:
		if i, err := strconv.ParseUint(v, 10, 64); err == nil {
			return i, true
		}
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case json.Number:
		if i, err := v.Int64(); err == nil && i >= 0 {
			return uint64(i), true
		}
	}
	return 0, false
}

// ConvertToFloat64 converts any value to float64.
// Delegates to internal core function to reduce code duplication.
// MAINTENANCE: Keep type switch cases in sync with ConvertToInt, ConvertToInt64, ConvertToUint64.
func ConvertToFloat64(value any) (float64, bool) {
	// Fast path: use core integer conversion
	if result := convertToInt64Core(value); result.ok {
		return float64(result.value), true
	}

	// Handle non-integer types
	switch v := value.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	case bool:
		if v {
			return 1.0, true
		}
		return 0.0, true
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f, true
		}
	}
	return 0.0, false
}

// ConvertToBool converts any value to bool.
// String conversion supports both standard formats and user-friendly formats:
// Standard: "1", "t", "T", "TRUE", "true", "True", "0", "f", "F", "FALSE", "false", "False"
// Extended: "yes", "on" -> true, "no", "off", "" -> false
// Delegates to internal core function to reduce code duplication
func ConvertToBool(value any) (bool, bool) {
	// Fast path: use core integer conversion for numeric types
	if result := convertToInt64Core(value); result.ok {
		return result.value != 0, true
	}

	// Handle non-integer types
	switch v := value.(type) {
	case bool:
		return v, true
	case float32:
		return v != 0.0, true
	case float64:
		return v != 0.0, true
	case string:
		// First try standard library format
		if result, err := strconv.ParseBool(v); err == nil {
			return result, true
		}
		// Then try extended user-friendly formats
		switch strings.ToLower(v) {
		case "yes", "on":
			return true, true
		case "no", "off", "":
			return false, true
		}
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return f != 0.0, true
		}
	}
	return false, false
}

// GetTypedWithProcessor retrieves a typed value from JSON using a specific processor.
// This is the core implementation used by GetTyped and GetTypedOr.
func GetTypedWithProcessor[T any](processor *Processor, jsonStr, path string, cfg ...Config) (T, error) {
	var zero T

	value, err := processor.Get(jsonStr, path, cfg...)
	if err != nil {
		return zero, err
	}

	if value == nil {
		return handleNullValue[T]()
	}

	if converted, ok := UnifiedTypeConversion[T](value); ok {
		return converted, nil
	}

	// Fallback: re-marshal and unmarshal for complex types
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return zero, &JsonsError{
			Op:      "get_typed",
			Path:    path,
			Message: fmt.Sprintf("failed to marshal value for type conversion: %v", err),
			Err:     ErrTypeMismatch,
		}
	}

	var finalResult T
	if err := json.Unmarshal(jsonBytes, &finalResult); err != nil {
		return zero, &JsonsError{
			Op:      "get_typed",
			Path:    path,
			Message: fmt.Sprintf("failed to convert value to type %T: %v", finalResult, err),
			Err:     ErrTypeMismatch,
		}
	}

	return finalResult, nil
}

// UnifiedTypeConversion provides optimized type conversion with comprehensive support
func UnifiedTypeConversion[T any](value any) (T, bool) {
	var zero T

	// Handle nil values
	if value == nil {
		return zero, true
	}

	// Direct type assertion (fastest path)
	if typedValue, ok := value.(T); ok {
		return typedValue, true
	}

	// Get target type information
	targetType := reflect.TypeOf(zero)
	if targetType == nil {
		return zero, false
	}

	// Handle pointer types
	if targetType.Kind() == reflect.Pointer {
		elemType := targetType.Elem()
		elemValue := reflect.New(elemType).Interface()
		if converted, ok := convertValue(value, elemValue); ok {
			if result, ok := converted.(T); ok {
				return result, true
			}
		}
		return zero, false
	}

	// Convert to target type
	if converted, ok := convertValue(value, zero); ok {
		if result, ok := converted.(T); ok {
			return result, true
		}
	}

	return zero, false
}

// convertValue handles the actual conversion logic
func convertValue(value any, target any) (any, bool) {
	targetType := reflect.TypeOf(target)

	switch targetType.Kind() {
	case reflect.String:
		// Inline string conversion - fix order to handle json.Number before fmt.Stringer
		switch v := value.(type) {
		case string:
			return v, true
		case []byte:
			return string(v), true
		case json.Number:
			return string(v), true
		case fmt.Stringer:
			return v.String(), true
		default:
			return fmt.Sprintf("%v", v), true
		}
	case reflect.Int:
		if i, ok := ConvertToInt(value); ok {
			return i, true
		}
	case reflect.Int64:
		if i, ok := ConvertToInt64(value); ok {
			return i, true
		}
	case reflect.Uint64:
		if i, ok := ConvertToUint64(value); ok {
			return i, true
		}
	case reflect.Float64:
		if f, ok := ConvertToFloat64(value); ok {
			return f, true
		}
	case reflect.Bool:
		if b, ok := ConvertToBool(value); ok {
			return b, true
		}
	case reflect.Slice:
		if s, ok := convertToSlice(value, targetType); ok {
			return s, true
		}
	case reflect.Map:
		if m, ok := convertToMap(value, targetType); ok {
			return m, true
		}
	}

	return nil, false
}

// convertToSlice converts value to slice type
func convertToSlice(value any, targetType reflect.Type) (any, bool) {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil, false
	}

	elemType := targetType.Elem()
	result := reflect.MakeSlice(targetType, rv.Len(), rv.Len())

	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i).Interface()
		if converted, ok := convertValue(elem, reflect.Zero(elemType).Interface()); ok {
			result.Index(i).Set(reflect.ValueOf(converted))
		} else {
			return nil, false
		}
	}

	return result.Interface(), true
}

// convertToMap converts value to map type
func convertToMap(value any, targetType reflect.Type) (any, bool) {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Map {
		return nil, false
	}

	keyType := targetType.Key()
	elemType := targetType.Elem()
	result := reflect.MakeMap(targetType)

	for _, key := range rv.MapKeys() {
		keyInterface := key.Interface()
		valueInterface := rv.MapIndex(key).Interface()

		convertedKey, keyOk := convertValue(keyInterface, reflect.Zero(keyType).Interface())
		convertedValue, valueOk := convertValue(valueInterface, reflect.Zero(elemType).Interface())

		if keyOk && valueOk {
			result.SetMapIndex(reflect.ValueOf(convertedKey), reflect.ValueOf(convertedValue))
		} else {
			return nil, false
		}
	}

	return result.Interface(), true
}

// SafeConvertToInt64 safely converts any value to int64 with error handling
func SafeConvertToInt64(value any) (int64, error) {
	if result, ok := ConvertToInt64(value); ok {
		return result, nil
	}
	return 0, fmt.Errorf("cannot convert %T to int64", value)
}

// SafeConvertToUint64 safely converts any value to uint64 with error handling
func SafeConvertToUint64(value any) (uint64, error) {
	if result, ok := ConvertToUint64(value); ok {
		return result, nil
	}
	return 0, fmt.Errorf("cannot convert %T to uint64", value)
}

// FormatNumber formats a number value as a string
func FormatNumber(value any) string {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ConvertToString converts any value to string (for backward compatibility)
func ConvertToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case json.Number:
		return string(v)
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// IsValidJSON quickly checks if a string is valid JSON
func IsValidJSON(jsonStr string) bool {
	decoder := newNumberPreservingDecoder(false)
	_, err := decoder.DecodeToAny(jsonStr)
	return err == nil
}

// IsValidPath checks if a path expression is valid
func IsValidPath(path string) bool {
	if path == "" {
		return false
	}
	if path == "." {
		return true
	}
	processor := getDefaultProcessor()
	if processor == nil {
		return false
	}
	err := processor.validatePath(path)
	return err == nil
}

// ValidatePath validates a path expression and returns detailed error information
func ValidatePath(path string) error {
	if path == "" {
		return &JsonsError{
			Op:      "validate_path",
			Path:    path,
			Message: "path cannot be empty",
			Err:     ErrInvalidPath,
		}
	}
	if path == "." {
		return nil
	}
	processor := getDefaultProcessor()
	if processor == nil {
		return &JsonsError{
			Op:      "validate_path",
			Path:    path,
			Message: "processor not available",
			Err:     ErrInternalError,
		}
	}
	return processor.validatePath(path)
}

// deepCopyMaxDepth is the maximum recursion depth for DeepCopy operations
// SECURITY: Prevents stack overflow from deeply nested structures
const deepCopyMaxDepth = 200

// DeepCopy creates a deep copy of JSON-compatible data
// Uses direct recursive copying for better performance (avoids marshal/unmarshal overhead)
// SECURITY: Added depth limit to prevent stack overflow
func DeepCopy(data any) (any, error) {
	return deepCopyValueWithDepth(data, 0)
}

// deepCopyValueWithDepth performs recursive deep copy with depth tracking
// SECURITY: Depth parameter prevents stack overflow from deeply nested structures
func deepCopyValueWithDepth(data any, depth int) (any, error) {
	// SECURITY: Check depth limit to prevent stack overflow
	if depth > deepCopyMaxDepth {
		return nil, fmt.Errorf("deep copy depth limit exceeded: maximum depth is %d", deepCopyMaxDepth)
	}

	if data == nil {
		return nil, nil
	}

	// Fast path for primitive types (no allocation needed)
	switch v := data.(type) {
	case bool:
		return v, nil
	case int:
		return v, nil
	case int8:
		return v, nil
	case int16:
		return v, nil
	case int32:
		return v, nil
	case int64:
		return v, nil
	case uint:
		return v, nil
	case uint8:
		return v, nil
	case uint16:
		return v, nil
	case uint32:
		return v, nil
	case uint64:
		return v, nil
	case float32:
		return v, nil
	case float64:
		return v, nil
	case string:
		return v, nil
	case json.Number:
		// json.Number is immutable, return as-is
		return v, nil
	}

	// Handle complex types with type-specific optimizations
	switch v := data.(type) {
	case map[string]any:
		return deepCopyMapWithDepth(v, depth)
	case []any:
		return deepCopySliceWithDepth(v, depth)
	case map[string]string:
		// Fast path for map[string]string - no recursion needed
		result := make(map[string]string, len(v))
		maps.Copy(result, v)
		return result, nil
	case []string:
		// Fast path for []string - no recursion needed
		result := make([]string, len(v))
		copy(result, v)
		return result, nil
	case []int:
		// Fast path for []int - no recursion needed
		result := make([]int, len(v))
		copy(result, v)
		return result, nil
	case []float64:
		// Fast path for []float64 - no recursion needed
		result := make([]float64, len(v))
		copy(result, v)
		return result, nil
	case []bool:
		// Fast path for []bool - no recursion needed
		result := make([]bool, len(v))
		copy(result, v)
		return result, nil
	default:
		// Fallback to marshal/unmarshal for unknown types (structs, custom types, etc.)
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data for deep copy: %v", err)
		}
		var result any
		if err := json.Unmarshal(jsonBytes, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data for deep copy: %v", err)
		}
		return result, nil
	}
}

// deepCopyMapWithDepth creates a deep copy of a map with depth tracking
// SECURITY: Pass depth to recursive calls for stack overflow protection
func deepCopyMapWithDepth(m map[string]any, depth int) (map[string]any, error) {
	result := make(map[string]any, len(m))
	for key, val := range m {
		copied, err := deepCopyValueWithDepth(val, depth+1)
		if err != nil {
			return nil, fmt.Errorf("error copying key '%s': %v", key, err)
		}
		result[key] = copied
	}
	return result, nil
}

// deepCopySliceWithDepth creates a deep copy of a slice with depth tracking
// SECURITY: Pass depth to recursive calls for stack overflow protection
func deepCopySliceWithDepth(s []any, depth int) ([]any, error) {
	result := make([]any, len(s))
	for i, val := range s {
		copied, err := deepCopyValueWithDepth(val, depth+1)
		if err != nil {
			return nil, fmt.Errorf("error copying index %d: %v", i, err)
		}
		result[i] = copied
	}
	return result, nil
}

// CompareJSON compares two JSON strings for equality by parsing and normalizing them.
// This function handles numeric precision differences and key ordering.
//
// Example:
//
//	equal, err := json.CompareJSON(`{"a":1}`, `{"a":1.0}`)
//	// equal == true
func CompareJSON(json1, json2 string) (bool, error) {
	decoder := newNumberPreservingDecoder(true)

	data1, err := decoder.DecodeToAny(json1)
	if err != nil {
		return false, fmt.Errorf("invalid JSON in first argument: %v", err)
	}

	data2, err := decoder.DecodeToAny(json2)
	if err != nil {
		return false, fmt.Errorf("invalid JSON in second argument: %v", err)
	}

	bytes1, err := json.Marshal(data1)
	if err != nil {
		return false, err
	}

	bytes2, err := json.Marshal(data2)
	if err != nil {
		return false, err
	}

	return string(bytes1) == string(bytes2), nil
}

// MergeJSON merges two JSON objects using deep merge strategy.
// For nested objects, it recursively merges keys according to the specified mode.
// For primitive values and arrays, the value from json2 takes precedence.
//
// Supported modes (optional, defaults to MergeUnion):
//   - MergeUnion: combines all keys from both objects (default)
//   - MergeIntersection: only keys present in both objects
//   - MergeDifference: keys in json1 but not in json2
//
// Example:
//
//	// Union merge (default)
//	result, err := json.MergeJSON(a, b)
//
//	// Intersection merge
//	result, err := json.MergeJSON(a, b, json.MergeIntersection)
//
//	// Difference merge
//	result, err := json.MergeJSON(a, b, json.MergeDifference)
func MergeJSON(json1, json2 string, mode ...MergeMode) (string, error) {
	m := MergeUnion
	if len(mode) > 0 {
		m = mode[0]
	}

	decoder := newNumberPreservingDecoder(true)

	data1, err := decoder.DecodeToAny(json1)
	if err != nil {
		return "", fmt.Errorf("invalid JSON in first argument: %v", err)
	}

	data2, err := decoder.DecodeToAny(json2)
	if err != nil {
		return "", fmt.Errorf("invalid JSON in second argument: %v", err)
	}

	obj1, ok1 := data1.(map[string]any)
	obj2, ok2 := data2.(map[string]any)

	if !ok1 {
		return "", fmt.Errorf("first JSON is not an object")
	}
	if !ok2 {
		return "", fmt.Errorf("second JSON is not an object")
	}

	merged := internal.DeepMergeWithMode(obj1, obj2, internal.MergeMode(m))

	// Convert library Number types to float64 for proper encoding
	converted := convertLibraryNumbers(merged)

	// Use library's Encode function to properly handle the result
	return Encode(converted)
}

// convertLibraryNumbers recursively converts the library's Number type to float64
// This is needed because the library's NumberPreservingDecoder returns Number (not json.Number)
func convertLibraryNumbers(data any) any {
	switch v := data.(type) {
	case Number:
		f, err := v.Float64()
		if err != nil {
			return v // Keep original if conversion fails
		}
		return f
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, value := range v {
			result[key] = convertLibraryNumbers(value)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = convertLibraryNumbers(item)
		}
		return result
	default:
		return data
	}
}

// MergeJSONMany merges multiple JSON objects with specified merge mode.
// Returns error if less than 2 JSON strings are provided.
//
// Example:
//
//	result, err := json.MergeJSONMany(json.MergeUnion, config1, config2, config3)
func MergeJSONMany(mode MergeMode, jsons ...string) (string, error) {
	if len(jsons) < 2 {
		return "", fmt.Errorf("MergeJSONMany requires at least 2 JSON strings, got %d", len(jsons))
	}

	result := jsons[0]
	for i := 1; i < len(jsons); i++ {
		var err error
		result, err = MergeJSON(result, jsons[i], mode)
		if err != nil {
			return "", fmt.Errorf("merge failed at index %d: %w", i, err)
		}
	}

	return result, nil
}

// getTypedWithProcessor retrieves a typed value from JSON using a specific processor
func getTypedWithProcessor[T any](processor *Processor, jsonStr, path string, opts ...Config) (T, error) {
	var zero T

	value, err := processor.Get(jsonStr, path, opts...)
	if err != nil {
		return zero, err
	}

	if value == nil {
		return handleNullValue[T]()
	}

	if converted, ok := UnifiedTypeConversion[T](value); ok {
		return converted, nil
	}

	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return zero, &JsonsError{
			Op:      "get_typed",
			Path:    path,
			Message: fmt.Sprintf("failed to marshal value for type conversion: %v", err),
			Err:     ErrTypeMismatch,
		}
	}

	var finalResult T
	if err := json.Unmarshal(jsonBytes, &finalResult); err != nil {
		return zero, &JsonsError{
			Op:      "get_typed",
			Path:    path,
			Message: fmt.Sprintf("failed to convert value to type %T: %v", finalResult, err),
			Err:     ErrTypeMismatch,
		}
	}

	return finalResult, nil
}

// convertValueToType converts a pre-parsed value to the target type T.
// Used by GetTypedOr to avoid re-parsing when the raw value is already available.
func convertValueToType[T any](value any, path string) (T, error) {
	var zero T

	if value == nil {
		return zero, ErrPathNotFound
	}

	if converted, ok := UnifiedTypeConversion[T](value); ok {
		return converted, nil
	}

	// Fallback: re-marshal and unmarshal for complex types
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return zero, &JsonsError{
			Op:      "get_typed",
			Path:    path,
			Message: fmt.Sprintf("failed to marshal value for type conversion: %v", err),
			Err:     ErrTypeMismatch,
		}
	}

	var finalResult T
	if err := json.Unmarshal(jsonBytes, &finalResult); err != nil {
		return zero, &JsonsError{
			Op:      "get_typed",
			Path:    path,
			Message: fmt.Sprintf("failed to convert value to type %T: %v", finalResult, err),
			Err:     ErrTypeMismatch,
		}
	}

	return finalResult, nil
}

// handleNullValue handles null values for different target types using direct type checking
func handleNullValue[T any]() (T, error) {
	var zero T

	// Use direct type checking instead of string reflection for better performance
	switch any(zero).(type) {
	case string:
		// Return empty string for null values
		if result, ok := any("").(T); ok {
			return result, nil
		}
	case *string:
		if result, ok := any((*string)(nil)).(T); ok {
			return result, nil
		}
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, bool:
		return zero, nil
	}

	// For all other types (including unhandled cases), return zero value
	// This provides consistent behavior without error for null values
	return zero, nil
}

// TypeSafeConvert attempts to convert a value to the target type safely
func TypeSafeConvert[T any](value any) (T, error) {
	var zero T

	if result, ok := value.(T); ok {
		return result, nil
	}

	targetType := fmt.Sprintf("%T", zero)
	return convertWithTypeInfo[T](value, targetType)
}

// convertWithTypeInfo handles type conversion with type information
func convertWithTypeInfo[T any](value any, targetType string) (T, error) {
	var zero T

	convResult, handled := handleLargeNumberConversion[T](value)
	if handled {
		return convResult.value, convResult.err
	}

	if str, ok := value.(string); ok {
		return convertStringToType[T](str)
	}

	return zero, fmt.Errorf("cannot convert %T to %s", value, targetType)
}

// convertStringToType converts string values to target types safely
func convertStringToType[T any](str string) (T, error) {
	var zero T

	switch any(zero).(type) {
	case int:
		if val, err := strconv.ParseInt(str, 10, 64); err == nil {
			if result, ok := any(int(val)).(T); ok {
				return result, nil
			}
		}
	case int64:
		if val, err := strconv.ParseInt(str, 10, 64); err == nil {
			if result, ok := any(val).(T); ok {
				return result, nil
			}
		}
	case float64:
		if val, err := strconv.ParseFloat(str, 64); err == nil {
			if result, ok := any(val).(T); ok {
				return result, nil
			}
		}
	case bool:
		if val, err := strconv.ParseBool(str); err == nil {
			if result, ok := any(val).(T); ok {
				return result, nil
			}
		}
	case string:
		if result, ok := any(str).(T); ok {
			return result, nil
		}
	}

	return zero, fmt.Errorf("cannot convert string %q to %T", str, zero)
}

// conversionResult holds the result of a type conversion attempt
type conversionResult[T any] struct {
	value T
	err   error
}

// handleLargeNumberConversion handles conversion of large numbers to specific types
func handleLargeNumberConversion[T any](value any) (conversionResult[T], bool) {
	var zero T

	switch any(zero).(type) {
	case int64:
		if converted, err := SafeConvertToInt64(value); err == nil {
			if typedResult, ok := any(converted).(T); ok {
				return conversionResult[T]{value: typedResult, err: nil}, true
			}
		} else {
			return conversionResult[T]{
				value: zero,
				err: &JsonsError{
					Op:      "get_typed",
					Path:    "type_conversion",
					Message: fmt.Sprintf("large number conversion failed: %v", err),
					Err:     ErrTypeMismatch,
				},
			}, true
		}

	case uint64:
		if converted, err := SafeConvertToUint64(value); err == nil {
			if typedResult, ok := any(converted).(T); ok {
				return conversionResult[T]{value: typedResult, err: nil}, true
			}
		} else {
			return conversionResult[T]{
				value: zero,
				err: &JsonsError{
					Op:      "get_typed",
					Path:    "type_conversion",
					Message: fmt.Sprintf("large number conversion failed: %v", err),
					Err:     ErrTypeMismatch,
				},
			}, true
		}

	case string:
		if strResult, ok := any(FormatNumber(value)).(T); ok {
			return conversionResult[T]{value: strResult, err: nil}, true
		}
	}

	return conversionResult[T]{value: zero, err: nil}, false
}

// IteratorControl represents control flags for iteration
type IteratorControl int

const (
	IteratorNormal IteratorControl = iota
	IteratorContinue
	IteratorBreak
)

// ============================================================================
// JSON KEY INTERNING
// Delegates to internal.KeyIntern (64-shard with hot cache) for concurrent performance.
// ============================================================================

// InternKey interns a string key for memory efficiency.
// Returns an interned version of the key that can be reused across operations.
//
// Example:
//
//	key := json.InternKey("user_id") // Returns interned string
func InternKey(key string) string {
	return internal.GlobalKeyIntern.Intern(key)
}

// ClearKeyInternCache clears the global key interning cache.
func ClearKeyInternCache() {
	internal.GlobalKeyIntern.Clear()
}

// GetKeyInternCacheSize returns the number of interned keys in the cache.
func GetKeyInternCacheSize() int {
	return internal.GlobalKeyIntern.Size()
}

// ============================================================================
// VALUE UTILITIES
// ============================================================================

// IsEmptyOrZero checks if a value is empty or its zero value.
// Supports all standard numeric types, bool, string, slices, maps, and json.Number.
// For slices and maps, returns true if nil or empty (len == 0).
//
// Example:
//
//	if json.IsEmptyOrZero(value) {
//	    // Handle empty or zero value
//	}
func IsEmptyOrZero(v any) bool {
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
	case Number:
		n, err := val.Int64()
		return err == nil && n == 0
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	case map[any]any:
		return len(val) == 0
	default:
		return false
	}
}
