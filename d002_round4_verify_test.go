package json

import (
	"strings"
	"testing"
)

// [D-002 第四轮] 回归测试:锁定本轮 6 项正确性修复。
// 移除任一修复,对应用例应失败(panic 或断言不通过)。

// TestD002R4_NumberPreservedInDeepCopy pins C1 (helpers.go deep-copy) + C2 (encoding.go encoder):
// in PreserveNumbers mode, Number values must survive Get/GetFromParsed and Parse-into-map
// as numeric types, not be corrupted to strings.
func TestD002R4_NumberPreservedInDeepCopy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PreserveNumbers = true
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	in := `{"big":9007199254740993,"arr":[1,2,3]}`

	// C1: GetFromParsed returns Number (deep-copy path safeCopyResult).
	pp, err := p.PreParse(in, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pp.Release()
	if v, _ := p.GetFromParsed(pp, "big"); v == nil {
		t.Fatal("GetFromParsed(big) nil")
	} else if _, ok := v.(Number); !ok {
		t.Errorf("C1 GetFromParsed(big) = %T, want Number (was string before fix)", v)
	}

	// C1: Get cache-hit returns Number (deepCopySubtree on the cached Number).
	p.Get(in, "big", cfg) // prime cache
	if v, _ := p.Get(in, "big", cfg); v == nil {
		t.Fatal("Get(big) nil")
	} else if _, ok := v.(Number); !ok {
		t.Errorf("C1 Get(big) cache-hit = %T, want Number (was string before fix)", v)
	}

	// C2: Parse into *map[string]any yields a numeric type, not a string.
	var m map[string]any
	if err := p.Parse(in, &m, cfg); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["big"].(string); ok {
		t.Errorf("C2 Parse(*map).big = string (corrupted), want numeric type (was string before fix)")
	}
}

// TestD002R4_NonBMPEscapeSurrogatePair pins C3 (encoding.go writeUnicodeEscape):
// EscapeUnicode must emit a UTF-16 surrogate pair for non-BMP runes, not truncate to 16 bits.
func TestD002R4_NonBMPEscapeSurrogatePair(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EscapeUnicode = true
	out, err := Marshal("😀", cfg)
	if err != nil {
		t.Fatal(err)
	}
	// U+1F600 must encode as the surrogate pair 😀, not the truncated .
	if !strings.Contains(string(out), "\\ud83d\\ude00") {
		t.Errorf("C3 emoji escape = %s, want surrogate pair \\ud83d\\ude00 (was \\uf600 before fix)", string(out))
	}
}

// TestD002R4_CompiledPathReverseSlice pins C4 (internal/compiled_path.go applySlice):
// CompiledPath [::-1] must reverse the array, not return empty.
func TestD002R4_CompiledPathReverseSlice(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	cp, err := p.CompilePath("[::-1]")
	if err != nil {
		t.Fatalf("CompilePath([::-1]) err=%v", err)
	}
	rv, err := cp.Get([]any{"a", "b", "c", "d"})
	if err != nil {
		t.Fatalf("CompiledPath [::-1] err=%v", err)
	}
	s, ok := rv.([]any)
	if !ok || len(s) != 4 || s[0] != "d" || s[3] != "a" {
		t.Errorf("C4 CompiledPath [::-1] = %v, want [d c b a] (was [] before fix)", rv)
	}
}

// TestD002R4_ParallelIteratorMapNoMutex pins the Map mutex removal:
// distinct-index writes are safe without a mutex; result must be correct.
func TestD002R4_ParallelIteratorMapNoMutex(t *testing.T) {
	it := NewParallelIterator([]any{1, 2, 3, 4, 5})
	defer it.Close()
	res, err := it.Map(func(i int, v any) (any, error) { return i * 2, nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 5 || res[0] != 0 || res[4] != 8 {
		t.Errorf("Map result = %v, want [0 2 4 6 8]", res)
	}
}
