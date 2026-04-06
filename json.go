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
	p, err := New()
	if err != nil {
		// This should never happen with DefaultConfig, but handle gracefully.
		// Create a minimal config with reduced limits as a last resort.
		cfg := DefaultConfig()
		cfg.MaxJSONSize = 1024 * 1024 // 1MB - minimal safe limit
		cfg.EnableCache = false       // Disable cache to reduce complexity
		p, err = New(cfg)
		if err != nil {
			// Final fallback: log and continue without fallback processor
			// Callers must handle nil processor gracefully
			fmt.Fprintf(os.Stderr, "json: warning: failed to create fallback processor: %v\n", err)
			return
		}
	}
	fallbackProcessor.Store(p)
}

// getDefaultProcessor returns the default processor in a panic-safe manner.
// SAFETY: Never panics - returns fallback processor on error.
// Uses sync.OnceValue pattern for efficient lazy initialization.
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
		// Last resort: create a minimal processor directly
		// This should never fail with DefaultConfig
		minimalP, minimalErr := New()
		if minimalErr != nil {
			// Return nil - callers must handle nil processor
			// This is safer than panicking in production
			return nil
		}
		return minimalP
	}
	defaultProcessor.Store(p)
	return p
}

// SetGlobalProcessor sets a custom global processor (thread-safe)
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

// ShutdownGlobalProcessor shuts down the global processor
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

// JSONLConfig holds configuration for JSONL processing
type JSONLConfig struct {
	BufferSize    int        // Buffer size for reading (default: 64KB)
	MaxLineSize   int        // Maximum line size (default: 1MB)
	SkipEmpty     bool       // Skip empty lines (default: true)
	SkipComments  bool       // Skip lines starting with # or // (default: false)
	ContinueOnErr bool       // Continue processing on parse errors (default: false)
	Processor     *Processor // Optional custom processor (default: global processor)
}

// DefaultJSONLConfig returns the default JSONL configuration
func DefaultJSONLConfig() JSONLConfig {
	return JSONLConfig{
		BufferSize:    64 * 1024,   // 64KB
		MaxLineSize:   1024 * 1024, // 1MB
		SkipEmpty:     true,
		SkipComments:  false,
		ContinueOnErr: false,
		Processor:     nil, // Use global processor
	}
}

// shouldSkipJSONLLine checks if a line should be skipped based on JSONLConfig.
// This is a standalone function for use without a JSONLProcessor instance.
func shouldSkipJSONLLine(line []byte, config JSONLConfig) bool {
	// Skip empty lines if configured
	if config.SkipEmpty && len(line) == 0 {
		return true
	}

	// Skip comments if configured
	if config.SkipComments && len(line) > 0 {
		if line[0] == '#' || (len(line) > 1 && line[0] == '/' && line[1] == '/') {
			return true
		}
	}

	return false
}

// StreamLinesInto processes JSONL data into a slice of typed values
// Uses generics for type-safe processing
func StreamLinesInto[T any](reader io.Reader, fn func(lineNum int, data T) error) ([]T, error) {
	return StreamLinesIntoWithConfig(reader, DefaultJSONLConfig(), fn)
}

// StreamLinesIntoWithConfig processes JSONL data into a slice of typed values with config
func StreamLinesIntoWithConfig[T any](reader io.Reader, config JSONLConfig, fn func(lineNum int, data T) error) ([]T, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, config.BufferSize), config.MaxLineSize)

	var results []T
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		// Use helper to check if line should be skipped
		if shouldSkipJSONLLine(line, config) {
			continue
		}

		var data T
		if err := json.Unmarshal(line, &data); err != nil {
			if config.ContinueOnErr {
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

// JSONLStats holds statistics for JSONL processing
type JSONLStats struct {
	LinesProcessed int64
	BytesRead      int64
	BytesWritten   int64
	CurrentLine    int
}

// JSONLWriter writes JSON Lines format to an io.Writer
type JSONLWriter struct {
	writer   io.Writer
	encoder  *json.Encoder
	lineNum  int
	err      error
	bytesOut int64
}

// NewJSONLWriter creates a new JSONL writer
func NewJSONLWriter(writer io.Writer) *JSONLWriter {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false) // Performance: skip HTML escaping
	return &JSONLWriter{
		writer:  writer,
		encoder: encoder,
	}
}

// Write writes a single JSON value as a line
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

// WriteAll writes multiple JSON values as lines
func (w *JSONLWriter) WriteAll(data []any) error {
	for _, d := range data {
		if err := w.Write(d); err != nil {
			return err
		}
	}
	return nil
}

// WriteRaw writes a raw JSON line (already encoded)
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

// Err returns any error encountered during writing
func (w *JSONLWriter) Err() error {
	return w.err
}

// Stats returns writing statistics
func (w *JSONLWriter) Stats() JSONLStats {
	return JSONLStats{
		LinesProcessed: int64(w.lineNum),
		BytesWritten:   w.bytesOut,
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

// ParseJSONL parses JSONL data from a byte slice
func ParseJSONL(data []byte) ([]any, error) {
	p, err := New()
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

// ToJSONL converts a slice of values to JSONL format
func ToJSONL(data []any) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	// Estimate buffer size
	estimatedSize := min(len(data)*64, 64*1024)

	buf := internal.GetEncoderBuffer()
	defer internal.PutEncoderBuffer(buf)
	buf.Reset()

	// Grow buffer if needed
	if buf.Cap() < estimatedSize {
		buf.Grow(estimatedSize - buf.Cap())
	}

	for _, item := range data {
		encoded, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		buf.Write(encoded)
		buf.WriteByte('\n')
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// ToJSONLString converts a slice of values to JSONL format string
func ToJSONLString(data []any) (string, error) {
	result, err := ToJSONL(data)
	if err != nil {
		return "", err
	}
	return string(result), nil
}
