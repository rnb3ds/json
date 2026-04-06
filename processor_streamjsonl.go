package json

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

// streamJSONLConfig holds configuration for JSONL streaming operations.
//
// Deprecated: Use the main Config struct with JSONL* fields instead.
// This type will be removed in v2.0.
//
// Example migration:
//
//	// Old:
//	cfg := json.defaultStreamJSONLConfig()
//	cfg.SkipEmpty = false
//	err := processor.StreamJSONLWithConfig(reader, cfg, fn)
//
//	// New:
//	cfg := json.DefaultConfig()
//	cfg.JSONLSkipEmpty = false
//	err := processor.StreamJSONL(reader, fn, cfg)
type streamJSONLConfig struct {
	// BufferSize is the buffer size for reading JSONL data
	// Default: 64KB
	BufferSize int

	// MaxLineSize is the maximum allowed line size
	// Default: 1MB
	MaxLineSize int

	// SkipEmpty skips empty lines during streaming
	// Default: true
	SkipEmpty bool

	// SkipComments skips lines starting with # or // during streaming
	// Default: false
	SkipComments bool

	// ContinueOnError continues processing at parse errors
	// Default: false
	ContinueOnError bool

	// Workers is the number of parallel workers for streaming
	// Default: 4
	Workers int

	// ChunkSize is the chunk size for batched processing
	// Default: 1000
	ChunkSize int
}

// DefaultstreamJSONLConfig returns the default configuration for StreamJSONL.
//
// Deprecated: Use DefaultConfig() and modify JSONL* fields instead.
// This function will be removed in v2.0.
func defaultStreamJSONLConfig() streamJSONLConfig {
	return streamJSONLConfig{
		BufferSize:      64 * 1024,      // 64KB
		MaxLineSize:     1024 * 1024,    // 1MB
		SkipEmpty:       true,
		SkipComments:    false,
		ContinueOnError: false,
		Workers:         4,
		ChunkSize:       1000,
	}
}

// Validate validates and applies defaults to the configuration
func (c *streamJSONLConfig) validate() {
	if c.BufferSize <= 0 {
		c.BufferSize = 64 * 1024
	}
	if c.MaxLineSize <= 0 {
		c.MaxLineSize = 1024 * 1024
	}
	if c.Workers <= 0 {
		c.Workers = 4
	}
	if c.ChunkSize <= 0 {
		c.ChunkSize = 1000
	}
}

// ToConfig converts streamJSONLConfig to the unified Config struct.
// This method helps with migration from the deprecated streamJSONLConfig type.
//
// Example:
//
//	// Old code using streamJSONLConfig
//	oldCfg := json.defaultStreamJSONLConfig()
//	oldCfg.SkipEmpty = false
//	oldCfg.Workers = 8
//
//	// Convert to new Config
//	cfg := oldCfg.ToConfig()
//	err := processor.StreamJSONL(reader, fn, cfg)
func (c streamJSONLConfig) toConfig() Config {
	cfg := DefaultConfig()
	cfg.JSONLBufferSize = c.BufferSize
	cfg.JSONLMaxLineSize = c.MaxLineSize
	cfg.JSONLSkipEmpty = c.SkipEmpty
	cfg.JSONLSkipComments = c.SkipComments
	cfg.JSONLContinueOnErr = c.ContinueOnError
	cfg.JSONLWorkers = c.Workers
	cfg.JSONLChunkSize = c.ChunkSize
	return cfg
}

// StreamJSONLResult holds the result of a JSONL streaming operation
type StreamJSONLResult struct {
	// LinesProcessed is the total number of lines processed
	LinesProcessed int64

	// BytesRead is the total bytes read
	BytesRead int64

	// Errors is the collection of errors encountered during streaming
	Errors []error

	// Stopped indicates if streaming was stopped early
	Stopped bool
}

// StreamJSONL streams JSONL data from a reader with IterableValue callback support.
//
// This method provides line-by-line processing of JSONL (NDJSON) files with
// full IterableValue support for type-safe data access.
//
// Example:
//
//	processor, _ := json.New()
//	defer processor.Close()
//
//	err := processor.StreamJSONL(reader, func(lineNum int, item *json.IterableValue) error {
//		name := item.GetString("name")
//		age := item.GetInt("age")
//		fmt.Printf("Line %d: name=%s, age=%d\n", lineNum, name, age)
//		return nil // continue processing
//		// return item.Break() // to stop iteration
//	})
func (p *Processor) StreamJSONL(reader io.Reader, fn func(lineNum int, item *IterableValue) error) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	config := defaultStreamJSONLConfig()
	return p.streamJSONLInternal(reader, config, fn)
}

// StreamJSONLWithConfig streams JSONL data with custom configuration.
//
// Deprecated: Use StreamJSONL(reader, fn, cfg...) instead.
// This method will be removed in v2.0.
//
// Example migration:
//
//	// Old:
//	cfg := json.defaultStreamJSONLConfig()
//	cfg.SkipEmpty = false
//	err := processor.StreamJSONLWithConfig(reader, cfg, fn)
//
//	// New:
//	cfg := json.DefaultConfig()
//	cfg.JSONLSkipEmpty = false
//	err := processor.StreamJSONL(reader, fn, cfg)
func (p *Processor) StreamJSONLWithConfig(reader io.Reader, config streamJSONLConfig, fn func(lineNum int, item *IterableValue) error) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	config.validate()
	return p.streamJSONLInternal(reader, config, fn)
}

// streamJSONLInternal is the internal implementation of JSONL streaming
func (p *Processor) streamJSONLInternal(reader io.Reader, config streamJSONLConfig, fn func(lineNum int, item *IterableValue) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, config.BufferSize), config.MaxLineSize)

	var lineNum int64

	for scanner.Scan() {
		line := scanner.Bytes()

		// Skip empty lines
		if config.SkipEmpty && len(strings.TrimSpace(string(line))) == 0 {
			continue
		}

		// Skip comment lines
		if config.SkipComments && len(line) > 0 {
			if line[0] == '#' || (len(line) > 1 && line[0] == '/' && line[1] == '/') {
				continue
			}
		}

		// Parse JSON line
		var data any
		if err := json.Unmarshal(line, &data); err != nil {
			if config.ContinueOnError {
				continue
			}
			return fmt.Errorf("line %d: %w", lineNum+1, err)
		}

		lineNum++

		// Create IterableValue and call user callback
		item := newIterableValue(data)
		if err := fn(int(lineNum), item); err != nil {
			if errors.Is(err, errBreak) {
				return nil // Clean stop
			}
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// StreamJSONLParallel processes JSONL data in parallel with multiple workers.
// This method provides parallel processing of JSONL files with configurable worker count.
//
// Example:
//
//	processor, _ := json.New()
//	defer processor.Close()
//
//	err := processor.StreamJSONLParallel(reader, 4, func(lineNum int, item *json.IterableValue) error {
//		// Process each item in parallel
//		return nil
//	})
func (p *Processor) StreamJSONLParallel(reader io.Reader, workers int, fn func(lineNum int, item *IterableValue) error) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	config := defaultStreamJSONLConfig()
	config.Workers = workers
	return p.StreamJSONLParallelWithConfig(reader, config, fn)
}

// StreamJSONLParallelWithConfig processes JSONL data in parallel with custom configuration.
//
// Deprecated: Use StreamJSONLParallel(reader, workers, fn) instead.
// This method will be removed in v2.0.
func (p *Processor) StreamJSONLParallelWithConfig(reader io.Reader, config streamJSONLConfig, fn func(lineNum int, item *IterableValue) error) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	config.validate()

	// Job structure for parallel processing
	type job struct {
		lineNum int
		data    any
	}

	jobs := make(chan job, config.Workers*2)

	// Error handling
	var firstErr atomic.Pointer[error]
	var errCount int32
	var wg sync.WaitGroup

	// Start workers
	for range config.Workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if atomic.LoadInt32(&errCount) > 0 {
					continue
				}
				item := newIterableValue(job.data)
				if jobErr := fn(job.lineNum, item); jobErr != nil {
					if !errors.Is(jobErr, errBreak) {
						if atomic.CompareAndSwapInt32(&errCount, 0, 1) {
							firstErr.Store(&jobErr)
						}
					}
				}
			}
		}()
	}

	// Feed jobs
	var lineNum int64
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, config.BufferSize), config.MaxLineSize)

	for scanner.Scan() {
		line := scanner.Bytes()

		// Skip empty lines
		if config.SkipEmpty && len(strings.TrimSpace(string(line))) == 0 {
			continue
		}

		// Skip comment lines
		if config.SkipComments && len(line) > 0 {
			if line[0] == '#' || (len(line) > 1 && line[0] == '/' && line[1] == '/') {
				continue
			}
		}

		// Parse JSON line
		var data any
		if err := json.Unmarshal(line, &data); err != nil {
			if config.ContinueOnError {
				continue
			}
			close(jobs)
			wg.Wait()
			return fmt.Errorf("line %d: %w", lineNum+1, err)
		}

		lineNum++

		// Check if error occurred before sending
		if atomic.LoadInt32(&errCount) > 0 {
			break
		}

		jobs <- job{lineNum: int(lineNum), data: data}
	}

	close(jobs)
	wg.Wait()

	if err := scanner.Err(); err != nil {
		return err
	}

	if storedErr := firstErr.Load(); storedErr != nil {
		return *storedErr
	}

	return nil
}

// StreamJSONLChunked processes JSONL data in chunks for memory-efficient processing
// This method provides chunked processing of JSONL files with configurable chunk size
//
// Example:
//
//	processor, _ := json.New()
//	defer processor.Close()
//
//	err := processor.StreamJSONLChunked(reader, 1000, func(chunk []*IterableValue) error {
//		// Process chunk of 1000 items
//		return nil
//	})
func (p *Processor) StreamJSONLChunked(reader io.Reader, chunkSize int, fn func(chunk []*IterableValue) error) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	config := defaultStreamJSONLConfig()
	config.ChunkSize = chunkSize
	return p.StreamJSONLChunkedWithConfig(reader, config, fn)
}

// StreamJSONLChunkedWithConfig processes JSONL data in chunks with custom configuration.
//
// Deprecated: Use StreamJSONLChunked(reader, chunkSize, fn) instead.
// This method will be removed in v2.0.
func (p *Processor) StreamJSONLChunkedWithConfig(reader io.Reader, config streamJSONLConfig, fn func(chunk []*IterableValue) error) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	config.validate()

	var chunk []*IterableValue

	err := p.streamJSONLInternal(reader, config, func(lineNum int, item *IterableValue) error {
		chunk = append(chunk, item)

		if len(chunk) >= config.ChunkSize {
			if err := fn(chunk); err != nil {
				return err
			}
			chunk = chunk[:0] // Reset chunk
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Process remaining chunk
	if len(chunk) > 0 {
		if err := fn(chunk); err != nil {
			return err
		}
	}

	return nil
}

// ForeachJSONL iterates over JSONL data with IterableValue callback (similar to Foreach)
//
// Example:
//
//	processor, _ := json.New()
//	defer processor.Close()
//
//	err := processor.ForeachJSONL(reader, func(lineNum int, item *json.IterableValue) error {
//		fmt.Printf("Line: %d, Value: %v\n", lineNum, item.GetData())
//		return nil
//	})
func (p *Processor) ForeachJSONL(reader io.Reader, fn func(lineNum int, item *IterableValue) error) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	return p.StreamJSONL(reader, fn)
}

// MapJSONL maps JSONL data into a new format using a mapping function
//
// Example:
//
//	processor, _ := json.New()
//	defer processor.Close()
//
//	result, err := processor.MapJSONL(reader, func(lineNum int, item *json.IterableValue) (any, error) {
//		// Transform each item
//		return map[string]any{
//			"name": item.GetString("name"),
//			"age":  item.GetInt("age"),
//		}, nil
//	})
func (p *Processor) MapJSONL(reader io.Reader, fn func(lineNum int, item *IterableValue) (any, error)) ([]any, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	var results []any

	err := p.StreamJSONL(reader, func(lineNum int, item *IterableValue) error {
		value, err := fn(lineNum, item)
		if err != nil {
			return err
		}
		results = append(results, value)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// ReduceJSONL reduces JSONL data to a single aggregated result using a reducer function
// The accumulator starts with the initial value and is updated by the reducer function.
//
// Example:
//
//	processor, _ := json.New()
//	defer processor.Close()
//
//	totalAge, err := processor.ReduceJSONL(reader, 0, func(acc any, item *json.IterableValue) any {
//		return acc.(int64) + int64(item.GetInt("age"))
//	})
func (p *Processor) ReduceJSONL(reader io.Reader, initial any, fn func(acc any, item *IterableValue) any) (any, error) {
	if err := p.checkClosed(); err != nil {
		return initial, err
	}

	acc := initial

	err := p.StreamJSONL(reader, func(lineNum int, item *IterableValue) error {
		acc = fn(acc, item)
		return nil
	})

	if err != nil {
		return initial, err
	}

	return acc, nil
}

// FilterJSONL filters JSONL data based on a predicate function
//
// Example:
//
//	processor, _ := json.New()
//	defer processor.Close()
//
//	adults, err := processor.FilterJSONL(reader, func(item *json.IterableValue) bool {
//		return item.GetInt("age") >= 18
//	})
func (p *Processor) FilterJSONL(reader io.Reader, predicate func(item *IterableValue) bool) ([]*IterableValue, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	var results []*IterableValue

	err := p.StreamJSONL(reader, func(lineNum int, item *IterableValue) error {
		if predicate(item) {
			results = append(results, item)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// StreamJSONLFile streams JSONL data from a file with IterableValue callback
//
// Example:
//
//	processor, _ := json.New()
//	defer processor.Close()
//
//	err := processor.StreamJSONLFile("data.jsonl", func(lineNum int, item *json.IterableValue) error {
//		fmt.Printf("Line %d: %v\n", lineNum, item.GetData())
//		return nil
//	})
func (p *Processor) StreamJSONLFile(filename string, fn func(lineNum int, item *IterableValue) error) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	// Use os.Open to read file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return p.StreamJSONL(file, fn)
}

// StreamJSONLFileWithConfig streams JSONL data from a file with custom configuration.
//
// Deprecated: Use StreamJSONLFile(filename, fn) instead.
// This method will be removed in v2.0.
func (p *Processor) StreamJSONLFileWithConfig(filename string, config streamJSONLConfig, fn func(lineNum int, item *IterableValue) error) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	// Use os.Open to read file
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return p.StreamJSONLWithConfig(file, config, fn)
}

// CollectJSONL collects all JSONL items into a slice
//
// Example:
//
//	processor, _ := json.New()
//	defer processor.Close()
//
//	items, err := processor.CollectJSONL(reader)
//	for _, item := range items {
//		fmt.Println(item.GetString("name"))
//	}
func (p *Processor) CollectJSONL(reader io.Reader) ([]*IterableValue, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	var items []*IterableValue

	err := p.StreamJSONL(reader, func(lineNum int, item *IterableValue) error {
		items = append(items, item)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return items, nil
}

// FirstJSONL returns the first JSONL item that matches a predicate
//
// Example:
//
//	processor, _ := json.New()
//	defer processor.Close()
//
//	user, found, err := processor.FirstJSONL(reader, func(item *json.IterableValue) bool {
//		return item.GetString("name") == "Alice"
//	})
func (p *Processor) FirstJSONL(reader io.Reader, predicate func(item *IterableValue) bool) (*IterableValue, bool, error) {
	if err := p.checkClosed(); err != nil {
		return nil, false, err
	}

	var result *IterableValue
	found := false

	err := p.StreamJSONL(reader, func(lineNum int, item *IterableValue) error {
		if predicate(item) {
			result = item
			found = true
			return errBreak
		}
		return nil
	})

	if err != nil {
		return nil, false, err
	}

	return result, found, nil
}
