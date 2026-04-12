package json

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// ============================================================================
// TestJSONPointerSet - JSON Pointer (RFC 6901) set operations
// ============================================================================

func TestJSONPointerSet(t *testing.T) {
	tests := []struct {
		name      string
		jsonStr   string
		path      string
		value     any
		cfg       Config
		wantErr   bool
		errSubstr string
		wantJSON  string
	}{
		{
			name:     "overwrite existing nested value",
			jsonStr:  `{"a":{"b":1}}`,
			path:     "/a/b",
			value:    2,
			wantJSON: `{"a":{"b":2}}`,
		},
		{
			name:     "add new key to existing object",
			jsonStr:  `{"a":{"b":1}}`,
			path:     "/a/c",
			value:    3,
			wantJSON: `{"a":{"b":1,"c":3}}`,
		},
		{
			name:      "set root path should error",
			jsonStr:   `{"a":1}`,
			path:      "/",
			value:     42,
			wantErr:   true,
			errSubstr: "cannot set root",
		},
		{
			name:     "create nested path with CreatePaths",
			jsonStr:  `{}`,
			path:     "/x/y/z",
			value:    "hello",
			cfg:      Config{CreatePaths: true},
			wantJSON: `{"x":{"y":{"z":"hello"}}}`,
		},
		{
			name:     "tilde slash escaping ~1 becomes slash",
			jsonStr:  `{}`,
			path:     "/a~1b",
			value:    "val",
			cfg:      Config{CreatePaths: true},
			wantJSON: `{"a/b":"val"}`,
		},
		{
			name:     "tilde escaping ~0 becomes tilde",
			jsonStr:  `{}`,
			path:     "/m~0n",
			value:    "val",
			cfg:      Config{CreatePaths: true},
			wantJSON: `{"m~n":"val"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Set(tt.jsonStr, tt.path, tt.value, tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Fatalf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tt.wantJSON, result)
		})
	}
}

// ============================================================================
// TestJSONPointerDelete - JSON Pointer delete operations
// ============================================================================

func TestJSONPointerDelete(t *testing.T) {
	tests := []struct {
		name      string
		jsonStr   string
		path      string
		wantErr   bool
		errSubstr string
		wantJSON  string
	}{
		{
			name:     "delete nested property",
			jsonStr:  `{"a":{"b":1,"c":2}}`,
			path:     "/a/b",
			wantJSON: `{"a":{"c":2}}`,
		},
		{
			name:      "delete root should error",
			jsonStr:   `{"a":1}`,
			path:      "/",
			wantErr:   true,
			errSubstr: "cannot delete root",
		},
		{
			name:     "delete top-level key leaving empty object",
			jsonStr:  `{"x":1}`,
			path:     "/x",
			wantJSON: `{}`,
		},
		{
			name:     "tilde escaping in delete path",
			jsonStr:  `{"a/b":"val","c":2}`,
			path:     "/a~1b",
			wantJSON: `{"c":2}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Delete(tt.jsonStr, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Fatalf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tt.wantJSON, result)
		})
	}
}

// ============================================================================
// TestAppendOperation - append syntax path[+]
// ============================================================================

func TestAppendOperation(t *testing.T) {
	tests := []struct {
		name      string
		jsonStr   string
		path      string
		value     any
		cfg       Config
		wantErr   bool
		errSubstr string
		wantJSON  string
	}{
		{
			name:     "append to existing array",
			jsonStr:  `{"items":[1,2]}`,
			path:     "items[+]",
			value:    3,
			wantJSON: `{"items":[1,2,3]}`,
		},
		{
			name:     "append to empty array",
			jsonStr:  `{"items":[]}`,
			path:     "items[+]",
			value:    1,
			wantJSON: `{"items":[1]}`,
		},
		{
			name:      "append to non-existent key without CreatePaths errors",
			jsonStr:   `{}`,
			path:      "items[+]",
			value:     1,
			cfg:       func() Config { c := DefaultConfig(); c.CreatePaths = false; return c }(),
			wantErr:   true,
			errSubstr: "cannot append",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Set(tt.jsonStr, tt.path, tt.value, tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Fatalf("expected error containing %q, got %q", tt.errSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tt.wantJSON, result)
		})
	}
}

// ============================================================================
// TestNegativeArrayIndex - negative array index handling
// ============================================================================

func TestNegativeArrayIndexEdgeCases(t *testing.T) {
	t.Run("get negative index returns last element", func(t *testing.T) {
		result, err := Get(`{"items":[1,2,3]}`, "items[-1]")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != float64(3) {
			t.Fatalf("expected 3, got %v", result)
		}
	})

	t.Run("delete negative index removes last element", func(t *testing.T) {
		result, err := Delete(`{"items":[1,2,3]}`, "items[-1]")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertJSONEqual(t, `{"items":[1,2]}`, result)
	})

	t.Run("set negative index sets last element", func(t *testing.T) {
		result, err := Set(`{"items":[1,2,3]}`, "items[-1]", 99)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertJSONEqual(t, `{"items":[1,2,99]}`, result)
	})
}

// ============================================================================
// TestArraySliceOperations - slice syntax [start:end:step]
// ============================================================================

func TestArraySliceOperationsEdgeCases(t *testing.T) {
	t.Run("get basic slice", func(t *testing.T) {
		result, err := Get(`{"items":[0,1,2,3,4]}`, "items[1:3]")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		arr, ok := result.([]any)
		if !ok {
			t.Fatalf("expected array, got %T", result)
		}
		if len(arr) != 2 {
			t.Fatalf("expected 2 elements, got %d", len(arr))
		}
		if arr[0] != float64(1) || arr[1] != float64(2) {
			t.Fatalf("expected [1,2], got %v", arr)
		}
	})

	t.Run("get slice with step", func(t *testing.T) {
		result, err := Get(`{"items":[0,1,2,3,4]}`, "items[::2]")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		arr, ok := result.([]any)
		if !ok {
			t.Fatalf("expected array, got %T", result)
		}
		if len(arr) != 3 {
			t.Fatalf("expected 3 elements, got %d: %v", len(arr), arr)
		}
		if arr[0] != float64(0) || arr[1] != float64(2) || arr[2] != float64(4) {
			t.Fatalf("expected [0,2,4], got %v", arr)
		}
	})

	t.Run("get reversed slice", func(t *testing.T) {
		result, err := Get(`{"items":[1,2,3]}`, "items[::-1]")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		arr, ok := result.([]any)
		if !ok {
			t.Fatalf("expected array, got %T", result)
		}
		if len(arr) != 3 {
			t.Fatalf("expected 3 elements, got %d", len(arr))
		}
		if arr[0] != float64(3) || arr[1] != float64(2) || arr[2] != float64(1) {
			t.Fatalf("expected [3,2,1], got %v", arr)
		}
	})

	t.Run("delete slice leaves remaining elements", func(t *testing.T) {
		result, err := Delete(`{"items":[0,1,2,3,4]}`, "items[1:3]")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertJSONEqual(t, `{"items":[0,3,4]}`, result)
	})
}

// ============================================================================
// TestTypedGettersEdgeCases - typed getters with edge cases
// ============================================================================

func TestTypedGettersEdgeCases(t *testing.T) {
	t.Run("GetString with json.Number value", func(t *testing.T) {
		// Use json.Number encoding to get a json.Number value
		// When UseNumber is enabled, numbers are decoded as json.Number
		// For standard parsing, numbers become float64, so GetString converts float64 to string
		result := GetString(`{"v":42}`, "v")
		if result != "42" {
			t.Fatalf("expected \"42\", got %q", result)
		}
	})

	t.Run("GetInt with whole float64", func(t *testing.T) {
		result := GetInt(`{"v":42.0}`, "v")
		if result != 42 {
			t.Fatalf("expected 42, got %d", result)
		}
	})

	t.Run("GetInt with fractional float64 returns zero", func(t *testing.T) {
		// 42.5 has a fractional part, so convertToInt returns (0, false)
		result := GetInt(`{"v":42.5}`, "v")
		if result != 0 {
			t.Fatalf("expected 0 for non-integer float, got %d", result)
		}
	})

	t.Run("GetBool with string true", func(t *testing.T) {
		result := GetBool(`{"v":"true"}`, "v")
		if result != true {
			t.Fatalf("expected true, got %v", result)
		}
	})

	t.Run("GetBool with string TRUE", func(t *testing.T) {
		result := GetBool(`{"v":"TRUE"}`, "v")
		if result != true {
			t.Fatalf("expected true, got %v", result)
		}
	})

	t.Run("GetBool with string 1", func(t *testing.T) {
		result := GetBool(`{"v":"1"}`, "v")
		if result != true {
			t.Fatalf("expected true, got %v", result)
		}
	})

	t.Run("GetBool with string false", func(t *testing.T) {
		result := GetBool(`{"v":"false"}`, "v")
		if result != false {
			t.Fatalf("expected false, got %v", result)
		}
	})

	t.Run("GetFloat with int value", func(t *testing.T) {
		result := GetFloat(`{"v":42}`, "v")
		if result != 42.0 {
			t.Fatalf("expected 42.0, got %v", result)
		}
	})

	t.Run("GetArray on non-array returns nil", func(t *testing.T) {
		result := GetArray(`{"v":"not_array"}`, "v")
		if result != nil {
			t.Fatalf("expected nil, got %v", result)
		}
	})

	t.Run("GetObject on non-object returns nil", func(t *testing.T) {
		result := GetObject(`{"v":42}`, "v")
		if result != nil {
			t.Fatalf("expected nil, got %v", result)
		}
	})

	t.Run("GetTyped on missing path returns zero value", func(t *testing.T) {
		result := GetTyped[string](`{"a":1}`, "missing_key")
		if result != "" {
			t.Fatalf("expected empty string, got %q", result)
		}
	})

	t.Run("GetTyped on missing path returns default value", func(t *testing.T) {
		result := GetTyped[int](`{"a":1}`, "missing_key", 99)
		if result != 99 {
			t.Fatalf("expected 99, got %d", result)
		}
	})
}

// ============================================================================
// TestEncodePrettyAPI - package-level EncodePretty function
// ============================================================================

func TestEncodePrettyAPI(t *testing.T) {
	t.Run("pretty print map", func(t *testing.T) {
		data := map[string]any{"name": "Alice", "age": float64(30)}
		result, err := EncodePretty(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "\n") {
			t.Fatalf("expected newlines in pretty output, got: %s", result)
		}
		if !strings.Contains(result, `"name"`) || !strings.Contains(result, `"age"`) {
			t.Fatalf("expected keys in output, got: %s", result)
		}
	})

	t.Run("pretty print nil returns null", func(t *testing.T) {
		result, err := EncodePretty(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertJSONEqual(t, "null", result)
	})

	t.Run("pretty print slice", func(t *testing.T) {
		data := []any{float64(1), float64(2), float64(3)}
		result, err := EncodePretty(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "\n") {
			t.Fatalf("expected newlines in pretty array output, got: %s", result)
		}
	})

	t.Run("pretty print struct", func(t *testing.T) {
		type Person struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}
		data := Person{Name: "Bob", Age: 25}
		result, err := EncodePretty(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "\n") {
			t.Fatalf("expected newlines in pretty struct output, got: %s", result)
		}
		if !strings.Contains(result, `"name"`) || !strings.Contains(result, "Bob") {
			t.Fatalf("expected struct fields in output, got: %s", result)
		}
	})

	t.Run("custom indent config", func(t *testing.T) {
		cfg := PrettyConfig()
		cfg.Indent = "    " // 4-space indent
		data := map[string]any{"key": "value"}
		result, err := EncodePretty(data, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "    ") {
			t.Fatalf("expected 4-space indent in output, got: %s", result)
		}
	})
}

// ============================================================================
// Helpers
// ============================================================================

// assertJSONEqual compares two JSON strings by parsing both and comparing
// the resulting data structures. This handles key ordering differences.
func assertJSONEqual(t *testing.T, expected, actual string) {
	t.Helper()
	var expData, actData any
	if err := json.Unmarshal([]byte(expected), &expData); err != nil {
		t.Fatalf("failed to parse expected JSON %q: %v", expected, err)
	}
	if err := json.Unmarshal([]byte(actual), &actData); err != nil {
		t.Fatalf("failed to parse actual JSON %q: %v", actual, err)
	}
	// Re-marshal both for canonical comparison
	expBytes, _ := json.Marshal(expData)
	actBytes, _ := json.Marshal(actData)
	if string(expBytes) != string(actBytes) {
		t.Fatalf("JSON mismatch:\nexpected: %s\nactual:   %s", string(expBytes), string(actBytes))
	}
}

// ============================================================================
// Additional edge case tests for coverage uplift
// ============================================================================

// TestDeleteComplexPaths tests delete operations on complex path patterns
func TestDeleteComplexPaths(t *testing.T) {
	t.Run("delete from nested array", func(t *testing.T) {
		result, err := Delete(`{"users":[{"name":"a"},{"name":"b"}]}`, "users[0]")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertJSONEqual(t, `{"users":[{"name":"b"}]}`, result)
	})

	t.Run("delete array slice", func(t *testing.T) {
		result, err := Delete(`{"items":[0,1,2,3,4,5]}`, "items[1:4]")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertJSONEqual(t, `{"items":[0,4,5]}`, result)
	})

}

// TestSetCreatePaths tests Set with CreatePaths for intermediate object creation
func TestSetCreatePaths(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		path     string
		value    any
		wantJSON string
	}{
		{
			name:     "create 3-level path",
			jsonStr:  `{}`,
			path:     "a.b.c",
			value:    42,
			wantJSON: `{"a":{"b":{"c":42}}}`,
		},
		{
			name:     "create path with array index",
			jsonStr:  `{"items":[]}`,
			path:     "items[0]",
			value:    "hello",
			wantJSON: `{"items":["hello"]}`,
		},
		{
			name:     "set on existing path",
			jsonStr:  `{"a":{"b":1}}`,
			path:     "a.b",
			value:    2,
			wantJSON: `{"a":{"b":2}}`,
		},
		{
			name:     "create path alongside existing",
			jsonStr:  `{"a":1}`,
			path:     "b.c",
			value:    3,
			wantJSON: `{"a":1,"b":{"c":3}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Set(tt.jsonStr, tt.path, tt.value, Config{CreatePaths: true})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tt.wantJSON, result)
		})
	}
}

// TestNormalizeNegativeIndex tests the normalizeNegativeIndex function
func TestNormalizeNegativeIndex(t *testing.T) {
	tests := []struct {
		index int
		len   int
		want  int
		err   bool
	}{
		{-1, 5, 4, false},
		{-5, 5, 0, false},
		{0, 5, 0, false},
		{4, 5, 4, false},
		{-6, 5, 0, true},
		{5, 5, 0, true},
		{-1, 0, 0, true},
		{-1, 1, 0, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("index_%d_len_%d", tt.index, tt.len), func(t *testing.T) {
			got, err := normalizeNegativeIndex(tt.index, tt.len)
			if tt.err {
				if err == nil {
					t.Errorf("expected error for index=%d len=%d", tt.index, tt.len)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("normalizeNegativeIndex(%d, %d) = %d, want %d", tt.index, tt.len, got, tt.want)
				}
			}
		})
	}
}
