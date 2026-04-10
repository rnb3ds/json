package json

import (
	"errors"
	"sync/atomic"
	"testing"
)

// ============================================================================
// ParallelIterator Tests
// ============================================================================

func TestParallelIteratorForEach(t *testing.T) {
	t.Run("AllItems", func(t *testing.T) {
		data := make([]any, 10)
		for i := range data {
			data[i] = i
		}

		cfg := DefaultConfig()
		cfg.MaxConcurrency = 4
		iter := NewParallelIterator(data, cfg)
		defer iter.Close()

		var processed int64
		err := iter.ForEach(func(idx int, val any) error {
			atomic.AddInt64(&processed, 1)
			return nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if processed != 10 {
			t.Errorf("expected 10 items processed, got %d", processed)
		}
	})

	t.Run("WithError", func(t *testing.T) {
		data := make([]any, 10)
		for i := range data {
			data[i] = i
		}

		cfg := DefaultConfig()
		cfg.MaxConcurrency = 4
		iter := NewParallelIterator(data, cfg)
		defer iter.Close()

		testErr := errors.New("test error")
		err := iter.ForEach(func(idx int, val any) error {
			if idx == 5 {
				return testErr
			}
			return nil
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, testErr) {
			t.Errorf("expected testErr, got %v", err)
		}
	})

	t.Run("EmptyData", func(t *testing.T) {
		data := []any{}

		iter := NewParallelIterator(data)
		defer iter.Close()

		err := iter.ForEach(func(idx int, val any) error {
			return nil
		})

		if err != nil {
			t.Fatalf("unexpected error for empty data: %v", err)
		}
	})
}

func TestParallelIteratorMap(t *testing.T) {
	t.Run("MultiplyByTwo", func(t *testing.T) {
		data := []any{float64(1), float64(2), float64(3), float64(4), float64(5)}
		cfg := DefaultConfig()
		cfg.MaxConcurrency = 4
		iter := NewParallelIterator(data, cfg)
		defer iter.Close()

		result, err := iter.Map(func(idx int, val any) (any, error) {
			return val.(float64) * 2, nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 5 {
			t.Fatalf("expected 5 results, got %d", len(result))
		}
		expected := []float64{2, 4, 6, 8, 10}
		for i, v := range result {
			if v.(float64) != expected[i] {
				t.Errorf("result[%d] = %v, want %v", i, v, expected[i])
			}
		}
	})

	t.Run("ErrorOnOneElement", func(t *testing.T) {
		data := []any{float64(1), float64(2), float64(3)}
		cfg := DefaultConfig()
		cfg.MaxConcurrency = 2
		iter := NewParallelIterator(data, cfg)
		defer iter.Close()

		testErr := errors.New("transform error")
		result, err := iter.Map(func(idx int, val any) (any, error) {
			if idx == 1 {
				return nil, testErr
			}
			return val, nil
		})

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, testErr) {
			t.Errorf("expected testErr, got %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result on error, got %v", result)
		}
	})
}

func TestParallelIteratorFilter(t *testing.T) {
	t.Run("KeepEvenValues", func(t *testing.T) {
		data := []any{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		cfg := DefaultConfig()
		cfg.MaxConcurrency = 4
		iter := NewParallelIterator(data, cfg)
		defer iter.Close()

		result := iter.Filter(func(idx int, val any) bool {
			return val.(int)%2 == 0
		})

		if len(result) != 5 {
			t.Fatalf("expected 5 results, got %d", len(result))
		}
		expected := []int{2, 4, 6, 8, 10}
		// Filter preserves order of insertion, but parallel execution means
		// order is not guaranteed. Check by presence instead.
		resultSet := map[int]bool{}
		for _, v := range result {
			resultSet[v.(int)] = true
		}
		for _, e := range expected {
			if !resultSet[e] {
				t.Errorf("expected %d in result", e)
			}
		}
	})
}

func TestParallelIteratorForEachBatch(t *testing.T) {
	t.Run("BatchSizes", func(t *testing.T) {
		data := make([]any, 10)
		for i := range data {
			data[i] = i
		}

		cfg := DefaultConfig()
		cfg.MaxConcurrency = 4
		iter := NewParallelIterator(data, cfg)
		defer iter.Close()

		var batchCalls int64
		var totalItems int64

		err := iter.ForEachBatch(3, func(batchIdx int, batch []any) error {
			atomic.AddInt64(&batchCalls, 1)
			for range batch {
				atomic.AddInt64(&totalItems, 1)
			}
			return nil
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if batchCalls != 4 {
			t.Errorf("expected 4 batch calls (3+3+3+1), got %d", batchCalls)
		}
		if totalItems != 10 {
			t.Errorf("expected 10 total items, got %d", totalItems)
		}
	})
}

func TestParallelIteratorCloseNoPanic(t *testing.T) {
	data := []any{1, 2, 3}
	iter := NewParallelIterator(data)

	// Close should not panic
	iter.Close()
}
