package json

import (
	"bytes"
	stdjson "encoding/json"
	"reflect"
	"strings"
	"testing"
)

// ============================================================================
// [D-005] Phase 1 — API Unification: package-level cfg passthrough
//
// These tests lock in the Phase 1 change: Marshal / Unmarshal / MarshalIndent /
// Valid / CompareJSON now accept an optional trailing Config, making the
// package-level API a true mirror of the Processor API.
//
// Two invariants are guarded:
//   1. Backward compatibility — calling without cfg behaves exactly as before
//      (drop-in encoding/json compatible).
//   2. Mirror — package Foo(v, cfg) produces the same result as p.Foo(v, cfg).
//
// Notes on test data:
//   - Byte-equality with encoding/json uses structs, not maps. Both libraries
//     encode structs in field-declaration order, so the comparison is stable.
//     Map key order is NOT guaranteed byte-identical (this library defaults
//     SortKeys=false; encoding/json sorts), so maps are only compared by value.
//   - stdjson aliases encoding/json to avoid shadowing the package's own name.
// ============================================================================

// unifyUser is a struct with deterministic field order in both libraries.
type unifyUser struct {
	Name   string `json:"name"`
	Age    int    `json:"age"`
	Active bool   `json:"active"`
}

func TestUnify_Marshal_NoConfig_MatchesStdlib(t *testing.T) {
	v := unifyUser{Name: "Alice", Age: 30, Active: true}
	want, err := stdjson.Marshal(v)
	if err != nil {
		t.Fatalf("stdlib Marshal: %v", err)
	}
	got, err := Marshal(v) // no cfg — must remain drop-in compatible
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("Marshal(v) without cfg drifted from encoding/json:\n got=%s\nwant=%s", got, want)
	}
}

func TestUnify_Marshal_WithConfig_AppliesCfg(t *testing.T) {
	v := unifyUser{Name: "Alice", Age: 30}

	compact, err := Marshal(v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	pretty, err := Marshal(v, PrettyConfig())
	if err != nil {
		t.Fatalf("Marshal(v, PrettyConfig()): %v", err)
	}
	if bytes.Equal(compact, pretty) {
		t.Errorf("cfg had no effect: compact==pretty (%s)", pretty)
	}
	if !bytes.Contains(pretty, []byte("\n")) {
		t.Errorf("PrettyConfig not honored: %s", pretty)
	}
}

func TestUnify_Marshal_MirrorsProcessor(t *testing.T) {
	cfg := PrettyConfig()
	v := unifyUser{Name: "Alice", Age: 30, Active: true}

	pkgOut, err := Marshal(v, cfg)
	if err != nil {
		t.Fatalf("package Marshal: %v", err)
	}
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()
	procOut, err := p.Marshal(v, cfg)
	if err != nil {
		t.Fatalf("processor Marshal: %v", err)
	}
	if !bytes.Equal(pkgOut, procOut) {
		t.Errorf("package and processor Marshal diverged:\n pkg=%s\nproc=%s", pkgOut, procOut)
	}
}

func TestUnify_Unmarshal_NoConfig_BehaviorUnchanged(t *testing.T) {
	src := `{"name":"Alice","age":30}`
	var std, ours map[string]any
	if err := stdjson.Unmarshal([]byte(src), &std); err != nil {
		t.Fatalf("stdlib Unmarshal: %v", err)
	}
	if err := Unmarshal([]byte(src), &ours); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if ours["name"] != std["name"] || ours["age"] != std["age"] {
		t.Errorf("Unmarshal without cfg diverged: got=%v want=%v", ours, std)
	}
}

func TestUnify_Unmarshal_WithConfig_Succeeds(t *testing.T) {
	src := `{"name":"Alice"}`
	var got map[string]any
	if err := Unmarshal([]byte(src), &got, DefaultConfig()); err != nil {
		t.Fatalf("Unmarshal with cfg: %v", err)
	}
	if got["name"] != "Alice" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestUnify_MarshalIndent_NoConfig_MatchesStdlib(t *testing.T) {
	v := unifyUser{Name: "Alice", Age: 30, Active: true}
	want, err := stdjson.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("stdlib MarshalIndent: %v", err)
	}
	got, err := MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("MarshalIndent without cfg drifted:\n got=%s\nwant=%s", got, want)
	}
}

func TestUnify_MarshalIndent_WithConfig_Succeeds(t *testing.T) {
	v := unifyUser{Name: "Alice"}
	got, err := MarshalIndent(v, "", "    ", DefaultConfig())
	if err != nil {
		t.Fatalf("MarshalIndent with cfg: %v", err)
	}
	if !bytes.Contains(got, []byte("    ")) {
		t.Errorf("indent not applied: %s", got)
	}
}

func TestUnify_Valid_NoConfig_BehaviorUnchanged(t *testing.T) {
	if !Valid([]byte(`{"a":1}`)) {
		t.Error("Valid should accept valid JSON")
	}
	if Valid([]byte(`{not json`)) {
		t.Error("Valid should reject invalid JSON")
	}
}

// TestUnify_Valid_WithConfig_AcceptsCfg confirms Valid accepts an optional
// Config and collapses validation/parse errors to false (its bool return type
// cannot surface them). Per-call options are honored: Processor.Valid resolves
// options from cfg and validates input via validateInputForOptions, so a
// caller-supplied MaxJSONSize / FullSecurityScan / etc. is enforced.
func TestUnify_Valid_WithConfig_AcceptsCfg(t *testing.T) {
	if !Valid([]byte(`{"a":1}`), DefaultConfig()) {
		t.Error("Valid(valid, cfg) should be true")
	}
	if Valid([]byte(`{not json`), DefaultConfig()) {
		t.Error("Valid(invalid, cfg) should be false")
	}
	if !Valid([]byte(`{"a":1}`), SecurityConfig()) {
		t.Error("Valid(valid, SecurityConfig) should be true")
	}

	// A caller-supplied MaxJSONSize is enforced on the per-call path: an input
	// larger than the configured limit is rejected even though it is valid JSON.
	small := DefaultConfig()
	small.MaxJSONSize = 2
	if Valid([]byte(`{"a":1}`), small) {
		t.Error("Valid should reject input exceeding cfg.MaxJSONSize")
	}
}

func TestUnify_CompareJSON_NoConfig_BehaviorUnchanged(t *testing.T) {
	eq, err := CompareJSON(`{"a":1}`, `{"a":1.0}`)
	if err != nil {
		t.Fatalf("CompareJSON: %v", err)
	}
	if !eq {
		t.Error("CompareJSON should treat 1 and 1.0 as equal")
	}
}

func TestUnify_CompareJSON_WithConfig_AgreesWithNoConfig(t *testing.T) {
	a, b := `{"x":[1,2,3]}`, `{"x":[1,2,3]}`
	eqPlain, err := CompareJSON(a, b)
	if err != nil {
		t.Fatalf("CompareJSON plain: %v", err)
	}
	eqCfg, err := CompareJSON(a, b, DefaultConfig())
	if err != nil {
		t.Fatalf("CompareJSON with cfg: %v", err)
	}
	if eqPlain != eqCfg {
		t.Errorf("cfg changed CompareJSON result: plain=%v cfg=%v", eqPlain, eqCfg)
	}
}

// ============================================================================
// [D-006] follow-up — Processor method mirrors for CompareJSON / MergeJSON / MergeMany
//
// These three previously existed only at package level, leaving a symmetry gap
// in the mirror principle. The mirrors are verified for correctness and for
// parity with the package-level functions (json.Foo(args, cfg) == p.Foo(args, cfg)).
//
// Map outputs are compared by parsed value (reflect.DeepEqual), never by bytes:
// the library defaults SortKeys=false, so map key order is not byte-stable.
// ============================================================================

// unifyToMap parses a JSON object string into map[string]any via the stdlib
// (numbers as float64), giving an order-independent representation for DeepEqual.
func unifyToMap(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := stdjson.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("unmarshal %q: %v", s, err)
	}
	return m
}

func TestUnify_CompareJSON_MethodMirror(t *testing.T) {
	proc, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Single-key cases keep the symmetric marshal byte-stable (SortKeys=false).
	cases := []struct {
		name      string
		a, b      string
		wantEqual bool
	}{
		{"numeric precision", `{"a":1}`, `{"a":1.0}`, true},
		{"nested equal", `{"x":[1,2,3]}`, `{"x":[1,2,3]}`, true},
		{"differ value", `{"a":1}`, `{"a":2}`, false},
		{"differ key", `{"a":1}`, `{"b":1}`, false},
	}
	for _, tc := range cases {
		got, err := proc.CompareJSON(tc.a, tc.b)
		if err != nil {
			t.Errorf("%s: proc.CompareJSON error: %v", tc.name, err)
			continue
		}
		if got != tc.wantEqual {
			t.Errorf("%s: proc.CompareJSON = %v, want %v", tc.name, got, tc.wantEqual)
		}
		// Mirror parity: method result must equal the package-level result.
		pkgEq, err := CompareJSON(tc.a, tc.b, DefaultConfig())
		if err != nil {
			t.Errorf("%s: package CompareJSON error: %v", tc.name, err)
			continue
		}
		if got != pkgEq {
			t.Errorf("%s: mirror parity broken: method=%v package=%v", tc.name, got, pkgEq)
		}
	}
}

func TestUnify_MergeJSON_MethodMirror(t *testing.T) {
	proc, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	a := `{"a":1,"nested":{"x":1}}`
	b := `{"b":2,"nested":{"y":2}}`

	got, err := proc.MergeJSON(a, b)
	if err != nil {
		t.Fatalf("proc.MergeJSON: %v", err)
	}

	// Correctness: union deep-merge contains keys from both sides, nested merged.
	want := map[string]any{
		"a":      1.0,
		"b":      2.0,
		"nested": map[string]any{"x": 1.0, "y": 2.0},
	}
	if !reflect.DeepEqual(unifyToMap(t, got), want) {
		t.Errorf("proc.MergeJSON value mismatch:\n got =%v\n want=%v", unifyToMap(t, got), want)
	}

	// Mirror parity: method output equals the package-level output (by value).
	pkgGot, err := MergeJSON(a, b)
	if err != nil {
		t.Fatalf("package MergeJSON: %v", err)
	}
	if !reflect.DeepEqual(unifyToMap(t, got), unifyToMap(t, pkgGot)) {
		t.Errorf("MergeJSON mirror parity broken:\n method=%s\n pkg   =%s", got, pkgGot)
	}
}

func TestUnify_MergeMany_MethodMirror(t *testing.T) {
	proc, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	jsons := []string{`{"a":1}`, `{"b":2}`, `{"c":3}`}
	got, err := proc.MergeMany(jsons)
	if err != nil {
		t.Fatalf("proc.MergeMany: %v", err)
	}

	// Correctness: all keys folded in.
	want := map[string]any{"a": 1.0, "b": 2.0, "c": 3.0}
	if !reflect.DeepEqual(unifyToMap(t, got), want) {
		t.Errorf("proc.MergeMany value mismatch:\n got =%v\n want=%v", unifyToMap(t, got), want)
	}

	// Mirror parity with the package-level function.
	pkgGot, err := MergeMany(jsons)
	if err != nil {
		t.Fatalf("package MergeMany: %v", err)
	}
	if !reflect.DeepEqual(unifyToMap(t, got), unifyToMap(t, pkgGot)) {
		t.Errorf("MergeMany mirror parity broken:\n method=%s\n pkg   =%s", got, pkgGot)
	}

	// Contract parity: fewer than 2 inputs errors on the method too.
	if _, err := proc.MergeMany([]string{`{"a":1}`}); err == nil {
		t.Error("proc.MergeMany(<2) should error, matching the package contract")
	}
}

// ============================================================================
// [D-005] Phase 2 — per-call Config is now actually enforced
//
// Phase 1 made package-level functions ACCEPT cfg; Phase 2 closes the gap that
// cfg was silently ignored for security limits. validateInput previously used
// the processor's baked-in config; validateInputForOptions now honors a
// caller-supplied cfg across Get/Set/Delete/Valid/Parse/GetMultiple/
// SetMultiple/Prettify/Compact/ValidateSchema/PreParse/WarmupCache.
//
// These tests assert the enforcement end-to-end via the package-level API,
// using MaxNestingDepthSecurity as the signal (its valid range [10,200] allows
// a small per-call limit that a deeply nested payload exceeds). The no-cfg
// path must keep accepting the same payload (default limit 200).
// ============================================================================

// nestedJSON builds a valid JSON object nested `depth` levels deep:
// {"a":{"a":...{"a":1}...}}. depth=50 is well under the default limit (200)
// but exceeds a per-call limit of 15.
func nestedJSON(depth int) string {
	return strings.Repeat(`{"a":`, depth) + `1` + strings.Repeat(`}`, depth)
}

// perCallNestingCfg returns a Config whose nesting limit (15) a 50-deep
// document exceeds but which survives Validate's clamp (min 10, max 200).
func perCallNestingCfg() Config {
	cfg := DefaultConfig()
	cfg.MaxNestingDepthSecurity = 15
	return cfg
}

func TestUnify_Valid_EnforcesPerCallCfg(t *testing.T) {
	nested := nestedJSON(50) // depth 50, valid under the default limit (200)

	// No cfg: default processor accepts it.
	if !Valid([]byte(nested)) {
		t.Error("Valid without cfg should accept 50-deep JSON (default limit 200)")
	}
	// Per-call cfg with a 15-deep limit must reject the 50-deep payload.
	if Valid([]byte(nested), perCallNestingCfg()) {
		t.Error("Valid(nested, cfg{nesting:15}) should reject 50-deep JSON; per-call cfg was not enforced")
	}
}

func TestUnify_Get_EnforcesPerCallCfg(t *testing.T) {
	nested := nestedJSON(50)

	// No cfg: Get succeeds (validateOperationInput uses processor default).
	if _, err := Get(nested, "a"); err != nil {
		t.Errorf("Get without cfg should accept 50-deep JSON: %v", err)
	}
	// Per-call cfg: the 50-deep payload is rejected before navigation.
	// This exercises the prepareOperation -> validateOperationInput path
	// shared by Get/Set/Delete.
	if _, err := Get(nested, "a", perCallNestingCfg()); err == nil {
		t.Error("Get(nested, \"a\", cfg{nesting:15}) should reject; per-call cfg was not enforced")
	}
}

func TestUnify_Parse_EnforcesPerCallCfg(t *testing.T) {
	nested := nestedJSON(50)
	var v any

	// No cfg fast path: parses fine.
	if err := Parse(nested, &v); err != nil {
		t.Errorf("Parse without cfg should accept 50-deep JSON: %v", err)
	}
	// Per-call cfg slow path: rejected before parsing.
	if err := Parse(nested, &v, perCallNestingCfg()); err == nil {
		t.Error("Parse(nested, &v, cfg{nesting:15}) should reject; per-call cfg was not enforced")
	}
}

func TestUnify_Set_EnforcesPerCallCfg(t *testing.T) {
	nested := nestedJSON(50)

	// No cfg: Set succeeds.
	if _, err := Set(nested, "a", 2); err != nil {
		t.Errorf("Set without cfg should accept 50-deep JSON: %v", err)
	}
	// Per-call cfg: rejected.
	if _, err := Set(nested, "a", 2, perCallNestingCfg()); err == nil {
		t.Error("Set(nested, \"a\", 2, cfg{nesting:15}) should reject; per-call cfg was not enforced")
	}
}

func TestUnify_NoCfg_Path_UnaffectedByProcessorConfig(t *testing.T) {
	// A processor built with a tight nesting limit must STILL enforce it when
	// its methods are called WITHOUT a per-call cfg (the fix must not loosen a
	// SecurityConfig processor's own limits). 50-deep exceeds 15.
	p, err := New(perCallNestingCfg())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()
	if _, err := p.Get(nestedJSON(50), "a"); err == nil {
		t.Error("SecurityConfig processor should reject 50-deep JSON on no-cfg Get (its own limit is 15)")
	}
}

// TestUnify_Encode_DeprecatedStillWorks locks in that Encode (deprecated in
// Phase 2 as identical to EncodeWithConfig) still functions and stays a mirror
// of EncodeWithConfig. Removal is deferred to a future major version.
func TestUnify_Encode_DeprecatedStillWorks(t *testing.T) {
	v := unifyUser{Name: "Alice", Age: 30}
	out, err := Encode(v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	want, err := EncodeWithConfig(v)
	if err != nil {
		t.Fatalf("EncodeWithConfig: %v", err)
	}
	if out != want {
		t.Errorf("Encode diverged from EncodeWithConfig:\n Encode=%q\n EWC   =%q", out, want)
	}
}

// ============================================================================
// [D-005] Phase 3 — Compact naming divergence resolved
//
// Phase 3 closes the one place the mirror principle was broken: package-level
// Compact (buffer form, encoding/json-compatible) and Processor.Compact (string
// form) shared a name but differed in signature AND behavior — json.Compact was
// the mirror of p.CompactBuffer, while p.Compact had no package-level mirror.
//
// Fix (additive): CompactString is introduced as the package-level mirror of
// Processor.Compact (json.CompactString(s) ↔ p.Compact(s)), symmetric with
// Prettify mirroring p.Prettify. json.Compact(dst, src) still mirrors
// p.CompactBuffer. No existing symbol changed.
// ============================================================================

func TestUnify_CompactString_RemovesWhitespace(t *testing.T) {
	in := "{\n\t\"name\": \"Alice\",\n\t\"age\": 30\n}"
	got, err := CompactString(in)
	if err != nil {
		t.Fatalf("CompactString: %v", err)
	}
	// Compact form must contain no insignificant whitespace. Map key order is
	// NOT byte-stable (see file header), so verify compactness by whitespace
	// absence and equivalence by parsed value — not by exact bytes.
	if strings.ContainsAny(got, "\n\t") {
		t.Errorf("CompactString left whitespace in output: %q", got)
	}
	var gotVal, wantVal any
	if err := stdjson.Unmarshal([]byte(got), &gotVal); err != nil {
		t.Fatalf("CompactString output is not valid JSON: %v", err)
	}
	if err := stdjson.Unmarshal([]byte(`{"name":"Alice","age":30}`), &wantVal); err != nil {
		t.Fatalf("expected value is not valid JSON: %v", err)
	}
	if !reflect.DeepEqual(gotVal, wantVal) {
		t.Errorf("CompactString value mismatch: got %q", got)
	}
}

func TestUnify_CompactString_MirrorsProcessor(t *testing.T) {
	// Map key order is non-deterministic when SortKeys=false (see file header),
	// so the package and processor outputs are compared by parsed value, not by
	// byte-equality — matching the contract this file documents for maps.
	in := "{\n  \"name\": \"Alice\",\n  \"age\": 30\n}"
	cfg := DefaultConfig()
	cfg.PreserveNumbers = true

	pkgOut, err := CompactString(in, cfg)
	if err != nil {
		t.Fatalf("package CompactString: %v", err)
	}
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()
	procOut, err := p.Compact(in, cfg)
	if err != nil {
		t.Fatalf("processor Compact: %v", err)
	}

	var pkgVal, procVal any
	if err := stdjson.Unmarshal([]byte(pkgOut), &pkgVal); err != nil {
		t.Fatalf("unmarshal package CompactString output: %v", err)
	}
	if err := stdjson.Unmarshal([]byte(procOut), &procVal); err != nil {
		t.Fatalf("unmarshal processor Compact output: %v", err)
	}
	if !reflect.DeepEqual(pkgVal, procVal) {
		t.Errorf("package CompactString and processor Compact diverged by value:\n pkg=%q\nproc=%q", pkgOut, procOut)
	}
}

// TestUnify_Compact_BufferVsString_Distinct locks in that the buffer form
// (json.Compact, mirror of p.CompactBuffer) and the string form (json.CompactString,
// mirror of p.Compact) coexist without colliding — the original divergence is
// resolved by naming, not by repurposing either symbol.
func TestUnify_Compact_BufferVsString_Distinct(t *testing.T) {
	src := []byte(`{ "a": 1 }`)

	// Buffer form (encoding/json-compatible).
	var buf bytes.Buffer
	if err := Compact(&buf, src); err != nil {
		t.Fatalf("Compact(buffer): %v", err)
	}

	// String form.
	s, err := CompactString(string(src))
	if err != nil {
		t.Fatalf("CompactString: %v", err)
	}
	if buf.String() != s {
		t.Errorf("buffer and string compact forms disagree: buf=%q str=%q", buf.String(), s)
	}
}

// ============================================================================
// [D-005] Phase 4 — iterate family cfg unified
//
// Before Phase 4 the iterate family was split: 5 package-level functions
// (Foreach, ForeachWithPath, ForeachWithPathAndControl, ForeachReturn,
// ForeachNested) accepted cfg, but 3 (ForeachWithError, ForeachNestedWithError,
// ForeachWithPathAndIterator) did not, and none of the *methods* took per-call
// cfg. Phase 4 adds a trailing cfg ...Config to all 8 Processor methods (threaded
// to the internal Get) and to the 3 missing package wrappers, making both layers
// uniform. cfg flows to Get, so per-call security limits apply (Phase 2 semantics).
// ============================================================================

func TestUnify_ForeachWithError_EnforcesPerCallCfg(t *testing.T) {
	nested := nestedJSON(50) // 50-deep, valid under default limit (200)

	// Method, no cfg: default processor accepts 50-deep.
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()
	if err := p.ForeachWithError(nested, ".", func(key any, item *IterableValue) error { return nil }); err != nil {
		t.Errorf("p.ForeachWithError without cfg should accept 50-deep JSON: %v", err)
	}
	// Method, per-call cfg (limit 15): rejected before iteration.
	if err := p.ForeachWithError(nested, ".", func(key any, item *IterableValue) error { return nil }, perCallNestingCfg()); err == nil {
		t.Error("p.ForeachWithError(nested, cfg{nesting:15}) should reject; per-call cfg was not enforced")
	}
}

func TestUnify_ForeachWithError_PackageForwardsCfg(t *testing.T) {
	nested := nestedJSON(50)

	// Package-level, no cfg: accepts.
	if err := ForeachWithError(nested, ".", func(key any, item *IterableValue) error { return nil }); err != nil {
		t.Errorf("package ForeachWithError without cfg should accept 50-deep JSON: %v", err)
	}
	// Package-level, per-call cfg (limit 15): rejected — confirms the wrapper
	// forwards cfg into the method (and thus into Get's per-call validation).
	if err := ForeachWithError(nested, ".", func(key any, item *IterableValue) error { return nil }, perCallNestingCfg()); err == nil {
		t.Error("package ForeachWithError(nested, cfg{nesting:15}) should reject; cfg was not forwarded")
	}
}

// TestUnify_Foreach_FamilyAcceptsCfg is a compile-time/behavior guard that every
// package-level iterate function now accepts a trailing cfg without changing the
// no-cfg behavior. It exercises one representative of each formerly-cfg-less
// wrapper plus a formerly-cfg-accepting one.
func TestUnify_Foreach_FamilyAcceptsCfg(t *testing.T) {
	data := `{"a":1,"b":2}`
	called := 0

	// Formerly cfg-less wrappers now accept cfg.
	if err := ForeachWithError(data, ".", func(key any, item *IterableValue) error { called++; return nil }, DefaultConfig()); err != nil {
		t.Errorf("ForeachWithError with cfg: %v", err)
	}
	if err := ForeachNestedWithError(data, func(key any, item *IterableValue) error { return nil }, DefaultConfig()); err != nil {
		t.Errorf("ForeachNestedWithError with cfg: %v", err)
	}
	if err := ForeachWithPathAndIterator(data, ".", func(key any, item *IterableValue, currentPath string) IteratorControl { return IteratorNormal }, DefaultConfig()); err != nil {
		t.Errorf("ForeachWithPathAndIterator with cfg: %v", err)
	}
	// Formerly cfg-accepting wrapper still works.
	Foreach(data, func(key any, item *IterableValue) {}, DefaultConfig())

	if called != 2 {
		t.Errorf("ForeachWithError callback count = %d, want 2", called)
	}
}

// ============================================================================
// [D-005] Phase 5 — JSONL/stream family cfg unified
//
// Before Phase 5 the 11 JSONL package wrappers (StreamJSONL, StreamJSONLParallel,
// StreamJSONLParallelWithContext, StreamJSONLChunked, ForeachJSONL, MapJSONL,
// ReduceJSONL, FilterJSONL, StreamJSONLFile, CollectJSONL, FirstJSONL) selected
// the default processor and had no way to honor a per-call Config. Phase 5 routes
// them through processorForCfg: with cfg omitted the default processor is used
// (behavior unchanged); with cfg supplied a config-cached processor is selected,
// whose baked-in JSONL settings (buffer/line sizes, memory limit, nesting cap)
// reflect cfg.
//
// The behavioral signal is MaxNestingDepthSecurity: Processor.StreamJSONL reads
// p.config.MaxNestingDepthSecurity and rejects lines deeper than it
// (processor_streamjsonl.go), so a cfg-cached processor with nesting=15 rejects a
// 50-deep line that the default processor (limit 200) accepts.
// ============================================================================

func TestUnify_StreamJSONL_NoCfg_BehaviorUnchanged(t *testing.T) {
	data := `{"id":1,"name":"a"}` + "\n" + `{"id":2,"name":"b"}` + "\n"
	var seen []int
	err := StreamJSONL(strings.NewReader(data), func(lineNum int, item *IterableValue) error {
		seen = append(seen, item.GetInt("id"))
		return nil
	})
	if err != nil {
		t.Fatalf("StreamJSONL without cfg: %v", err)
	}
	if len(seen) != 2 || seen[0] != 1 || seen[1] != 2 {
		t.Errorf("StreamJSONL processed wrong ids: %v", seen)
	}
}

func TestUnify_StreamJSONL_EnforcesPerCallCfg(t *testing.T) {
	// One 50-deep line: accepted by the default processor (limit 200).
	nested := nestedJSON(50)
	stream := nested + "\n"

	if err := StreamJSONL(strings.NewReader(stream), func(lineNum int, item *IterableValue) error { return nil }); err != nil {
		t.Errorf("StreamJSONL without cfg should accept 50-deep line: %v", err)
	}
	// Same stream with a per-call cfg (nesting 15): the config-cached processor
	// rejects the line — confirming cfg flows into processor selection.
	if err := StreamJSONL(strings.NewReader(stream), func(lineNum int, item *IterableValue) error { return nil }, perCallNestingCfg()); err == nil {
		t.Error("StreamJSONL(nested, cfg{nesting:15}) should reject; per-call cfg was not enforced")
	}
}

func TestUnify_CollectJSONL_AcceptsCfgAndMirrors(t *testing.T) {
	data := `{"x":1}` + "\n" + `{"x":2}` + "\n"

	pkgPlain, err := CollectJSONL(strings.NewReader(data))
	if err != nil {
		t.Fatalf("CollectJSONL no cfg: %v", err)
	}
	pkgCfg, err := CollectJSONL(strings.NewReader(data), DefaultConfig())
	if err != nil {
		t.Fatalf("CollectJSONL with cfg: %v", err)
	}
	if len(pkgPlain) != 2 || len(pkgCfg) != 2 {
		t.Errorf("CollectJSONL counts wrong: plain=%d cfg=%d", len(pkgPlain), len(pkgCfg))
	}
	// Mirror: package CollectJSONL(r, cfg) matches a processor built from cfg.
	p, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()
	procOut, err := p.CollectJSONL(strings.NewReader(data))
	if err != nil {
		t.Fatalf("processor CollectJSONL: %v", err)
	}
	if len(procOut) != len(pkgCfg) {
		t.Errorf("package and processor CollectJSONL diverged: pkg=%d proc=%d", len(pkgCfg), len(procOut))
	}
}

// TestUnify_JSONL_FamilyAcceptsCfg is a smoke guard that every formerly-cfg-less
// JSONL wrapper now accepts a trailing cfg and still produces correct results.
func TestUnify_JSONL_FamilyAcceptsCfg(t *testing.T) {
	data := `{"n":1}` + "\n" + `{"n":2}` + "\n"
	r := strings.NewReader

	vals, err := MapJSONL(r(data), func(lineNum int, item *IterableValue) (any, error) {
		return item.GetInt("n"), nil
	}, DefaultConfig())
	if err != nil {
		t.Errorf("MapJSONL with cfg: %v", err)
	}
	if len(vals) != 2 {
		t.Errorf("MapJSONL count = %d, want 2", len(vals))
	}

	item, ok, err := FirstJSONL(r(data), func(item *IterableValue) bool { return true }, DefaultConfig())
	if err != nil || !ok || item == nil {
		t.Errorf("FirstJSONL with cfg: ok=%v err=%v", ok, err)
	}

	filt, err := FilterJSONL(r(data), func(item *IterableValue) bool { return true }, DefaultConfig())
	if err != nil {
		t.Errorf("FilterJSONL with cfg: %v", err)
	}
	if len(filt) != 2 {
		t.Errorf("FilterJSONL count = %d, want 2", len(filt))
	}

	acc, err := ReduceJSONL(r(data), 0, func(acc any, item *IterableValue) any {
		return acc.(int) + item.GetInt("n")
	}, DefaultConfig())
	if err != nil {
		t.Errorf("ReduceJSONL with cfg: %v", err)
	}
	if acc.(int) != 3 {
		t.Errorf("ReduceJSONL sum = %v, want 3", acc)
	}

	if err := ForeachJSONL(r(data), func(lineNum int, item *IterableValue) error { return nil }, DefaultConfig()); err != nil {
		t.Errorf("ForeachJSONL with cfg: %v", err)
	}
	if err := StreamJSONLChunked(r(data), 2, func(chunk []*IterableValue) error { return nil }, DefaultConfig()); err != nil {
		t.Errorf("StreamJSONLChunked with cfg: %v", err)
	}
	if err := StreamJSONLParallel(r(data), 2, func(lineNum int, item *IterableValue) error { return nil }, DefaultConfig()); err != nil {
		t.Errorf("StreamJSONLParallel with cfg: %v", err)
	}
}
