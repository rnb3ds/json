package json

import (
	"context"
	"strings"
	"testing"
)

// ============================================================================
// Boundary tests for processor.go / processor_iterate.go / processor_streamjsonl.go
// low-coverage paths: closed-processor error guards, invalid JSONL, and
// context cancellation.
// ============================================================================

// --- closed-processor error guards (checkClosed branches) ---

func TestProcessor_ClosedState_Boundary(t *testing.T) {
	p, _ := New()
	p.Close()

	noop := func(key any, item *IterableValue) {}
	noopLine := func(lineNum int, item *IterableValue) error { return nil }
	reduceFn := func(acc any, item *IterableValue) any { return acc }

	if err := p.ForeachWithPath(`{"a":1}`, "a", noop); err == nil {
		t.Error("expected error from ForeachWithPath on closed processor")
	}
	if _, err := p.ForeachReturn(`{"a":1}`, noop); err == nil {
		t.Error("expected error from ForeachReturn on closed processor")
	}
	if _, err := p.CompilePath("a"); err == nil {
		t.Error("expected error from CompilePath on closed processor")
	}
	if _, err := p.GetCompiled(`{"a":1}`, nil); err == nil {
		t.Error("expected error from GetCompiled on closed processor")
	}
	if err := p.ForeachJSONL(strings.NewReader(`{"a":1}`), noopLine); err == nil {
		t.Error("expected error from ForeachJSONL on closed processor")
	}
	if _, err := p.ReduceJSONL(strings.NewReader(`{"a":1}`), nil, reduceFn); err == nil {
		t.Error("expected error from ReduceJSONL on closed processor")
	}
	if _, err := p.CollectJSONL(strings.NewReader(`{"a":1}`)); err == nil {
		t.Error("expected error from CollectJSONL on closed processor")
	}
}

// --- StreamJSONL invalid-JSONL error path (processor_streamjsonl.go) ---

func TestStreamJSONL_InvalidJSON(t *testing.T) {
	p, _ := New()
	defer p.Close()
	err := p.StreamJSONL(strings.NewReader("not json\n"), func(int, *IterableValue) error { return nil })
	if err == nil {
		t.Error("expected error for invalid JSONL input")
	}
}

// --- StreamJSONLParallelWithContext context cancellation (processor_streamjsonl.go:163) ---

func TestStreamJSONLParallelWithContext_Cancel(t *testing.T) {
	p, _ := New()
	defer p.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before work starts
	_ = p.StreamJSONLParallelWithContext(ctx, strings.NewReader(`{"a":1}`+"\n"), 2,
		func(int, *IterableValue) error { return nil })
	// A pre-cancelled context must not panic; it surfaces an error or short-circuits.
}
