// Package json provides a high-performance, thread-safe JSON processing library
// with 100% encoding/json compatibility and advanced path operations.
//
// Key Features:
//   - 100% encoding/json compatibility - drop-in replacement
//   - High-performance path operations with smart caching
//   - Thread-safe concurrent operations
//   - Type-safe generic operations with Go 1.18+ generics
//   - Memory-efficient resource pooling
//   - Production-ready error handling and validation
//
// Basic Usage:
//
//	// Simple operations (100% compatible with encoding/json)
//	data, err := json.Marshal(value)
//	err = json.Unmarshal(data, &target)
//
//	// Advanced path operations
//	value, err := json.Get(`{"user":{"name":"John"}}`, "user.name")
//	result, err := json.Set(`{"user":{}}`, "user.age", 30)
//
//	// Type-safe operations
//	name, err := json.GetString(jsonStr, "user.name")
//	age, err := json.GetInt(jsonStr, "user.age")
//
//	// Advanced processor for complex operations
//	processor := json.New() // Use default config
//	defer processor.Close()
//	value, err := processor.Get(jsonStr, "complex.path[0].field")
package json

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/cybergodev/json/internal"
)

var (
	defaultProcessor   atomic.Pointer[Processor]
	defaultProcessorMu sync.Mutex
	fallbackProcessor  atomic.Pointer[Processor] // SAFETY: fallback for error recovery
)

// init initializes the fallback processor at package load time.
// SAFETY: Ensures a fallback processor is always available for error recovery.
func init() {
	// Single attempt with default config - should always succeed
	p, err := New()
	if err != nil {
		// This should never happen with DefaultConfig (all defaults are validated).
		// Log warning and continue - callers must handle nil processor gracefully.
		fmt.Fprintf(os.Stderr, "json: warning: failed to create fallback processor: %v\n", err)
		return
	}
	fallbackProcessor.Store(p)
}

// getDefaultProcessor returns the default processor in a panic-safe manner.
// SAFETY: Never panics - returns fallback processor on error.
// Uses sync.OnceValue pattern for efficient lazy initialization.
// IMPORTANT: Callers MUST check for nil return value and handle appropriately.
// FIX: Removed redundant New() call that could cause infinite loop on repeated failures.
func getDefaultProcessor() *Processor {
	// Fast path: check if processor exists and is not closed
	if p := defaultProcessor.Load(); p != nil && !p.IsClosed() {
		return p
	}

	// Slow path: need to create or replace processor
	defaultProcessorMu.Lock()
	defer defaultProcessorMu.Unlock()

	// Double-check after acquiring lock
	if p := defaultProcessor.Load(); p != nil && !p.IsClosed() {
		return p
	}

	// Create new processor
	p, err := New()
	if err != nil {
		// SAFETY: Return fallback processor instead of panicking
		// This ensures the library never terminates the calling program
		if fb := fallbackProcessor.Load(); fb != nil && !fb.IsClosed() {
			return fb
		}
		// FIX: Return nil instead of retrying New() which already failed
		// Callers must handle nil processor gracefully
		return nil
	}
	defaultProcessor.Store(p)
	return p
}

// SetGlobalProcessor sets a custom global processor for package-level operations.
// The processor is used by all package-level functions (Get, Set, Delete, Marshal, etc.).
// Passing nil is a no-op. The previous processor is closed before being replaced.
// This function is thread-safe.
//
// Example:
//
//	cfg := json.DefaultConfig()
//	cfg.EnableCache = true
//	processor, _ := json.New(cfg)
//	json.SetGlobalProcessor(processor)
//
//	// Now all package-level operations use the custom processor
//	data, err := json.Get(`{"name":"Alice"}`, "name")
func SetGlobalProcessor(processor *Processor) {
	if processor == nil {
		return
	}

	defaultProcessorMu.Lock()
	defer defaultProcessorMu.Unlock()

	if old := defaultProcessor.Swap(processor); old != nil {
		old.Close()
	}
}

// ShutdownGlobalProcessor closes and removes the global processor.
// After calling this, package-level operations will create a new default processor
// on first use. Call this for clean shutdown in long-running services.
// This function is thread-safe.
//
// Example:
//
//	// At application shutdown
//	json.ShutdownGlobalProcessor()
func ShutdownGlobalProcessor() {
	defaultProcessorMu.Lock()
	defer defaultProcessorMu.Unlock()

	if old := defaultProcessor.Swap(nil); old != nil {
		old.Close()
	}
}

// Package-level API functions are organized in the following files:
//
//   - api.go       : All exported functions (Get*, Set*, Delete, Marshal, etc.)
//   - encoding.go  : Encoding/decoding implementation details
//   - file.go      : File I/O operations
//   - helpers.go   : Type conversion and utility functions
//   - processor.go : Core Processor implementation
//
// All functions remain in package json and maintain 100% API compatibility.

// ============================================================================
// JSONL (JSON LINES) SUPPORT
// Efficient processing for line-delimited JSON format
// Commonly used for: logs, data pipelines, streaming data
// ============================================================================

// shouldSkipJSONLLineFromConfig checks if a line should be skipped based on Config.
// Uses the unified Config struct with JSONL* fields.
func shouldSkipJSONLLineFromConfig(line []byte, cfg Config) bool {
	// Skip empty lines if configured
	if cfg.JSONLSkipEmpty && len(line) == 0 {
		return true
	}

	// Skip comments if configured
	if cfg.JSONLSkipComments && len(line) > 0 {
		if line[0] == '#' || (len(line) > 1 && line[0] == '/' && line[1] == '/') {
			return true
		}
	}

	return false
}

// StreamLinesInto processes JSONL data into a slice of typed values.
// Uses generics for type-safe processing.
// The optional cfg parameter allows customization using the unified Config pattern.
//
// Example:
//
//	// Default settings
//	results, err := json.StreamLinesInto[MyType](reader, func(lineNum int, data MyType) error {
//	    fmt.Printf("Line %d: %+v\n", lineNum, data)
//	    return nil
//	})
//
//	// With custom configuration
//	cfg := json.DefaultConfig()
//	cfg.JSONLSkipEmpty = false
//	cfg.JSONLSkipComments = true
//	results, err := json.StreamLinesInto[MyType](reader, processFunc, cfg)
func StreamLinesInto[T any](reader io.Reader, fn func(lineNum int, data T) error, cfg ...Config) ([]T, error) {
	var config Config
	if len(cfg) > 0 {
		config = cfg[0]
	} else {
		config = DefaultConfig()
	}

	// Get buffer and line size from config
	bufSize := config.JSONLBufferSize
	if bufSize <= 0 {
		bufSize = 64 * 1024
	}
	maxLineSize := config.JSONLMaxLineSize
	if maxLineSize <= 0 {
		maxLineSize = 1024 * 1024
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, bufSize), maxLineSize)

	var results []T
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		// Use helper to check if line should be skipped
		if shouldSkipJSONLLineFromConfig(line, config) {
			continue
		}

		var data T
		if err := json.Unmarshal(line, &data); err != nil {
			if config.JSONLContinueOnErr {
				continue
			}
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		if fn != nil {
			if err := fn(lineNum, data); err != nil {
				return nil, err
			}
		}

		results = append(results, data)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// ============================================================================
// JSONL WRITER - For writing JSONL format
// ============================================================================

// JSONLStats holds statistics for JSONL (JSON Lines) processing.
// Used by JSONLWriter to track lines and bytes written.
type JSONLStats struct {
	LinesProcessed int64
	BytesRead      int64
	BytesWritten   int64
	CurrentLine    int
}

// JSONLWriter writes JSON Lines (NDJSON) format to an io.Writer.
// Each value is written as a single line of JSON, suitable for
// log files, data pipelines, and streaming applications.
//
// Example:
//
//	file, _ := os.Create("output.jsonl")
//	writer := json.NewJSONLWriter(file)
//	defer file.Close()
//
//	writer.Write(map[string]any{"name": "Alice", "id": 1})
//	writer.Write(map[string]any{"name": "Bob", "id": 2})
type JSONLWriter struct {
	writer   io.Writer
	encoder  *json.Encoder
	lineNum  int
	err      error
	bytesOut int64
}

// NewJSONLWriter creates a new JSONL writer that writes to the provided io.Writer.
// HTML escaping is controlled by Config.EscapeHTML (default: false for performance).
//
// Example:
//
//	var buf bytes.Buffer
//	writer := json.NewJSONLWriter(&buf)
//	writer.Write(map[string]any{"event": "login", "user": "alice"})
//
// With custom config:
//
//	cfg := json.DefaultConfig()
//	cfg.EscapeHTML = true
//	writer := json.NewJSONLWriter(&buf, cfg)
func NewJSONLWriter(writer io.Writer, cfg ...Config) *JSONLWriter {
	config := getConfigOrDefault(cfg...)
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(config.EscapeHTML)
	return &JSONLWriter{
		writer:  writer,
		encoder: encoder,
	}
}

// Write writes a single JSON value as a line to the underlying writer.
// Each value is followed by a newline character.
// Returns any error encountered during writing.
//
// Example:
//
//	writer.Write(map[string]any{"name": "Alice"})  // Writes: {"name":"Alice"}\n
//	writer.Write([]int{1, 2, 3})                   // Writes: [1,2,3]\n
func (w *JSONLWriter) Write(data any) error {
	if w.err != nil {
		return w.err
	}

	if err := w.encoder.Encode(data); err != nil {
		w.err = err
		return err
	}

	w.lineNum++
	return nil
}

// WriteAll writes multiple JSON values as separate lines.
// Stops and returns the first error encountered.
//
// Example:
//
//	writer.WriteAll([]any{
//	    map[string]any{"id": 1},
//	    map[string]any{"id": 2},
//	})
func (w *JSONLWriter) WriteAll(data []any) error {
	for _, d := range data {
		if err := w.Write(d); err != nil {
			return err
		}
	}
	return nil
}

// WriteRaw writes a raw JSON line that is already encoded.
// A newline is appended if the line does not already end with one.
// Use this for pre-encoded JSON to avoid re-encoding overhead.
//
// Example:
//
//	writer.WriteRaw([]byte(`{"pre":"encoded"}`))
func (w *JSONLWriter) WriteRaw(line []byte) error {
	if w.err != nil {
		return w.err
	}

	n, err := w.writer.Write(line)
	if err != nil {
		w.err = err
		return err
	}
	w.bytesOut += int64(n)

	// Add newline if not present
	if len(line) == 0 || line[len(line)-1] != '\n' {
		if _, err := w.writer.Write([]byte{'\n'}); err != nil {
			w.err = err
			return err
		}
		w.bytesOut++
	}

	w.lineNum++
	return nil
}

// Err returns any error encountered during previous write operations.
// Returns nil if no errors have occurred.
func (w *JSONLWriter) Err() error {
	return w.err
}

// Stats returns writing statistics including lines processed and bytes written.
func (w *JSONLWriter) Stats() JSONLStats {
	return JSONLStats{
		LinesProcessed: int64(w.lineNum),
		BytesWritten:   w.bytesOut,
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// ParseJSONL parses JSONL data from a byte slice.
// Uses Config.JSONLSkipComments and Config.JSONLContinueOnErr for processing options.
//
// Example:
//
//	// Basic usage
//	data, err := json.ParseJSONL(jsonlBytes)
//
//	// With custom config
//	cfg := json.DefaultConfig()
//	cfg.JSONLSkipComments = true
//	cfg.JSONLContinueOnErr = true
//	data, err := json.ParseJSONL(jsonlBytes, cfg)
func ParseJSONL(data []byte, cfg ...Config) ([]any, error) {
	config := getConfigOrDefault(cfg...)
	p, err := New(config)
	if err != nil {
		return nil, err
	}
	defer p.Close()

	var results []any
	err = p.StreamJSONL(bytes.NewReader(data), func(lineNum int, item *IterableValue) error {
		results = append(results, item.GetData())
		return nil
	})

	return results, err
}

// ToJSONL converts a slice of values to JSONL format.
// Uses Config.EscapeHTML for encoding options.
//
// Example:
//
//	// Basic usage
//	jsonl, err := json.ToJSONL([]any{map[string]any{"id": 1}, map[string]any{"id": 2}})
//
//	// With custom config
//	cfg := json.DefaultConfig()
//	cfg.EscapeHTML = true
//	jsonl, err := json.ToJSONL(data, cfg)
func ToJSONL(data []any, cfg ...Config) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	config := getConfigOrDefault(cfg...)

	// Estimate buffer size
	estimatedSize := min(len(data)*64, 64*1024)

	buf := internal.GetEncoderBuffer()
	defer internal.PutEncoderBuffer(buf)
	buf.Reset()

	// Grow buffer if needed
	if buf.Cap() < estimatedSize {
		buf.Grow(estimatedSize - buf.Cap())
	}

	// Use processor for encoding with config
	p, err := New(config)
	if err != nil {
		return nil, err
	}
	defer p.Close()

	for _, item := range data {
		encoded, err := p.EncodeWithConfig(item, cfg...)
		if err != nil {
			return nil, err
		}
		buf.WriteString(encoded)
		buf.WriteByte('\n')
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// ToJSONLString converts a slice of values to JSONL format string.
// Uses Config.EscapeHTML and Config.Pretty for encoding options.
//
// Example:
//
//	// Basic usage
//	jsonlStr, err := json.ToJSONLString(data)
//
//	// With custom config
//	cfg := json.DefaultConfig()
//	cfg.EscapeHTML = true
//	jsonlStr, err := json.ToJSONLString(data, cfg)
func ToJSONLString(data []any, cfg ...Config) (string, error) {
	result, err := ToJSONL(data, cfg...)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
