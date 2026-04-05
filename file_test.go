package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// FILE OPERATIONS TESTS
// Tests for file.go: LoadFromFile, SaveToFile, LargeFileProcessor,
// NDJSONProcessor, ChunkedWriter, SamplingReader, LazyParser
// ============================================================================

// TestLoadFromFile tests the LoadFromFile method
func TestLoadFromFile(t *testing.T) {
	processor, _ := New()
	defer processor.Close()

	t.Run("ValidFile", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.json")
		testData := `{"name":"test","value":123}`

		err := os.WriteFile(filePath, []byte(testData), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		loaded, err := processor.LoadFromFile(filePath)
		if err != nil {
			t.Errorf("LoadFromFile failed: %v", err)
		}
		if loaded != testData {
			t.Errorf("Loaded data = %q, want %q", loaded, testData)
		}
	})

	t.Run("LoadFromFileAsData", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test_data.json")
		testData := `{"name":"test","nested":{"key":"value"}}`

		err := os.WriteFile(filePath, []byte(testData), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		data, err := processor.LoadFromFileAsData(filePath)
		if err != nil {
			t.Errorf("LoadFromFileAsData failed: %v", err)
		}

		result, ok := data.(map[string]interface{})
		if !ok {
			t.Errorf("Expected map[string]interface{}, got %T", data)
		}
		if result["name"] != "test" {
			t.Errorf("name = %v, want test", result["name"])
		}
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := processor.LoadFromFile("/non/existent/file.json")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("EmptyFilePath", func(t *testing.T) {
		_, err := processor.LoadFromFile("")
		if err == nil {
			t.Error("Expected error for empty file path")
		}
	})
}

// TestSaveToFile tests the SaveToFile method
func TestSaveToFile(t *testing.T) {
	processor, _ := New()
	defer processor.Close()

	t.Run("SaveAndLoad", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "save_test.json")
		testData := map[string]interface{}{"name": "test", "value": 123}

		cfg := DefaultConfig()
		cfg.Pretty = false
		err := processor.SaveToFile(filePath, testData, cfg)
		if err != nil {
			t.Errorf("SaveToFile failed: %v", err)
		}

		loaded, err := processor.LoadFromFile(filePath)
		if err != nil {
			t.Errorf("LoadFromFile failed: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(loaded), &parsed); err != nil {
			t.Errorf("Failed to parse loaded data: %v", err)
		}

		if parsed["name"] != "test" {
			t.Errorf("name = %v, want test", parsed["name"])
		}
	})

	t.Run("PrettyPrint", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "pretty_test.json")
		testData := map[string]interface{}{"name": "test", "nested": map[string]interface{}{"key": "value"}}

		cfg := DefaultConfig()
		cfg.Pretty = true
		err := processor.SaveToFile(filePath, testData, cfg)
		if err != nil {
			t.Errorf("SaveToFile with pretty failed: %v", err)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}

		if !strings.Contains(string(content), "\n") {
			t.Error("Expected pretty-printed JSON with newlines")
		}
	})

	t.Run("CreateDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "subdir", "nested", "test.json")
		testData := map[string]interface{}{"created": true}

		cfg := DefaultConfig()
		cfg.Pretty = false
		err := processor.SaveToFile(filePath, testData, cfg)
		if err != nil {
			t.Errorf("SaveToFile with directory creation failed: %v", err)
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("File was not created")
		}
	})

	t.Run("SaveStringData", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "string_test.json")
		testData := `{"already":"json"}`

		cfg := DefaultConfig()
		cfg.Pretty = false
		err := processor.SaveToFile(filePath, testData, cfg)
		if err != nil {
			t.Errorf("SaveToFile with string data failed: %v", err)
		}
	})
}

// TestLoadFromReader tests the LoadFromReader method
func TestLoadFromReader(t *testing.T) {
	processor, _ := New()
	defer processor.Close()

	t.Run("ValidReader", func(t *testing.T) {
		testData := `{"name":"reader_test"}`
		reader := strings.NewReader(testData)

		loaded, err := processor.LoadFromReader(reader)
		if err != nil {
			t.Errorf("LoadFromReader failed: %v", err)
		}
		if loaded != testData {
			t.Errorf("Loaded data = %q, want %q", loaded, testData)
		}
	})

	t.Run("LoadFromReaderAsData", func(t *testing.T) {
		testData := `{"key":"value","number":42}`
		reader := strings.NewReader(testData)

		data, err := processor.LoadFromReaderAsData(reader)
		if err != nil {
			t.Errorf("LoadFromReaderAsData failed: %v", err)
		}

		_, ok := data.(map[string]interface{})
		if !ok {
			t.Errorf("Expected map[string]interface{}, got %T", data)
		}
	})
}

// TestSaveToWriter tests the SaveToWriter method
func TestSaveToWriter(t *testing.T) {
	processor, _ := New()
	defer processor.Close()

	t.Run("SaveToWriter", func(t *testing.T) {
		testData := map[string]interface{}{"writer": "test"}
		var buf bytes.Buffer

		cfg := DefaultConfig()
		cfg.Pretty = false
		err := processor.SaveToWriter(&buf, testData, cfg)
		if err != nil {
			t.Errorf("SaveToWriter failed: %v", err)
		}

		if buf.Len() == 0 {
			t.Error("Expected non-empty buffer")
		}
	})

	t.Run("SaveToWriterPretty", func(t *testing.T) {
		testData := map[string]interface{}{"pretty": true}
		var buf bytes.Buffer

		cfg := DefaultConfig()
		cfg.Pretty = true
		err := processor.SaveToWriter(&buf, testData, cfg)
		if err != nil {
			t.Errorf("SaveToWriter with pretty failed: %v", err)
		}

		if !strings.Contains(buf.String(), "\n") {
			t.Error("Expected pretty-printed JSON")
		}
	})
}

// TestMarshalToFile tests the MarshalToFile method
func TestMarshalToFile(t *testing.T) {
	processor, _ := New()
	defer processor.Close()

	t.Run("MarshalAndUnmarshal", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "marshal_test.json")

		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		testData := TestStruct{Name: "marshal_test", Value: 42}

		cfg := DefaultConfig()
		cfg.Pretty = false
		err := processor.MarshalToFile(filePath, testData, cfg)
		if err != nil {
			t.Errorf("MarshalToFile failed: %v", err)
		}

		var loaded TestStruct
		err = processor.UnmarshalFromFile(filePath, &loaded)
		if err != nil {
			t.Errorf("UnmarshalFromFile failed: %v", err)
		}

		if loaded.Name != testData.Name || loaded.Value != testData.Value {
			t.Errorf("Loaded = %+v, want %+v", loaded, testData)
		}
	})

	t.Run("MarshalPretty", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "marshal_pretty.json")

		testData := map[string]interface{}{"pretty": true}

		cfg := DefaultConfig()
		cfg.Pretty = true
		err := processor.MarshalToFile(filePath, testData, cfg)
		if err != nil {
			t.Errorf("MarshalToFile with pretty failed: %v", err)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
		}

		if !strings.Contains(string(content), "\n") {
			t.Error("Expected pretty-printed JSON")
		}
	})
}

// ============================================================================
// LARGE FILE PROCESSOR TESTS
// ============================================================================

// TestLargeFileProcessor tests the large file processing methods on Processor
func TestLargeFileProcessor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file tests in short mode")
	}

	t.Run("ForeachFile", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "large_test.json")

		// Create a large JSON array
		var items []map[string]interface{}
		for i := 0; i < 100; i++ {
			items = append(items, map[string]interface{}{
				"id":    i,
				"name":  fmt.Sprintf("Item%d", i),
				"value": i * 10,
			})
		}
		data, _ := json.Marshal(items)
		os.WriteFile(filePath, data, 0644)

		processor, err := New()
		if err != nil {
			t.Fatalf("Failed to create processor: %v", err)
		}

		count := 0
		err = processor.ForeachFile(filePath, func(key any, item *IterableValue) error {
			count++
			// Verify we can access fields using IterableValue
			id := item.GetInt("id")
			name := item.GetString("name")
			if id < 0 || name == "" {
				t.Errorf("Unexpected values: id=%d, name=%s", id, name)
			}
			return nil
		})

		if err != nil {
			t.Errorf("ForeachFile failed: %v", err)
		}
		if count != 100 {
			t.Errorf("Processed %d items, want 100", count)
		}
	})

	t.Run("ForeachFileChunked", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "chunked_test.json")

		// Create a large JSON array
		var items []map[string]interface{}
		for i := 0; i < 50; i++ {
			items = append(items, map[string]interface{}{
				"id":    i,
				"value": i * 2,
			})
		}
		data, _ := json.Marshal(items)
		os.WriteFile(filePath, data, 0644)

		processor, err := New()
		if err != nil {
			t.Fatalf("Failed to create processor: %v", err)
		}

		chunks := 0
		totalItems := 0
		err = processor.ForeachFileChunked(filePath, 10, func(chunk []*IterableValue) error {
			chunks++
			totalItems += len(chunk)
			// Verify we can access fields using IterableValue
			for _, item := range chunk {
				_ = item.GetInt("id")
			}
			return nil
		})

		if err != nil {
			t.Errorf("ForeachFileChunked failed: %v", err)
		}
		if chunks < 5 {
			t.Errorf("Expected at least 5 chunks, got %d", chunks)
		}
		if totalItems != 50 {
			t.Errorf("Total items = %d, want 50", totalItems)
		}
	})
}

// TestDefaultConfigLargeFile tests that DefaultConfig has proper large file settings
func TestDefaultConfigLargeFile(t *testing.T) {
	config := DefaultConfig()

	if config.ChunkSize <= 0 {
		t.Error("ChunkSize should be positive")
	}
	if config.MaxMemory <= 0 {
		t.Error("MaxMemory should be positive")
	}
	if config.BufferSize <= 0 {
		t.Error("BufferSize should be positive")
	}
}

// ============================================================================
// NDJSON PROCESSOR TESTS
// ============================================================================

// TestNDJSONProcessor tests the NDJSONProcessor
func TestNDJSONProcessor(t *testing.T) {
	t.Run("ProcessFile", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.ndjson")

		// Create NDJSON file
		content := `{"id":1,"name":"first"}
{"id":2,"name":"second"}
{"id":3,"name":"third"}
`
		os.WriteFile(filePath, []byte(content), 0644)

		np := NewNDJSONProcessor(1024)

		lines := 0
		err := np.ProcessFile(filePath, func(lineNum int, obj map[string]interface{}) error {
			lines++
			return nil
		})

		if err != nil {
			t.Errorf("ProcessFile failed: %v", err)
		}
		if lines != 3 {
			t.Errorf("Processed %d lines, want 3", lines)
		}
	})

	t.Run("ProcessReader", func(t *testing.T) {
		content := `{"id":1}
{"id":2}
`
		reader := strings.NewReader(content)

		np := NewNDJSONProcessor(1024)

		lines := 0
		err := np.ProcessReader(reader, func(lineNum int, obj map[string]interface{}) error {
			lines++
			return nil
		})

		if err != nil {
			t.Errorf("ProcessReader failed: %v", err)
		}
		if lines != 2 {
			t.Errorf("Processed %d lines, want 2", lines)
		}
	})

	t.Run("SkipInvalidLines", func(t *testing.T) {
		content := `{"id":1}
invalid json
{"id":2}
`
		reader := strings.NewReader(content)

		np := NewNDJSONProcessor(1024)

		lines := 0
		err := np.ProcessReader(reader, func(lineNum int, obj map[string]interface{}) error {
			lines++
			return nil
		})

		if err != nil {
			t.Errorf("ProcessReader failed: %v", err)
		}
		// Should only process valid lines
		if lines != 2 {
			t.Errorf("Processed %d lines, want 2 (skipping invalid)", lines)
		}
	})

	t.Run("SkipEmptyLines", func(t *testing.T) {
		content := `{"id":1}

{"id":2}
`
		reader := strings.NewReader(content)

		np := NewNDJSONProcessor(1024)

		lines := 0
		err := np.ProcessReader(reader, func(lineNum int, obj map[string]interface{}) error {
			lines++
			return nil
		})

		if err != nil {
			t.Errorf("ProcessReader failed: %v", err)
		}
		if lines != 2 {
			t.Errorf("Processed %d lines, want 2 (skipping empty)", lines)
		}
	})
}

// ============================================================================
// CHUNKED WRITER TESTS
// ============================================================================

// TestChunkedWriter tests the ChunkedWriter
func TestChunkedWriter(t *testing.T) {
	t.Run("WriteArray", func(t *testing.T) {
		var buf bytes.Buffer
		cw := NewChunkedWriter(&buf, 1024, true)

		for i := 0; i < 5; i++ {
			if err := cw.WriteItem(map[string]interface{}{"id": i}); err != nil {
				t.Errorf("WriteItem failed: %v", err)
			}
		}

		if err := cw.Flush(true); err != nil {
			t.Errorf("Flush failed: %v", err)
		}

		result := buf.String()
		if result[0] != '[' || result[len(result)-1] != ']' {
			t.Errorf("Result should be a JSON array, got: %s", result)
		}

		if cw.Count() != 5 {
			t.Errorf("Count = %d, want 5", cw.Count())
		}
	})

	t.Run("WriteObject", func(t *testing.T) {
		var buf bytes.Buffer
		cw := NewChunkedWriter(&buf, 1024, false)

		for i := 0; i < 3; i++ {
			if err := cw.WriteKeyValue(fmt.Sprintf("key%d", i), i); err != nil {
				t.Errorf("WriteKeyValue failed: %v", err)
			}
		}

		if err := cw.Flush(true); err != nil {
			t.Errorf("Flush failed: %v", err)
		}

		result := buf.String()
		if result[0] != '{' || result[len(result)-1] != '}' {
			t.Errorf("Result should be a JSON object, got: %s", result)
		}
	})

	t.Run("AutoFlush", func(t *testing.T) {
		var buf bytes.Buffer
		// Small chunk size to trigger auto-flush
		cw := NewChunkedWriter(&buf, 50, true)

		for i := 0; i < 100; i++ {
			if err := cw.WriteItem(map[string]interface{}{"id": i, "data": strings.Repeat("x", 100)}); err != nil {
				t.Errorf("WriteItem failed: %v", err)
			}
		}

		// Buffer should have content even before final flush
		if buf.Len() == 0 {
			t.Error("Expected auto-flush to write data")
		}
	})
}

// ============================================================================
// STREAMING PROCESSOR FILE TESTS
// ============================================================================

// TestStreamingProcessorWithFile tests streaming with file-like data
func TestStreamingProcessorWithFile(t *testing.T) {
	t.Run("StreamLargeArray", func(t *testing.T) {
		// Create a large array
		var buf bytes.Buffer
		buf.WriteString("[")
		for i := 0; i < 1000; i++ {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(fmt.Sprintf(`{"id":%d}`, i))
		}
		buf.WriteString("]")

		sp := newStreamingProcessor(bytes.NewReader(buf.Bytes()), 0)

		count := 0
		err := sp.StreamArray(func(index int, item interface{}) bool {
			count++
			return true
		})

		if err != nil {
			t.Errorf("StreamArray failed: %v", err)
		}
		if count != 1000 {
			t.Errorf("Streamed %d items, want 1000", count)
		}
	})

	t.Run("StreamWithEarlyStop", func(t *testing.T) {
		var buf bytes.Buffer
		buf.WriteString("[")
		for i := 0; i < 100; i++ {
			if i > 0 {
				buf.WriteString(",")
			}
			buf.WriteString(fmt.Sprintf(`%d`, i))
		}
		buf.WriteString("]")

		sp := newStreamingProcessor(bytes.NewReader(buf.Bytes()), 0)

		count := 0
		err := sp.StreamArray(func(index int, item interface{}) bool {
			count++
			return count < 10 // Stop after 10
		})

		if err != nil {
			t.Errorf("StreamArray failed: %v", err)
		}
		if count != 10 {
			t.Errorf("Streamed %d items, want 10", count)
		}
	})
}


// ============================================================================
// Processor Streaming Methods Tests
// ============================================================================

func TestProcessorStreamArray(t *testing.T) {
	processor, err := New()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	jsonData := `["a", "b", "c", "d", "e"]`
	reader := strings.NewReader(jsonData)

	var items []string
	err = processor.StreamArray(reader, func(index int, item any) bool {
		if s, ok := item.(string); ok {
			items = append(items, s)
		}
		return true
	})
	if err != nil {
		t.Fatalf("StreamArray failed: %v", err)
	}

	if len(items) != 5 {
		t.Errorf("Expected 5 items, got %d", len(items))
	}
}

func TestProcessorStreamObject(t *testing.T) {
	processor, err := New()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	jsonData := `{"a": 1, "b": 2, "c": 3}`
	reader := strings.NewReader(jsonData)

	result := make(map[string]any)
	err = processor.StreamObject(reader, func(key string, value any) bool {
		result[key] = value
		return true
	})
	if err != nil {
		t.Fatalf("StreamObject failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(result))
	}
}

func TestProcessorStreamArrayChunked(t *testing.T) {
	processor, err := New()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	// Create JSON array with 10 elements
	var buf bytes.Buffer
	buf.WriteString("[")
	for i := 0; i < 10; i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf(`%d`, i))
	}
	buf.WriteString("]")
	reader := bytes.NewReader(buf.Bytes())

	var totalBatches int
	var totalItems int
	err = processor.StreamArrayChunked(reader, 3, func(chunk []any) error {
		totalBatches++
		totalItems += len(chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamArrayChunked failed: %v", err)
	}

	if totalItems != 10 {
		t.Errorf("Expected 10 items, got %d", totalItems)
	}

	// Should have 4 batches: 3, 3, 3, 1
	if totalBatches != 4 {
		t.Errorf("Expected 4 batches, got %d", totalBatches)
	}
}

func TestProcessorStreamObjectChunked(t *testing.T) {
	processor, err := New()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	jsonData := `{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5}`
	reader := strings.NewReader(jsonData)

	var totalBatches int
	var totalKeys int
	err = processor.StreamObjectChunked(reader, 2, func(chunk map[string]any) error {
		totalBatches++
		totalKeys += len(chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamObjectChunked failed: %v", err)
	}

	if totalKeys != 5 {
		t.Errorf("Expected 5 keys, got %d", totalKeys)
	}

	// Should have 3 batches: 2, 2, 1
	if totalBatches != 3 {
		t.Errorf("Expected 3 batches, got %d", totalBatches)
	}
}

func TestProcessorStreamArrayWithStats(t *testing.T) {
	processor, err := New()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	jsonData := `[1, 2, 3, 4, 5]`
	reader := strings.NewReader(jsonData)

	stats, err := processor.StreamArrayWithStats(reader, func(index int, item any) bool {
		return true
	})
	if err != nil {
		t.Fatalf("StreamArrayWithStats failed: %v", err)
	}

	if stats.ItemsProcessed != 5 {
		t.Errorf("Expected 5 items processed, got %d", stats.ItemsProcessed)
	}
}

func TestProcessorStreamArrayEarlyStop(t *testing.T) {
	processor, err := New()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}
	defer processor.Close()

	jsonData := `[1, 2, 3, 4, 5]`
	reader := strings.NewReader(jsonData)

	var count int
	err = processor.StreamArray(reader, func(index int, item any) bool {
		count++
		return count < 3 // Stop after 3 items
	})
	if err != nil {
		t.Fatalf("StreamArray failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 items, got %d", count)
	}
}

func TestProcessorStreamOnClosedProcessor(t *testing.T) {
	processor, err := New()
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Close immediately
	processor.Close()

	jsonData := `[1, 2, 3]`
	reader := strings.NewReader(jsonData)

	err = processor.StreamArray(reader, func(index int, item any) bool {
		return true
	})
	if err == nil {
		t.Error("Expected error on closed processor")
	}
}
