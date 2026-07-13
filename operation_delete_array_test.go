package json

import (
	"testing"
)

// operation_delete_array_test.go covers Delete on array elements via index,
// shorthand index, wildcard, slice, and multi-field extract — the scenarios
// that previously failed (ERR or silent no-op) before Delete was unified onto
// the recursive processor.

const deleteArrayInput = `[
  {"name_cn": "万国数据", "name_en": "GDS Holdings Limited", "name_hk": "万国数据", "symbol": "GDS.US"},
  {"name_cn": "极氪", "name_en": "ZEEKR Intelligent Technology Holding Limited", "name_hk": "極氪", "symbol": "ZK.US"}
]`

// assertField verifies whether a field exists on the given array element after
// parsing the result JSON. Used to assert deletion precisely without depending
// on key ordering in the serialized output.
func assertField(t *testing.T, result string, elemIdx int, field string, wantExists bool) {
	t.Helper()
	var arr []map[string]any
	if err := Unmarshal([]byte(result), &arr); err != nil {
		t.Fatalf("result is not a JSON array: %v\nresult: %s", err, result)
	}
	if elemIdx < 0 || elemIdx >= len(arr) {
		t.Fatalf("element index %d out of range (len %d)", elemIdx, len(arr))
	}
	_, exists := arr[elemIdx][field]
	if exists != wantExists {
		t.Errorf("elem[%d].%q exists=%v, want %v", elemIdx, field, exists, wantExists)
	}
}

func TestDeleteArrayElementScenarios(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		check   func(t *testing.T, result string)
	}{
		{
			name: "bracket index [0].name_cn removes only elem 0 field",
			path: "[0].name_cn",
			check: func(t *testing.T, result string) {
				assertField(t, result, 0, "name_cn", false)
				assertField(t, result, 1, "name_cn", true) // untouched
				assertField(t, result, 0, "symbol", true)  // untouched
			},
		},
		{
			name: "shorthand index 0.name_cn removes only elem 0 field",
			path: "0.name_cn",
			check: func(t *testing.T, result string) {
				assertField(t, result, 0, "name_cn", false)
				assertField(t, result, 1, "name_cn", true)
			},
		},
		{
			name: "wildcard [*].name_cn removes field from all elements",
			path: "[*].name_cn",
			check: func(t *testing.T, result string) {
				assertField(t, result, 0, "name_cn", false)
				assertField(t, result, 1, "name_cn", false)
				assertField(t, result, 0, "symbol", true)
			},
		},
		{
			name: "slice [0:2].name_cn removes field from ranged elements",
			path: "[0:2].name_cn",
			check: func(t *testing.T, result string) {
				assertField(t, result, 0, "name_cn", false)
				assertField(t, result, 1, "name_cn", false)
			},
		},
		{
			name: "multi-field wildcard [*].{name_cn,symbol} removes both from all",
			path: "[*].{name_cn,symbol}",
			check: func(t *testing.T, result string) {
				assertField(t, result, 0, "name_cn", false)
				assertField(t, result, 0, "symbol", false)
				assertField(t, result, 1, "name_cn", false)
				assertField(t, result, 1, "symbol", false)
				assertField(t, result, 0, "name_en", true) // untouched
				assertField(t, result, 0, "name_hk", true)
			},
		},
		{
			name: "multi-field index [1].{name_cn,symbol} removes both from elem 1 only",
			path: "[1].{name_cn,symbol}",
			check: func(t *testing.T, result string) {
				assertField(t, result, 0, "name_cn", true)  // untouched
				assertField(t, result, 0, "symbol", true)   // untouched
				assertField(t, result, 1, "name_cn", false) // removed
				assertField(t, result, 1, "symbol", false)  // removed
			},
		},
		{
			name:    "precise complex missing path reports error (contract preserved)",
			path:    "[0].nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Delete(deleteArrayInput, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for path %q, got nil; result: %s", tt.path, result)
				}
				return
			}
			if err != nil {
				t.Fatalf("Delete(%q) unexpected error: %v", tt.path, err)
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

// TestDeleteMultiFieldExtractTolerant verifies that batch multi-field extract
// delete tolerates elements missing the target field (idempotent, no error),
// matching Go's delete() semantics on absent keys.
func TestDeleteMultiFieldExtractTolerant(t *testing.T) {
	// Element 1 lacks "symbol" — deleting it must not error.
	input := `[
	  {"name_cn": "甲", "symbol": "A.US"},
	  {"name_cn": "乙"}
	]`
	result, err := Delete(input, "[*].{name_cn,symbol}")
	if err != nil {
		t.Fatalf("expected no error on tolerant batch delete, got: %v", err)
	}
	assertField(t, result, 0, "name_cn", false)
	assertField(t, result, 0, "symbol", false)
	assertField(t, result, 1, "name_cn", false)
}

// TestDeleteWildcardMissingFieldTolerant verifies that a wildcard property
// delete over elements where some lack the property stays silent (no error).
func TestDeleteWildcardMissingFieldTolerant(t *testing.T) {
	input := `[{"a":1},{"b":2}]`
	// [*].a: elem 0 has a (deleted), elem 1 lacks a (tolerated).
	if _, err := Delete(input, "[*].a"); err != nil {
		t.Fatalf("expected no error for wildcard delete with missing field, got: %v", err)
	}
}
