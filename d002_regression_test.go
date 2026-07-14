package json

import (
	stdjson "encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/cybergodev/json/internal"
)

// This file locks in the behavior fixes from task D-002 so they cannot regress.
// Each test corresponds to a specific finding (C1, M1–M9) and would fail on the
// pre-fix code.

// asStr renders a Get result with %v so assertions are shape-agnostic.
func asStr(v any) string { return d002fmt(v) }

func d002fmt(v any) string {
	if v == nil {
		return "<nil>"
	}
	b, err := stdjson.Marshal(v)
	if err != nil {
		return "<unmarshalable>"
	}
	return string(b)
}

// M2: non-last-segment array slice must honor step (a[0:5:2].b → indices 0,2,4).
func TestD002_NonLastSegmentSlice_HonorsStep(t *testing.T) {
	const doc = `{"arr":[{"b":0},{"b":1},{"b":2},{"b":3},{"b":4}]}`
	got, err := Get(doc, "arr[0:5:2].b")
	if err != nil {
		t.Fatalf("Get err: %v", err)
	}
	if want := "[0,2,4]"; asStr(got) != want {
		t.Fatalf("M2 step ignored: got %q want %q", asStr(got), want)
	}
}

// M3: {extract}[slice] must honor step and reverse.
func TestD002_ExtractThenSlice_HonorsStepAndReverse(t *testing.T) {
	const doc = `{"items":[{"id":1},{"id":2},{"id":3},{"id":4},{"id":5}]}`

	got, err := Get(doc, "items{id}[0:5:2]")
	if err != nil {
		t.Fatalf("Get step err: %v", err)
	}
	if want := "[1,3,5]"; asStr(got) != want {
		t.Fatalf("M3 step ignored: got %q want %q", asStr(got), want)
	}

	rev, err := Get(doc, "items{id}[::-1]")
	if err != nil {
		t.Fatalf("Get reverse err: %v", err)
	}
	if want := "[5,4,3,2,1]"; asStr(rev) != want {
		t.Fatalf("M3 reverse ignored: got %q want %q", asStr(rev), want)
	}
}

// M4: {extract}[slice] delete must honor step.
func TestD002_ExtractThenSliceDelete_HonorsStep(t *testing.T) {
	const doc = `{"items":[{"tags":["a","b","c","d","e"]}]}`
	out, err := Delete(doc, "items{tags}[0:5:2]")
	if err != nil {
		t.Fatalf("Delete err: %v", err)
	}
	got, err := Get(out, "items[0].tags")
	if err != nil {
		t.Fatalf("Get after delete err: %v", err)
	}
	// Deleting indices 0,2,4 leaves ["b","d"]. Pre-fix this emptied the array.
	if want := `["b","d"]`; asStr(got) != want {
		t.Fatalf("M4 delete step ignored: got %q want %q", asStr(got), want)
	}
}

// M5: convertToUint64 must accept json.Number values in (MaxInt64, MaxUint64].
func TestD002_ConvertToUint64_LargeJSONNumber(t *testing.T) {
	cases := []string{
		"9223372036854775808",  // MaxInt64 + 1
		"18446744073709551615", // MaxUint64
	}
	for _, s := range cases {
		v, ok := convertToUint64(stdjson.Number(s))
		if !ok {
			t.Fatalf("M5: convertToUint64(%s) rejected a valid uint64 (pre-fix behavior)", s)
		}
		if got, want := v, mustParseUint64(s); got != want {
			t.Fatalf("M5: convertToUint64(%s) = %d, want %d", s, got, want)
		}
	}
	// Negative still rejected.
	if _, ok := convertToUint64(stdjson.Number("-1")); ok {
		t.Fatalf("M5: negative json.Number must not convert to uint64")
	}
}

func mustParseUint64(s string) uint64 {
	var u uint64
	for _, c := range s {
		u = u*10 + uint64(c-'0')
	}
	return u
}

// M8: Marshal must reject NaN/Inf (invalid JSON) instead of emitting "NaN"/"+Inf".
func TestD002_Marshal_RejectsNaNAndInf(t *testing.T) {
	for name, val := range map[string]float64{
		"NaN":  math.NaN(),
		"+Inf": math.Inf(1),
		"-Inf": math.Inf(-1),
	} {
		_, err := Marshal(struct{ X float64 }{X: val})
		if err == nil {
			t.Fatalf("M8: Marshal accepted %s (would emit invalid JSON)", name)
		}
	}
}

// C1: streaming Decoder must enforce MaxJSONSize on an unterminated string, not
// grow the buffer without bound.
func TestD002_StreamingDecoder_EnforcesMaxBytes(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxJSONSize = 1024
	huge := `"` + strings.Repeat("A", 1_000_000) // 1MB unterminated string

	dec := NewDecoder(strings.NewReader(huge), cfg)
	var v any
	err := dec.Decode(&v)
	if err == nil {
		t.Fatalf("C1: streaming Decode accepted an oversized unterminated string")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("C1: expected size-limit error, got: %v", err)
	}
}

// M1: cache Get must agree with Set on long (>MaxCacheKeyLength) keys.
func TestD002_Cache_LongKeyRoundTrip(t *testing.T) {
	cm := internal.NewCacheManager(true, 10000, 0)
	longKey := strings.Repeat("k", 2048) // > MaxCacheKeyLength (1024)
	cm.Set(longKey, "hit")
	v, ok := cm.Get(longKey)
	if !ok {
		t.Fatalf("M1: long cache key write-only (pre-fix: Set and Get shard mismatched)")
	}
	if s, _ := v.(string); s != "hit" {
		t.Fatalf("M1: long key round-trip mismatch: got %v want hit", v)
	}
}
