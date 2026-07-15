package json

import (
	"strings"
	"testing"

	"github.com/cybergodev/json/internal"
)

// ============================================================================
// Table-driven coverage tests for core functions with 0% or low coverage.
// Target: core ≥ 90%, utils ≥ 80%
// ============================================================================

// --- parseSurrogatePair (encoding.go:656, 0% coverage) ---
// Tested indirectly via Decode with strings containing surrogate pairs.

func TestDecoderSurrogatePair(t *testing.T) {
	// parseSurrogatePair is called from parseString via the Token() method
	// Build surrogate pair strings at byte level
	validPair := string([]byte{
		'"', '\\', 'u', 'D', '8', '3', 'D',
		'\\', 'u', 'D', 'E', '0', '0', '"',
	})

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid surrogate pair", input: validPair, wantErr: false},
		{name: "valid emoji UTF8", input: `"😀"`, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec := NewDecoder(strings.NewReader(tt.input))
			tok, err := dec.Token()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tok == nil {
				t.Error("expected non-nil token")
			}
		})
	}
}

// --- encodeJSONNumber: tested in coverage_test.go TestEncodeJSONNumber ---

// --- validateInputEssential (processor.go:1736, 0% coverage) ---
// Tested via Processor with SkipValidation enabled and oversized input.

func TestValidateInputEssential(t *testing.T) {
	t.Run("oversized input with SkipValidation", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxJSONSize = 100
		cfg.SkipValidation = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		largeJSON := `{"a":"` + strings.Repeat("x", 200) + `"}`
		_, err = p.Get(largeJSON, "a")
		if err == nil {
			t.Error("expected error for oversized JSON even with SkipValidation")
		}
	})

	t.Run("valid input with SkipValidation", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.SkipValidation = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		val, err := p.Get(`{"key":"value"}`, "key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "value" {
			t.Errorf("val = %v, want value", val)
		}
	})
}

// --- Recursive processor: array index with delete ---

func TestRecursiveArrayIndexDelete(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "delete middle element", json: `{"arr":[1,2,3]}`, path: "arr[1]", want: `{"arr":[1,3]}`},
		{name: "delete first element", json: `{"arr":[10,20,30]}`, path: "arr[0]", want: `{"arr":[20,30]}`},
		{name: "delete last by negative index", json: `{"arr":[1,2,3]}`, path: "arr[-1]", want: `{"arr":[1,2]}`},
		{name: "delete nested array element", json: `{"a":{"b":[1,2,3]}}`, path: "a.b[1]", want: `{"a":{"b":[1,3]}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Delete(tt.json, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tt.want, result)
		})
	}
}

// --- Recursive processor: array slice with delete ---

func TestRecursiveArraySliceDelete(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "delete slice range", json: `{"arr":[1,2,3,4,5]}`, path: "arr[1:3]", want: `{"arr":[1,4,5]}`},
		{name: "delete slice with step", json: `{"arr":[1,2,3,4,5]}`, path: "arr[0:5:2]", want: `{"arr":[2,4]}`},
		{name: "delete all elements via slice", json: `{"arr":[1,2,3]}`, path: "arr[0:3]", want: `{"arr":[]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Delete(tt.json, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertJSONEqual(t, tt.want, result)
		})
	}
}

// --- Wildcard/extract/slice: tested in recursive_test.go and operation_test.go ---

// --- PreParse and Release ---

func TestPreParseAndRelease(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer p.Close()

	parsed, err := p.PreParse(`{"a":1,"b":[2,3],"c":{"d":4}}`)
	if err != nil {
		t.Fatalf("PreParse failed: %v", err)
	}

	val, err := p.GetFromParsed(parsed, "a")
	if err != nil {
		t.Fatalf("GetFromParsed failed: %v", err)
	}
	if val != float64(1) {
		t.Errorf("val = %v, want 1", val)
	}

	val, err = p.GetFromParsed(parsed, "c.d")
	if err != nil {
		t.Fatalf("GetFromParsed nested failed: %v", err)
	}
	if val != float64(4) {
		t.Errorf("val = %v, want 4", val)
	}

	parsed.Release()
	parsed.Release() // double release should not panic
}

// --- checkRateLimit (processor.go:1267, 18.2% coverage) ---

func TestProcessorRateLimit(t *testing.T) {
	t.Run("rate limit enforcement", func(t *testing.T) {
		cfg := DefaultConfig()
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		// operationWindow is the internal max-ops/sec; a small window makes the
		// minimum interval between operations large enough that a second call in
		// immediate succession must be rejected by checkRateLimit (processor_get.go).
		p.metrics.operationWindow = 100 // 100 ops/sec => 10ms minimum interval

		// First call records lastOperationTime and succeeds.
		if _, err := p.Get(`{"a":1}`, "a"); err != nil {
			t.Fatalf("first Get failed: %v", err)
		}

		// Second call fires well within the 10ms window => rate-limited.
		_, err = p.Get(`{"a":1}`, "a")
		if err == nil {
			t.Fatal("second Get should have been rejected by the rate limit, got nil")
		}
		if !strings.Contains(err.Error(), "rate limit") {
			t.Errorf("expected a rate-limit error, got: %v", err)
		}
	})

	t.Run("disabled by default", func(t *testing.T) {
		p, err := New(DefaultConfig())
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		// operationWindow defaults to 0 => rate limiting disabled; rapid calls are fine.
		for range 5 {
			if _, err := p.Get(`{"a":1}`, "a"); err != nil {
				t.Fatalf("Get failed with rate limiting disabled: %v", err)
			}
		}
	})
}

// ============================================================================
// operation_set.go: array auto-extension & extraction-set paths (low coverage)
// Confirmed behaviors probed via the public Set API.
// ============================================================================

func TestSetArrayExtension_Coverage(t *testing.T) {
	// Auto-extension on out-of-bounds index: the array grows with nulls to fit.
	t.Run("auto extend with nulls", func(t *testing.T) {
		result, err := Set(`{"arr":[1,2]}`, "arr[5]", 9)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		assertJSONEqual(t, `{"arr":[1,2,null,null,null,9]}`, result)
	})

	t.Run("in bounds append via index", func(t *testing.T) {
		result, err := Set(`{"arr":[1,2]}`, "arr[2]", 3)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		assertJSONEqual(t, `{"arr":[1,2,3]}`, result)
	})

	t.Run("slice set auto extends", func(t *testing.T) {
		result, err := Set(`{"arr":[1,2]}`, "arr[0:5]", 9)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		assertJSONEqual(t, `{"arr":[9,9,9,9,9]}`, result)
	})

	t.Run("negative index sets last element", func(t *testing.T) {
		result, err := Set(`{"arr":[1,2]}`, "arr[-1]", 9)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		assertJSONEqual(t, `{"arr":[1,9]}`, result)
	})

	t.Run("create nested path", func(t *testing.T) {
		result, err := Set(`{}`, "a.b.c", 1)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		assertJSONEqual(t, `{"a":{"b":{"c":1}}}`, result)
	})
}

// TestSetExtraction_Coverage exercises the extraction-Set code paths
// (setValueForExtract, setValueForArrayExtract, setValueForArrayExtractFlat,
// navigateToExtraction) via {field} extraction syntax on objects and arrays.
func TestSetExtraction_Coverage(t *testing.T) {
	t.Run("extract set on single object", func(t *testing.T) {
		result, err := Set(`{"u":{}}`, "u{name}", "x")
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		assertJSONEqual(t, `{"u":{"name":"x"}}`, result)
	})

	t.Run("extract set on array of objects", func(t *testing.T) {
		result, err := Set(`{"us":[{"a":1},{"a":2}]}`, "us{a}", 99)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		assertJSONEqual(t, `{"us":[{"a":99},{"a":99}]}`, result)
	})

	t.Run("flat extract set", func(t *testing.T) {
		result, err := Set(`{"us":[{"t":[1]},{"t":[2]}]}`, "us{flat:t}", 99)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		assertJSONEqual(t, `{"us":[{"t":99},{"t":99}]}`, result)
	})
}

// ============================================================================
// operation_array.go: multi-field / post-extraction Get paths (low coverage)
// ============================================================================

func TestGetExtraction_Coverage(t *testing.T) {
	t.Run("extract then slice", func(t *testing.T) {
		got, err := Get(`{"us":[{"id":1},{"id":2},{"id":3}]}`, "us{id}[0:2]")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		arr, ok := got.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T (%v)", got, got)
		}
		if len(arr) != 2 || arr[0] != 1.0 || arr[1] != 2.0 {
			t.Errorf("extract-then-slice = %v, want [1 2]", got)
		}
	})

	t.Run("multi-field extract on object", func(t *testing.T) {
		got, err := Get(`{"u":{"id":1,"name":"n","email":"e"}}`, "u{id,name,email}")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		m, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T (%v)", got, got)
		}
		if m["id"] != 1.0 || m["name"] != "n" || m["email"] != "e" {
			t.Errorf("multi-field extract = %v, want id/name/email", got)
		}
	})

	t.Run("extract on array yields array of maps", func(t *testing.T) {
		got, err := Get(`{"us":[{"id":1,"name":"a"},{"id":2,"name":"b"}]}`, "us{id,name}")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		arr, ok := got.([]any)
		if !ok || len(arr) != 2 {
			t.Fatalf("expected 2-element []any, got %T (%v)", got, got)
		}
		first, ok := arr[0].(map[string]any)
		if !ok || first["id"] != 1.0 || first["name"] != "a" {
			t.Errorf("first extracted map = %v, want id=1 name=a", arr[0])
		}
	})
}

// ============================================================================
// White-box coverage of operation_set.go / operation_array.go handlers that
// the PUBLIC Get/Set API bypasses.
//
// The library ships TWO parallel path engines: path.go + operation_*.go, and
// recursive.go. The public Get/Set default to recursive.go, so the
// extraction handlers in operation_set.go (setValueForExtract family) and
// operation_array.go (handleMultiFieldExtraction, handleStructAccess) are
// never reached through the public API. They are pure handler functions, so
// calling them directly — exactly as core_test.go does for handleExtraction —
// is the only way to cover them.
// ============================================================================

func TestSetExtractHandlers_Coverage(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	t.Run("setValueForExtract on object", func(t *testing.T) {
		obj := map[string]any{"a": 1}
		seg := internal.PathSegment{Type: internal.ExtractSegment, Key: "name"}
		if err := p.setValueForExtract(obj, seg, "x", true); err != nil {
			t.Fatalf("setValueForExtract: %v", err)
		}
		if obj["name"] != "x" {
			t.Errorf("field not set: %v", obj)
		}
	})

	t.Run("setValueForExtract on array (non-flat)", func(t *testing.T) {
		arr := []any{map[string]any{"a": 1}, "notmap"}
		seg := internal.PathSegment{Type: internal.ExtractSegment, Key: "k"}
		if err := p.setValueForExtract(arr, seg, 9, true); err != nil {
			t.Fatalf("setValueForExtract: %v", err)
		}
		// both elements now carry k=9; the non-map element is promoted to a map
		first, ok := arr[0].(map[string]any)
		if !ok || first["k"] != 9 {
			t.Errorf("first element not set: %v", arr[0])
		}
		second, ok := arr[1].(map[string]any)
		if !ok || second["k"] != 9 {
			t.Errorf("second element not promoted: %v", arr[1])
		}
	})

	t.Run("setValueForExtract on array (flat)", func(t *testing.T) {
		arr := []any{map[string]any{"tags": []any{"a"}}}
		seg := internal.PathSegment{Type: internal.ExtractSegment, Key: "tags", Flags: internal.FlagIsFlat}
		if err := p.setValueForExtract(arr, seg, "b", true); err != nil {
			t.Fatalf("setValueForExtract flat: %v", err)
		}
		got := arr[0].(map[string]any)["tags"].([]any)
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Errorf("flat extract set failed: %v", got)
		}
	})

	t.Run("setValueForExtract rejects non-container", func(t *testing.T) {
		seg := internal.PathSegment{Type: internal.ExtractSegment, Key: "name"}
		if err := p.setValueForExtract(42, seg, "x", true); err == nil {
			t.Error("expected error for extraction set on a number")
		}
	})

	t.Run("setValueForExtract rejects empty key", func(t *testing.T) {
		seg := internal.PathSegment{Type: internal.ExtractSegment, Key: ""}
		if err := p.setValueForExtract(map[string]any{}, seg, "x", true); err == nil {
			t.Error("expected error for empty extraction key")
		}
	})

	t.Run("navigateToExtraction on array returns current", func(t *testing.T) {
		arr := []any{1, 2}
		seg := internal.PathSegment{Type: internal.ExtractSegment, Key: "a"}
		got, err := p.navigateToExtraction(arr, seg, true, nil, 0)
		if err != nil {
			t.Fatalf("navigateToExtraction: %v", err)
		}
		// Array extraction is delegated to distributed operations: the current
		// array is returned unchanged (slices are not comparable, so check by
		// identity of contents).
		out, ok := got.([]any)
		if !ok || len(out) != 2 || out[0] != 1 || out[1] != 2 {
			t.Errorf("array navigation should return current unchanged, got %v", got)
		}
	})

	t.Run("navigateToExtraction on missing field with createPaths", func(t *testing.T) {
		obj := map[string]any{}
		segs := []internal.PathSegment{{Type: internal.ExtractSegment, Key: "name"}}
		// currentIndex is the last segment => createContainerForNextSegment
		// returns nil (the slot will be filled by the caller's value).
		got, err := p.navigateToExtraction(obj, segs[0], true, segs, 0)
		if err != nil {
			t.Fatalf("navigateToExtraction: %v", err)
		}
		if got != nil {
			t.Errorf("last-segment container should be nil, got %v", got)
		}
		if _, ok := obj["name"]; !ok {
			t.Errorf("missing field should be created: %v", obj)
		}
	})

	t.Run("navigateToExtraction missing field without createPaths errors", func(t *testing.T) {
		seg := internal.PathSegment{Type: internal.ExtractSegment, Key: "nope"}
		if _, err := p.navigateToExtraction(map[string]any{}, seg, false, nil, 0); err == nil {
			t.Error("expected error for missing field without createPaths")
		}
	})
}

// TestArrayExtractHandlers_Coverage directly exercises operation_array.go's
// multi-field extraction and struct-access handlers (bypassed by the public
// recursive engine).
func TestArrayExtractHandlers_Coverage(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	t.Run("handleMultiFieldExtraction on object", func(t *testing.T) {
		data := map[string]any{"id": 1, "name": "n", "extra": "x"}
		got, err := p.handleMultiFieldExtraction(data, "id,name", false)
		if err != nil {
			t.Fatalf("handleMultiFieldExtraction: %v", err)
		}
		m, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T (%v)", got, got)
		}
		if m["id"] != 1 || m["name"] != "n" {
			t.Errorf("multi-field object extract = %v", got)
		}
		if _, ok := m["extra"]; ok {
			t.Errorf("unrequested field leaked into result: %v", m)
		}
	})

	t.Run("handleMultiFieldExtraction on array (flat and non-flat)", func(t *testing.T) {
		data := []any{
			map[string]any{"id": 1, "name": "a"},
			map[string]any{"id": 2, "name": "b"},
		}
		// Non-flat: each item becomes a sub-map.
		got, err := p.handleMultiFieldExtraction(data, "id,name", false)
		if err != nil {
			t.Fatalf("handleMultiFieldExtraction: %v", err)
		}
		if arr, ok := got.([]any); !ok || len(arr) != 2 {
			t.Errorf("non-flat array extract = %v", got)
		}
		// Flat flag exercises the flattenValue branch for completeness.
		if _, err := p.handleMultiFieldExtraction(data, "id,name", true); err != nil {
			t.Fatalf("handleMultiFieldExtraction flat: %v", err)
		}
	})

	t.Run("handleStructAccess direct, case-insensitive, nil, pointer", func(t *testing.T) {
		type sample struct {
			Name string
			Age  int
		}
		s := sample{Name: "x", Age: 5}

		if v := p.handleStructAccess(s, "Name"); v != "x" {
			t.Errorf("direct field = %v, want x", v)
		}
		if v := p.handleStructAccess(s, "age"); v != 5 {
			t.Errorf("case-insensitive field = %v, want 5", v)
		}
		if v := p.handleStructAccess(s, "missing"); v != nil {
			t.Errorf("missing field = %v, want nil", v)
		}
		if v := p.handleStructAccess(nil, "x"); v != nil {
			t.Errorf("nil data = %v, want nil", v)
		}
		if v := p.handleStructAccess(42, "x"); v != nil {
			t.Errorf("non-struct = %v, want nil", v)
		}
		if v := p.handleStructAccess((*sample)(nil), "x"); v != nil {
			t.Errorf("nil pointer = %v, want nil", v)
		}
		if v := p.handleStructAccess(&sample{Name: "p"}, "Name"); v != "p" {
			t.Errorf("non-nil pointer field = %v, want p", v)
		}
	})
}
