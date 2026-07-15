package json

import (
	stdjson "encoding/json"
	"strings"
	"testing"
	"time"
)

// This file locks in the behavior fixes from the D-002 round-2 review so they
// cannot regress. Each subtest corresponds to a specific finding and would fail
// on the pre-fix code.
//
// Formerly split across d002_m5_verify_test.go, d002_f6_verify_test.go,
// d002_f7_verify_test.go, and d002_f8_verify_test.go; consolidated here by
// topic (reverse slices, encoding compat, wildcards, path-creation override,
// extract-then-slice, distributed null, and the F6/F7/F8 struct-encoding
// fixes).

// ---------------------------------------------------------------------------
// Reverse / negative-step slices (C1/C2/C3, M1, M3) — previously panics or
// silent no-ops because the opSet/opDelete loops assumed a positive step.
// ---------------------------------------------------------------------------

func TestD002R2_ReverseSlices(t *testing.T) {
	// C1: Set with reverse slice on an inner array must not panic.
	t.Run("C1_set_reverse_no_panic", func(t *testing.T) {
		doc := `{"items":[{"arr":[1,2,3]},{"arr":[4,5,6]}]}`
		r, err := Set(doc, "items[0].arr[::-1]", 99)
		if err != nil {
			t.Fatalf("Set err: %v", err)
		}
		got, _ := Get(r, "items[0].arr")
		if s := asStr(got); s != "[99,99,99]" {
			t.Fatalf("got %s want [99,99,99]", s)
		}
	})

	// C2: Delete with full reverse slice empties the array.
	t.Run("C2_delete_reverse_all", func(t *testing.T) {
		r, err := Delete(`{"a":[1,2,3,4,5]}`, "a[::-1]")
		if err != nil {
			t.Fatalf("Delete err: %v", err)
		}
		got, _ := Get(r, "a")
		if s := asStr(got); s != "[]" {
			t.Fatalf("got %s want []", s)
		}
	})

	// M1: Delete a[3:0:-1] removes indices 3,2,1 (Python [3:0:-1]==[4,3,2]).
	t.Run("M1_delete_3_0_step_neg1", func(t *testing.T) {
		r, err := Delete(`{"a":[1,2,3,4,5]}`, "a[3:0:-1]")
		if err != nil {
			t.Fatalf("Delete err: %v", err)
		}
		got, _ := Get(r, "a")
		if s := asStr(got); s != "[1,5]" {
			t.Fatalf("got %s want [1,5]", s)
		}
	})

	// C3: Delete on {extract}[slice] with reverse step must not panic.
	t.Run("C3_delete_extract_then_slice_reverse", func(t *testing.T) {
		r, err := Delete(`{"data":[{"arr":[1,2,3]}]}`, "data{arr}[::-1]")
		if err != nil {
			t.Fatalf("Delete err: %v", err)
		}
		got, _ := Get(r, "data[0].arr")
		if s := asStr(got); s != "[]" {
			t.Fatalf("got %s want []", s)
		}
	})

	// M3: very-negative start + reverse step yields [] (Python), not [1].
	t.Run("M3_get_very_neg_reverse_empty", func(t *testing.T) {
		got, err := Get(`{"a":[1,2,3,4,5]}`, "a[-10::-1]")
		if err != nil {
			t.Fatalf("Get err: %v", err)
		}
		if s := asStr(got); s != "[]" {
			t.Fatalf("got %s want []", s)
		}
	})

	// Regression guard: positive-step slice Set still honors step.
	t.Run("regress_positive_step_set", func(t *testing.T) {
		r, err := Set(`{"a":[{"b":1},{"b":2},{"b":3},{"b":4},{"b":5}]}`, "a[0:5:2].b", 99)
		if err != nil {
			t.Fatalf("Set err: %v", err)
		}
		got, _ := Get(r, "a")
		if s := asStr(got); s != `[{"b":99},{"b":2},{"b":99},{"b":4},{"b":99}]` {
			t.Fatalf("got %s", s)
		}
	})
}

// ---------------------------------------------------------------------------
// encoding/json compatibility (F1-F5) on the default Marshal / custom-encoder
// paths. Each diverged from encoding/json before the fix.
// ---------------------------------------------------------------------------

func TestD002R2_EncodingCompat(t *testing.T) {
	// F1: map keys are sorted lexicographically (encoding/json always sorts).
	t.Run("F1_map_keys_sorted", func(t *testing.T) {
		m := map[string]any{"z": 1, "a": 2, "m": 3, "b": 4}
		got, err := Marshal(m)
		if err != nil {
			t.Fatalf("Marshal err: %v", err)
		}
		want, _ := stdjson.Marshal(m)
		if string(got) != string(want) {
			t.Fatalf("got %s want %s", got, want)
		}
	})

	// F2: time.Time preserves sub-second precision (RFC3339Nano).
	t.Run("F2_time_nano_precision", func(t *testing.T) {
		ts := time.Date(2024, 1, 1, 12, 30, 45, 123456789, time.UTC)
		got, err := Marshal(ts)
		if err != nil {
			t.Fatalf("Marshal err: %v", err)
		}
		want, _ := stdjson.Marshal(ts)
		if string(got) != string(want) {
			t.Fatalf("got %s want %s", got, want)
		}
	})

	// F3 (small-magnitude floats use 'e' notation, matching encoding/json) is
	// covered by TestFloatEncoding_MatchesStdlib in encoding_test.go (incl. 9e-7).

	// F4: Decoder.Token returns float64 for numbers (not int64).
	t.Run("F4_token_float64", func(t *testing.T) {
		dec := NewDecoder(strings.NewReader("42"))
		tok, err := dec.Token()
		if err != nil {
			t.Fatalf("Token err: %v", err)
		}
		if _, ok := tok.(float64); !ok {
			t.Fatalf("Token returned %T, want float64", tok)
		}
	})

	// F5: nil slice/map encode as null (not [] / {}).
	t.Run("F5_nil_slice_map_null", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EscapeHTML = false
		var nilSlice []int
		var nilMap map[string]int
		gotS, _ := EncodeWithConfig(nilSlice, cfg)
		gotM, _ := EncodeWithConfig(nilMap, cfg)
		wantS, _ := stdjson.Marshal(nilSlice)
		wantM, _ := stdjson.Marshal(nilMap)
		if gotS != string(wantS) {
			t.Fatalf("nil slice: got %s want %s", gotS, wantS)
		}
		if gotM != string(wantM) {
			t.Fatalf("nil map: got %s want %s", gotM, wantM)
		}
	})
}

// ---------------------------------------------------------------------------
// Wildcards, per-call CreatePaths, extract-then-slice, reverse-step Set,
// distributed null (M5/M7/M4/M2/M6) — formerly d002_m5_verify_test.go.
// ---------------------------------------------------------------------------

// M5: bare '*' must be a wildcard for Set/Delete too (Get already treats it as
// one). Pre-fix, Set created a literal "*" key and Delete failed "path not
// found: *".
func TestD002R2_M5_BareWildcard(t *testing.T) {
	// Set *.v distributes over all object values.
	t.Run("set_wildcard_prop", func(t *testing.T) {
		doc := `{"x":{"v":1},"y":{"v":2}}`
		r, err := Set(doc, "*.v", 99)
		if err != nil {
			t.Fatalf("Set err: %v", err)
		}
		x, _ := Get(r, "x.v")
		y, _ := Get(r, "y.v")
		if x != float64(99) || y != float64(99) {
			t.Fatalf("M5: x.v=%v y.v=%v, want 99/99 (no literal '*' key should be created)", x, y)
		}
		// A literal "*" key must NOT have been created.
		if lit, _ := Get(r, "*"); lit != nil {
			// "*" is itself a wildcard now, so it distributes; ensure no key
			// literally named "*" exists by checking the object shape.
			obj, _ := Get(r, "")
			if m, ok := obj.(map[string]any); ok {
				if _, exists := m["*"]; exists {
					t.Fatalf("M5: literal '*' key was created: %v", m)
				}
			}
		}
	})

	// Set bare "*" on an object sets every value.
	t.Run("set_bare_wildcard", func(t *testing.T) {
		doc := `{"x":1,"y":2}`
		r, err := Set(doc, "*", 99)
		if err != nil {
			t.Fatalf("Set err: %v", err)
		}
		x, _ := Get(r, "x")
		y, _ := Get(r, "y")
		if x != float64(99) || y != float64(99) {
			t.Fatalf("M5: x=%v y=%v, want 99/99", x, y)
		}
	})

	// Delete *.v removes v from every object.
	t.Run("delete_wildcard_prop", func(t *testing.T) {
		doc := `{"x":{"v":1,"a":0},"y":{"v":2,"a":0}}`
		r, err := Delete(doc, "*.v")
		if err != nil {
			t.Fatalf("Delete err: %v", err)
		}
		if v, _ := Get(r, "x.v"); v != nil {
			t.Fatalf("M5: x.v still present after delete: %v", v)
		}
		if v, _ := Get(r, "y.v"); v != nil {
			t.Fatalf("M5: y.v still present after delete: %v", v)
		}
		// Non-targeted key preserved.
		if a, _ := Get(r, "x.a"); a != float64(0) {
			t.Fatalf("M5: x.a should be preserved, got %v", a)
		}
	})
}

// M7: a per-call cfg.CreatePaths=false must be honored even when the processor
// (default, CreatePaths=true) would otherwise enable it. Pre-fix the OR
// (options.CreatePaths || p.config.CreatePaths) forced it back on.
func TestD002R2_M7_PerCallCreatePaths(t *testing.T) {
	p, _ := New(DefaultConfig()) // CreatePaths=true (default)
	defer p.Close()

	cfg := DefaultConfig()
	cfg.CreatePaths = false

	// Slice end out of bounds: with CreatePaths=false this must error, not
	// silently extend the array to length 10.
	_, err := p.Set(`{"a":[1,2,3]}`, "a[0:10]", 99, cfg)
	if err == nil {
		t.Fatalf("M7: expected error for out-of-bounds slice with CreatePaths=false, got nil")
	}
	if !strings.Contains(err.Error(), "out of bounds") && !strings.Contains(err.Error(), "not found") {
		t.Logf("M7 error: %v", err)
	}

	// Sanity: with CreatePaths=true (default, no cfg) extension still happens.
	r, err := p.Set(`{"a":[1,2,3]}`, "a[0:5]", 99)
	if err != nil {
		t.Fatalf("M7: default CreatePaths=true Set should succeed: %v", err)
	}
	got, _ := Get(r, "a")
	if asStr(got) != "[99,99,99,99,99]" {
		t.Fatalf("M7: default extension got %s", asStr(got))
	}
}

// M4: Set on {extract}[slice] must actually write (previously a silent no-op
// because handleExtractThenSlice had no opSet branch).
func TestD002R2_M4_ExtractThenSliceSet(t *testing.T) {
	doc := `{"items":[{"v":[1,2,3]},{"v":[4,5,6]}]}`
	r, err := Set(doc, "items{v}[0:2]", 99)
	if err != nil {
		t.Fatalf("Set err: %v", err)
	}
	v0, _ := Get(r, "items[0].v")
	v1, _ := Get(r, "items[1].v")
	if asStr(v0) != "[99,99,3]" {
		t.Fatalf("M4: items[0].v=%s want [99,99,3]", asStr(v0))
	}
	if asStr(v1) != "[99,99,6]" {
		t.Fatalf("M4: items[1].v=%s want [99,99,6]", asStr(v1))
	}
}

// M2: Set with a reverse-step terminal slice must honor the step, not silently
// flip it to +1 (default config, CreatePaths=true, dot-notation path).
func TestD002R2_M2_ReverseStepSet(t *testing.T) {
	// [::-2] on [1,2,3,4,5] visits indices 4,2,0 -> set those to 99.
	r, err := Set(`{"a":[1,2,3,4,5]}`, "a[::-2]", 99)
	if err != nil {
		t.Fatalf("Set err: %v", err)
	}
	got, _ := Get(r, "a")
	if asStr(got) != "[99,2,99,4,99]" {
		t.Fatalf("M2: got %s want [99,2,99,4,99] (step -2 must be honored)", asStr(got))
	}
}

// M6: distributed Get must preserve explicit JSON null values (previously
// dropped). Top-level/mid-path access on a non-container still returns nil with
// no error (that contract is unchanged).
func TestD002R2_M6_DistributedGetKeepsNull(t *testing.T) {
	v, err := Get(`[{"a":null},{"a":1}]`, "a")
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if asStr(v) != "[null,1]" {
		t.Fatalf("M6: got %s want [null,1] (null must be preserved)", asStr(v))
	}
	// Contract preserved: property access on a non-container returns nil, no error.
	v2, err2 := Get(`{"a":1}`, "a.b")
	if err2 != nil || v2 != nil {
		t.Fatalf("M6 contract: Get({\"a\":1},\"a.b\") want (nil,nil), got (%v,%v)", v2, err2)
	}
}

// ---------------------------------------------------------------------------
// Struct encoding fixes (F6/F7/F8) — formerly d002_f6/f7/f8_verify_test.go.
// ---------------------------------------------------------------------------

func TestD002R2_F6_EmbeddedStruct(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EscapeHTML = false

	// Plain embedding: Inner's fields promoted to top level.
	type Inner2 struct {
		X int `json:"x"`
	}
	type Plain struct {
		Inner2
		Y int `json:"y"`
	}
	p := Plain{Inner2: Inner2{X: 1}, Y: 2}
	got, _ := EncodeWithConfig(p, cfg)
	want, _ := stdjson.Marshal(p)
	if got != string(want) {
		t.Fatalf("F6 plain: got %s want %s", got, want)
	}

	// Tagged embedding: TaggedInner has a json name -> nested, NOT promoted.
	type TaggedInner struct {
		M int `json:"m"`
	}
	type WithTag struct {
		TaggedInner `json:"tagged"`
		Y           int `json:"y"`
	}
	w := WithTag{TaggedInner: TaggedInner{M: 5}, Y: 2}
	got2, _ := EncodeWithConfig(w, cfg)
	want2, _ := stdjson.Marshal(w)
	if got2 != string(want2) {
		t.Fatalf("F6 tagged: got %s want %s", got2, want2)
	}

	// omitempty on a promoted field is honored.
	type InnerOE struct {
		Z int `json:"z,omitempty"`
	}
	type WithOE struct {
		InnerOE
		Y int `json:"y"`
	}
	got3, _ := EncodeWithConfig(WithOE{InnerOE: InnerOE{}, Y: 1}, cfg)
	if strings.Contains(got3, "z") {
		t.Fatalf("F6 omitempty on promoted field not honored: %s", got3)
	}

	// Pointer embedding.
	type WithPtr struct {
		*Inner2
		Y int `json:"y"`
	}
	got4, _ := EncodeWithConfig(WithPtr{Inner2: &Inner2{X: 7}, Y: 2}, cfg)
	want4, _ := stdjson.Marshal(WithPtr{Inner2: &Inner2{X: 7}, Y: 2})
	if got4 != string(want4) {
		t.Fatalf("F6 ptr embed: got %s want %s", got4, want4)
	}
}

func TestD002R2_F7_StringTag(t *testing.T) {
	type S struct {
		Count  int     `json:"count,string"`
		Rate   float64 `json:"rate,string"`
		Active bool    `json:"active,string"`
		Name   string  `json:"name"`
		Named  int     `json:"string"` // field NAME is "string", no option -> no wrapping
	}
	s := S{Count: 42, Rate: 1.5, Active: true, Name: "hi", Named: 7}
	cfg := DefaultConfig()
	cfg.EscapeHTML = false
	got, err := EncodeWithConfig(s, cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want, _ := stdjson.Marshal(s)
	if got != string(want) {
		t.Fatalf("F7:\n got %s\nwant %s", got, want)
	}
}

// F8: a type whose MarshalJSON has a POINTER receiver only.
type d002R2PtrMarshaler struct{ N int }

func (m *d002R2PtrMarshaler) MarshalJSON() ([]byte, error) {
	return stdjson.Marshal(map[string]int{"custom": m.N})
}

type d002R2Wrapper struct {
	Data d002R2PtrMarshaler `json:"data"`
}

func TestD002R2_F8_PtrReceiverMarshaler(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EscapeHTML = false
	got, err := EncodeWithConfig(&d002R2Wrapper{Data: d002R2PtrMarshaler{42}}, cfg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want, _ := stdjson.Marshal(&d002R2Wrapper{Data: d002R2PtrMarshaler{42}})
	if got != string(want) {
		t.Fatalf("F8: got %s want %s (pointer-receiver MarshalJSON missed)", got, want)
	}
}
