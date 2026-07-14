package json

import (
	"reflect"
	"testing"
)

// TestGet_DefaultCacheCopiesOnHit verifies the default (safe) behavior: two
// cache-hit Gets return independent deep copies, so mutating one cannot affect
// the other. This guards against regressing the default safety guarantee.
func TestGet_DefaultCacheCopiesOnHit(t *testing.T) {
	p, _ := New()
	defer p.Close()

	jsonStr := `{"items":[1,2,3]}`

	// Prime the result cache (first call is a cache miss).
	_, _ = p.Get(jsonStr, "items")

	// Two cache-hit Gets must each deep-copy the cached slice.
	a, err := p.Get(jsonStr, "items")
	if err != nil {
		t.Fatalf("Get #1: %v", err)
	}
	b, err := p.Get(jsonStr, "items")
	if err != nil {
		t.Fatalf("Get #2: %v", err)
	}

	aArr, aOK := a.([]any)
	bArr, bOK := b.([]any)
	if !aOK || !bOK {
		t.Fatalf("expected []any, got %T and %T", a, b)
	}
	if reflect.ValueOf(aArr).Pointer() == reflect.ValueOf(bArr).Pointer() {
		t.Error("default cache: two cache-hit Gets share a backing array; expected independent deep copies")
	}
}

// TestGet_SharedCacheReturnsSameReference verifies that with
// CacheSharedResults enabled, cache-hit Gets skip the defensive deep copy and
// return the shared cached value directly (pointer identity).
func TestGet_SharedCacheReturnsSameReference(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CacheSharedResults = true
	p, _ := New(cfg)
	defer p.Close()

	jsonStr := `{"items":[1,2,3]}`

	// Prime the result cache.
	_, _ = p.Get(jsonStr, "items")

	a, err := p.Get(jsonStr, "items")
	if err != nil {
		t.Fatalf("Get #1: %v", err)
	}
	b, err := p.Get(jsonStr, "items")
	if err != nil {
		t.Fatalf("Get #2: %v", err)
	}

	aArr, aOK := a.([]any)
	bArr, bOK := b.([]any)
	if !aOK || !bOK {
		t.Fatalf("expected []any, got %T and %T", a, b)
	}
	if reflect.ValueOf(aArr).Pointer() != reflect.ValueOf(bArr).Pointer() {
		t.Error("CacheSharedResults: two cache-hit Gets returned distinct backing arrays; expected a shared reference")
	}
}

// TestGet_SharedCacheStillReturnsCorrectValue ensures the opt-in fast path
// returns the right value, not just any reference.
func TestGet_SharedCacheStillReturnsCorrectValue(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CacheSharedResults = true
	p, _ := New(cfg)
	defer p.Close()

	jsonStr := `{"user":{"name":"Alice","roles":["admin","user"]}}`

	roles, err := p.Get(jsonStr, "user.roles")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	arr := roles.([]any)
	if len(arr) != 2 || arr[0] != "admin" || arr[1] != "user" {
		t.Errorf("unexpected roles: %v", arr)
	}

	name, err := p.Get(jsonStr, "user.name")
	if err != nil {
		t.Fatalf("Get name: %v", err)
	}
	if name != "Alice" {
		t.Errorf("unexpected name: %v", name)
	}
}

// TestGetFromParsed_SharedCacheSkipsCopy mirrors the Get behavior for the
// pre-parsed read path.
func TestGetFromParsed_SharedCacheSkipsCopy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CacheSharedResults = true
	p, _ := New(cfg)
	defer p.Close()

	jsonStr := `{"items":[1,2,3]}`
	parsed, err := p.PreParse(jsonStr)
	if err != nil {
		t.Fatalf("PreParse: %v", err)
	}
	defer parsed.Release()

	a, err := p.GetFromParsed(parsed, "items")
	if err != nil {
		t.Fatalf("GetFromParsed #1: %v", err)
	}
	b, err := p.GetFromParsed(parsed, "items")
	if err != nil {
		t.Fatalf("GetFromParsed #2: %v", err)
	}
	aArr := a.([]any)
	bArr := b.([]any)
	if reflect.ValueOf(aArr).Pointer() != reflect.ValueOf(bArr).Pointer() {
		t.Error("CacheSharedResults: GetFromParsed returned distinct backing arrays; expected a shared reference")
	}
}

// TestCacheSharedResults_InConfigFieldRegistry guards against the
// maintainability hazard of forgetting to register a new Config field in
// configFieldList: two configs differing only in CacheSharedResults must
// compare unequal and hash differently, so their cache keys never collide.
func TestCacheSharedResults_InConfigFieldRegistry(t *testing.T) {
	a := DefaultConfig()
	b := DefaultConfig()
	b.CacheSharedResults = true

	if configFieldsEqual(a, b) {
		t.Error("configFieldsEqual must distinguish CacheSharedResults (field missing from configFieldList?)")
	}
	if hashConfig(a) == hashConfig(b) {
		t.Error("hashConfig must differ for CacheSharedResults (field missing from configFieldList?)")
	}

	// isDefaultConfig must reject the shared-results config as non-default.
	if isDefaultConfig(b) {
		t.Error("isDefaultConfig must return false when CacheSharedResults is true")
	}
}
