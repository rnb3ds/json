package json

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

// TestIsEmptyOrZeroExtended provides additional edge-case coverage for isEmptyOrZero
// beyond the existing TestIsEmptyOrZero in core_test.go.
func TestIsEmptyOrZeroExtended(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		// Integer types not covered by existing tests
		{"zero int8", int8(0), true},
		{"non-zero int8", int8(1), false},
		{"zero int16", int16(0), true},
		{"non-zero int16", int16(100), false},
		{"zero int32", int32(0), true},
		{"non-zero int32", int32(999), false},
		{"zero uint", uint(0), true},
		{"non-zero uint", uint(42), false},
		{"zero uint8", uint8(0), true},
		{"non-zero uint8", uint8(255), false},
		{"zero uint16", uint16(0), true},
		{"non-zero uint16", uint16(1000), false},
		{"zero uint32", uint32(0), true},
		{"non-zero uint32", uint32(5000), false},
		{"zero uint64", uint64(0), true},
		{"non-zero uint64", uint64(999999), false},
		{"zero float32", float32(0), true},
		{"non-zero float32", float32(1.5), false},

		// Number type
		{"Number zero", Number("0"), true},
		{"Number non-zero", Number("42"), false},
		{"Number non-parseable", Number("abc"), false},

		// nil slice and map
		{"nil slice", ([]any)(nil), true},
		{"nil map", (map[string]any)(nil), true},
		{"nil any-key map", (map[any]any)(nil), true},
		{"empty any-key map", map[any]any{}, true},
		{"non-empty any-key map", map[any]any{"k": 1}, false},

		// Unsupported types should return false
		{"struct type", struct{ X int }{X: 0}, false},
		{"chan type", make(chan int), false},
		{"func type", func() {}, false},
		{"uintptr", uintptr(0), false},
		{"complex64", complex64(0), false},
		{"complex128", complex128(0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmptyOrZero(tt.input)
			if result != tt.expected {
				t.Errorf("isEmptyOrZero(%v) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestConvertToStringExtended provides additional edge-case coverage for convertToString.
func TestConvertToStringExtended(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"nil value", nil, "<nil>"},
		{"string empty", "", ""},
		{"string non-empty", "test", "test"},
		{"byte slice empty", []byte{}, ""},
		{"byte slice non-empty", []byte("data"), "data"},
		{"json.Number integer", json.Number("42"), "42"},
		{"json.Number float", json.Number("3.14"), "3.14"},
		{"int zero", 0, "0"},
		{"int negative", -99, "-99"},
		{"bool false", false, "false"},
		{"bool true", true, "true"},
		{"float64", 2.5, "2.5"},
		{"slice", []any{1, 2}, "[1 2]"},
		{"map", map[string]any{"a": 1}, "map[a:1]"},
		{"stringer", myStringer{}, "stringer-value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToString(tt.input)
			if result != tt.expected {
				t.Errorf("convertToString(%v) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// myStringer is a test helper implementing fmt.Stringer.
type myStringer struct{}

func (myStringer) String() string { return "stringer-value" }

// TestFormatNumberExtended provides additional edge-case coverage for formatNumber.
func TestFormatNumberExtended(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"int zero", 0, "0"},
		{"int negative", -1, "-1"},
		{"int64 max", int64(math.MaxInt64), "9223372036854775807"},
		{"int64 min", int64(math.MinInt64), "-9223372036854775808"},
		{"uint64 zero", uint64(0), "0"},
		{"float64 integer", 42.0, "42"},
		{"float64 negative", -3.5, "-3.5"},
		{"json.Number integer", json.Number("100"), "100"},
		{"json.Number scientific", json.Number("1e10"), "1e10"},
		{"string fallback", "hello", "hello"},
		{"bool fallback", true, "true"},
		{"nil fallback", nil, "<nil>"},
		{"struct fallback", struct{}{}, "{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("formatNumber(%v) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSafeConvertToInt64Extended provides extended coverage for safeConvertToInt64
// beyond the basic TestSafeConvertToInt64 in types_test.go.
func TestSafeConvertToInt64Extended(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expected    int64
		expectError bool
	}{
		// Boundary values
		{"int64 max", int64(math.MaxInt64), math.MaxInt64, false},
		{"int64 min", int64(math.MinInt64), math.MinInt64, false},
		{"int zero", 0, 0, false},
		{"int negative", -42, -42, false},

		// All integer types
		{"int8", int8(8), 8, false},
		{"int16", int16(16), 16, false},
		{"int32", int32(32), 32, false},
		{"uint8", uint8(8), 8, false},
		{"uint16", uint16(16), 16, false},
		{"uint32", uint32(32), 32, false},
		{"uint64 within range", uint64(math.MaxInt64), math.MaxInt64, false},
		{"uint64 overflow", uint64(math.MaxInt64) + 1, 0, true},

		// Float types
		{"float64 exact", float64(42.0), 42, false},
		{"float64 fractional", float64(3.14), 0, true},
		{"float32 exact", float32(7.0), 7, false},
		{"float32 fractional", float32(2.5), 0, true},

		// String parsing
		{"string valid", "123", 123, false},
		{"string negative", "-456", -456, false},
		{"string max int64", "9223372036854775807", math.MaxInt64, false},
		{"string overflow", "9223372036854775808", 0, true},
		{"string invalid", "abc", 0, true},
		{"string empty", "", 0, true},
		{"string float", "3.14", 0, true},

		// Bool
		{"bool true", true, 1, false},
		{"bool false", false, 0, false},

		// json.Number
		{"json.Number valid", json.Number("999"), 999, false},
		{"json.Number float", json.Number("3.14"), 0, true},
		{"json.Number invalid", json.Number("notnum"), 0, true},

		// Unsupported types
		{"nil", nil, 0, true},
		{"slice", []any{1}, 0, true},
		{"map", map[string]any{"a": 1}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := safeConvertToInt64(tt.input)
			if tt.expectError && err == nil {
				t.Errorf("safeConvertToInt64(%v): expected error, got nil", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("safeConvertToInt64(%v): unexpected error: %v", tt.input, err)
			}
			if !tt.expectError && result != tt.expected {
				t.Errorf("safeConvertToInt64(%v) = %d; want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSafeConvertToUint64Extended provides extended coverage for safeConvertToUint64.
func TestSafeConvertToUint64Extended(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expected    uint64
		expectError bool
	}{
		// Boundary values
		{"uint64 max", uint64(math.MaxUint64), math.MaxUint64, false},
		{"uint64 zero", uint64(0), 0, false},
		{"uint", uint(42), 42, false},

		// Signed integer types
		{"int positive", 100, 100, false},
		{"int8 positive", int8(10), 10, false},
		{"int16 positive", int16(100), 100, false},
		{"int32 positive", int32(1000), 1000, false},
		{"int64 positive", int64(10000), 10000, false},
		{"int8 negative", int8(-1), 0, true},
		{"int16 negative", int16(-1), 0, true},
		{"int32 negative", int32(-1), 0, true},
		{"int64 negative", int64(-1), 0, true},

		// Unsigned integer types
		{"uint8", uint8(255), 255, false},
		{"uint16", uint16(65535), 65535, false},
		{"uint32", uint32(100000), 100000, false},

		// Float types
		{"float64 exact positive", float64(42.0), 42, false},
		{"float64 fractional", float64(3.14), 0, true},
		{"float64 negative", float64(-1.0), 0, true},
		{"float32 exact positive", float32(7.0), 7, false},
		{"float32 fractional", float32(2.5), 0, true},
		{"float32 negative", float32(-1.0), 0, true},

		// String parsing
		{"string valid", "123", 123, false},
		{"string max uint64", "18446744073709551615", math.MaxUint64, false},
		{"string negative", "-1", 0, true},
		{"string overflow", "18446744073709551616", 0, true},
		{"string invalid", "abc", 0, true},
		{"string empty", "", 0, true},
		{"string float", "3.14", 0, true},

		// Bool
		{"bool true", true, 1, false},
		{"bool false", false, 0, false},

		// json.Number
		{"json.Number valid", json.Number("999"), 999, false},
		{"json.Number negative", json.Number("-1"), 0, true},
		{"json.Number float", json.Number("3.14"), 0, true},
		{"json.Number invalid", json.Number("notnum"), 0, true},

		// Unsupported types
		{"nil", nil, 0, true},
		{"slice", []any{1}, 0, true},
		{"map", map[string]any{"a": 1}, 0, true},
		{"struct", struct{}{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := safeConvertToUint64(tt.input)
			if tt.expectError && err == nil {
				t.Errorf("safeConvertToUint64(%v): expected error, got nil", tt.input)
			}
			if !tt.expectError && err != nil {
				t.Errorf("safeConvertToUint64(%v): unexpected error: %v", tt.input, err)
			}
			if !tt.expectError && result != tt.expected {
				t.Errorf("safeConvertToUint64(%v) = %d; want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestDeepCopyExtended provides extended coverage for deepCopy beyond TestDeepCopy
// in coverage_test.go, including channels, functions, nil, and deep nesting.
func TestDeepCopyExtended(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		result, err := deepCopy(nil)
		if err != nil {
			t.Fatalf("deepCopy(nil) error: %v", err)
		}
		if result != nil {
			t.Errorf("deepCopy(nil) = %v; want nil", result)
		}
	})

	t.Run("bool values", func(t *testing.T) {
		for _, val := range []any{true, false} {
			result, err := deepCopy(val)
			if err != nil {
				t.Fatalf("deepCopy(%v) error: %v", val, err)
			}
			if result != val {
				t.Errorf("deepCopy(%v) = %v; want %v", val, result, val)
			}
		}
	})

	t.Run("all integer types", func(t *testing.T) {
		tests := []any{
			int(42), int8(8), int16(16), int32(32), int64(64),
			uint(42), uint8(8), uint16(16), uint32(32), uint64(64),
		}
		for _, val := range tests {
			result, err := deepCopy(val)
			if err != nil {
				t.Fatalf("deepCopy(%v) error: %v", val, err)
			}
			if result != val {
				t.Errorf("deepCopy(%v) = %v; want %v", val, result, val)
			}
		}
	})

	t.Run("float types", func(t *testing.T) {
		tests := []any{float32(1.5), float64(3.14)}
		for _, val := range tests {
			result, err := deepCopy(val)
			if err != nil {
				t.Fatalf("deepCopy(%v) error: %v", val, err)
			}
			if result != val {
				t.Errorf("deepCopy(%v) = %v; want %v", val, result, val)
			}
		}
	})

	t.Run("string", func(t *testing.T) {
		result, err := deepCopy("hello")
		if err != nil {
			t.Fatalf("deepCopy error: %v", err)
		}
		if result != "hello" {
			t.Errorf("deepCopy(\"hello\") = %v; want \"hello\"", result)
		}
	})

	t.Run("json.Number", func(t *testing.T) {
		num := json.Number("42")
		result, err := deepCopy(num)
		if err != nil {
			t.Fatalf("deepCopy error: %v", err)
		}
		if result != num {
			t.Errorf("deepCopy(json.Number(\"42\")) = %v; want %v", result, num)
		}
	})

	t.Run("map[string]string", func(t *testing.T) {
		original := map[string]string{"a": "1", "b": "2"}
		result, err := deepCopy(original)
		if err != nil {
			t.Fatalf("deepCopy error: %v", err)
		}
		copied := result.(map[string]string)
		original["a"] = "modified"
		if copied["a"] != "1" {
			t.Error("deepCopy map[string]string should be independent")
		}
	})

	t.Run("[]string", func(t *testing.T) {
		original := []string{"x", "y", "z"}
		result, err := deepCopy(original)
		if err != nil {
			t.Fatalf("deepCopy error: %v", err)
		}
		copied := result.([]string)
		original[0] = "modified"
		if copied[0] != "x" {
			t.Error("deepCopy []string should be independent")
		}
	})

	t.Run("[]int", func(t *testing.T) {
		original := []int{10, 20, 30}
		result, err := deepCopy(original)
		if err != nil {
			t.Fatalf("deepCopy error: %v", err)
		}
		copied := result.([]int)
		original[0] = 999
		if copied[0] != 10 {
			t.Error("deepCopy []int should be independent")
		}
	})

	t.Run("[]float64", func(t *testing.T) {
		original := []float64{1.1, 2.2, 3.3}
		result, err := deepCopy(original)
		if err != nil {
			t.Fatalf("deepCopy error: %v", err)
		}
		copied := result.([]float64)
		original[0] = 999.9
		if copied[0] != 1.1 {
			t.Error("deepCopy []float64 should be independent")
		}
	})

	t.Run("[]bool", func(t *testing.T) {
		original := []bool{true, false, true}
		result, err := deepCopy(original)
		if err != nil {
			t.Fatalf("deepCopy error: %v", err)
		}
		copied := result.([]bool)
		original[0] = false
		if copied[0] != true {
			t.Error("deepCopy []bool should be independent")
		}
	})

	t.Run("deep nesting within limit", func(t *testing.T) {
		// Create a moderately nested structure (10 levels)
		val := any("leaf")
		for i := 0; i < 10; i++ {
			val = map[string]any{"level": val}
		}
		result, err := deepCopy(val)
		if err != nil {
			t.Fatalf("deepCopy nested map error: %v", err)
		}
		if result == nil {
			t.Error("deepCopy of nested map should not be nil")
		}
	})

	t.Run("depth limit exceeded", func(t *testing.T) {
		// Create a structure that exceeds deepCopyMaxDepth (200)
		val := any("leaf")
		for i := 0; i <= deepCopyMaxDepth+5; i++ {
			val = []any{val}
		}
		_, err := deepCopy(val)
		if err == nil {
			t.Error("deepCopy should return error when depth limit exceeded")
		}
	})

	t.Run("struct fallback via marshal", func(t *testing.T) {
		type testStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		original := testStruct{Name: "test", Value: 42}
		result, err := deepCopy(original)
		if err != nil {
			t.Fatalf("deepCopy struct error: %v", err)
		}
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("deepCopy struct result type = %T; want map[string]any", result)
		}
		if resultMap["name"] != "test" {
			t.Errorf("deepCopy struct name = %v; want test", resultMap["name"])
		}
	})

	t.Run("channel fallback via marshal", func(t *testing.T) {
		ch := make(chan int)
		_, err := deepCopy(ch)
		if err == nil {
			t.Error("deepCopy of channel should return error (channels are not JSON-serializable)")
		}
	})

	t.Run("function fallback via marshal", func(t *testing.T) {
		fn := func() {}
		_, err := deepCopy(fn)
		if err == nil {
			t.Error("deepCopy of function should return error (functions are not JSON-serializable)")
		}
	})

	t.Run("empty map[string]any", func(t *testing.T) {
		result, err := deepCopy(map[string]any{})
		if err != nil {
			t.Fatalf("deepCopy error: %v", err)
		}
		copied := result.(map[string]any)
		if len(copied) != 0 {
			t.Error("deepCopy of empty map should produce empty map")
		}
	})

	t.Run("empty []any", func(t *testing.T) {
		result, err := deepCopy([]any{})
		if err != nil {
			t.Fatalf("deepCopy error: %v", err)
		}
		copied := result.([]any)
		if len(copied) != 0 {
			t.Error("deepCopy of empty slice should produce empty slice")
		}
	})

	t.Run("deeply nested map modification independence", func(t *testing.T) {
		original := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"value": "original",
				},
			},
		}
		result, err := deepCopy(original)
		if err != nil {
			t.Fatalf("deepCopy error: %v", err)
		}
		copied := result.(map[string]any)
		original["level1"].(map[string]any)["level2"].(map[string]any)["value"] = "modified"
		deepVal := copied["level1"].(map[string]any)["level2"].(map[string]any)["value"]
		if deepVal != "original" {
			t.Errorf("deepCopy nested modification: got %q; want \"original\"", deepVal)
		}
	})
}

// TestConvertToSliceExtended tests the convertToSlice function with reflect.Type.
func TestConvertToSliceExtended(t *testing.T) {
	sliceType := reflect.TypeOf([]string{})

	t.Run("non-slice input returns false", func(t *testing.T) {
		result, ok := convertToSlice(42, sliceType)
		if ok {
			t.Error("convertToSlice(42) should return false")
		}
		if result != nil {
			t.Errorf("convertToSlice(42) result = %v; want nil", result)
		}
	})

	t.Run("non-slice map input returns false", func(t *testing.T) {
		_, ok := convertToSlice(map[string]any{"a": 1}, sliceType)
		if ok {
			t.Error("convertToSlice(map) should return false")
		}
	})

	t.Run("non-slice nil input returns false", func(t *testing.T) {
		_, ok := convertToSlice(nil, sliceType)
		if ok {
			t.Error("convertToSlice(nil) should return false")
		}
	})

	t.Run("non-slice string input returns false", func(t *testing.T) {
		_, ok := convertToSlice("hello", sliceType)
		if ok {
			t.Error("convertToSlice(string) should return false")
		}
	})

	t.Run("valid []any to []string", func(t *testing.T) {
		input := []any{"a", "b", "c"}
		result, ok := convertToSlice(input, sliceType)
		if !ok {
			t.Fatal("convertToSlice should succeed for []any to []string")
		}
		strSlice := result.([]string)
		if len(strSlice) != 3 {
			t.Errorf("len = %d; want 3", len(strSlice))
		}
		if strSlice[0] != "a" || strSlice[1] != "b" || strSlice[2] != "c" {
			t.Errorf("result = %v; want [a b c]", strSlice)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		input := []any{}
		result, ok := convertToSlice(input, sliceType)
		if !ok {
			t.Fatal("convertToSlice should succeed for empty slice")
		}
		strSlice := result.([]string)
		if len(strSlice) != 0 {
			t.Errorf("len = %d; want 0", len(strSlice))
		}
	})

	t.Run("element conversion failure", func(t *testing.T) {
		intSliceType := reflect.TypeOf([]int{})
		input := []any{"not_a_number", "also_not"}
		result, ok := convertToSlice(input, intSliceType)
		if ok {
			t.Error("convertToSlice should fail when element conversion fails")
		}
		if result != nil {
			t.Errorf("result = %v; want nil on failure", result)
		}
	})

	t.Run("[]any to []int", func(t *testing.T) {
		intSliceType := reflect.TypeOf([]int{})
		input := []any{1, 2, 3}
		result, ok := convertToSlice(input, intSliceType)
		if !ok {
			t.Fatal("convertToSlice should succeed for []any to []int")
		}
		intSlice := result.([]int)
		if len(intSlice) != 3 {
			t.Errorf("len = %d; want 3", len(intSlice))
		}
		if intSlice[0] != 1 || intSlice[1] != 2 || intSlice[2] != 3 {
			t.Errorf("result = %v; want [1 2 3]", intSlice)
		}
	})

	t.Run("array input", func(t *testing.T) {
		input := [3]string{"x", "y", "z"}
		result, ok := convertToSlice(input, sliceType)
		if !ok {
			t.Fatal("convertToSlice should succeed for array input")
		}
		strSlice := result.([]string)
		if len(strSlice) != 3 {
			t.Errorf("len = %d; want 3", len(strSlice))
		}
	})
}

// TestConvertToMapExtended tests the convertToMap function with reflect.Type.
func TestConvertToMapExtended(t *testing.T) {
	mapType := reflect.TypeOf(map[string]string{})

	t.Run("non-map input returns false", func(t *testing.T) {
		_, ok := convertToMap(42, mapType)
		if ok {
			t.Error("convertToMap(42) should return false")
		}
	})

	t.Run("non-map slice input returns false", func(t *testing.T) {
		_, ok := convertToMap([]any{1, 2}, mapType)
		if ok {
			t.Error("convertToMap(slice) should return false")
		}
	})

	t.Run("non-map string input returns false", func(t *testing.T) {
		_, ok := convertToMap("hello", mapType)
		if ok {
			t.Error("convertToMap(string) should return false")
		}
	})

	t.Run("non-map nil input returns false", func(t *testing.T) {
		_, ok := convertToMap(nil, mapType)
		if ok {
			t.Error("convertToMap(nil) should return false")
		}
	})

	t.Run("valid map[string]any to map[string]string", func(t *testing.T) {
		input := map[string]any{"a": "1", "b": "2"}
		result, ok := convertToMap(input, mapType)
		if !ok {
			t.Fatal("convertToMap should succeed for map[string]any to map[string]string")
		}
		strMap := result.(map[string]string)
		if len(strMap) != 2 {
			t.Errorf("len = %d; want 2", len(strMap))
		}
		if strMap["a"] != "1" || strMap["b"] != "2" {
			t.Errorf("result = %v; want {a:1 b:2}", strMap)
		}
	})

	t.Run("empty map", func(t *testing.T) {
		input := map[string]any{}
		result, ok := convertToMap(input, mapType)
		if !ok {
			t.Fatal("convertToMap should succeed for empty map")
		}
		strMap := result.(map[string]string)
		if len(strMap) != 0 {
			t.Errorf("len = %d; want 0", len(strMap))
		}
	})

	t.Run("value conversion failure", func(t *testing.T) {
		intMapType := reflect.TypeOf(map[string]int{})
		input := map[string]any{"a": "not_a_number"}
		_, ok := convertToMap(input, intMapType)
		if ok {
			t.Error("convertToMap should fail when value conversion fails")
		}
	})

	t.Run("key conversion failure", func(t *testing.T) {
		intKeyMapType := reflect.TypeOf(map[int]string{})
		input := map[string]any{"not_a_number": "value"}
		_, ok := convertToMap(input, intKeyMapType)
		if ok {
			t.Error("convertToMap should fail when key conversion fails")
		}
	})

	t.Run("map[string]any to map[string]int", func(t *testing.T) {
		intMapType := reflect.TypeOf(map[string]int{})
		input := map[string]any{"a": 1, "b": 2}
		result, ok := convertToMap(input, intMapType)
		if !ok {
			t.Fatal("convertToMap should succeed for map[string]any to map[string]int")
		}
		intMap := result.(map[string]int)
		if intMap["a"] != 1 || intMap["b"] != 2 {
			t.Errorf("result = %v; want {a:1 b:2}", intMap)
		}
	})
}

// TestInternKey tests the internKey function for string interning.
func TestInternKey(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"simple key", "name"},
		{"nested path", "user.profile.name"},
		{"array index", "items[0]"},
		{"special chars", "key-with.special_chars"},
		{"unicode key", "\u4e2d\u6587"},
		{"long key", "very_long_key_name_that_exceeds_typical_lengths_for_testing_purposes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := internKey(tt.input)
			if result != tt.input {
				t.Errorf("internKey(%q) = %q; want %q", tt.input, result, tt.input)
			}
		})
	}

	t.Run("deduplication returns same instance", func(t *testing.T) {
		// Calling internKey twice with the same value should return identical strings
		key := "dedup_test_key"
		first := internKey(key)
		second := internKey(key)
		if first != second {
			t.Errorf("internKey should return identical values for same input: %q vs %q", first, second)
		}
	})
}

// TestConvertToFloatCore tests the internal convertToFloatCore function.
func TestConvertToFloatCore(t *testing.T) {
	tests := []struct {
		name          string
		input         any
		expected      float64
		shouldSucceed bool
	}{
		{"float32 value", float32(1.5), float64(float32(1.5)), true},
		{"float64 value", float64(3.14), 3.14, true},
		{"float32 zero", float32(0), 0, true},
		{"float64 zero", float64(0), 0, true},
		{"float64 negative", float64(-2.5), -2.5, true},
		{"int not float", 42, 0, false},
		{"string not float", "3.14", 0, false},
		{"nil not float", nil, 0, false},
		{"bool not float", true, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := convertToFloatCore(tt.input)
			if ok != tt.shouldSucceed {
				t.Errorf("convertToFloatCore(%v) success = %v; want %v", tt.input, ok, tt.shouldSucceed)
			}
			if ok && result != tt.expected {
				t.Errorf("convertToFloatCore(%v) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestConvertToInt64Core tests the internal convertToInt64Core function.
func TestConvertToInt64Core(t *testing.T) {
	tests := []struct {
		name          string
		input         any
		expected      int64
		shouldSucceed bool
	}{
		// All integer types
		{"int", int(42), 42, true},
		{"int8", int8(8), 8, true},
		{"int16", int16(16), 16, true},
		{"int32", int32(32), 32, true},
		{"int64", int64(64), 64, true},
		{"uint", uint(42), 42, true},
		{"uint8", uint8(8), 8, true},
		{"uint16", uint16(16), 16, true},
		{"uint32", uint32(32), 32, true},
		{"uint64 in range", uint64(100), 100, true},
		{"uint64 overflow", uint64(math.MaxInt64) + 1, 0, false},
		{"uint overflow", uint(math.MaxInt64) + 1, 0, false},
		{"int negative", int(-42), -42, true},

		// Non-integer types should fail
		{"float64", float64(3.14), 0, false},
		{"string", "42", 0, false},
		{"bool", true, 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToInt64Core(tt.input)
			if result.ok != tt.shouldSucceed {
				t.Errorf("convertToInt64Core(%v) ok = %v; want %v", tt.input, result.ok, tt.shouldSucceed)
			}
			if result.ok && result.value != tt.expected {
				t.Errorf("convertToInt64Core(%v) = %d; want %d", tt.input, result.value, tt.expected)
			}
		})
	}
}

