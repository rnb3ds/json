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
//	age, err := json.GetAsInt(jsonStr, "user.age")
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

// JSONLProcessor processes JSON Lines format data
// PERFORMANCE: Uses buffered reading and object pooling for efficiency
type JSONLProcessor struct {
	scanner    *bufio.Scanner
	config     JSONLConfig
	lineNum    int
	err        error
	processor  *Processor
	stopped    atomic.Bool
	bytesRead  int64
	linesCount int64
}

// NewJSONLProcessor creates a new JSONL processor with default configuration.
func NewJSONLProcessor(reader io.Reader) *JSONLProcessor {
	return NewJSONLProcessorWithConfig(reader, DefaultJSONLConfig())
}

// NewJSONLProcessorWithConfig creates a new JSONL processor with the specified configuration.
//
// Example:
//
//	cfg := json.DefaultJSONLConfig()
//	cfg.SkipEmpty = false
//	cfg.Processor = customProcessor
//	proc := json.NewJSONLProcessorWithConfig(reader, cfg)
func NewJSONLProcessorWithConfig(reader io.Reader, config JSONLConfig) *JSONLProcessor {
	// Apply config defaults
	if config.BufferSize <= 0 {
		config.BufferSize = 64 * 1024
	}
	if config.MaxLineSize <= 0 {
		config.MaxLineSize = 1024 * 1024
	}

	processor := config.Processor
	if processor == nil {
		processor = getDefaultProcessor()
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, config.BufferSize), config.MaxLineSize)

	// Create processor with all fields initialized in one step
	// stopped defaults to false (atomic.Bool zero value) - ready for processing
	return &JSONLProcessor{
		scanner:   scanner,
		config:    config,
		processor: processor,
		// stopped, lineNum, err, bytesRead, linesCount all default to zero values
	}
}

// shouldSkipLine checks if a line should be skipped based on configuration.
// Returns true if the line should be skipped (empty or comment).
func (j *JSONLProcessor) shouldSkipLine(line []byte) bool {
	return shouldSkipJSONLLine(line, j.config)
}

// parseLine parses a JSON line and returns the parsed data.
// Returns an error if parsing fails and ContinueOnErr is false.
func (j *JSONLProcessor) parseLine(line []byte, lineNum int) (any, error) {
	var data any
	if err := json.Unmarshal(line, &data); err != nil {
		if j.config.ContinueOnErr {
			return nil, nil // Signal to continue (error is suppressed)
		}
		return nil, fmt.Errorf("line %d: %w", lineNum, err)
	}
	return data, nil
}

// StreamLines processes JSONL data line by line
// The callback function receives the line number and parsed data
// Return false from the callback to stop iteration
func (j *JSONLProcessor) StreamLines(fn func(lineNum int, data any) bool) error {
	for j.scanner.Scan() {
		if j.stopped.Load() {
			break
		}

		j.lineNum++
		line := j.scanner.Bytes()
		j.bytesRead += int64(len(line))

		// Use helper to check if line should be skipped
		if j.shouldSkipLine(line) {
			continue
		}

		// Use helper to parse line
		data, err := j.parseLine(line, j.lineNum)
		if err != nil {
			return err
		}
		if data == nil {
			// ContinueOnErr case - error was suppressed
			continue
		}

		j.linesCount++

		if !fn(j.lineNum, data) {
			break
		}
	}

	if err := j.scanner.Err(); err != nil {
		j.err = err
		return err
	}

	return nil
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

// StreamLinesParallel processes JSONL data in parallel using worker pool
// PERFORMANCE: Parallel processing for CPU-bound operations on JSONL data
func (j *JSONLProcessor) StreamLinesParallel(fn func(lineNum int, data any) error, workers int) (err error) {
	if workers <= 0 {
		workers = 4
	}

	// Channel for distributing work
	type lineJob struct {
		lineNum int
		data    any
	}
	jobs := make(chan lineJob, workers*2)

	// Error handling
	var firstErr error
	var errCount int32
	var wg sync.WaitGroup

	// Start workers
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if atomic.LoadInt32(&errCount) > 0 {
					continue
				}
				if jobErr := fn(job.lineNum, job.data); jobErr != nil {
					if atomic.CompareAndSwapInt32(&errCount, 0, 1) {
						firstErr = jobErr
					}
				}
			}
		}()
	}

	// SAFETY: Ensure cleanup and error propagation on all exit paths
	defer func() {
		close(jobs)
		wg.Wait()
		// Propagate worker error if no other error occurred
		if err == nil && firstErr != nil {
			err = firstErr
		}
	}()

	// Feed jobs
	for j.scanner.Scan() {
		if j.stopped.Load() {
			break
		}

		j.lineNum++
		line := j.scanner.Bytes()
		j.bytesRead += int64(len(line))

		// Use helper to check if line should be skipped
		if j.shouldSkipLine(line) {
			continue
		}

		// Use helper to parse line
		data, parseErr := j.parseLine(line, j.lineNum)
		if parseErr != nil {
			return parseErr // defer handles cleanup and error propagation
		}
		if data == nil {
			// ContinueOnErr case - error was suppressed
			continue
		}

		j.linesCount++

		// Check if error occurred before sending
		if atomic.LoadInt32(&errCount) > 0 {
			break
		}

		jobs <- lineJob{lineNum: j.lineNum, data: data}
	}

	if scanErr := j.scanner.Err(); scanErr != nil {
		j.err = scanErr
		return scanErr
	}

	return nil // defer will set err = firstErr if worker had error
}

// Stop stops the JSONL processor
func (j *JSONLProcessor) Stop() {
	j.stopped.Store(true)
}

// Err returns any error encountered during processing
func (j *JSONLProcessor) Err() error {
	return j.err
}

// Stats returns processing statistics
type JSONLStats struct {
	LinesProcessed int64
	BytesRead      int64
	BytesWritten   int64
	CurrentLine    int
}

// GetStats returns current processing statistics
func (j *JSONLProcessor) GetStats() JSONLStats {
	return JSONLStats{
		LinesProcessed: j.linesCount,
		BytesRead:      j.bytesRead,
		CurrentLine:    j.lineNum,
	}
}

// Release releases resources held by the JSONLProcessor.
//
// IMPORTANT:
//   - Call this after all processing is complete
//   - For StreamLinesParallel, ensure workers have finished before calling Release
//   - After Release, the processor must not be used
//
// Resource Management:
//   - Sets stopped flag to halt any pending operations
//   - Clears internal processor reference to allow garbage collection
func (j *JSONLProcessor) Release() {
	// Signal stop to any running operations first
	j.stopped.Store(true)

	// Clear all references including processor for GC
	j.scanner = nil
	j.processor = nil
}

// ============================================================================
// JSONL WRITER - For writing JSONL format
// ============================================================================

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
	processor := NewJSONLProcessor(bytes.NewReader(data))
	defer processor.Release()

	var results []any
	err := processor.StreamLines(func(lineNum int, data any) bool {
		results = append(results, data)
		return true
	})

	return results, err
}

// ParseJSONLInto parses JSONL data into typed values
func ParseJSONLInto[T any](data []byte) ([]T, error) {
	return StreamLinesInto[T](bytes.NewReader(data), nil)
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
