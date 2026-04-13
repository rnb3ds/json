package json

import (
	"strings"
	"testing"
)

// ============================================================================
// StreamIterator Tests
// ============================================================================

func TestStreamIterator(t *testing.T) {
	t.Run("IterateArray", func(t *testing.T) {
		input := `[1,2,3]`
		iter := NewStreamIterator(strings.NewReader(input))

		var values []any
		var indices []int
		for iter.Next() {
			values = append(values, iter.Value())
			indices = append(indices, iter.Index())
		}

		if err := iter.Err(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(values) != 3 {
			t.Fatalf("expected 3 values, got %d", len(values))
		}
		// JSON numbers decode as float64
		expectedValues := []any{float64(1), float64(2), float64(3)}
		for i, v := range values {
			if v.(float64) != expectedValues[i].(float64) {
				t.Errorf("values[%d] = %v, want %v", i, v, expectedValues[i])
			}
		}
		expectedIndices := []int{0, 1, 2}
		for i, idx := range indices {
			if idx != expectedIndices[i] {
				t.Errorf("indices[%d] = %d, want %d", i, idx, expectedIndices[i])
			}
		}
	})

	t.Run("EmptyArray", func(t *testing.T) {
		input := `[]`
		iter := NewStreamIterator(strings.NewReader(input))

		if iter.Next() {
			t.Error("expected Next() to return false for empty array")
		}
		if err := iter.Err(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		input := `not json`
		iter := NewStreamIterator(strings.NewReader(input))

		if iter.Next() {
			t.Error("expected Next() to return false for invalid JSON")
		}
		if err := iter.Err(); err == nil {
			t.Error("expected Err() to return non-nil for invalid JSON")
		}
	})

	t.Run("ValueBeforeNext", func(t *testing.T) {
		input := `[1,2,3]`
		iter := NewStreamIterator(strings.NewReader(input))

		val := iter.Value()
		if val != nil {
			t.Errorf("expected Value() before Next() to return nil, got %v", val)
		}
	})

	t.Run("ErrAfterCleanIteration", func(t *testing.T) {
		input := `[1,2,3]`
		iter := NewStreamIterator(strings.NewReader(input))

		for iter.Next() {
			// drain all elements
		}

		if err := iter.Err(); err != nil {
			t.Errorf("expected Err() to be nil after clean iteration, got %v", err)
		}
	})

	t.Run("LargeArray", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteByte('[')
		const n = 1000
		for i := 0; i < n; i++ {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString("1")
		}
		sb.WriteByte(']')

		iter := NewStreamIterator(strings.NewReader(sb.String()))

		count := 0
		for iter.Next() {
			count++
		}

		if err := iter.Err(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != n {
			t.Errorf("expected %d elements, got %d", n, count)
		}
	})
}

// ============================================================================
// StreamObjectIterator Tests
// ============================================================================

func TestStreamObjectIterator(t *testing.T) {
	t.Run("IterateObject", func(t *testing.T) {
		input := `{"a":1,"b":2}`
		iter := NewStreamObjectIterator(strings.NewReader(input))

		keys := map[string]any{}
		for iter.Next() {
			keys[iter.Key()] = iter.Value()
		}

		if err := iter.Err(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 2 {
			t.Fatalf("expected 2 keys, got %d", len(keys))
		}
		if v, ok := keys["a"]; !ok || v.(float64) != 1 {
			t.Errorf("key 'a' = %v, want 1", v)
		}
		if v, ok := keys["b"]; !ok || v.(float64) != 2 {
			t.Errorf("key 'b' = %v, want 2", v)
		}
	})

	t.Run("EmptyObject", func(t *testing.T) {
		input := `{}`
		iter := NewStreamObjectIterator(strings.NewReader(input))

		if iter.Next() {
			t.Error("expected Next() to return false for empty object")
		}
		if err := iter.Err(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		input := `not json`
		iter := NewStreamObjectIterator(strings.NewReader(input))

		if iter.Next() {
			t.Error("expected Next() to return false for invalid JSON")
		}
		if err := iter.Err(); err == nil {
			t.Error("expected Err() to return non-nil for invalid JSON")
		}
	})

	t.Run("KeyValueBeforeNext", func(t *testing.T) {
		input := `{"a":1}`
		iter := NewStreamObjectIterator(strings.NewReader(input))

		if key := iter.Key(); key != "" {
			t.Errorf("expected Key() before Next() to return empty string, got %q", key)
		}
		if val := iter.Value(); val != nil {
			t.Errorf("expected Value() before Next() to return nil, got %v", val)
		}
	})

	t.Run("NestedValues", func(t *testing.T) {
		input := `{"str":"hello","num":42,"bool":true,"arr":[1,2],"obj":{"nested":1}}`
		iter := NewStreamObjectIterator(strings.NewReader(input))

		result := map[string]any{}
		for iter.Next() {
			result[iter.Key()] = iter.Value()
		}

		if err := iter.Err(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if v, ok := result["str"]; !ok || v.(string) != "hello" {
			t.Errorf("key 'str' = %v, want 'hello'", v)
		}
		if v, ok := result["num"]; !ok || v.(float64) != 42 {
			t.Errorf("key 'num' = %v, want 42", v)
		}
		if v, ok := result["bool"]; !ok || v.(bool) != true {
			t.Errorf("key 'bool' = %v, want true", v)
		}
		if v, ok := result["arr"]; !ok {
			t.Errorf("key 'arr' missing")
		} else if arr, ok := v.([]any); !ok || len(arr) != 2 {
			t.Errorf("key 'arr' = %v, want array of 2 elements", v)
		}
		if v, ok := result["obj"]; !ok {
			t.Errorf("key 'obj' missing")
		} else if obj, ok := v.(map[string]any); !ok || obj["nested"].(float64) != 1 {
			t.Errorf("key 'obj' = %v, want object with nested=1", v)
		}
	})
}

// ============================================================================
// BatchIterator Tests
// ============================================================================

func TestBatchIterator(t *testing.T) {
	t.Run("ExactMultiple", func(t *testing.T) {
		data := []any{float64(1), float64(2), float64(3), float64(4), float64(5), float64(6)}
		cfg := DefaultConfig()
		cfg.MaxBatchSize = 3
		iter := NewBatchIterator(data, cfg)

		batch1 := iter.NextBatch()
		if len(batch1) != 3 {
			t.Fatalf("batch 1: expected 3 items, got %d", len(batch1))
		}

		batch2 := iter.NextBatch()
		if len(batch2) != 3 {
			t.Fatalf("batch 2: expected 3 items, got %d", len(batch2))
		}

		if iter.HasNext() {
			t.Error("expected HasNext() to be false after all batches consumed")
		}

		batch3 := iter.NextBatch()
		if batch3 != nil {
			t.Errorf("expected nil after all batches, got %v", batch3)
		}
	})

	t.Run("NonMultiple", func(t *testing.T) {
		data := []any{1, 2, 3, 4, 5, 6, 7}
		cfg := DefaultConfig()
		cfg.MaxBatchSize = 3
		iter := NewBatchIterator(data, cfg)

		batchSizes := []int{}
		for iter.HasNext() {
			batch := iter.NextBatch()
			batchSizes = append(batchSizes, len(batch))
		}

		expected := []int{3, 3, 1}
		if len(batchSizes) != len(expected) {
			t.Fatalf("expected %d batches, got %d", len(expected), len(batchSizes))
		}
		for i, size := range batchSizes {
			if size != expected[i] {
				t.Errorf("batch %d: size=%d, want %d", i, size, expected[i])
			}
		}
	})

	t.Run("EmptyData", func(t *testing.T) {
		data := []any{}
		cfg := DefaultConfig()
		cfg.MaxBatchSize = 3
		iter := NewBatchIterator(data, cfg)

		batch := iter.NextBatch()
		if batch != nil {
			t.Errorf("expected nil for empty data, got %v", batch)
		}
		if iter.HasNext() {
			t.Error("expected HasNext() to be false for empty data")
		}
	})

	t.Run("SingleItem", func(t *testing.T) {
		data := []any{"only"}
		cfg := DefaultConfig()
		cfg.MaxBatchSize = 3
		iter := NewBatchIterator(data, cfg)

		batch := iter.NextBatch()
		if len(batch) != 1 {
			t.Fatalf("expected 1 item, got %d", len(batch))
		}
		if batch[0] != "only" {
			t.Errorf("expected 'only', got %v", batch[0])
		}
		if iter.HasNext() {
			t.Error("expected HasNext() to be false after single item")
		}
	})

	t.Run("Reset", func(t *testing.T) {
		data := []any{1, 2, 3, 4}
		cfg := DefaultConfig()
		cfg.MaxBatchSize = 2
		iter := NewBatchIterator(data, cfg)

		// Consume first batch
		batch1 := iter.NextBatch()
		if len(batch1) != 2 {
			t.Fatalf("first batch: expected 2 items, got %d", len(batch1))
		}

		// Reset
		iter.Reset()

		if !iter.HasNext() {
			t.Error("expected HasNext() to be true after Reset()")
		}
		if iter.CurrentIndex() != 0 {
			t.Errorf("expected CurrentIndex()=0 after Reset(), got %d", iter.CurrentIndex())
		}
		if iter.Remaining() != 4 {
			t.Errorf("expected Remaining()=4 after Reset(), got %d", iter.Remaining())
		}

		// Consume again from start
		batchAfterReset := iter.NextBatch()
		if len(batchAfterReset) != 2 {
			t.Fatalf("batch after reset: expected 2 items, got %d", len(batchAfterReset))
		}
		if batchAfterReset[0] != 1 || batchAfterReset[1] != 2 {
			t.Errorf("batch after reset: expected [1,2], got %v", batchAfterReset)
		}
	})

	t.Run("TotalBatches", func(t *testing.T) {
		tests := []struct {
			name      string
			dataLen   int
			batchSize int
			expected  int
		}{
			{"exact_multiple", 6, 3, 2},
			{"non_multiple", 7, 3, 3},
			{"single_batch", 3, 5, 1},
			{"empty", 0, 3, 0},
			{"one_item", 1, 5, 1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data := make([]any, tt.dataLen)
				cfg := DefaultConfig()
				cfg.MaxBatchSize = tt.batchSize
				iter := NewBatchIterator(data, cfg)

				total := iter.TotalBatches()
				if total != tt.expected {
					t.Errorf("TotalBatches() = %d, want %d", total, tt.expected)
				}
			})
		}
	})

	t.Run("CurrentIndexAndRemaining", func(t *testing.T) {
		data := []any{1, 2, 3, 4, 5}
		cfg := DefaultConfig()
		cfg.MaxBatchSize = 2
		iter := NewBatchIterator(data, cfg)

		if idx := iter.CurrentIndex(); idx != 0 {
			t.Errorf("initial CurrentIndex() = %d, want 0", idx)
		}
		if rem := iter.Remaining(); rem != 5 {
			t.Errorf("initial Remaining() = %d, want 5", rem)
		}

		iter.NextBatch() // consumes 2 items

		if idx := iter.CurrentIndex(); idx != 2 {
			t.Errorf("after 1 batch CurrentIndex() = %d, want 2", idx)
		}
		if rem := iter.Remaining(); rem != 3 {
			t.Errorf("after 1 batch Remaining() = %d, want 3", rem)
		}

		iter.NextBatch() // consumes 2 more items

		if idx := iter.CurrentIndex(); idx != 4 {
			t.Errorf("after 2 batches CurrentIndex() = %d, want 4", idx)
		}
		if rem := iter.Remaining(); rem != 1 {
			t.Errorf("after 2 batches Remaining() = %d, want 1", rem)
		}

		iter.NextBatch() // consumes last item

		if idx := iter.CurrentIndex(); idx != 5 {
			t.Errorf("after 3 batches CurrentIndex() = %d, want 5", idx)
		}
		if rem := iter.Remaining(); rem != 0 {
			t.Errorf("after 3 batches Remaining() = %d, want 0", rem)
		}
	})
}
