package json

import "testing"

// ============================================================================
// Boundary tests for path.go low-coverage paths.
// ============================================================================

// --- preservingUnmarshal (path.go:712, 48% coverage) ---

func TestPreservingUnmarshal_Boundary(t *testing.T) {
	t.Run("non_preserve_into_any", func(t *testing.T) {
		var v any
		if err := preservingUnmarshal([]byte(`{"a":1}`), &v, false); err != nil {
			t.Fatalf("err: %v", err)
		}
	})
	t.Run("preserve_into_any", func(t *testing.T) {
		var v any
		if err := preservingUnmarshal([]byte(`{"a":42}`), &v, true); err != nil {
			t.Fatalf("err: %v", err)
		}
	})
	t.Run("preserve_into_map", func(t *testing.T) {
		var m map[string]any
		if err := preservingUnmarshal([]byte(`{"a":42}`), &m, true); err != nil {
			t.Fatalf("err: %v", err)
		}
	})
	t.Run("preserve_into_slice", func(t *testing.T) {
		var s []any
		if err := preservingUnmarshal([]byte(`[1,2,3]`), &s, true); err != nil {
			t.Fatalf("err: %v", err)
		}
	})
	t.Run("preserve_into_struct", func(t *testing.T) {
		type S struct {
			A int `json:"a"`
		}
		var s S
		if err := preservingUnmarshal([]byte(`{"a":42}`), &s, true); err != nil {
			t.Fatalf("err: %v", err)
		}
		if s.A != 42 {
			t.Errorf("A = %d, want 42", s.A)
		}
	})
	t.Run("invalid_json", func(t *testing.T) {
		var v any
		if err := preservingUnmarshal([]byte("not json"), &v, true); err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}
