package internal

import (
	"bytes"
	"testing"
)

// TestMarshalJSON tests the MarshalJSON function
func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		pretty   bool
		prefix   string
		indent   string
		wantErr  bool
		contains string
	}{
		{
			name:     "simple object compact",
			value:    map[string]any{"name": "test"},
			pretty:   false,
			contains: `"name"`,
		},
		{
			name:     "simple object pretty",
			value:    map[string]any{"name": "test"},
			pretty:   true,
			prefix:   "",
			indent:   "  ",
			contains: "\n",
		},
		{
			name:     "array compact",
			value:    []any{1, 2, 3},
			pretty:   false,
			contains: "[",
		},
		{
			name:     "array pretty",
			value:    []any{1, 2, 3},
			pretty:   true,
			prefix:   "  ",
			indent:   "  ",
			contains: "\n",
		},
		{
			name:     "nil value",
			value:    nil,
			pretty:   false,
			contains: "null",
		},
		{
			name:     "boolean true",
			value:    true,
			pretty:   false,
			contains: "true",
		},
		{
			name:     "boolean false",
			value:    false,
			pretty:   false,
			contains: "false",
		},
		{
			name:     "number",
			value:    42,
			pretty:   false,
			contains: "42",
		},
		{
			name:     "string",
			value:    "hello",
			pretty:   false,
			contains: `"hello"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MarshalJSON(tt.value, tt.pretty, tt.prefix, tt.indent)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.contains != "" && !bytes.Contains([]byte(result), []byte(tt.contains)) {
				t.Errorf("MarshalJSON() = %v, want to contain %v", result, tt.contains)
			}
		})
	}
}

// TestEncoderBuffer tests the buffer pool operations
func TestEncoderBuffer(t *testing.T) {
	t.Run("Get and Put", func(t *testing.T) {
		buf := GetEncoderBuffer()
		if buf == nil {
			t.Fatal("GetEncoderBuffer returned nil")
		}

		buf.WriteString("test data")
		if buf.String() != "test data" {
			t.Errorf("Buffer content = %q, want %q", buf.String(), "test data")
		}

		PutEncoderBuffer(buf)
	})

	t.Run("Reset on reuse", func(t *testing.T) {
		buf1 := GetEncoderBuffer()
		buf1.WriteString("first data")
		PutEncoderBuffer(buf1)

		buf2 := GetEncoderBuffer()
		// Buffer should be reset
		if buf2.Len() != 0 {
			t.Errorf("Buffer length = %d, want 0", buf2.Len())
		}
		PutEncoderBuffer(buf2)
	})

	t.Run("Nil buffer", func(t *testing.T) {
		// Should not panic
		PutEncoderBuffer(nil)
	})

	t.Run("Oversized buffer discarded", func(t *testing.T) {
		buf := GetEncoderBuffer()
		// Grow beyond max size
		buf.Grow(16 * 1024)
		buf.WriteString("oversized")
		// This should be discarded, not pooled
		PutEncoderBuffer(buf)
	})
}

// TestStringToBytes tests string to bytes conversion
func TestStringToBytes(t *testing.T) {
	tests := []string{
		"",
		"hello",
		"test with spaces",
		"unicode: 你好世界",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			result := StringToBytes(tt)
			if string(result) != tt {
				t.Errorf("StringToBytes(%q) = %q, want %q", tt, string(result), tt)
			}
		})
	}
}

// TestParseIntFast tests fast integer parsing
func TestParseIntFast(t *testing.T) {
	tests := []struct {
		s        string
		expected int
		ok       bool
	}{
		{"0", 0, true},
		{"1", 1, true},
		{"9", 9, true},
		{"10", 10, true},
		{"99", 99, true},
		{"100", 100, true},
		{"12345", 12345, true},
		{"-1", -1, true},
		{"-99", -99, true},
		{"-123", -123, true},
		{"", 0, false},
		{"-", 0, false},
		{"abc", 0, false},
		{"12a", 0, false},
		{"a12", 0, false},
		{"--1", 0, false},
		{"1.5", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got, ok := ParseIntFast(tt.s)
			if ok != tt.ok {
				t.Errorf("ParseIntFast(%q) ok = %v, want %v", tt.s, ok, tt.ok)
			}
			if ok && got != tt.expected {
				t.Errorf("ParseIntFast(%q) = %d, want %d", tt.s, got, tt.expected)
			}
		})
	}
}

// TestIntToStringFast tests fast integer to string conversion
func TestIntToStringFast(t *testing.T) {
	tests := []struct {
		n        int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "10"},
		{50, "50"},
		{99, "99"},
		{100, "100"},
		{1000, "1000"},
		{-1, "-1"},
		{-99, "-99"},
		{-100, "-100"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := IntToStringFast(tt.n); got != tt.expected {
				t.Errorf("IntToStringFast(%d) = %q, want %q", tt.n, got, tt.expected)
			}
		})
	}
}

// Benchmark tests for encoding functions
func BenchmarkMarshalJSON_Compact(b *testing.B) {
	data := map[string]any{"name": "test", "value": 42}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = MarshalJSON(data, false, "", "")
	}
}

func BenchmarkMarshalJSON_Pretty(b *testing.B) {
	data := map[string]any{"name": "test", "value": 42}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = MarshalJSON(data, true, "", "  ")
	}
}

func BenchmarkParseIntFast(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseIntFast("12345")
	}
}

func BenchmarkIntToStringFast(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IntToStringFast(50)
	}
}
