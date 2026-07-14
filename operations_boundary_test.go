package json

import "testing"

// ============================================================================
// Boundary tests for recursive.go (69%) and operation_set.go (71%) uncovered
// branches, exercised via the public Set/Get/Delete API with createPaths,
// distributed ops, slices, extracts, and security limits.
// ============================================================================

// --- handleArrayIndexSegmentUnified / handleArraySliceSegmentUnified (recursive.go) ---

func TestRecursive_ArrayIndex_Boundary(t *testing.T) {
	t.Run("index_on_map_no_panic", func(t *testing.T) {
		// Array index access on a non-array (object) must not panic; it surfaces
		// as a not-found result (nil and/or error) rather than crashing.
		v, err := Get(`{"a":1}`, "[0]")
		if err == nil && v != nil {
			t.Errorf("expected nil-or-error for array index on object, got (%v, %v)", v, err)
		}
	})
	t.Run("oob_index_extends_with_createPaths", func(t *testing.T) {
		// Default config has CreatePaths=true: OOB index extends the array.
		r, err := Set(`{"a":[1,2]}`, "a[5]", 99)
		if err != nil {
			t.Fatalf("Set err: %v", err)
		}
		v, _ := Get(r, "a[5]")
		if v != float64(99) {
			t.Errorf("a[5] = %v, want 99", v)
		}
	})
	t.Run("slice_delete", func(t *testing.T) {
		r, err := Delete(`{"a":[1,2,3]}`, "a[0:1]")
		if err != nil {
			t.Fatalf("Delete err: %v", err)
		}
		v, _ := Get(r, "a")
		arr, ok := v.([]any)
		if !ok || len(arr) != 2 || arr[0] != float64(2) || arr[1] != float64(3) {
			t.Errorf("after slice delete got %v, want [2 3]", v)
		}
	})
	t.Run("distributed_oob_set_no_panic", func(t *testing.T) {
		// Distributed set on a slice of slices with an OOB index must not panic.
		_, _ = Set(`{"items":[[1,2],[3,4]]}`, "items[5]", 99)
	})
}

// --- handleMultiFieldExtractSegment / handleExtractThenSlice (recursive.go) ---

func TestRecursive_Extract_Boundary(t *testing.T) {
	t.Run("multifield_delete", func(t *testing.T) {
		r, err := Delete(`{"users":[{"id":1,"name":"a","x":2}]}`, "users.{id,name}")
		if err != nil {
			t.Fatalf("Delete err: %v", err)
		}
		x, _ := Get(r, "users[0].x")
		if x != float64(2) {
			t.Errorf("x should be preserved, got %v", x)
		}
		id, _ := Get(r, "users[0].id")
		if id != nil {
			t.Errorf("id should be deleted, got %v", id)
		}
	})
	t.Run("multifield_set_with_map", func(t *testing.T) {
		r, err := Set(`{"users":[{"id":1}]}`, "users.{id,name}", map[string]any{"id": 2, "name": "b"})
		if err != nil {
			t.Fatalf("Set err: %v", err)
		}
		id, _ := Get(r, "users[0].id")
		name, _ := Get(r, "users[0].name")
		if id != float64(2) || name != "b" {
			t.Errorf("got id=%v name=%v, want 2/b", id, name)
		}
	})
	t.Run("extract_set_non_map_value", func(t *testing.T) {
		// Setting an extract with a non-map scalar completes without panicking;
		// the branch is exercised regardless of how the scalar is applied.
		if _, err := Set(`{"users":[{"id":1}]}`, "users.{id}", "not a map"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
	t.Run("extract_then_slice_on_non_container", func(t *testing.T) {
		// {extract}[slice] on a non-container (JSON number) must not panic.
		_, _ = Get("42", "{tags}[0:1]")
	})
	t.Run("extract_then_slice_set", func(t *testing.T) {
		r, err := Set(`{"items":[{"tags":["a","b"]}]}`, "items{tags}[0:1]", "x")
		if err != nil {
			t.Fatalf("Set err: %v", err)
		}
		v, _ := Get(r, "items[0].tags")
		arr, ok := v.([]any)
		if !ok || len(arr) == 0 || arr[0] != "x" {
			t.Errorf("extract-then-slice set got %v, want first element x", v)
		}
	})
}

// --- ProcessRecursivelyWithOptions empty-path (recursive.go:37) ---

func TestRecursive_EmptyPath_Boundary(t *testing.T) {
	if _, err := Set(`{"a":1}`, "", 99); err == nil {
		t.Error("expected error for Set with empty path")
	}
	if _, err := Delete(`{"a":1}`, ""); err == nil {
		t.Error("expected error for Delete with empty path")
	}
}

// --- handleArrayExtensionAndSet / extendArrayAndSet* (operation_set.go) ---

func TestOperationSet_Extension_Boundary(t *testing.T) {
	t.Run("slice_extension_security_limit", func(t *testing.T) {
		// Slice end far beyond maxArrayExtension must be rejected.
		if _, err := Set(`{"a":[]}`, "a[0:999999999]", []any{99}); err == nil {
			t.Error("expected error for slice extension beyond security limit")
		}
	})
	t.Run("index_extension_security_limit", func(t *testing.T) {
		// Index far beyond maxArrayExtension must be rejected.
		if _, err := Set(`{"a":[]}`, "a[9999999999]", 99); err == nil {
			t.Error("expected error for index extension beyond security limit")
		}
	})
	t.Run("root_array_oob_no_panic", func(t *testing.T) {
		// OOB index on a root-level array must not panic (errors or rejects).
		_, _ = Set(`[1,2]`, "[5]", 99)
	})
}

// --- setValueForArrayExtractFlat (operation_set.go:909) ---

func TestOperationSet_ArrayExtractFlat_Boundary(t *testing.T) {
	t.Run("merge_into_existing_array", func(t *testing.T) {
		r, err := Set(`{"items":[{"tags":["a"]}]}`, "items{flat:tags}", []any{"b", "c"})
		if err != nil {
			t.Fatalf("Set err: %v", err)
		}
		v, _ := Get(r, "items[0].tags")
		arr, ok := v.([]any)
		if !ok || len(arr) < 2 {
			t.Errorf("flat-merge got %v, want merged slice with b,c", v)
		}
	})
	t.Run("convert_non_array_existing", func(t *testing.T) {
		// Existing value is a scalar string -> converted to a slice.
		r, err := Set(`{"items":[{"tags":"x"}]}`, "items{flat:tags}", "y")
		if err != nil {
			t.Fatalf("Set err: %v", err)
		}
		v, _ := Get(r, "items[0].tags")
		if v == nil {
			t.Error("expected converted tags value, got nil")
		}
	})
	t.Run("non_map_item", func(t *testing.T) {
		// items[0] is a number, not a map -> cannot set {flat:tags}; must not panic.
		_, _ = Set(`{"items":[42]}`, "items{flat:tags}", "y")
	})
}
