package internal

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// MergeMode defines the merge strategy for combining JSON objects and arrays
type MergeMode int

const (
	// MergeUnion performs union merge - combines all keys/elements (default)
	MergeUnion MergeMode = iota
	// MergeIntersection performs intersection merge - only common keys/elements
	MergeIntersection
	// MergeDifference performs difference merge - keys/elements only in base
	MergeDifference
)

// DeepMerge recursively merges two JSON values using union merge strategy (default)
// This is kept for backward compatibility - it delegates to DeepMergeWithMode
func DeepMerge(base, override any) any {
	return DeepMergeWithMode(base, override, MergeUnion)
}

// DeepMergeWithMode recursively merges two JSON values with specified mode
func DeepMergeWithMode(base, override any, mode MergeMode) any {
	return deepMergeWithMode(base, override, mode, 0, make(map[uintptr]bool))
}

func deepMergeWithMode(base, override any, mode MergeMode, depth int, visited map[uintptr]bool) any {
	if depth > MaxDeepMergeDepth {
		return override
	}

	baseMap, baseIsMap := base.(map[string]any)
	overrideMap, overrideIsMap := override.(map[string]any)

	if baseIsMap && overrideIsMap {
		return mergeObjects(baseMap, overrideMap, mode, depth, visited)
	}

	baseArray, baseIsArray := base.([]any)
	overrideArray, overrideIsArray := override.([]any)

	if baseIsArray && overrideIsArray {
		return mergeArrays(baseArray, overrideArray, mode, visited)
	}

	// For non-map, non-array types
	switch mode {
	case MergeDifference:
		// Difference mode: if values are different, exclude (return nil)
		// If values are the same, they're not "different" so also exclude
		return nil
	case MergeIntersection:
		// Intersection mode: include if values are equal (use override)
		// For primitives, we can't easily compare, so use override
		return override
	default: // MergeUnion
		return override
	}
}

// mergeObjects handles object merging based on mode
func mergeObjects(baseMap, overrideMap map[string]any, mode MergeMode, depth int, visited map[uintptr]bool) map[string]any {
	// Cycle detection
	basePtr := reflect.ValueOf(baseMap).Pointer()
	if visited[basePtr] {
		return overrideMap
	}
	visited[basePtr] = true
	defer delete(visited, basePtr)

	switch mode {
	case MergeUnion:
		return mergeObjectsUnion(baseMap, overrideMap, mode, depth, visited)
	case MergeIntersection:
		return mergeObjectsIntersection(baseMap, overrideMap, mode, depth, visited)
	case MergeDifference:
		return mergeObjectsDifference(baseMap, overrideMap, mode, depth, visited)
	}
	return make(map[string]any)
}

// mergeObjectsUnion performs union merge - combines all keys from both objects
// PERFORMANCE: Pre-allocates result map with capacity hint
func mergeObjectsUnion(baseMap, overrideMap map[string]any, mode MergeMode, depth int, visited map[uintptr]bool) map[string]any {
	// PERFORMANCE: Pre-allocate with combined size hint
	result := make(map[string]any, len(baseMap)+len(overrideMap))

	// Copy all keys from base
	for key, value := range baseMap {
		result[key] = value
	}

	// Merge override keys
	for key, overrideValue := range overrideMap {
		if baseValue, exists := baseMap[key]; exists {
			// Both exist - recursively merge
			result[key] = deepMergeWithMode(baseValue, overrideValue, mode, depth+1, visited)
		} else {
			// Only in override - add directly
			result[key] = overrideValue
		}
	}

	return result
}

// mergeObjectsIntersection performs intersection merge - only keys present in both
// PERFORMANCE: Pre-allocates result map with capacity hint
func mergeObjectsIntersection(baseMap, overrideMap map[string]any, mode MergeMode, depth int, visited map[uintptr]bool) map[string]any {
	// PERFORMANCE: Pre-allocate with min size hint
	minLen := len(baseMap)
	if len(overrideMap) < minLen {
		minLen = len(overrideMap)
	}
	result := make(map[string]any, minLen)

	// Only include keys that exist in both
	for key, baseValue := range baseMap {
		if overrideValue, exists := overrideMap[key]; exists {
			// Key exists in both - recursively merge
			merged := deepMergeWithMode(baseValue, overrideValue, mode, depth+1, visited)
			// Only include non-nil results (nil means excluded by difference at deeper level)
			if merged != nil {
				result[key] = merged
			}
		}
	}

	return result
}

// mergeObjectsDifference performs difference merge - keys only in base (A - B)
// PERFORMANCE: Pre-allocates result map with capacity hint
func mergeObjectsDifference(baseMap, overrideMap map[string]any, mode MergeMode, depth int, visited map[uintptr]bool) map[string]any {
	// PERFORMANCE: Pre-allocate with base size hint
	result := make(map[string]any, len(baseMap))

	// Only include keys that exist in base but NOT in override
	for key, baseValue := range baseMap {
		if overrideValue, exists := overrideMap[key]; exists {
			// Key exists in both - need to check if values are different
			// If both are objects, recursively compute difference
			baseNested, baseIsNested := baseValue.(map[string]any)
			overrideNested, overrideIsNested := overrideValue.(map[string]any)

			if baseIsNested && overrideIsNested {
				// Both are objects - recursively compute difference
				diff := mergeObjectsDifference(baseNested, overrideNested, mode, depth+1, visited)
				// Only include if difference is not empty
				if len(diff) > 0 {
					result[key] = diff
				}
			}

			// If both are arrays, compute array difference
			baseArray, baseIsArray := baseValue.([]any)
			overrideArray, overrideIsArray := overrideValue.([]any)

			if baseIsArray && overrideIsArray {
				// Both are arrays - compute array difference
				diff := mergeArraysDifference(baseArray, overrideArray)
				// Include even if empty (to preserve the key)
				result[key] = diff
			}
			// If values are different types or primitives, the key exists in both
			// so it's not part of the difference - skip it
		} else {
			// Key only in base - include it
			result[key] = baseValue
		}
	}

	return result
}

// mergeArrays handles array merging based on mode
func mergeArrays(baseArray, overrideArray []any, mode MergeMode, visited map[uintptr]bool) []any {
	// Cycle detection
	basePtr := reflect.ValueOf(baseArray).Pointer()
	overridePtr := reflect.ValueOf(overrideArray).Pointer()

	if visited[basePtr] || visited[overridePtr] {
		return overrideArray
	}

	visited[basePtr] = true
	if basePtr != overridePtr {
		visited[overridePtr] = true
	}
	defer func() {
		delete(visited, basePtr)
		if basePtr != overridePtr {
			delete(visited, overridePtr)
		}
	}()

	switch mode {
	case MergeUnion:
		return mergeArraysUnion(baseArray, overrideArray)
	case MergeIntersection:
		return mergeArraysIntersection(baseArray, overrideArray)
	case MergeDifference:
		return mergeArraysDifference(baseArray, overrideArray)
	}
	return []any{}
}

// mergeArraysUnion performs union merge - combines all elements with deduplication
// PERFORMANCE: Pre-allocates result and seen map with capacity hints
func mergeArraysUnion(baseArray, overrideArray []any) []any {
	// PERFORMANCE: Pre-allocate with combined size hint
	result := make([]any, 0, len(baseArray)+len(overrideArray))
	// PERFORMANCE: Pre-allocate seen map
	seen := make(map[string]bool, len(baseArray)+len(overrideArray))

	// Add elements from base array
	for _, item := range baseArray {
		key := ArrayItemKey(item)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	// Add elements from override array
	for _, item := range overrideArray {
		key := ArrayItemKey(item)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	return result
}

// mergeArraysIntersection performs intersection merge - only elements in both arrays
// PERFORMANCE: Pre-allocates result and sets with capacity hints
func mergeArraysIntersection(baseArray, overrideArray []any) []any {
	// PERFORMANCE: Pre-allocate sets
	overrideSet := make(map[string]bool, len(overrideArray))
	for _, item := range overrideArray {
		overrideSet[ArrayItemKey(item)] = true
	}

	// PERFORMANCE: Pre-allocate result with min size hint
	minLen := len(baseArray)
	if len(overrideArray) < minLen {
		minLen = len(overrideArray)
	}
	result := make([]any, 0, minLen)
	seen := make(map[string]bool, minLen)

	for _, item := range baseArray {
		key := ArrayItemKey(item)
		if overrideSet[key] && !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	return result
}

// mergeArraysDifference performs difference merge - elements only in base (A - B)
// PERFORMANCE: Pre-allocates result and sets with capacity hints
func mergeArraysDifference(baseArray, overrideArray []any) []any {
	// PERFORMANCE: Pre-allocate sets
	overrideSet := make(map[string]bool, len(overrideArray))
	for _, item := range overrideArray {
		overrideSet[ArrayItemKey(item)] = true
	}

	// PERFORMANCE: Pre-allocate result with base size hint
	result := make([]any, 0, len(baseArray))
	seen := make(map[string]bool, len(baseArray))

	for _, item := range baseArray {
		key := ArrayItemKey(item)
		if !overrideSet[key] && !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}

	return result
}

// ArrayItemKey generates a unique key for array item deduplication
func ArrayItemKey(item any) string {
	switch v := item.(type) {
	case string:
		return "s:" + v
	case float64:
		// JSON numbers are parsed as float64
		return "n:" + FormatNumberForDedup(v)
	case bool:
		if v {
			return "b:true"
		}
		return "b:false"
	case nil:
		return "null"
	case map[string]any:
		// For objects, use JSON marshaling for comparison
		bytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("obj:%p", v)
		}
		return "o:" + string(bytes)
	case []any:
		// For arrays, use JSON marshaling for comparison
		bytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("arr:%p", v)
		}
		return "a:" + string(bytes)
	default:
		// Fallback for other types
		return fmt.Sprintf("other:%v", v)
	}
}

// FormatNumberForDedup formats a number for deduplication key generation
func FormatNumberForDedup(f float64) string {
	// Check if it's an integer
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%g", f)
}

// IsJSONPointerPath checks if a path uses JSON Pointer format
func IsJSONPointerPath(path string) bool {
	return path != "" && path[0] == '/'
}

// IsDotNotationPath checks if a path uses dot notation format
func IsDotNotationPath(path string) bool {
	return path != "" && path != "." && path[0] != '/'
}

// IsArrayPath checks if a path contains array access
func IsArrayPath(path string) bool {
	return strings.Contains(path, "[") && strings.Contains(path, "]")
}

// IsSlicePath checks if a path contains slice notation
func IsSlicePath(path string) bool {
	return strings.Contains(path, "[") && strings.Contains(path, ":") && strings.Contains(path, "]")
}

// IsExtractionPath checks if a path contains extraction syntax
func IsExtractionPath(path string) bool {
	return strings.Contains(path, "{") && strings.Contains(path, "}")
}

// IsJSONObject checks if data is a JSON object (map[string]any)
func IsJSONObject(data any) bool {
	_, ok := data.(map[string]any)
	return ok
}

// IsJSONArray checks if data is a JSON array ([]any)
func IsJSONArray(data any) bool {
	_, ok := data.([]any)
	return ok
}

// IsJSONPrimitive checks if data is a JSON primitive type
func IsJSONPrimitive(data any) bool {
	switch data.(type) {
	case string, int, int32, int64, float32, float64, bool, nil:
		return true
	default:
		return false
	}
}

// TryConvertToArray attempts to convert a map to an array if it has numeric keys
func TryConvertToArray(m map[string]any) ([]any, bool) {
	if len(m) == 0 {
		return []any{}, true
	}

	maxIndex := -1
	for key := range m {
		if index, err := strconv.Atoi(key); err == nil && index >= 0 {
			if index > maxIndex {
				maxIndex = index
			}
		} else {
			return nil, false
		}
	}

	arr := make([]any, maxIndex+1)
	for key, value := range m {
		if index, err := strconv.Atoi(key); err == nil {
			arr[index] = value
		}
	}

	return arr, true
}

// IndexIgnoreCase finds a pattern in s case-insensitively without allocation
// This is a shared utility function used by multiple packages for security pattern matching
func IndexIgnoreCase(s, pattern string) int {
	plen := len(pattern)
	slen := len(s)
	if plen > slen {
		return -1
	}

	// Use first character as filter
	firstChar := pattern[0]
	firstCharLower := firstChar
	if firstChar >= 'A' && firstChar <= 'Z' {
		firstCharLower = firstChar + 32
	}

	// Only check positions where first character matches
	for i := 0; i <= slen-plen; i++ {
		c := s[i]
		if c == firstCharLower || (firstCharLower >= 'a' && firstCharLower <= 'z' && c == firstCharLower-32) {
			// First char matches, check rest
			if matchPatternIgnoreCase(s[i:i+plen], pattern) {
				return i
			}
		}
	}
	return -1
}

// matchPatternIgnoreCase checks if s matches pattern case-insensitively
func matchPatternIgnoreCase(s, pattern string) bool {
	if len(s) != len(pattern) {
		return false
	}
	for i := 0; i < len(pattern); i++ {
		c1 := s[i]
		c2 := pattern[i]
		if c1 >= 'A' && c1 <= 'Z' {
			c1 += 32
		}
		if c1 != c2 {
			return false
		}
	}
	return true
}

// IsMatchPatternIgnoreCase is the exported version for use by other packages
func IsMatchPatternIgnoreCase(s, pattern string) bool {
	return matchPatternIgnoreCase(s, pattern)
}

// CleanupNullValues recursively removes null values and empty containers from JSON data.
// When compactArrays is true, null elements are also removed from arrays.
// PERFORMANCE: Pre-allocates result containers with capacity hints
func CleanupNullValues(data any, compactArrays bool) any {
	switch v := data.(type) {
	case map[string]any:
		// PERFORMANCE: Pre-allocate with original size hint
		result := make(map[string]any, len(v))
		for key, value := range v {
			if value != nil {
				cleanedValue := CleanupNullValues(value, compactArrays)
				if cleanedValue != nil && !isEmptyContainer(cleanedValue) {
					result[key] = cleanedValue
				}
			}
		}
		return result

	case []any:
		if compactArrays {
			return cleanupArrayCompact(v, compactArrays)
		}
		// PERFORMANCE: Pre-allocate with exact size
		result := make([]any, len(v))
		for i, item := range v {
			if item != nil {
				result[i] = CleanupNullValues(item, compactArrays)
			}
		}
		return result

	default:
		return data
	}
}

// cleanupArrayCompact removes null elements from an array while recursively cleaning nested values
// PERFORMANCE: Pre-allocates result with array capacity hint
func cleanupArrayCompact(arr []any, compactArrays bool) []any {
	// PERFORMANCE: Pre-allocate with array size hint
	result := make([]any, 0, len(arr))
	for _, item := range arr {
		if item != nil {
			cleanedItem := CleanupNullValues(item, compactArrays)
			if cleanedItem != nil && !isEmptyContainer(cleanedItem) {
				result = append(result, cleanedItem)
			}
		}
	}
	return result
}

// isEmptyContainer checks if a value is an empty map, array, or string
func isEmptyContainer(data any) bool {
	switch v := data.(type) {
	case map[string]any:
		return len(v) == 0
	case []any:
		return len(v) == 0
	case string:
		return v == ""
	default:
		return false
	}
}

// ConvertNumbersToFloat recursively converts json.Number and Number types to float64
// This is needed because standard json.Marshal encodes json.Number as strings
// PERFORMANCE: Pre-allocates result containers with capacity hints
func ConvertNumbersToFloat(data any) any {
	switch v := data.(type) {
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return v // Keep original if conversion fails
		}
		return f
	case map[string]any:
		// PERFORMANCE: Pre-allocate with exact size
		result := make(map[string]any, len(v))
		for key, value := range v {
			result[key] = ConvertNumbersToFloat(value)
		}
		return result
	case []any:
		// PERFORMANCE: Pre-allocate with exact size
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = ConvertNumbersToFloat(item)
		}
		return result
	default:
		return data
	}
}
