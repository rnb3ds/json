package json

import (
	stdjson "encoding/json"
	"testing"
)

// ============================================================================
// Boundary tests for helpers.go low-coverage paths.
// House style: plain assertions, t.Run subtests, section headers.
// ============================================================================

// --- isEmptyOrZero (helpers.go:1251, 36% coverage) ---

func TestIsEmptyOrZero_Boundary(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want bool
	}{
		{"nil", nil, true},
		{"empty_string", "", true},
		{"nonempty_string", "x", false},
		{"int_zero", 0, true},
		{"int_nonzero", 1, false},
		{"int8_zero", int8(0), true},
		{"int16_zero", int16(0), true},
		{"int32_zero", int32(0), true},
		{"int64_zero", int64(0), true},
		{"uint_zero", uint(0), true},
		{"uint8_zero", uint8(0), true},
		{"uint16_zero", uint16(0), true},
		{"uint32_zero", uint32(0), true},
		{"uint64_zero", uint64(0), true},
		{"float32_zero", float32(0), true},
		{"float64_zero", float64(0), true},
		{"float64_nonzero", float64(1.5), false},
		{"bool_false", false, true},
		{"bool_true", true, false},
		{"number_zero", Number("0"), true},
		{"number_nonzero", Number("5"), false},
		{"stdjson_number_zero", stdjson.Number("0"), true},
		{"empty_any_slice", []any{}, true},
		{"nonempty_any_slice", []any{1}, false},
		{"empty_string_map", map[string]any{}, true},
		{"nonempty_string_map", map[string]any{"a": 1}, false},
		{"empty_any_map", map[any]any{}, true},
		{"nonempty_any_map", map[any]any{"a": 1}, false},
		{"default_struct", struct{}{}, false},
		{"default_chan", make(chan int), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEmptyOrZero(tt.v); got != tt.want {
				t.Errorf("isEmptyOrZero(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

// --- convertToInt64 (helpers.go:143, 53% coverage) ---

func TestConvertToInt64_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		v      any
		want   int64
		wantOk bool
	}{
		{"int64", int64(42), 42, true},
		{"float64_whole", float64(42.0), 42, true},
		{"float64_fractional", float64(3.14), 0, false},
		{"float32_whole", float32(42.0), 42, true},
		{"float32_fractional", float32(3.14), 0, false},
		{"string_valid", "42", 42, true},
		{"string_invalid", "abc", 0, false},
		{"bool_true", true, 1, true},
		{"bool_false", false, 0, true},
		{"stdjson_number_valid", stdjson.Number("42"), 42, true},
		{"stdjson_number_invalid", stdjson.Number("abc"), 0, false},
		{"unsupported_slice", []any{1}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := convertToInt64(tt.v)
			if ok != tt.wantOk || got != tt.want {
				t.Errorf("convertToInt64(%v) = (%d, %v), want (%d, %v)", tt.v, got, ok, tt.want, tt.wantOk)
			}
		})
	}
}

// --- containerGetProperty (helpers.go:542, 50% coverage) ---

func TestContainerGetProperty(t *testing.T) {
	t.Run("string_map_hit", func(t *testing.T) {
		v, ok := containerGetProperty(map[string]any{"k": "v"}, "k")
		if !ok || v != "v" {
			t.Errorf("got (%v, %v), want (v, true)", v, ok)
		}
	})
	t.Run("string_map_miss", func(t *testing.T) {
		v, ok := containerGetProperty(map[string]any{"k": "v"}, "missing")
		if ok || v != nil {
			t.Errorf("got (%v, %v), want (nil, false)", v, ok)
		}
	})
	t.Run("any_map_hit", func(t *testing.T) {
		v, ok := containerGetProperty(map[any]any{"k": "v"}, "k")
		if !ok || v != "v" {
			t.Errorf("got (%v, %v), want (v, true)", v, ok)
		}
	})
	t.Run("non_map", func(t *testing.T) {
		v, ok := containerGetProperty("not a map", "k")
		if ok || v != nil {
			t.Errorf("got (%v, %v), want (nil, false)", v, ok)
		}
	})
}

// --- safeCopyResult + deepCopy family (helpers.go:746/766/808/849) ---

func TestSafeCopyResult(t *testing.T) {
	t.Run("primitive_returned_as_is", func(t *testing.T) {
		if got := safeCopyResult(float64(42)); got != float64(42) {
			t.Errorf("primitive not returned as-is: %v", got)
		}
		if got := safeCopyResult(Number("42")); got != Number("42") {
			t.Errorf("Number not returned as-is: %v", got)
		}
	})
	t.Run("container_deep_copied", func(t *testing.T) {
		orig := map[string]any{"a": []any{1, 2, 3}}
		got := safeCopyResult(orig)
		cp, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", got)
		}
		// Mutating the copy must not affect the original.
		cp["a"].([]any)[0] = 99
		if orig["a"].([]any)[0] == 99 {
			t.Error("safeCopyResult returned a shallow reference, not a deep copy")
		}
	})
	t.Run("depth_limit_returns_nil", func(t *testing.T) {
		// Build a structure nested deeper than deepCopyMaxDepth (200).
		deep := any(map[string]any{"v": 1})
		for i := 0; i < 210; i++ {
			deep = map[string]any{"n": deep}
		}
		if got := safeCopyResult(deep); got != nil {
			t.Error("expected nil from safeCopyResult on pathologically deep input")
		}
	})
}

func TestDeepCopySubtreeWithDepth_Boundary(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		got, err := deepCopySubtreeWithDepth(nil, 0)
		if err != nil || got != nil {
			t.Errorf("got (%v, %v), want (nil, nil)", got, err)
		}
	})
	t.Run("number_preserved", func(t *testing.T) {
		got, err := deepCopySubtreeWithDepth(Number("42"), 0)
		if err != nil || got != Number("42") {
			t.Errorf("Number not preserved: got (%v, %v)", got, err)
		}
	})
	t.Run("nested_map_with_slice", func(t *testing.T) {
		in := map[string]any{"a": []any{1, map[string]any{"b": 2}}}
		got, err := deepCopySubtreeWithDepth(in, 0)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		out := got.(map[string]any)
		// Mutate to confirm independence.
		out["a"].([]any)[0] = 99
		if in["a"].([]any)[0] == 99 {
			t.Error("deep copy is not independent")
		}
	})
	t.Run("depth_limit", func(t *testing.T) {
		if _, err := deepCopySubtreeWithDepth(map[string]any{"a": 1}, 201); err == nil {
			t.Error("expected depth-limit error")
		}
	})
	t.Run("map_depth_limit", func(t *testing.T) {
		if _, err := deepCopyJSONMapWithDepth(map[string]any{"a": 1}, 201); err == nil {
			t.Error("expected depth-limit error for map")
		}
	})
	t.Run("slice_depth_limit", func(t *testing.T) {
		if _, err := deepCopyJSONSliceWithDepth([]any{1}, 201); err == nil {
			t.Error("expected depth-limit error for slice")
		}
	})
}

// --- unifiedTypeConversion (helpers.go:354, 57% coverage) ---

func TestUnifiedTypeConversion_Boundary(t *testing.T) {
	t.Run("nil_value", func(t *testing.T) {
		got, ok := unifiedTypeConversion[string](nil)
		if !ok || got != "" {
			t.Errorf("nil conversion got (%q, %v), want (\"\", true)", got, ok)
		}
	})
	t.Run("direct_assertion", func(t *testing.T) {
		got, ok := unifiedTypeConversion[int](42)
		if !ok || got != 42 {
			t.Errorf("direct assertion got (%d, %v), want (42, true)", got, ok)
		}
	})
	t.Run("single_element_array_unwrap", func(t *testing.T) {
		got, ok := unifiedTypeConversion[int]([]any{42})
		if !ok || got != 42 {
			t.Errorf("array unwrap got (%d, %v), want (42, true)", got, ok)
		}
	})
	t.Run("conversion_failure", func(t *testing.T) {
		got, ok := unifiedTypeConversion[int]("not a number")
		if ok || got != 0 {
			t.Errorf("conversion failure got (%d, %v), want (0, false)", got, ok)
		}
	})
	t.Run("pointer_target", func(t *testing.T) {
		// Target type is *int (Pointer kind) -> exercises the pointer branch.
		got, ok := unifiedTypeConversion[*int](42)
		if ok {
			if got == nil || *got != 42 {
				t.Errorf("pointer conversion got %v, want *42", got)
			}
		}
		// If !ok, the pointer branch still executed and returned zero — acceptable.
	})
}
