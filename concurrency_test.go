package json

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/cybergodev/json/internal"
)

// TestConcurrentSampling tests concurrent access to SamplingReader
// This test verifies that globalRand is thread-safe
func TestConcurrentSampling(t *testing.T) {
	// Create a large JSON array for sampling
	items := make([]map[string]any, 100)
	for i := range items {
		items[i] = map[string]any{
			"id":    i,
			"value": "test item",
		}
	}
	jsonData, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	// Run concurrent sampling operations
	var wg sync.WaitGroup
	concurrency := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				reader := NewSamplingReader(bytes.NewReader(jsonData), 10)
				count := 0
				err := reader.Sample(func(index int, item any) bool {
					count++
					return true
				})
				if err != nil {
					t.Errorf("Sample failed: %v", err)
					return
				}
				if count > 10 {
					t.Errorf("Expected at most 10 samples, got %d", count)
				}
			}
		}()
	}
	wg.Wait()
}

// TestConcurrentPathTypeCache tests concurrent access to pathTypeCacheShards
func TestConcurrentPathTypeCache(t *testing.T) {
	var wg sync.WaitGroup
	concurrency := 20
	iterations := 100

	paths := []string{
		"simple",
		"nested.path",
		"array[0]",
		"complex.nested[1].key",
		"very.deep.nested.path.with.many.segments",
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				for _, path := range paths {
					_ = getPathType(path)
				}
			}
		}()
	}
	wg.Wait()
}

// TestConcurrentKeyInternMap tests concurrent access to key interning
func TestConcurrentKeyInternMap(t *testing.T) {
	var wg sync.WaitGroup
	concurrency := 20
	iterations := 100

	keys := []string{
		"id",
		"name",
		"value",
		"timestamp",
		"metadata",
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				for _, key := range keys {
					InternKey(key)
				}
			}
		}()
	}
	wg.Wait()
}

// TestConcurrentDefaultProcessor tests concurrent access to default processor
func TestConcurrentDefaultProcessor(t *testing.T) {
	var wg sync.WaitGroup
	concurrency := 20

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p := getDefaultProcessor()
			if p == nil {
				t.Error("Expected non-nil processor")
				return
			}
			_, err := p.Get(`{"test": 1}`, "test")
			if err != nil {
				t.Errorf("Get failed: %v", err)
			}
		}()
	}
	wg.Wait()
}

// TestConcurrentPathSegmentCache tests concurrent access to GlobalPathIntern
func TestConcurrentPathSegmentCache(t *testing.T) {
	var wg sync.WaitGroup
	concurrency := 20
	iterations := 50

	paths := []string{
		"simple",
		"nested.path",
		"array[0].item",
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				for _, path := range paths {
					if segments, ok := internal.GlobalPathIntern.Get(path); ok {
						if len(segments) == 0 {
							t.Errorf("Empty segments for path %s", path)
						}
					}
				}
			}
		}()
	}
	wg.Wait()
}

// TestConcurrentCompiledPathCache tests concurrent access to globalCompiledPathCache
func TestConcurrentCompiledPathCache(t *testing.T) {
	cache := internal.GetGlobalCompiledPathCache()
	var wg sync.WaitGroup
	concurrency := 20
	iterations := 50

	paths := []string{
		"simple",
		"nested.path",
		"array[0]",
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				for _, path := range paths {
					cp, err := cache.Get(path)
					if err != nil {
						t.Errorf("Get failed for path %s: %v", path, err)
						continue
					}
					if cp == nil {
						t.Errorf("expected non-nil CompiledPath for %s", path)
					}
					cp.Release()
				}
			}
		}()
	}
	wg.Wait()
}

// TestConcurrentValidationCache tests concurrent access to security validator cache
func TestConcurrentValidationCache(t *testing.T) {
	p, err := New()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer p.Close()

	var wg sync.WaitGroup
	concurrency := 10
	iterations := 50

	jsonStrings := []string{
		`{"test": 1}`,
		`{"nested": {"key": "value"}}`,
		`{"array": [1, 2, 3]}`,
	}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				for _, jsonStr := range jsonStrings {
					_, err := p.Get(jsonStr, "test")
					if err != nil && !strings.Contains(err.Error(), "not found") {
						t.Errorf("Get failed: %v", err)
					}
				}
			}
		}()
	}
	wg.Wait()
}
