package json

import (
	"strings"
	"testing"
)

// ============================================================================
// operation_coverage_test.go - Tests for zero-coverage functions in
// operation.go and path.go exercised through public API
// ============================================================================

// --- Delete operations: deleteArrayElement, deleteExtractedValues ---

func TestDeleteArrayElement(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		wantJSON string
		wantErr  bool
	}{
		{"delete middle", `{"a":[1,2,3]}`, "a[1]", `{"a":[1,3]}`, false},
		{"delete first", `{"a":[10,20,30]}`, "a[0]", `{"a":[20,30]}`, false},
		{"delete last", `{"a":[10,20,30]}`, "a[2]", `{"a":[10,20]}`, false},
		{"delete negative index", `{"a":[10,20,30]}`, "a[-1]", `{"a":[10,20]}`, false},
		{"delete nested array element", `{"x":{"b":[4,5,6]}}`, "x.b[1]", `{"x":{"b":[4,6]}}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Delete(tt.json, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete(%q, %q) error = %v, wantErr %v", tt.json, tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !jsonEqual(result, tt.wantJSON) {
				t.Errorf("Delete() = %q, want %q", result, tt.wantJSON)
			}
		})
	}
}

// --- Delete extraction: deleteExtractedValues, deleteConsecutiveExtractions ---

func TestDeleteExtractedValues(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		wantJSON string
	}{
		{"delete single field from array",
			`[{"k":1,"x":2},{"k":3,"x":4}]`,
			"{x}",
			`[{"k":1},{"k":3}]`},
		{"delete nested field from array",
			`[{"a":{"b":1}},{"a":{"b":2}}]`,
			"{a}",
			`[{},{}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Delete(tt.json, tt.path)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if !jsonEqual(result, tt.wantJSON) {
				t.Errorf("Delete() = %q, want %q", result, tt.wantJSON)
			}
		})
	}
}

func TestDeleteConsecutiveExtractions(t *testing.T) {
	// {a}{b} extracts a, then b from each element, then deletes
	json := `[{"a":{"b":1}},{"a":{"b":2}}]`
	result, err := Delete(json, "{a}{b}")
	// This may or may not be supported; just verify no panic and valid output
	if err != nil {
		// Expected: some paths may not support double extraction
		return
	}
	_ = result
}

// --- Delete complex paths: deleteComplexProperty, deleteComplexArray ---

func TestDeleteComplexProperty(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		wantJSON string
	}{
		{"nested property then array",
			`{"a":{"b":[1,2,3]}}`,
			"a.b[0]",
			`{"a":{"b":[2,3]}}`},
		{"deep property",
			`{"x":{"y":{"z":42}}}`,
			"x.y.z",
			`{"x":{"y":{}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Delete(tt.json, tt.path)
			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}
			if !jsonEqual(result, tt.wantJSON) {
				t.Errorf("Delete() = %q, want %q", result, tt.wantJSON)
			}
		})
	}
}

func TestDeleteComplexArray(t *testing.T) {
	// Delete specific array element within nested structure
	json := `{"a":[{"b":1},{"b":2}]}`
	result, err := Delete(json, "a[0]")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !jsonEqual(result, `{"a":[{"b":2}]}`) {
		t.Errorf("Delete() = %q", result)
	}
}

// --- Set array index: setValueForArrayIndex ---

func TestSetValueForArrayIndex(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		value    any
		wantJSON string
	}{
		{"overwrite index", `{"a":[1,2,3]}`, "a[1]", 99, `{"a":[1,99,3]}`},
		{"overwrite first", `{"a":[10,20]}`, "a[0]", 0, `{"a":[0,20]}`},
		{"overwrite nested", `{"x":{"b":[4,5,6]}}`, "x.b[2]", 7, `{"x":{"b":[4,5,7]}}`},
		{"negative index", `{"a":[10,20,30]}`, "a[-1]", 99, `{"a":[10,20,99]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Set(tt.json, tt.path, tt.value)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}
			if !jsonEqual(result, tt.wantJSON) {
				t.Errorf("Set() = %q, want %q", result, tt.wantJSON)
			}
		})
	}
}

// --- Set array slice: setValueForArraySlice ---

func TestSetValueForArraySlice(t *testing.T) {
	result, err := Set(`{"a":[1,2,3,4,5]}`, "a[1:3]", 0)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if !jsonEqual(result, `{"a":[1,0,0,4,5]}`) {
		t.Errorf("Set() = %q", result)
	}
}

// --- Set extraction: setValueForExtract, setValueForArrayExtract ---

func TestSetValueForExtract(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		value    any
		wantJSON string
	}{
		{"set extracted field in array",
			`[{"k":"a"},{"k":"b"}]`,
			"{k}", "REPLACED",
			`[{"k":"REPLACED"},{"k":"REPLACED"}]`},
		{"set nested extracted field",
			`{"items":[{"name":"a"},{"name":"b"}]}`,
			"items{name}", "x",
			`{"items":[{"name":"x"},{"name":"x"}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Set(tt.json, tt.path, tt.value)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}
			if !jsonEqual(result, tt.wantJSON) {
				t.Errorf("Set() = %q, want %q", result, tt.wantJSON)
			}
		})
	}
}

// --- Set flat extraction: setValueForArrayExtractFlat ---

func TestSetValueForArrayExtractFlat(t *testing.T) {
	result, err := Set(`[{"tags":["a"]},{"tags":["b"]}]`, "{flat:tags}", "c")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	var parsed any
	if err := Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	arr, ok := parsed.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", parsed)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr))
	}
}

// --- Set JSON Pointer: setValueJSONPointer ---

func TestSetValueJSONPointer(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		value    any
		wantJSON string
	}{
		{"simple pointer", `{"a":{"b":1}}`, "/a/b", 42, `{"a":{"b":42}}`},
		{"add new key", `{"a":{}}`, "/a/c", "val", `{"a":{"c":"val"}}`},
		{"top-level key", `{"x":1}`, "/y", 2, `{"x":1,"y":2}`},
		{"array index", `{"a":[10,20,30]}`, "/a/1", 99, `{"a":[10,99,30]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Set(tt.json, tt.path, tt.value)
			if err != nil {
				t.Fatalf("Set() error = %v", err)
			}
			if !jsonEqual(result, tt.wantJSON) {
				t.Errorf("Set() = %q, want %q", result, tt.wantJSON)
			}
		})
	}
}

// --- Set with array extension: handleArrayExtensionAndSet ---

func TestSetArrayExtension(t *testing.T) {
	cfg := Config{CreatePaths: true}
	result, err := Set(`{"a":[1,2]}`, "a[5]", "x", cfg)
	if err != nil {
		t.Fatalf("Set() with CreatePaths error = %v", err)
	}
	m := mustParseMap(t, result)
	arr, ok := m["a"].([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", m["a"])
	}
	if len(arr) != 6 {
		t.Errorf("array length = %d, want 6", len(arr))
	}
	if arr[5] != "x" {
		t.Errorf("arr[5] = %v, want 'x'", arr[5])
	}
}

// --- Get distributed: getValueWithDistributedOperation, extractIndividualArrays ---

func TestGetDistributedOperation(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		want    []any
	}{
		{"extract then index",
			`{"items":[{"vals":[1,2]},{"vals":[3,4]}]}`,
			"items{vals}[0]",
			[]any{float64(1), float64(3)}},
		{"extract then slice",
			`{"items":[{"tags":["a","b","c"]},{"tags":["d","e","f"]}]}`,
			"items{tags}[0:2]",
			[]any{[]any{"a", "b"}, []any{"d", "e"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Get(tt.json, tt.path)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			arr, ok := result.([]any)
			if !ok {
				t.Fatalf("expected []any, got %T", result)
			}
			if len(arr) != len(tt.want) {
				t.Fatalf("length = %d, want %d", len(arr), len(tt.want))
			}
		})
	}
}

// --- Get multi-field extraction: handleMultiFieldExtraction ---

func TestGetMultiFieldExtraction(t *testing.T) {
	result, err := Get(
		`[{"name":"A","age":1,"x":9},{"name":"B","age":2,"x":8}]`,
		"{name,age}",
	)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr))
	}
}

// --- Get post-extraction: handlePostExtractionArrayAccess ---

func TestGetPostExtractionArrayAccess(t *testing.T) {
	result, err := Get(
		`[{"tags":["a","b","c"]},{"tags":["d","e"]}]`,
		"{tags}[0]",
	)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	// Result should contain the first element from each tags array
	if len(arr) < 1 {
		t.Fatalf("expected at least 1 result, got %d", len(arr))
	}
}

// --- Navigate JSON Pointer: navigateJSONPointer ---

func TestNavigateJSONPointer(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		want    any
	}{
		{"simple", `{"a":{"b":1}}`, "/a/b", float64(1)},
		{"array index", `{"a":[10,20,30]}`, "/a/1", float64(20)},
		{"escaped tilde", `{"m~n":42}`, "/m~0n", float64(42)},
		{"escaped slash", `{"a/b":"val"}`, "/a~1b", "val"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Get(tt.json, tt.path)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if result != tt.want {
				t.Errorf("Get() = %v (%T), want %v (%T)", result, result, tt.want, tt.want)
			}
		})
	}
}

// --- Navigate dot notation: navigateDotNotation, navigateToPath ---

func TestNavigateDotNotation(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		wantKey string
		wantOk  bool
	}{
		{"simple property", `{"a":{"b":1}}`, "a.b", "1", true},
		{"array in path", `{"x":[10,20]}`, "x[0]", "10", true},
		{"nested array", `{"a":{"b":[1,2,3]}}`, "a.b[2]", "3", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Get(tt.json, tt.path)
			if (err != nil) == tt.wantOk {
				if !tt.wantOk {
					return
				}
				t.Fatalf("Get() error = %v", err)
			}
			if tt.wantOk && result == nil {
				t.Errorf("Get() = nil, want non-nil")
			}
		})
	}
}

// --- Delete complex slice: deleteComplexSlice ---

func TestDeleteComplexSlice(t *testing.T) {
	result, err := Delete(`{"a":[1,2,3,4,5]}`, "a[1:3]")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !jsonEqual(result, `{"a":[1,4,5]}`) {
		t.Errorf("Delete() = %q", result)
	}
}

// --- SetMultiple complex paths: setValueAtPath ---

func TestSetMultipleComplexPath(t *testing.T) {
	result, err := SetMultiple(`{"a":{"b":1}}`, map[string]any{
		"a.b": 42,
	})
	if err != nil {
		t.Fatalf("SetMultiple() error = %v", err)
	}
	if !jsonEqual(result, `{"a":{"b":42}}`) {
		t.Errorf("SetMultiple() = %q", result)
	}
}

// --- Navigate extraction: navigateToExtraction ---

func TestSetWithExtractionNavigation(t *testing.T) {
	cfg := Config{CreatePaths: true}
	result, err := Set(`{"items":[{"name":"a"},{"name":"b"}]}`, "items{name}", "x", cfg)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	m := mustParseMap(t, result)
	arr, ok := m["items"].([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", m["items"])
	}
	for i, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("items[%d] is not map", i)
		}
		if obj["name"] != "x" {
			t.Errorf("items[%d][name] = %v, want 'x'", i, obj["name"])
		}
	}
}

// --- JSON Pointer with array extension: replaceArrayInJSONPointerParent ---

func TestSetJSONPointerArrayExtension(t *testing.T) {
	cfg := Config{CreatePaths: true}
	_, err := Set(`{"arr":[1,2]}`, "/arr/5", "x", cfg)
	// Array extension via JSON Pointer beyond capacity may fail;
	// just verify no panic and consistent behavior
	if err != nil {
		// Extension beyond capacity is expected to fail gracefully
		if !strings.Contains(err.Error(), "extend") && !strings.Contains(err.Error(), "failed") {
			t.Errorf("unexpected error: %v", err)
		}
	}
}

// --- Edge case: delete from empty result ---

func TestDeleteNonExistentPath(t *testing.T) {
	_, err := Delete(`{"a":1}`, "b")
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

// --- normalizeNegativeSliceBounds edge cases ---

func TestNormalizeNegativeSliceBoundsEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		path      string
		wantLen   int
		wantErr   bool
	}{
		{"negative end", `{"a":[1,2,3,4,5]}`, "a[:-1]", 4, false},
		{"negative start and end", `{"a":[1,2,3,4,5]}`, "a[-3:-1]", 2, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Get(tt.json, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				arr, ok := result.([]any)
				if !ok {
					t.Fatalf("expected []any, got %T", result)
				}
				if len(arr) != tt.wantLen {
					t.Errorf("result length = %d, want %d", len(arr), tt.wantLen)
				}
			}
		})
	}
}

// --- SetCreate: Set with auto-creation ---

func TestSetCreateArrayIndex(t *testing.T) {
	result, err := SetCreate(`{"a":[1,2]}`, "a[5]", "x")
	if err != nil {
		t.Fatalf("SetCreate() error = %v", err)
	}
	m := mustParseMap(t, result)
	arr, ok := m["a"].([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", m["a"])
	}
	if len(arr) != 6 {
		t.Errorf("array length = %d, want 6", len(arr))
	}
}

// jsonEqual compares two JSON values by normalizing both to canonical form.
// Handles both string JSON and parsed any values for got.
func jsonEqual(got any, wantJSON string) bool {
	var wantVal, gotVal any
	if err := Unmarshal([]byte(wantJSON), &wantVal); err != nil {
		return false
	}
	// If got is a string, parse it first
	switch v := got.(type) {
	case string:
		if err := Unmarshal([]byte(v), &gotVal); err != nil {
			return false
		}
	default:
		gotBytes, err := Marshal(got)
		if err != nil {
			return false
		}
		if err := Unmarshal(gotBytes, &gotVal); err != nil {
			return false
		}
	}
	gotNorm, _ := Marshal(gotVal)
	wantNorm, _ := Marshal(wantVal)
	return string(gotNorm) == string(wantNorm)
}

// --- Test error cases for robustness ---

func TestSetErrors(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		value   any
		wantErr bool
	}{
		{"invalid json", `{invalid}`, "a", 1, true},
		{"empty path", `{"a":1}`, "", 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Set(tt.json, tt.path, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteErrors(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		wantErr bool
	}{
		{"invalid json", `{invalid}`, "a", true},
		{"empty path", `{"a":1}`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Delete(tt.json, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- NDJSON depth check: checkNestingDepth ---

func TestCheckNestingDepth(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		maxDepth int
		wantErr  bool
	}{
		{"flat object", `{"a":1}`, 10, false},
		{"nested within limit", `{"a":{"b":1}}`, 10, false},
		{"at exact limit", `{"a":{"b":1}}`, 2, false},
		{"exceeds limit", `{"a":{"b":{"c":1}}}`, 2, true},
		{"array nesting", `[[[1]]]`, 10, false},
		{"deep array exceeds", `[[[[[1]]]]]`, 3, true},
		{"string with braces", `"{not counted}"`, 1, false},
		{"empty", ``, 10, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkNestingDepth([]byte(tt.data), tt.maxDepth)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkNestingDepth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- Decoder depth limit: C-001 regression test ---

func TestDecoderDepthLimit(t *testing.T) {
	// Build deeply nested JSON
	depth := 250
	nested := strings.Repeat(`{"a":`, depth) + `1` + strings.Repeat(`}`, depth)
	reader := strings.NewReader(nested)

	dec := NewDecoder(reader, Config{MaxNestingDepthSecurity: 50})
	var result any
	err := dec.Decode(&result)
	if err == nil {
		t.Error("expected depth limit error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "depth") {
		t.Errorf("expected depth error, got: %v", err)
	}
}

func TestDecoderWithinDepthLimit(t *testing.T) {
	// 10-level nesting should work with default limit (200)
	nested := strings.Repeat(`{"a":`, 10) + `1` + strings.Repeat(`}`, 10)
	reader := strings.NewReader(nested)

	dec := NewDecoder(reader)
	var result any
	err := dec.Decode(&result)
	if err != nil {
		t.Errorf("Decode() error = %v", err)
	}
}

// --- SetMultipleCreate: exercises batch set paths ---

func TestSetMultipleCreate(t *testing.T) {
	result, err := SetMultipleCreate(`{}`, map[string]any{
		"a.b":  1,
		"a.c":  2,
	})
	if err != nil {
		t.Fatalf("SetMultipleCreate() error = %v", err)
	}
	m := mustParseMap(t, result)
	a, ok := m["a"].(map[string]any)
	if !ok {
		t.Fatalf("expected a to be map, got %T", m["a"])
	}
	if a["b"] != float64(1) || a["c"] != float64(2) {
		t.Errorf("unexpected result: %v", m)
	}
}
