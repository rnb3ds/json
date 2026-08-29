package internal

import (
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

		}()
	}
	wg.Wait()
}
