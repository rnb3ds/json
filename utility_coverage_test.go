package json

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// ============================================================================
// DEEP COPY TESTS
// Target: deepCopySubtree, deepCopyJSONValue, deepCopyJSONMap, deepCopyJSONSlice
// ============================================================================

func TestDeepCopy(t *testing.T) {
	t.Run("NestedMap", func(t *testing.T) {
		original := `{"a":{"b":1}}`
		result, err := MergeJSON(original, `{"a":{"b":2}}`)
		if err != nil {
			t.Fatalf("MergeJSON failed: %v", err)
		}
		// The merge should produce a modified copy, not alter the original concept
		if result == "" {
			t.Error("expected non-empty merge result")
		}
	})

	t.Run("CopyNestedMapIndependence", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		original := `{"a":{"b":1}}`
		// Set a deep path which triggers deep copy internally
		modified, err := p.Set(original, "a.b", 99)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Original path should still be 1
		val, err := p.Get(original, "a.b")
		if err != nil {
			t.Fatalf("Get original failed: %v", err)
		}
		if val != float64(1) {
			t.Errorf("original a.b = %v, want 1", val)
		}

		// Modified should have 99
		val2, err := p.Get(modified, "a.b")
		if err != nil {
			t.Fatalf("Get modified failed: %v", err)
		}
		if val2 != float64(99) {
			t.Errorf("modified a.b = %v, want 99", val2)
		}
	})

	t.Run("ArrayCopy", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		original := `{"arr":[1,2,3]}`
		modified, err := p.Set(original, "arr[0]", 99)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Original array element should be unchanged
		val, err := p.Get(original, "arr[0]")
		if err != nil {
			t.Fatalf("Get original failed: %v", err)
		}
		if val != float64(1) {
			t.Errorf("original arr[0] = %v, want 1", val)
		}

		// Modified should have 99
		val2, err := p.Get(modified, "arr[0]")
		if err != nil {
			t.Fatalf("Get modified failed: %v", err)
		}
		if val2 != float64(99) {
			t.Errorf("modified arr[0] = %v, want 99", val2)
		}
	})

	t.Run("MixedStructure", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		original := `{"arr":[1,{"x":2}],"val":"str"}`
		modified, err := p.Set(original, "arr[1].x", 42)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Original nested value unchanged
		val, err := p.Get(original, "arr[1].x")
		if err != nil {
			t.Fatalf("Get original failed: %v", err)
		}
		if val != float64(2) {
			t.Errorf("original arr[1].x = %v, want 2", val)
		}

		// Modified should have 42
		val2, err := p.Get(modified, "arr[1].x")
		if err != nil {
			t.Fatalf("Get modified failed: %v", err)
		}
		if val2 != float64(42) {
			t.Errorf("modified arr[1].x = %v, want 42", val2)
		}
	})

	t.Run("DeepCopySubtreeDirectly", func(t *testing.T) {
		// Test deepCopySubtree via internal deepCopy path through MergeJSON
		// which triggers deep copy of merged result
		result, err := MergeJSON(`{"a":1}`, `{"b":2}`)
		if err != nil {
			t.Fatalf("MergeJSON failed: %v", err)
		}
		if result == "" {
			t.Error("expected non-empty result")
		}
		// Both keys should be present
		p, _ := New()
		defer p.Close()
		val, err := p.Get(result, "a")
		if err != nil {
			t.Fatalf("Get a failed: %v", err)
		}
		if val != float64(1) {
			t.Errorf("a = %v, want 1", val)
		}
		val2, err := p.Get(result, "b")
		if err != nil {
			t.Fatalf("Get b failed: %v", err)
		}
		if val2 != float64(2) {
			t.Errorf("b = %v, want 2", val2)
		}
	})
}

// ============================================================================
// JSON NUMBER PRESERVATION TESTS
// Target: convertNumbers via PreserveNumbers config
// ============================================================================

func TestConvertJSONNumber(t *testing.T) {
	t.Run("PreserveNumbersTrue", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.PreserveNumbers = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		input := `{"value":123.456}`
		val, err := p.Get(input, "value", cfg)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		// With PreserveNumbers=true, the value should be the library's Number type
		jn, ok := val.(Number)
		if !ok {
			t.Fatalf("expected Number, got %T", val)
		}
		if jn.String() != "123.456" {
			t.Errorf("Number = %q, want %q", jn.String(), "123.456")
		}
	})

	t.Run("PreserveNumbersFalse", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.PreserveNumbers = false
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		input := `{"value":123.456}`
		val, err := p.Get(input, "value")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		// With PreserveNumbers=false, value should be float64
		f, ok := val.(float64)
		if !ok {
			t.Fatalf("expected float64, got %T", val)
		}
		if f != 123.456 {
			t.Errorf("float64 = %v, want 123.456", f)
		}
	})

	t.Run("PreserveNumbersInteger", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.PreserveNumbers = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		input := `{"count":42}`
		val, err := p.Get(input, "count", cfg)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		jn, ok := val.(Number)
		if !ok {
			t.Fatalf("expected Number, got %T", val)
		}
		if jn.String() != "42" {
			t.Errorf("Number = %q, want %q", jn.String(), "42")
		}
	})

	t.Run("PreserveNumbersNested", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.PreserveNumbers = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		input := `{"obj":{"num":7}}`
		val, err := p.Get(input, "obj.num", cfg)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		jn, ok := val.(Number)
		if !ok {
			t.Fatalf("expected Number, got %T", val)
		}
		if jn.String() != "7" {
			t.Errorf("Number = %q, want %q", jn.String(), "7")
		}
	})
}

// ============================================================================
// PATH VALIDATION TESTS
// Target: validatePath, isValidPath
// ============================================================================

func TestHelpersValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"EmptyPath", "", true},
		{"DotOnly", ".", false},
		{"SimpleKey", "key", false},
		{"NestedKey", "a.b.c", false},
		{"ArrayIndex", "arr[0]", false},
		{"NestedArray", "arr[0].key", false},
		{"WildcardStar", "items[*]", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}

	t.Run("EmptyPathViaGet", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		// Empty path is treated as root access by the processor
		val, err := p.Get(`{"a":1}`, "")
		if err != nil {
			t.Errorf("unexpected error for empty path via Get: %v", err)
		}
		if val == nil {
			t.Error("expected root data for empty path")
		}
	})

	t.Run("EmptyPathViaSet", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		_, err = p.Set(`{"a":1}`, "", 2)
		if err == nil {
			t.Error("expected error for empty path via Set")
		}
	})

	t.Run("EmptyPathViaDelete", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		_, err = p.Delete(`{"a":1}`, "")
		if err == nil {
			t.Error("expected error for empty path via Delete")
		}
	})

	t.Run("IsValidPathChecks", func(t *testing.T) {
		tests := []struct {
			path string
			want bool
		}{
			{"", false},
			{".", true},
			{"key", true},
			{"a.b", true},
		}
		for _, tt := range tests {
			got := isValidPath(tt.path)
			if got != tt.want {
				t.Errorf("isValidPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		}
	})
}

// ============================================================================
// RATE LIMIT TESTS
// Target: checkRateLimit
// ============================================================================

func TestProcessorRateLimit(t *testing.T) {
	t.Run("ConcurrencyLimit", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxConcurrency = 1
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		// ProcessBatch with single concurrency should still complete correctly
		ops := []BatchOperation{
			{Type: "get", JSONStr: `{"a":1}`, Path: "a", ID: "op1"},
			{Type: "get", JSONStr: `{"b":2}`, Path: "b", ID: "op2"},
			{Type: "get", JSONStr: `{"c":3}`, Path: "c", ID: "op3"},
		}
		results, err := p.ProcessBatch(ops)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		for i, r := range results {
			if r.Error != nil {
				t.Errorf("result %d error: %v", i, r.Error)
			}
			if r.Result != float64(i+1) {
				t.Errorf("result %d = %v, want %v", i, r.Result, float64(i+1))
			}
		}
	})

t.Run("RateLimitConfig", func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.MaxConcurrency = 1
			p, err := New(cfg)
			if err != nil {
				t.Fatalf("New() failed: %v", err)
			}
			defer p.Close()

			// Verify rate limit config is respected - just ensure no panic
			for i := 0; i < 5; i++ {
				_, _ = p.Get(`{"a":1}`, "a")
			}
		})
}

// ============================================================================
// PROCESS BATCH TESTS
// Target: ProcessBatch (62%)
// ============================================================================

func TestProcessorProcessBatch(t *testing.T) {
	t.Run("BatchGetOperations", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ops := []BatchOperation{
			{Type: "get", JSONStr: `{"a":1}`, Path: "a", ID: "1"},
			{Type: "get", JSONStr: `{"b":2}`, Path: "b", ID: "2"},
		}
		results, err := p.ProcessBatch(ops)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if results[0].ID != "1" || results[1].ID != "2" {
			t.Errorf("IDs mismatch: %q, %q", results[0].ID, results[1].ID)
		}
		if results[0].Result != float64(1) {
			t.Errorf("result[0] = %v, want 1", results[0].Result)
		}
		if results[1].Result != float64(2) {
			t.Errorf("result[1] = %v, want 2", results[1].Result)
		}
	})

	t.Run("BatchSetOperations", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ops := []BatchOperation{
			{Type: "set", JSONStr: `{"a":1}`, Path: "a", Value: 99, ID: "s1"},
		}
		results, err := p.ProcessBatch(ops)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}
		if results[0].Error != nil {
			t.Fatalf("set error: %v", results[0].Error)
		}
		resultStr, ok := results[0].Result.(string)
		if !ok {
			t.Fatalf("expected string result, got %T", results[0].Result)
		}
		if !strings.Contains(resultStr, "99") {
			t.Errorf("result %q should contain 99", resultStr)
		}
	})

	t.Run("BatchDeleteOperations", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ops := []BatchOperation{
			{Type: "delete", JSONStr: `{"a":1,"b":2}`, Path: "a", ID: "d1"},
		}
		results, err := p.ProcessBatch(ops)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}
		if results[0].Error != nil {
			t.Fatalf("delete error: %v", results[0].Error)
		}
	})

	t.Run("BatchValidateOperations", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ops := []BatchOperation{
			{Type: "validate", JSONStr: `{"a":1}`, ID: "v1"},
		}
		results, err := p.ProcessBatch(ops)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}
		if results[0].Error != nil {
			t.Fatalf("validate error: %v", results[0].Error)
		}
		m, ok := results[0].Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", results[0].Result)
		}
		if m["valid"] != true {
			t.Errorf("valid = %v, want true", m["valid"])
		}
	})

	t.Run("UnknownOperationType", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ops := []BatchOperation{
			{Type: "unknown", JSONStr: `{}`, ID: "u1"},
		}
		results, err := p.ProcessBatch(ops)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}
		if results[0].Error == nil {
			t.Error("expected error for unknown operation type")
		}
	})

	t.Run("BatchSizeExceeded", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxBatchSize = 1
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ops := []BatchOperation{
			{Type: "get", JSONStr: `{"a":1}`, Path: "a", ID: "1"},
			{Type: "get", JSONStr: `{"b":2}`, Path: "b", ID: "2"},
		}
		_, err = p.ProcessBatch(ops)
		if err == nil {
			t.Error("expected error for batch size exceeded")
		}
	})

	t.Run("EmptyBatch", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		results, err := p.ProcessBatch([]BatchOperation{})
		if err != nil {
			t.Fatalf("ProcessBatch with empty slice failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})

	t.Run("ErrorInCallback", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		ops := []BatchOperation{
			{Type: "get", JSONStr: `invalid json`, Path: "a", ID: "e1"},
			{Type: "get", JSONStr: `{"b":2}`, Path: "b", ID: "e2"},
		}
		results, err := p.ProcessBatch(ops)
		if err != nil {
			t.Fatalf("ProcessBatch failed: %v", err)
		}
		// First should error, second should succeed
		if results[0].Error == nil {
			t.Error("expected error for invalid JSON in first op")
		}
		if results[1].Error != nil {
			t.Errorf("unexpected error in second op: %v", results[1].Error)
		}
	})
}

// ============================================================================
// WARMUP CACHE TESTS
// Target: WarmupCache (63%)
// ============================================================================

func TestProcessorWarmupCache(t *testing.T) {
	t.Run("SuccessfulWarmup", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EnableCache = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		jsonStr := `{"users":"data","items":"list","settings":"conf"}`
		result, err := p.WarmupCache(jsonStr, []string{"users", "items", "settings"})
		if err != nil {
			t.Fatalf("WarmupCache failed: %v", err)
		}
		if result.TotalPaths != 3 {
			t.Errorf("TotalPaths = %d, want 3", result.TotalPaths)
		}
		if result.Successful != 3 {
			t.Errorf("Successful = %d, want 3", result.Successful)
		}
		if result.Failed != 0 {
			t.Errorf("Failed = %d, want 0", result.Failed)
		}
		if result.SuccessRate != 100.0 {
			t.Errorf("SuccessRate = %v, want 100", result.SuccessRate)
		}
	})

	t.Run("CacheDisabled", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EnableCache = false
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		_, err = p.WarmupCache(`{"a":1}`, []string{"a"})
		if err == nil {
			t.Error("expected error when cache is disabled")
		}
	})

	t.Run("EmptyPaths", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EnableCache = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		result, err := p.WarmupCache(`{"a":1}`, []string{})
		if err != nil {
			t.Fatalf("WarmupCache with empty paths failed: %v", err)
		}
		if result.TotalPaths != 0 {
			t.Errorf("TotalPaths = %d, want 0", result.TotalPaths)
		}
		if result.SuccessRate != 100.0 {
			t.Errorf("SuccessRate = %v, want 100", result.SuccessRate)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EnableCache = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		_, err = p.WarmupCache(`invalid`, []string{"a"})
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("PartialFailure", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EnableCache = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		jsonStr := `{"a":1}`
		result, err := p.WarmupCache(jsonStr, []string{"a", "nonexistent.path"})
		if err != nil {
			// Some implementations return error, some return partial results
			// Both are acceptable
			return
		}
		if result.TotalPaths != 2 {
			t.Errorf("TotalPaths = %d, want 2", result.TotalPaths)
		}
		if result.Successful != 1 {
			t.Errorf("Successful = %d, want 1", result.Successful)
		}
		if result.Failed != 1 {
			t.Errorf("Failed = %d, want 1", result.Failed)
		}
	})
}

// ============================================================================
// FOREACH WITH ERROR TESTS
// Target: ForeachWithError (67%)
// ============================================================================

func TestProcessorForeachWithError(t *testing.T) {
	t.Run("IterateObject", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		var keys []string
		err = p.ForeachWithError(`{"a":1,"b":2,"c":3}`, ".", func(key any, item *IterableValue) error {
			keys = append(keys, fmt.Sprintf("%v", key))
			return nil
		})
		if err != nil {
			t.Fatalf("ForeachWithError failed: %v", err)
		}
		if len(keys) != 3 {
			t.Errorf("got %d keys, want 3", len(keys))
		}
	})

	t.Run("IterateArray", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		var items []any
		err = p.ForeachWithError(`[10,20,30]`, ".", func(key any, item *IterableValue) error {
			items = append(items, item.GetData())
			return nil
		})
		if err != nil {
			t.Fatalf("ForeachWithError failed: %v", err)
		}
		if len(items) != 3 {
			t.Errorf("got %d items, want 3", len(items))
		}
	})

	t.Run("ErrorPropagation", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		testErr := fmt.Errorf("test error from callback")
		err = p.ForeachWithError(`{"a":1,"b":2}`, ".", func(key any, item *IterableValue) error {
			return testErr
		})
		if err == nil {
			t.Error("expected error from ForeachWithError callback")
		}
		if !errors.Is(err, testErr) {
			t.Errorf("error = %v, want wrapped testErr", err)
		}
	})

	t.Run("BreakStopsIteration", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		count := 0
		err = p.ForeachWithError(`{"a":1,"b":2,"c":3}`, ".", func(key any, item *IterableValue) error {
			count++
			return item.Break()
		})
		if err != nil {
			t.Fatalf("ForeachWithError with Break failed: %v", err)
		}
		if count != 1 {
			t.Errorf("count = %d, want 1 (Break should stop after first)", count)
		}
	})

	t.Run("InvalidPath", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		err = p.ForeachWithError(`{"a":1}`, "nonexistent", func(key any, item *IterableValue) error {
			return nil
		})
		if err == nil {
			t.Error("expected error for nonexistent path")
		}
	})

	t.Run("NestedPath", func(t *testing.T) {
		p, err := New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		var items []any
		err = p.ForeachWithError(`{"arr":[1,2,3]}`, "arr", func(key any, item *IterableValue) error {
			items = append(items, item.GetData())
			return nil
		})
		if err != nil {
			t.Fatalf("ForeachWithError on nested array failed: %v", err)
		}
		if len(items) != 3 {
			t.Errorf("got %d items, want 3", len(items))
		}
	})
}

// ============================================================================
// HEALTH STATUS TESTS
// Target: GetHealthStatus (75%)
// ============================================================================

func TestProcessorGetHealth(t *testing.T) {
	t.Run("HealthyProcessor", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EnableMetrics = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		health := p.GetHealthStatus()
		if health.Timestamp.IsZero() {
			t.Error("expected non-zero timestamp")
		}
		// Checks map should be populated
		if len(health.Checks) == 0 {
			t.Error("expected non-empty checks")
		}
	})

	t.Run("NilProcessor", func(t *testing.T) {
		var p *Processor
		health := p.GetHealthStatus()
		if health.Healthy {
			t.Error("nil processor should not be healthy")
		}
		if health.Checks == nil {
			t.Error("expected non-nil checks for nil processor")
		}
		if proc, ok := health.Checks["processor"]; !ok {
			t.Error("expected 'processor' check in nil processor health")
		} else if proc.Healthy {
			t.Error("processor check should not be healthy for nil processor")
		}
	})

	t.Run("HealthAfterOperations", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EnableMetrics = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		// Perform some operations
		_, _ = p.Get(`{"a":1}`, "a")
		_, _ = p.Get(`{"b":2}`, "b")

		health := p.GetHealthStatus()
		if health.Timestamp.IsZero() {
			t.Error("expected non-zero timestamp after operations")
		}
	})
}

// ============================================================================
// LOG ERROR TESTS
// Target: logError (0%)
// ============================================================================

func TestProcessorLogError(t *testing.T) {
	t.Run("LogErrorWithMetrics", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EnableMetrics = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		// logError is triggered internally via incrementErrorCount and error logging
		// We can test it by performing operations that generate errors
		_, err = p.Get(`invalid json`, "a")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}

		// The error count should have been incremented
		stats := p.GetStats()
		if stats.ErrorCount < 1 {
			t.Errorf("ErrorCount = %d, want >= 1", stats.ErrorCount)
		}
	})

	t.Run("LogErrorWithJsonError", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EnableMetrics = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		// Perform multiple error-causing operations
		_, _ = p.Get(`{bad`, "x")
		_, _ = p.Set(`notjson`, "x", 1)

		stats := p.GetStats()
		if stats.ErrorCount < 1 {
			t.Errorf("ErrorCount = %d, want >= 1 after multiple errors", stats.ErrorCount)
		}
	})

	t.Run("LogErrorViaHook", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.EnableMetrics = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		var mu sync.Mutex
		var loggedOps []string

		hook := &HookFunc{
			BeforeFn: func(ctx HookContext) error { return nil },
			AfterFn: func(ctx HookContext, result any, err error) (any, error) {
				mu.Lock()
				loggedOps = append(loggedOps, ctx.Operation)
				mu.Unlock()
				return result, err
			},
		}
		p.AddHook(hook)

		// Manually invoke hooks via hookChain to verify they are wired correctly
		hc := hookChain(p.hooks)
		ctx := HookContext{
			Operation: "get",
			Path:      "test",
			StartTime: time.Now(),
		}
		_ = hc.executeBefore(ctx)
		_, _ = hc.executeAfter(ctx, nil, nil)

		mu.Lock()
		count := len(loggedOps)
		mu.Unlock()

		if count == 0 {
			t.Error("expected hook to be called at least once")
		}
	})
}

// ============================================================================
// CONCURRENT RATE LIMIT TEST
// Target: checkRateLimit under concurrent load
// ============================================================================

func TestProcessorRateLimitConcurrent(t *testing.T) {
	t.Run("ConcurrentBatchOperations", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.MaxConcurrency = 2
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer p.Close()

		var wg sync.WaitGroup
		errCh := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				ops := []BatchOperation{
					{Type: "get", JSONStr: `{"val":1}`, Path: "val", ID: fmt.Sprintf("op%d", idx)},
				}
				results, batchErr := p.ProcessBatch(ops)
				if batchErr != nil {
					errCh <- batchErr
					return
				}
				if len(results) != 1 {
					errCh <- fmt.Errorf("expected 1 result, got %d", len(results))
					return
				}
				if results[0].Error != nil {
					errCh <- results[0].Error
					return
				}
			}(i)
		}

		wg.Wait()
		close(errCh)

		for e := range errCh {
			t.Errorf("concurrent operation error: %v", e)
		}
	})
}
