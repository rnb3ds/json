package json

import (
	"bytes"
	"strings"
	"sync/atomic"
	"testing"
)

// ============================================================================
// JSONL CONFIG TESTS
// ============================================================================

func TestDefaultJSONLConfig(t *testing.T) {
	config := DefaultJSONLConfig()

	if config.BufferSize <= 0 {
		t.Error("BufferSize should be positive")
	}
	if config.MaxLineSize <= 0 {
		t.Error("MaxLineSize should be positive")
	}
	if !config.SkipEmpty {
		t.Error("SkipEmpty should be true by default")
	}
	if config.SkipComments {
		t.Error("SkipComments should be false by default")
	}
	if config.ContinueOnErr {
		t.Error("ContinueOnErr should be false by default")
	}
}

func TestShouldSkipJSONLLine(t *testing.T) {
	tests := []struct {
		name     string
		line     []byte
		config   JSONLConfig
		expected bool
	}{
		{
			name:     "empty_line_skip_true",
			line:     []byte{},
			config:   JSONLConfig{SkipEmpty: true},
			expected: true,
		},
		{
			name:     "empty_line_skip_false",
			line:     []byte{},
			config:   JSONLConfig{SkipEmpty: false},
			expected: false,
		},
		{
			name:     "hash_comment_skip",
			line:     []byte("# this is a comment"),
			config:   JSONLConfig{SkipComments: true},
			expected: true,
		},
		{
			name:     "double_slash_comment_skip",
			line:     []byte("// this is a comment"),
			config:   JSONLConfig{SkipComments: true},
			expected: true,
		},
		{
			name:     "comment_skip_disabled",
			line:     []byte("# this is a comment"),
			config:   JSONLConfig{SkipComments: false},
			expected: false,
		},
		{
			name:     "normal_json_not_skipped",
			line:     []byte(`{"key":"value"}`),
			config:   DefaultJSONLConfig(),
			expected: false,
		},
		{
			name:     "single_slash_not_comment",
			line:     []byte(`/path/to/file`),
			config:   JSONLConfig{SkipComments: true},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSkipJSONLLine(tt.line, tt.config)
			if result != tt.expected {
				t.Errorf("shouldSkipJSONLLine() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ============================================================================
// JSONL PROCESSOR TESTS
// ============================================================================

func TestNewJSONLProcessor(t *testing.T) {
	input := `{"a":1}
{"b":2}
{"c":3}`

	processor := NewJSONLProcessor(strings.NewReader(input))
	defer processor.Release()

	if processor == nil {
		t.Fatal("NewJSONLProcessor returned nil")
	}
}

func TestNewJSONLProcessorWithOptions(t *testing.T) {
	input := `{"a":1}`

	customConfig := JSONLConfig{
		BufferSize:   32 * 1024,
		MaxLineSize:  512 * 1024,
		SkipEmpty:    false,
		SkipComments: true,
	}

	processor := NewJSONLProcessorWithConfig(
		strings.NewReader(input),
		customConfig,
	)
	defer processor.Release()

	if processor == nil {
		t.Fatal("NewJSONLProcessorWithOptions returned nil")
	}
	if processor.config.BufferSize != 32*1024 {
		t.Errorf("BufferSize = %d, want %d", processor.config.BufferSize, 32*1024)
	}
}

func TestJSONLProcessor_StreamLines(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		config        JSONLConfig
		expectedCount int
		wantErr       bool
	}{
		{
			name:          "basic_jsonl",
			input:         "{\"name\":\"Alice\"}\n{\"name\":\"Bob\"}\n{\"name\":\"Charlie\"}",
			config:        DefaultJSONLConfig(),
			expectedCount: 3,
			wantErr:       false,
		},
		{
			name:          "skip_empty_lines",
			input:         "{\"a\":1}\n\n{\"b\":2}\n\n{\"c\":3}",
			config:        JSONLConfig{SkipEmpty: true},
			expectedCount: 3,
			wantErr:       false,
		},
		{
			name:          "keep_empty_lines_errors",
			input:         "{\"a\":1}\n\n{\"b\":2}",
			config:        JSONLConfig{SkipEmpty: false},
			expectedCount: 0, // Will error on empty line
			wantErr:       true,
		},
		{
			name:          "skip_comments",
			input:         "# comment\n{\"a\":1}\n// another comment\n{\"b\":2}",
			config:        JSONLConfig{SkipEmpty: true, SkipComments: true},
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name:          "continue_on_error",
			input:         "{\"valid\":1}\n{invalid}\n{\"valid\":2}",
			config:        JSONLConfig{ContinueOnErr: true, SkipEmpty: true},
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name:          "stop_on_error",
			input:         "{\"valid\":1}\n{invalid}\n{\"valid\":2}",
			config:        JSONLConfig{ContinueOnErr: false, SkipEmpty: true},
			expectedCount: 0,
			wantErr:       true,
		},
		{
			name:          "empty_input",
			input:         "",
			config:        DefaultJSONLConfig(),
			expectedCount: 0,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewJSONLProcessorWithConfig(
				strings.NewReader(tt.input),
				tt.config,
			)
			defer processor.Release()

			var count int
			err := processor.StreamLines(func(lineNum int, data any) bool {
				count++
				return true
			})

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if count != tt.expectedCount {
					t.Errorf("Expected %d lines, got %d", tt.expectedCount, count)
				}
			}
		})
	}
}

func TestJSONLProcessor_StreamLines_EarlyStop(t *testing.T) {
	input := "{\"a\":1}\n{\"b\":2}\n{\"c\":3}\n{\"d\":4}\n{\"e\":5}"

	processor := NewJSONLProcessor(strings.NewReader(input))
	defer processor.Release()

	var count int
	err := processor.StreamLines(func(lineNum int, data any) bool {
		count++
		return lineNum < 3 // Stop after 3 lines
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 lines processed, got %d", count)
	}
}

func TestJSONLProcessor_StreamLinesParallel(t *testing.T) {
	// Create input with 100 lines
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, `{"id":`+string(rune('0'+i%10))+`}`)
	}
	input := strings.Join(lines, "\n")

	processor := NewJSONLProcessor(strings.NewReader(input))
	defer processor.Release()

	var count int32
	err := processor.StreamLinesParallel(func(lineNum int, data any) error {
		atomic.AddInt32(&count, 1)
		return nil
	}, 4)

	if err != nil {
		t.Errorf("StreamLinesParallel error: %v", err)
	}
	if count != 100 {
		t.Errorf("Expected 100 lines, got %d", count)
	}
}

func TestJSONLProcessor_StreamLinesParallel_WithError(t *testing.T) {
	input := "{\"a\":1}\n{\"b\":2}\n{\"c\":3}"

	processor := NewJSONLProcessor(strings.NewReader(input))
	defer processor.Release()

	err := processor.StreamLinesParallel(func(lineNum int, data any) error {
		if lineNum == 2 {
			return ErrOperationFailed
		}
		return nil
	}, 2)

	if err == nil {
		t.Error("Expected error from worker")
	}
}

func TestJSONLProcessor_Stop(t *testing.T) {
	input := strings.Repeat("{\"a\":1}\n", 1000)

	processor := NewJSONLProcessor(strings.NewReader(input))
	defer processor.Release()

	// Stop before processing
	processor.Stop()

	if !processor.stopped.Load() {
		t.Error("Processor should be stopped")
	}
}

func TestJSONLProcessor_GetStats(t *testing.T) {
	input := "{\"a\":1}\n{\"b\":2}\n{\"c\":3}"

	processor := NewJSONLProcessor(strings.NewReader(input))

	_ = processor.StreamLines(func(lineNum int, data any) bool {
		return true
	})

	stats := processor.GetStats()
	processor.Release()

	if stats.LinesProcessed != 3 {
		t.Errorf("LinesProcessed = %d, want 3", stats.LinesProcessed)
	}
	if stats.BytesRead == 0 {
		t.Error("BytesRead should be > 0")
	}
}

func TestJSONLProcessor_Err(t *testing.T) {
	input := "invalid json line"

	processor := NewJSONLProcessor(strings.NewReader(input))
	processor.config.SkipEmpty = false

	_ = processor.StreamLines(func(lineNum int, data any) bool {
		return true
	})
	processor.Release()

	// Err() returns scanner errors, not parse errors
	// Parse errors are returned directly from StreamLines
}

// ============================================================================
// JSONL WRITER TESTS
// ============================================================================

func TestNewJSONLWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewJSONLWriter(&buf)

	if writer == nil {
		t.Fatal("NewJSONLWriter returned nil")
	}
}

func TestJSONLWriter_Write(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		wantErr  bool
		checkOut func(t *testing.T, output string)
	}{
		{
			name:    "simple_map",
			data:    map[string]any{"key": "value"},
			wantErr: false,
			checkOut: func(t *testing.T, output string) {
				if !strings.Contains(output, `"key"`) || !strings.Contains(output, `"value"`) {
					t.Errorf("Output should contain key and value, got: %s", output)
				}
			},
		},
		{
			name:    "simple_array",
			data:    []any{1, 2, 3},
			wantErr: false,
			checkOut: func(t *testing.T, output string) {
				if !strings.Contains(output, "[") {
					t.Errorf("Output should be an array, got: %s", output)
				}
			},
		},
		{
			name:    "nested_object",
			data:    map[string]any{"inner": map[string]any{"deep": "value"}},
			wantErr: false,
			checkOut: func(t *testing.T, output string) {
				if !strings.Contains(output, "inner") || !strings.Contains(output, "deep") {
					t.Errorf("Output should contain nested keys, got: %s", output)
				}
			},
		},
		{
			name:    "primitive_string",
			data:    "hello",
			wantErr: false,
			checkOut: func(t *testing.T, output string) {
				if !strings.Contains(output, "hello") {
					t.Errorf("Output should contain string, got: %s", output)
				}
			},
		},
		{
			name:    "primitive_number",
			data:    42,
			wantErr: false,
			checkOut: func(t *testing.T, output string) {
				if !strings.Contains(output, "42") {
					t.Errorf("Output should contain number, got: %s", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := NewJSONLWriter(&buf)

			err := writer.Write(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				tt.checkOut(t, buf.String())
			}
		})
	}
}

func TestJSONLWriter_WriteAll(t *testing.T) {
	var buf bytes.Buffer
	writer := NewJSONLWriter(&buf)

	data := []any{
		map[string]any{"id": 1},
		map[string]any{"id": 2},
		map[string]any{"id": 3},
	}

	err := writer.WriteAll(data)
	if err != nil {
		t.Errorf("WriteAll error: %v", err)
	}

	output := buf.String()
	lines := strings.Count(output, "\n")
	if lines != 3 {
		t.Errorf("Expected 3 lines, got %d", lines)
	}
}

func TestJSONLWriter_WriteRaw(t *testing.T) {
	tests := []struct {
		name     string
		line     []byte
		expected string
	}{
		{
			name:     "with_newline",
			line:     []byte("{\"key\":\"value\"}\n"),
			expected: "{\"key\":\"value\"}\n",
		},
		{
			name:     "without_newline",
			line:     []byte("{\"key\":\"value\"}"),
			expected: "{\"key\":\"value\"}\n",
		},
		{
			name:     "empty_line",
			line:     []byte{},
			expected: "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := NewJSONLWriter(&buf)

			err := writer.WriteRaw(tt.line)
			if err != nil {
				t.Errorf("WriteRaw error: %v", err)
			}

			if buf.String() != tt.expected {
				t.Errorf("Output = %q, want %q", buf.String(), tt.expected)
			}
		})
	}
}

func TestJSONLWriter_Stats(t *testing.T) {
	var buf bytes.Buffer
	writer := NewJSONLWriter(&buf)

	_ = writer.Write(map[string]any{"a": 1})
	_ = writer.Write(map[string]any{"b": 2})
	_ = writer.WriteRaw([]byte("{\"c\":3}"))

	stats := writer.Stats()

	if stats.LinesProcessed != 3 {
		t.Errorf("LinesProcessed = %d, want 3", stats.LinesProcessed)
	}
}

func TestJSONLWriter_Err(t *testing.T) {
	var buf bytes.Buffer
	writer := NewJSONLWriter(&buf)

	// First write succeeds
	_ = writer.Write(map[string]any{"a": 1})

	// No error yet
	if writer.Err() != nil {
		t.Errorf("Err() = %v, want nil", writer.Err())
	}
}

// ============================================================================
// UTILITY FUNCTION TESTS
// ============================================================================

func TestParseJSONL(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCount int
		wantErr       bool
	}{
		{
			name:          "basic",
			input:         "{\"a\":1}\n{\"b\":2}\n{\"c\":3}",
			expectedCount: 3,
			wantErr:       false,
		},
		{
			name:          "with_empty_lines",
			input:         "{\"a\":1}\n\n{\"b\":2}",
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name:          "empty_input",
			input:         "",
			expectedCount: 0,
			wantErr:       false,
		},
		{
			name:          "invalid_json",
			input:         "{\"a\":1}\n{invalid}\n{\"c\":3}",
			expectedCount: 0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ParseJSONL([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(results) != tt.expectedCount {
					t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
				}
			}
		})
	}
}

func TestParseJSONLInto(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	input := "{\"name\":\"Alice\",\"age\":30}\n{\"name\":\"Bob\",\"age\":25}"

	results, err := ParseJSONLInto[Person]([]byte(input))
	if err != nil {
		t.Fatalf("ParseJSONLInto error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results[0].Name != "Alice" {
		t.Errorf("results[0].Name = %q, want %q", results[0].Name, "Alice")
	}
	if results[1].Age != 25 {
		t.Errorf("results[1].Age = %d, want 25", results[1].Age)
	}
}

func TestToJSONL(t *testing.T) {
	tests := []struct {
		name          string
		data          []any
		expectedLines int
		wantErr       bool
	}{
		{
			name:          "basic",
			data:          []any{map[string]any{"a": 1}, map[string]any{"b": 2}},
			expectedLines: 2,
			wantErr:       false,
		},
		{
			name:          "empty",
			data:          []any{},
			expectedLines: 0,
			wantErr:       false,
		},
		{
			name:          "with_primitives",
			data:          []any{"hello", 42, true},
			expectedLines: 3,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToJSONL(tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if tt.expectedLines == 0 {
					if len(result) != 0 {
						t.Errorf("Expected empty result, got %d bytes", len(result))
					}
				} else {
					lines := strings.Count(string(result), "\n")
					if lines != tt.expectedLines {
						t.Errorf("Expected %d lines, got %d", tt.expectedLines, lines)
					}
				}
			}
		})
	}
}

func TestToJSONLString(t *testing.T) {
	data := []any{
		map[string]any{"name": "Alice"},
		map[string]any{"name": "Bob"},
	}

	result, err := ToJSONLString(data)
	if err != nil {
		t.Fatalf("ToJSONLString error: %v", err)
	}

	lines := strings.Count(result, "\n")
	if lines != 2 {
		t.Errorf("Expected 2 lines, got %d", lines)
	}
}

// ============================================================================
// STREAM LINES INTO TESTS
// ============================================================================

func TestStreamLinesInto(t *testing.T) {
	type Item struct {
		ID    int    `json:"id"`
		Value string `json:"value"`
	}

	input := "{\"id\":1,\"value\":\"first\"}\n{\"id\":2,\"value\":\"second\"}"

	results, err := StreamLinesInto[Item](strings.NewReader(input), func(lineNum int, data Item) error {
		if data.ID <= 0 {
			return ErrInvalidJSON
		}
		return nil
	})

	if err != nil {
		t.Fatalf("StreamLinesInto error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	if results[0].Value != "first" {
		t.Errorf("results[0].Value = %q, want %q", results[0].Value, "first")
	}
}

func TestStreamLinesIntoWithConfig(t *testing.T) {
	type Record struct {
		Key string `json:"key"`
	}

	input := "# comment\n{\"key\":\"value1\"}\n\n{\"key\":\"value2\"}"

	config := JSONLConfig{
		BufferSize:   64 * 1024,
		MaxLineSize:  1024 * 1024,
		SkipEmpty:    true,
		SkipComments: true,
	}

	results, err := StreamLinesIntoWithConfig[Record](strings.NewReader(input), config, nil)
	if err != nil {
		t.Fatalf("StreamLinesIntoWithConfig error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results (comments and empty lines skipped), got %d", len(results))
	}
}

func TestStreamLinesInto_CallbackError(t *testing.T) {
	type Data struct {
		Value int `json:"value"`
	}

	input := "{\"value\":1}\n{\"value\":2}\n{\"value\":3}"

	_, err := StreamLinesInto[Data](strings.NewReader(input), func(lineNum int, data Data) error {
		if data.Value == 2 {
			return ErrOperationFailed
		}
		return nil
	})

	if err == nil {
		t.Error("Expected error from callback")
	}
}

// ============================================================================
// WITH PROCESSOR CONFIG TEST
// ============================================================================

func TestJSONLProcessorWithCustomProcessor(t *testing.T) {
	input := `{"a":1}`

	customProcessor, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer customProcessor.Close()

	config := DefaultJSONLConfig()
	config.Processor = customProcessor

	processor := NewJSONLProcessorWithConfig(
		strings.NewReader(input),
		config,
	)
	defer processor.Release()

	if processor.processor != customProcessor {
		t.Error("Config.Processor was not set correctly")
	}
}

func TestJSONLProcessorWithNilProcessor(t *testing.T) {
	input := `{"a":1}`

	config := DefaultJSONLConfig()
	config.Processor = nil

	processor := NewJSONLProcessorWithConfig(
		strings.NewReader(input),
		config,
	)
	defer processor.Release()

	// Should use default processor, not crash
	if processor == nil {
		t.Error("Processor should not be nil")
	}
}
