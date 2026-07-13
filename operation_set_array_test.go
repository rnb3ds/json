package json

import (
	"testing"
)

// operation_set_array_test.go covers Set on array elements via index, shorthand
// index, wildcard, slice range, and multi-field extract — the scenarios that
// previously failed (silent partial update, hard error, or silent no-op) before
// the Set routing in operation_set.go was corrected and the multi-field extract
// opSet branch was added to recursive.go.
//
// Companion to operation_delete_array_test.go (the Delete side of the same
// array-element scenarios).

const setArrayInput = `[
  {"name_cn": "万国数据", "name_en": "GDS Holdings Limited", "name_hk": "万国数据", "symbol": "GDS.US"},
  {"name_cn": "极氪", "name_en": "ZEEKR Intelligent Technology Holding Limited", "name_hk": "極氪", "symbol": "ZK.US"}
]`

// assertSetField verifies the value of a field on the given array element after
// parsing the result JSON. wantOK=false asserts the field is absent.
func assertSetField(t *testing.T, result string, elemIdx int, field string, want any, wantOK bool) {
	t.Helper()
	var arr []map[string]any
	if err := Unmarshal([]byte(result), &arr); err != nil {
		t.Fatalf("result is not a JSON array: %v\nresult: %s", err, result)
	}
	if elemIdx < 0 || elemIdx >= len(arr) {
		t.Fatalf("element index %d out of range (len %d)", elemIdx, len(arr))
	}
	got, exists := arr[elemIdx][field]
	if exists != wantOK {
		t.Errorf("elem[%d].%q exists=%v, want exists=%v", elemIdx, field, exists, wantOK)
		return
	}
	if wantOK && got != want {
		t.Errorf("elem[%d].%q = %v, want %v", elemIdx, field, got, want)
	}
}

func TestSetArrayElementScenarios(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		value   any
		wantErr bool
		check   func(t *testing.T, result string)
	}{
		{
			// Pre-existing behavior (dot-notation path): only elem 0 changes.
			name:  "bracket index [0].name_cn sets only elem 0",
			path:  "[0].name_cn",
			value: "aa",
			check: func(t *testing.T, result string) {
				assertSetField(t, result, 0, "name_cn", "aa", true)
				assertSetField(t, result, 1, "name_cn", "极氪", true) // untouched
			},
		},
		{
			name:  "shorthand index 0.name_cn sets only elem 0",
			path:  "0.name_cn",
			value: "bb",
			check: func(t *testing.T, result string) {
				assertSetField(t, result, 0, "name_cn", "bb", true)
				assertSetField(t, result, 1, "name_cn", "极氪", true)
			},
		},
		{
			// REGRESSION: previously only elem 0 changed (splitPath parsed [*]
			// as index 0 on the dot-notation path). Now distributed across all.
			name:  "wildcard [*].name_cn sets all elements",
			path:  "[*].name_cn",
			value: "X",
			check: func(t *testing.T, result string) {
				assertSetField(t, result, 0, "name_cn", "X", true)
				assertSetField(t, result, 1, "name_cn", "X", true)
			},
		},
		{
			// REGRESSION: previously errored "array slice not supported as
			// intermediate path segment". Now distributed across the range.
			name:  "slice range [0:2].name_cn sets all elements in range",
			path:  "[0:2].name_cn",
			value: "Y",
			check: func(t *testing.T, result string) {
				assertSetField(t, result, 0, "name_cn", "Y", true)
				assertSetField(t, result, 1, "name_cn", "Y", true)
			},
		},
		{
			name:  "partial slice range [0:1].name_cn sets only elem 0",
			path:  "[0:1].name_cn",
			value: "P",
			check: func(t *testing.T, result string) {
				assertSetField(t, result, 0, "name_cn", "P", true)
				assertSetField(t, result, 1, "name_cn", "极氪", true)
			},
		},
		{
			// REGRESSION: previously a silent no-op (handleMultiFieldExtractSegment
			// had no opSet branch). Now sets every listed field per element.
			name:  "multi-field [*].{name_cn,symbol} sets listed fields on all elements",
			path:  "[*].{name_cn,symbol}",
			value: map[string]any{"name_cn": "N", "symbol": "S", "ignored": "Z"},
			check: func(t *testing.T, result string) {
				assertSetField(t, result, 0, "name_cn", "N", true)
				assertSetField(t, result, 1, "name_cn", "N", true)
				assertSetField(t, result, 0, "symbol", "S", true)
				assertSetField(t, result, 1, "symbol", "S", true)
				// Fields outside the extract list must NOT be created.
				assertSetField(t, result, 0, "ignored", nil, false)
				// Unlisted fields are untouched.
				assertSetField(t, result, 0, "name_en", "GDS Holdings Limited", true)
			},
		},
		{
			name:    "multi-field set with non-map value returns type error",
			path:    "[*].{name_cn,symbol}",
			value:   "scalar",
			wantErr: true,
		},
		{
			// Multi-field extract on a single (non-array) object element via index.
			name:  "multi-field [0].{name_cn,symbol} sets listed fields on elem 0",
			path:  "[0].{name_cn,symbol}",
			value: map[string]any{"name_cn": "N0", "symbol": "S0"},
			check: func(t *testing.T, result string) {
				assertSetField(t, result, 0, "name_cn", "N0", true)
				assertSetField(t, result, 0, "symbol", "S0", true)
				assertSetField(t, result, 1, "name_cn", "极氪", true) // untouched
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := New(DefaultConfig())
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			result, err := p.Set(setArrayInput, tt.path, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil; result=%s", result)
				}
				return
			}
			if err != nil {
				t.Fatalf("Set(%q): %v", tt.path, err)
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// TestSetArrayElementCreatePathsOff ensures the index/wildcard/slice Set fixes
// also hold when CreatePaths is disabled (the package default enables it; this
// exercises the other routing branch where createPaths is false).
func TestSetArrayElementCreatePathsOff(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CreatePaths = false
	p, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	t.Run("wildcard", func(t *testing.T) {
		out, err := p.Set(setArrayInput, "[*].name_cn", "X")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		assertSetField(t, out, 0, "name_cn", "X", true)
		assertSetField(t, out, 1, "name_cn", "X", true)
	})

	t.Run("slice_range", func(t *testing.T) {
		out, err := p.Set(setArrayInput, "[0:2].name_cn", "Y")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		assertSetField(t, out, 0, "name_cn", "Y", true)
		assertSetField(t, out, 1, "name_cn", "Y", true)
	})
}
