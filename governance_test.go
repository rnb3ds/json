package json

import (
	"context"
	"strconv"
	"strings"
	"testing"
)

// TestGovernance_EncodeFunnelNoNestedAcquire locks in that governing the encode
// funnel (encodeWithConfigToBytes) does NOT cause nested semaphore acquisition in
// limited-concurrency mode. Marshal/MarshalIndent/EncodeWithConfig/EncodeStream/
// EncodeBatch all funnel through exactly one governed call; none of them govern
// themselves. Under MaxConcurrency=1 a nested acquire would self-reject with
// ErrConcurrencyLimit, so a spurious failure here is the canary for a nesting bug.
func TestGovernance_EncodeFunnelNoNestedAcquire(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConcurrency = 1 // tightest legal limit: a nested acquire would fail at once
	p, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	data := map[string]any{"k": []int{1, 2, 3}}

	if _, err := p.Marshal(data); err != nil {
		t.Fatalf("Marshal under MaxConcurrency=1: %v", err)
	}
	if _, err := p.MarshalIndent(data, "", "  "); err != nil {
		t.Fatalf("MarshalIndent under MaxConcurrency=1: %v", err)
	}
	if _, err := p.EncodeWithConfig(data); err != nil {
		t.Fatalf("EncodeWithConfig under MaxConcurrency=1: %v", err)
	}
	// EncodeStream funnels through EncodeWithConfig -> encodeWithConfigToBytes.
	if _, err := p.EncodeStream([]any{data, data, data}); err != nil {
		t.Fatalf("EncodeStream under MaxConcurrency=1: %v", err)
	}
	if _, err := p.EncodeBatch(map[string]any{"a": 1, "b": 2}); err != nil {
		t.Fatalf("EncodeBatch under MaxConcurrency=1: %v", err)
	}
}

// TestGovernance_StreamJSONLWorksUnderConcurrencyLimit confirms that acquiring
// governance once for the whole stream does not break normal streaming and does
// not self-reject when MaxConcurrency is small.
func TestGovernance_StreamJSONLWorksUnderConcurrencyLimit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConcurrency = 2
	p, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	in := `{"i":1}
{"i":2}
{"i":3}
`
	count := 0
	err = p.StreamJSONL(strings.NewReader(in), func(lineNum int, item *IterableValue) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatalf("StreamJSONL under MaxConcurrency=2: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 lines, got %d", count)
	}
}

// TestGovernance_StreamJSONLParallelUnderLimit exercises the parallel stream: the
// scanner goroutine holds the single in-flight slot for the whole run while worker
// goroutines run unregistered. Under MaxConcurrency=1 this must still succeed —
// workers must not contend the limit, only the outer stream call registers.
func TestGovernance_StreamJSONLParallelUnderLimit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConcurrency = 1
	p, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	var sb strings.Builder
	for i := 1; i <= 20; i++ {
		sb.WriteString(`{"i":`)
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("}\n")
	}

	count := 0
	err = p.StreamJSONLParallelWithContext(
		context.Background(),
		strings.NewReader(sb.String()),
		4,
		func(lineNum int, item *IterableValue) error {
			count++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("StreamJSONLParallelWithContext under MaxConcurrency=1: %v", err)
	}
	if count != 20 {
		t.Fatalf("expected 20 lines, got %d", count)
	}
}

// TestGovernance_StreamJSONLChunkedUnderLimit confirms the chunked stream variant
// is also governed exactly once and works under a tight limit.
func TestGovernance_StreamJSONLChunkedUnderLimit(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConcurrency = 1
	p, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer p.Close()

	var sb strings.Builder
	for i := 1; i <= 25; i++ {
		sb.WriteString(`{"i":`)
		sb.WriteString(strconv.Itoa(i))
		sb.WriteString("}\n")
	}

	processed := 0
	err = p.StreamJSONLChunked(strings.NewReader(sb.String()), 10, func(chunk []*IterableValue) error {
		processed += len(chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamJSONLChunked under MaxConcurrency=1: %v", err)
	}
	if processed != 25 {
		t.Fatalf("expected 25 items, got %d", processed)
	}
}
