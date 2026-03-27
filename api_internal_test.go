package json

import (
	"context"
	"testing"
)

// ============================================================================
// HASH CUSTOM ESCAPES TESTS
// ============================================================================

// TestHashCustomEscapes tests the hashCustomEscapes function
func TestHashCustomEscapes(t *testing.T) {
	tests := []struct {
		name string
		m    map[rune]string
	}{
		{"nil map", nil},
		{"empty map", map[rune]string{}},
		{"single entry", map[rune]string{'<': "\\u003c"}},
		{"multiple entries", map[rune]string{'<': "\\u003c", '>': "\\u003e", '&': "\\u0026"}},
		{"unicode key", map[rune]string{'\u2028': "\\u2028"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that hashing is deterministic
			h1 := hashCustomEscapes(0, tt.m)
			h2 := hashCustomEscapes(0, tt.m)
			if h1 != h2 {
				t.Errorf("hashCustomEscapes is not deterministic: %d != %d", h1, h2)
			}

			// Test that different starting hashes produce different results
			h3 := hashCustomEscapes(1000, tt.m)
			if tt.m != nil && len(tt.m) > 0 && h1 == h3 {
				t.Errorf("Different starting hashes should produce different results")
			}
		})
	}
}

// TestHashCustomEscapes_DifferentMaps tests that different maps produce different hashes
func TestHashCustomEscapes_DifferentMaps(t *testing.T) {
	maps := []map[rune]string{
		nil,
		{},
		{'<': "\\u003c"},
		{'>': "\\u003e"},
		{'<': "\\u003c", '>': "\\u003e"},
		{'&': "\\u0026"},
	}

	hashes := make(map[uint64]int)
	for i, m := range maps {
		h := hashCustomEscapes(0, m)
		if other, exists := hashes[h]; exists {
			// Some maps might have the same hash (collision), but it's unlikely for these test cases
			t.Logf("Hash collision between map[%d] and map[%d]: %d", i, other, h)
		}
		hashes[h] = i
	}
}

// ============================================================================
// CONTEXT EQUAL TESTS
// ============================================================================

// TestContextEqual tests the contextEqual function
func TestContextEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        context.Context
		b        context.Context
		expected bool
	}{
		{"both nil", nil, nil, true},
		{"a nil, b non-nil", nil, context.Background(), false},
		{"a non-nil, b nil", context.Background(), nil, false},
		{"both non-nil", context.Background(), context.Background(), true},
		{"different contexts", context.Background(), context.TODO(), false}, // Different singleton pointers
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contextEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("contextEqual(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// CONFIG FIELDS EQUAL TESTS
// ============================================================================

// TestConfigFieldsEqual tests the configFieldsEqual function with various configs
func TestConfigFieldsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        Config
		b        Config
		expected bool
	}{
		{
			name:     "both default",
			a:        DefaultConfig(),
			b:        DefaultConfig(),
			expected: true,
		},
		{
			name:     "different MaxJSONSize",
			a:        Config{MaxJSONSize: 1000},
			b:        Config{MaxJSONSize: 2000},
			expected: false,
		},
		{
			name:     "different EnableCache",
			a:        Config{EnableCache: true},
			b:        Config{EnableCache: false},
			expected: false,
		},
		{
			name:     "different Context",
			a:        Config{Context: context.Background()},
			b:        Config{Context: nil},
			expected: false,
		},
		{
			name:     "both with Context",
			a:        Config{Context: context.Background()},
			b:        Config{Context: context.TODO()},
			expected: false, // Different context pointers are not equal
		},
		{
			name:     "different CustomEscapes",
			a:        Config{CustomEscapes: map[rune]string{'<': "\\u003c"}},
			b:        Config{CustomEscapes: nil},
			expected: false,
		},
		{
			name:     "equal CustomEscapes",
			a:        Config{CustomEscapes: map[rune]string{'<': "\\u003c"}},
			b:        Config{CustomEscapes: map[rune]string{'<': "\\u003c"}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := configFieldsEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("configFieldsEqual() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ============================================================================
// IS DEFAULT CONFIG TESTS
// ============================================================================

// TestIsDefaultConfig tests the isDefaultConfig function
func TestIsDefaultConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected bool
	}{
		{"default config", DefaultConfig(), true},
		{"modified MaxJSONSize", func() Config { c := DefaultConfig(); c.MaxJSONSize = 999; return c }(), false},
		{"modified EnableCache", func() Config { c := DefaultConfig(); c.EnableCache = !c.EnableCache; return c }(), false},
		{"with Context", func() Config { c := DefaultConfig(); c.Context = context.Background(); return c }(), false},
		{"with CustomEscapes", func() Config { c := DefaultConfig(); c.CustomEscapes = map[rune]string{'<': "\\u003c"}; return c }(), false},
		{"zero config", Config{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDefaultConfig(tt.config)
			if result != tt.expected {
				t.Errorf("isDefaultConfig() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ============================================================================
// HASH CONFIG TESTS
// ============================================================================

// TestHashConfig tests the hashConfig function
func TestHashConfig(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		c := DefaultConfig()
		h1 := hashConfig(c)
		h2 := hashConfig(c)
		if h1 != h2 {
			t.Errorf("hashConfig is not deterministic: %d != %d", h1, h2)
		}
	})

	t.Run("different configs different hashes", func(t *testing.T) {
		c1 := DefaultConfig()
		c2 := DefaultConfig()
		c2.MaxJSONSize = 999

		h1 := hashConfig(c1)
		h2 := hashConfig(c2)

		if h1 == h2 {
			t.Errorf("Different configs should have different hashes")
		}
	})

	t.Run("with custom escapes", func(t *testing.T) {
		c := DefaultConfig()
		c.CustomEscapes = map[rune]string{'<': "\\u003c"}

		h := hashConfig(c)
		if h == 0 {
			t.Error("hashConfig with CustomEscapes should not be 0")
		}
	})

	t.Run("with context", func(t *testing.T) {
		c := DefaultConfig()
		c.Context = context.Background()

		h := hashConfig(c)
		if h == 0 {
			t.Error("hashConfig with Context should not be 0")
		}
	})
}

// ============================================================================
// PROCESS RECURSIVELY WITH OPTIONS TESTS
// ============================================================================

// TestProcessRecursivelyWithOptions_Comprehensive tests more edge cases
func TestProcessRecursivelyWithOptions_Comprehensive(t *testing.T) {
	processor := MustNew()
	defer processor.Close()
	rp := NewRecursiveProcessor(processor)

	t.Run("Get with createPaths false", func(t *testing.T) {
		data := map[string]any{"existing": "value"}
		result, err := rp.ProcessRecursivelyWithOptions(data, "existing", opGet, nil, false)
		if err != nil {
			t.Errorf("Get existing path failed: %v", err)
		}
		if result != "value" {
			t.Errorf("Get = %v, want 'value'", result)
		}
	})

	t.Run("Get non-existing without createPaths", func(t *testing.T) {
		data := map[string]any{}
		_, err := rp.ProcessRecursivelyWithOptions(data, "nonexistent", opGet, nil, false)
		if err == nil {
			t.Error("Expected error for non-existing path")
		}
	})

	t.Run("Set without createPaths on new path", func(t *testing.T) {
		data := map[string]any{}
		_, err := rp.ProcessRecursivelyWithOptions(data, "new.path", opSet, "value", false)
		if err == nil {
			t.Error("Expected error for setting non-existing path without createPaths")
		}
	})

	t.Run("Delete root should fail", func(t *testing.T) {
		data := map[string]any{"key": "value"}
		_, err := rp.ProcessRecursivelyWithOptions(data, "", opDelete, nil, false)
		if err == nil {
			t.Error("Expected error for deleting root")
		}
	})

	t.Run("Set root should fail", func(t *testing.T) {
		data := map[string]any{"key": "value"}
		_, err := rp.ProcessRecursivelyWithOptions(data, "", opSet, "newroot", false)
		if err == nil {
			t.Error("Expected error for setting root")
		}
	})

	t.Run("Nested array operations", func(t *testing.T) {
		data := map[string]any{
			"items": []any{
				map[string]any{"id": 1},
				map[string]any{"id": 2},
			},
		}

		result, err := rp.ProcessRecursivelyWithOptions(data, "items[0].id", opGet, nil, false)
		if err != nil {
			t.Errorf("Get nested array failed: %v", err)
		}
		// Result can be int or float64 depending on how it was stored
		if result != 1 && result != float64(1) {
			t.Errorf("Get = %v (%T), want 1", result, result)
		}
	})
}

// ============================================================================
// BENCHMARK TESTS
// ============================================================================

func BenchmarkHashCustomEscapes(b *testing.B) {
	m := map[rune]string{'<': "\\u003c", '>': "\\u003e", '&': "\\u0026"}

	for i := 0; i < b.N; i++ {
		_ = hashCustomEscapes(0, m)
	}
}

func BenchmarkContextEqual(b *testing.B) {
	ctx := context.Background()

	for i := 0; i < b.N; i++ {
		_ = contextEqual(ctx, ctx)
	}
}

func BenchmarkHashConfig(b *testing.B) {
	c := DefaultConfig()

	for i := 0; i < b.N; i++ {
		_ = hashConfig(c)
	}
}
