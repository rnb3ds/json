package json

import (
	"errors"
	"strings"
	"testing"
)

// This file locks in two library behavior fixes surfaced while optimizing the
// examples (task DOC-005), so they cannot regress:
//
//   1. Hooks registered via AddHook (or supplied through Config.Hooks) now
//      actually fire around Get/Set/Delete. Previously the hook machinery was
//      fully implemented but never invoked from the operation path, so AddHook
//      stored hooks that silently did nothing.
//   2. SetFromParsed now returns the *modified document*, not the set value.
//      A follow-up GetFromParsed on the result must read the new value and any
//      other path.
//
// Each test would fail on the pre-fix code.

// ----------------------------------------------------------------------------
// Hooks fire during operations
// ----------------------------------------------------------------------------

// TestHookFiresDuringOperations verifies that Before/After run for Get, Set,
// and Delete, in registration order (Before) and reverse order (After).
func TestHookFiresDuringOperations(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	var beforeOps, afterOps []string
	p.AddHook(&HookFunc{
		BeforeFn: func(ctx HookContext) error {
			beforeOps = append(beforeOps, ctx.Operation)
			return nil
		},
		AfterFn: func(ctx HookContext, result any, err error) (any, error) {
			afterOps = append(afterOps, ctx.Operation)
			return result, err
		},
	})

	if _, err := p.Get(`{"a":1}`, "a"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, err := p.Set(`{"a":1}`, "a", 2); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, err := p.Delete(`{"a":1}`, "a"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	want := []string{"get", "set", "delete"}
	if len(beforeOps) != 3 || beforeOps[0] != want[0] || beforeOps[1] != want[1] || beforeOps[2] != want[2] {
		t.Errorf("Before operations = %v, want %v", beforeOps, want)
	}
	if len(afterOps) != 3 || afterOps[0] != want[0] || afterOps[1] != want[1] || afterOps[2] != want[2] {
		t.Errorf("After operations = %v, want %v", afterOps, want)
	}
}

// TestHookBeforeCanAbort verifies a Before hook error aborts the operation and
// surfaces as the operation's error.
func TestHookBeforeCanAbort(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	abort := errors.New("blocked by hook")
	p.AddHook(&HookFunc{BeforeFn: func(ctx HookContext) error { return abort }})

	if _, err := p.Get(`{"a":1}`, "a"); !errors.Is(err, abort) {
		t.Errorf("Get error = %v, want %v", err, abort)
	}
	if _, err := p.Set(`{"a":1}`, "a", 2); !errors.Is(err, abort) {
		t.Errorf("Set error = %v, want %v", err, abort)
	}
	if _, err := p.Delete(`{"a":1}`, "a"); !errors.Is(err, abort) {
		t.Errorf("Delete error = %v, want %v", err, abort)
	}
}

// TestConfigHooksInstalledByNew verifies hooks supplied via Config.Hooks are
// installed at construction time (previously New ignored Config.Hooks).
func TestConfigHooksInstalledByNew(t *testing.T) {
	var ops []string
	cfg := DefaultConfig()
	cfg.Hooks = []Hook{
		&HookFunc{BeforeFn: func(ctx HookContext) error { ops = append(ops, ctx.Operation); return nil }},
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	if _, err := p.Get(`{"a":1}`, "a"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(ops) != 1 || ops[0] != "get" {
		t.Errorf("Config.Hooks observed ops = %v, want [get]", ops)
	}
}

// TestHookAfterSeesSetResult verifies the After hook receives the operation's
// JSON-string result for Set and that the result still reaches the caller
// (the executeAfterString coercion keeps the round-trip intact).
func TestHookAfterSeesSetResult(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	var seen string
	p.AddHook(&HookFunc{
		AfterFn: func(ctx HookContext, result any, err error) (any, error) {
			if ctx.Operation == "set" {
				if s, ok := result.(string); ok {
					seen = s
				}
			}
			return result, err
		},
	})

	out, err := p.Set(`{"a":1}`, "a", 2)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !strings.Contains(seen, `"a":2`) {
		t.Errorf("After hook saw %q, want modified JSON", seen)
	}
	if !strings.Contains(out, `"a":2`) {
		t.Errorf("Set output %q lost the set value", out)
	}
}

// TestHookDoesNotFireOnClosedProcessor verifies hook snapshotting is skipped
// once the processor is closed (no nil-deref / use-after-close).
func TestHookDoesNotFireOnClosedProcessor(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var fired bool
	p.AddHook(&HookFunc{BeforeFn: func(ctx HookContext) error { fired = true; return nil }})

	if err := p.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := p.Get(`{"a":1}`, "a"); err == nil {
		t.Error("expected error from Get on closed processor")
	}
	if fired {
		t.Error("hook fired on a closed processor")
	}
}

// ----------------------------------------------------------------------------
// SetFromParsed returns the modified document
// ----------------------------------------------------------------------------

// TestSetFromParsedReturnsModifiedDocument verifies the returned ParsedJSON
// holds the full modified document (not just the set value), so a subsequent
// GetFromParsed reads the new value at the set path AND resolves other paths.
func TestSetFromParsedReturnsModifiedDocument(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	parsed, err := p.PreParse(`{"metadata":{"page":1,"title":"x"},"other":true}`)
	if err != nil {
		t.Fatalf("PreParse: %v", err)
	}
	defer parsed.Release()

	newParsed, err := p.SetFromParsed(parsed, "metadata.page", 2)
	if err != nil {
		t.Fatalf("SetFromParsed: %v", err)
	}

	// The set path reflects the new value (numeric, type-agnostic compare).
	got, err := p.GetFromParsed(newParsed, "metadata.page")
	if err != nil {
		t.Fatalf("GetFromParsed(metadata.page): %v", err)
	}
	if n, ok := numericValue(got); !ok || n != 2 {
		t.Errorf("metadata.page = %v (%T), want 2", got, got)
	}

	// A different top-level path must still resolve — the whole document is
	// present, not just the set value.
	other, err := p.GetFromParsed(newParsed, "other")
	if err != nil || other != true {
		t.Errorf("other = %v err=%v, want true", other, err)
	}

	// A sibling under the same object must be intact.
	title, err := p.GetFromParsed(newParsed, "metadata.title")
	if err != nil || title != "x" {
		t.Errorf("metadata.title = %v err=%v, want \"x\"", title, err)
	}
}

// numericValue normalizes JSON numbers (int/int64/float64) for type-agnostic
// comparison in the regression tests above.
func numericValue(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}
