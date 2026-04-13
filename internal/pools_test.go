package internal

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestStringBuilderPool(t *testing.T) {
	t.Run("get put round trip", func(t *testing.T) {
		sb := GetStringBuilder()
		if sb == nil {
			t.Fatal("GetStringBuilder returned nil")
		}
		sb.WriteString("test")
		PutStringBuilder(sb)

		sb2 := GetStringBuilder()
		if sb2.Len() != 0 {
			t.Error("reused builder should be reset")
		}
		PutStringBuilder(sb2)
	})

	t.Run("nil input", func(t *testing.T) {
		PutStringBuilder(nil) // should not panic
	})

	t.Run("oversized builder discarded", func(t *testing.T) {
		sb := GetStringBuilder()
		sb.Grow(16*1024 + 1)
		PutStringBuilder(sb) // should not be returned to pool
	})
}

func TestResultsSlicePool(t *testing.T) {
	tests := []struct {
		name string
		hint int
	}{
		{"small", 4},
		{"medium", 16},
		{"large", 64},
		{"direct", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := GetResultsSlice(tt.hint)
			if s == nil {
				t.Fatal("GetResultsSlice returned nil")
			}
			if len(*s) != 0 {
				t.Error("slice should be empty")
			}
			*s = append(*s, "test")
			PutResultsSlice(s)
		})
	}

	t.Run("nil input", func(t *testing.T) {
		PutResultsSlice(nil) // should not panic
	})

	t.Run("oversized slice not pooled", func(t *testing.T) {
		s := GetResultsSlice(10)
		for i := 0; i < 300; i++ {
			*s = append(*s, i)
		}
		PutResultsSlice(s) // cap > maxSliceCap, should not pool
	})
}

func TestErrorSlicePool(t *testing.T) {
	t.Run("get put round trip", func(t *testing.T) {
		s := GetErrorSlice()
		if s == nil {
			t.Fatal("GetErrorSlice returned nil")
		}
		if len(*s) != 0 {
			t.Error("slice should be empty")
		}
		*s = append(*s, errTest)
		PutErrorSlice(s)
	})

	t.Run("nil input", func(t *testing.T) {
		PutErrorSlice(nil)
	})

	t.Run("oversized not pooled", func(t *testing.T) {
		s := GetErrorSlice()
		for i := 0; i < 300; i++ {
			*s = append(*s, errTest)
		}
		PutErrorSlice(s)
	})
}

func TestPathSegmentSlicePool(t *testing.T) {
	tests := []struct {
		name string
		hint int
	}{
		{"small", 2},
		{"medium", 6},
		{"large", 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := GetPathSegmentSlice(tt.hint)
			if s == nil {
				t.Fatal("GetPathSegmentSlice returned nil")
			}
			if len(*s) != 0 {
				t.Error("slice should be empty")
			}
			*s = append(*s, PathSegment{Type: PropertySegment, Key: "test"})
			PutPathSegmentSlice(s)
		})
	}

	t.Run("nil input", func(t *testing.T) {
		PutPathSegmentSlice(nil)
	})
}

func TestStreamingMapPool(t *testing.T) {
	tests := []struct {
		name string
		hint int
	}{
		{"small", 4},
		{"medium", 16},
		{"large", 64},
		{"direct", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := GetStreamingMap(tt.hint)
			if m == nil {
				t.Fatal("GetStreamingMap returned nil")
			}
			if len(m) != 0 {
				t.Error("map should be empty")
			}
			m["key"] = "value"
			PutStreamingMap(m)
		})
	}

	t.Run("nil input", func(t *testing.T) {
		PutStreamingMap(nil)
	})
}

func TestBatchResultsMapPool(t *testing.T) {
	tests := []struct {
		name string
		hint int
	}{
		{"small", 4},
		{"medium", 12},
		{"direct", 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := GetBatchResultsMap(tt.hint)
			if m == nil {
				t.Fatal("GetBatchResultsMap returned nil")
			}
			if len(m) != 0 {
				t.Error("map should be empty")
			}
			m["key"] = "value"
			PutBatchResultsMap(m)
		})
	}

	t.Run("nil input", func(t *testing.T) {
		PutBatchResultsMap(nil)
	})

	t.Run("oversized not pooled", func(t *testing.T) {
		m := GetBatchResultsMap(4)
		for i := 0; i < 40; i++ {
			m[strings.Repeat("k", i+1)] = i
		}
		PutBatchResultsMap(m) // len > 32, should not pool
	})
}

func TestPoolsConcurrent(t *testing.T) {
	const goroutines = 50
	var wg sync.WaitGroup

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			sb := GetStringBuilder()
			sb.WriteString("concurrent")
			PutStringBuilder(sb)

			s := GetResultsSlice(10)
			PutResultsSlice(s)

			m := GetStreamingMap(8)
			PutStreamingMap(m)
		}()
	}
	wg.Wait()
}

var errTest = errors.New("test error")
