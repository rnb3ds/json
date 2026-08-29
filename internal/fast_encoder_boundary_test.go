package internal

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"
)

// ============================================================================
// Boundary tests for internal/fast_encoder.go low-coverage paths.
// ============================================================================

// --- ClearStructEncoderCache (fast_encoder.go:1435, 0% coverage) ---

func TestClearStructEncoderCache(t *testing.T) {
	type cached struct {
		A int    `json:"a"`
		B string `json:"b"`
	}
	// Encoding a struct populates the struct-encoder cache via getEncodeFn.
	if _, err := FastMarshal(cached{A: 1, B: "x"}); err != nil {
		t.Fatalf("FastMarshal err: %v", err)
	}
	// Clearing must not panic and must leave the cache usable.
	ClearStructEncoderCache()
	if _, err := FastMarshal(cached{A: 2, B: "y"}); err != nil {
		t.Fatalf("FastMarshal after clear err: %v", err)
	}
}

// --- isEmptyValue (fast_encoder.go:1580, 0% coverage) ---

func TestIsEmptyValue(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want bool
	}{
		{"zero int", 0, true},
		{"nonzero int", 42, false},
		{"empty string", "", true},
		{"nonempty string", "x", false},
		{"false", false, true},
		{"true", true, false},
		{"nil pointer", (*int)(nil), true},
		{"non-nil pointer", ptrInt(7), false},
		{"zero float", float64(0), true},
		{"nan", math.NaN(), false},
		{"empty slice", []int{}, true},
		{"nonempty slice", []int{1}, false},
		{"empty map", map[string]int{}, true},
		{"nonempty map", map[string]int{"a": 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEmptyValue(reflect.ValueOf(tt.v)); got != tt.want {
				t.Errorf("isEmptyValue(%v) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}

func ptrInt(n int) *int { return &n }

// --- FastParseFloat special-value rejection (fast_encoder.go:1298, 69%) ---

func TestFastParseFloat_SpecialValues(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{"1.5", false},
		{"42", false},
		{"-3.14", false},
		{"nan", true},
		{"NaN", true},
		{"inf", true},
		{"Infinity", true},
		{"-inf", true},
		{"+nan", true},
		{"-Inf", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			_, err := FastParseFloat([]byte(tt.in))
			if tt.wantErr && err == nil {
				t.Errorf("FastParseFloat(%q): expected error, got nil", tt.in)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("FastParseFloat(%q): unexpected error: %v", tt.in, err)
			}
		})
	}
}

// --- getEncodeFn diverse field types (fast_encoder.go:1444, 5% coverage) ---

func TestFastEncoder_DiverseFieldTypes(t *testing.T) {
	type diverse struct {
		I8  int8    `json:"i8"`
		I16 int16   `json:"i16"`
		I32 int32   `json:"i32"`
		I64 int64   `json:"i64"`
		U8  uint8   `json:"u8"`
		U16 uint16  `json:"u16"`
		U32 uint32  `json:"u32"`
		U64 uint64  `json:"u64"`
		F32 float32 `json:"f32"`
		F64 float64 `json:"f64"`
		B   bool    `json:"b"`
		S   string  `json:"s"`
	}
	v := diverse{
		I8: -8, I16: -16, I32: -32, I64: -64,
		U8: 8, U16: 16, U32: 32, U64: 64,
		F32: 1.5, F64: 2.5, B: true, S: "hi",
	}
	got, err := FastMarshal(v)
	if err != nil {
		t.Fatalf("FastMarshal err: %v", err)
	}
	want, _ := json.Marshal(v)
	if string(got) != string(want) {
		t.Errorf("diverse types mismatch:\n got %s\nwant %s", got, want)
	}
}

// --- EncodeArray error propagation (fast_encoder.go:980, 69%) ---

func TestFastEncoder_EncodeArray_ErrorPropagation(t *testing.T) {
	e := GetEncoder()
	defer PutEncoder(e)
	// An invalid json.Number inside an array forces EncodeValue to error,
	// which EncodeArray must propagate.
	err := e.EncodeArray([]any{json.Number("not-a-number")})
	if err == nil {
		t.Error("expected EncodeArray to propagate error for invalid json.Number")
	}
}

// --- GetStructEncoder + getEncodeFn (fast_encoder.go:1385/1444, 5% coverage) ---
//
// GetStructEncoder is exported public API; calling it builds per-field
// encoders via getEncodeFn for every field type, and populates the cache that
// ClearStructEncoderCache empties.

func TestGetStructEncoder_DiverseFields(t *testing.T) {
	type nested struct {
		X int `json:"x"`
	}
	type fields struct {
		I8  int8           `json:"i8"`
		I64 int64          `json:"i64"`
		U8  uint8          `json:"u8"`
		U64 uint64         `json:"u64"`
		F32 float32        `json:"f32"`
		F64 float64        `json:"f64"`
		B   bool           `json:"b"`
		S   string         `json:"s"`
		Sl  []int          `json:"sl"`
		M   map[string]int `json:"m"`
		P   *int           `json:"p"`
		N   nested         `json:"n"`
	}
	ft := reflect.TypeOf(fields{})
	info := GetStructEncoder(ft)
	if len(info) != 12 {
		t.Fatalf("GetStructEncoder returned %d fields, want 12", len(info))
	}
	// Cached lookup returns the same entry without rebuilding.
	if info2 := GetStructEncoder(ft); len(info2) != len(info) {
		t.Error("cached GetStructEncoder returned a different field count")
	}
	// Clear empties the cache; a fresh rebuild still yields the same shape.
	ClearStructEncoderCache()
	if info3 := GetStructEncoder(ft); len(info3) != 12 {
		t.Fatalf("after clear, GetStructEncoder returned %d fields, want 12", len(info3))
	}
}

// --- EncodeValue top-level type branches (fast_encoder.go:155, 71% coverage) ---

func TestFastEncoder_TopLevelValues(t *testing.T) {
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		v    any
	}{
		{"int8", int8(8)},
		{"int16", int16(16)},
		{"int32", int32(32)},
		{"uint8", uint8(8)},
		{"uint16", uint16(16)},
		{"uint32", uint32(32)},
		{"float32", float32(1.5)},
		{"[]int", []int{1, 2, 3}},
		{"[]int64", []int64{1, 2}},
		{"[]float64", []float64{1.5, 2.5}},
		{"[]byte", []byte("hello")},
		{"map[string]int64", map[string]int64{"a": 1}},
		{"map[string]float64", map[string]float64{"a": 1.5}},
		{"time", now},
		{"json.Number", json.Number("42")},
		{"json.RawMessage", json.RawMessage(`{"k":1}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FastMarshal(tt.v)
			if err != nil {
				t.Fatalf("FastMarshal(%s) err: %v", tt.name, err)
			}
			want, werr := json.Marshal(tt.v)
			if werr != nil {
				t.Fatalf("stdlib marshal(%s) err: %v", tt.name, werr)
			}
			if string(got) != string(want) {
				t.Errorf("%s mismatch:\n got %s\nwant %s", tt.name, got, want)
			}
		})
	}
}

// --- EncodeStringSlice large pre-allocation (fast_encoder.go:1008, 58%) ---

func TestFastEncoder_EncodeStringSlice_Large(t *testing.T) {
	e := GetEncoder()
	defer PutEncoder(e)
	// >8 elements exercises the large-slice pre-allocation path.
	big := make([]string, 20)
	for i := range big {
		big[i] = "x"
	}
	e.EncodeStringSlice(big)
	want, _ := json.Marshal(big)
	if string(e.buf) != string(want) {
		t.Errorf("EncodeStringSlice mismatch:\n got %s\nwant %s", e.buf, want)
	}
}

// --- EncodeBase64 buffer growth (fast_encoder.go:1123, 70%) ---

func TestFastEncoder_EncodeBase64_Large(t *testing.T) {
	e := GetEncoder()
	defer PutEncoder(e)
	// Larger than the encoder's initial 512-byte buffer forces capacity growth.
	big := make([]byte, 1024)
	for i := range big {
		big[i] = byte(i)
	}
	e.EncodeBase64(big)
	want, _ := json.Marshal(big)
	if string(e.buf) != string(want) {
		t.Errorf("EncodeBase64 mismatch:\n got %s\nwant %s", e.buf, want)
	}
}
