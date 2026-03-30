package json

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cybergodev/json/internal"
)

// Processor state constants for lifecycle management
const (
	processorStateActive  int32 = iota // 0: Processor is active and accepting operations
	processorStateClosing              // 1: Processor is closing, no new operations
	processorStateClosed               // 2: Processor is fully closed
)

// Processor is the main JSON processing engine with thread safety and performance optimization
type Processor struct {
	config            Config
	cache             *internal.CacheManager
	state             int32
	cleanupOnce       sync.Once
	resources         *processorResources
	metrics           *processorMetrics
	resourceMonitor   *resourceMonitor
	logger            atomic.Value // *slog.Logger - thread-safe logger storage
	securityValidator *securityValidator
	// Cached recursiveProcessor for reuse across operations (performance optimization)
	recursiveProcessor *recursiveProcessor
	// Wait group for tracking active operations during Close()
	activeOps sync.WaitGroup
	// OPTIMIZED: Hash cache for large JSON strings to avoid repeated hash calculations
	// Uses uint64 keys directly instead of hex strings for memory efficiency
	hashCache   map[uint64]*hashCacheEntry
	hashCacheMu sync.RWMutex
	// Extension points for hooks
	hooks []Hook
}

// hashCacheEntry stores a cached hash with its last access time for LRU eviction
type hashCacheEntry struct {
	hash       uint64 // The actual hash value of the JSON string
	lastAccess int64  // Last access timestamp for LRU eviction
	expiresAt  int64  // Expiration timestamp for TTL-based cleanup
}

// hashCacheTTL is the time-to-live for hash cache entries (5 minutes)
// Entries older than this are recomputed on next access
const hashCacheTTL = int64(5 * 60 * 1000 * 1000 * 1000) // 5 minutes in nanoseconds

// hashCacheMaxSize is the maximum number of entries in the hash cache
const hashCacheMaxSize = 64

type processorResources struct {
	lastPoolReset   int64
	lastMemoryCheck int64
	memoryPressure  int32
}

type processorMetrics struct {
	operationCount       int64
	errorCount           int64
	lastOperationTime    int64
	operationWindow      int64
	concurrencySemaphore chan struct{}
	collector            *internal.MetricsCollector
	enabled              bool // Flag to enable/disable metrics collection
}

// New creates a new JSON processor with the given configuration.
// If no configuration is provided, uses default configuration.
// Returns an error if the configuration is invalid.
//
// Example:
//
//	// Using default configuration
//	processor, err := json.New()
//
//	// With custom configuration
//	cfg := json.DefaultConfig()
//	cfg.CreatePaths = true
//	processor, err := json.New(cfg)
//
//	// Using preset configuration
//	processor, err := json.New(json.SecurityConfig())
func New(cfg ...Config) (*Processor, error) {
	var config Config
	if len(cfg) > 0 {
		config = cfg[0]
	} else {
		config = DefaultConfig()
	}

	// Validate configuration and apply corrections for invalid values
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	p := &Processor{
		config:          config,
		cache:           internal.NewCacheManager(&config),
		resourceMonitor: newResourceMonitor(),
		securityValidator: newSecurityValidator(
			config.MaxJSONSize,
			MaxPathLength,
			config.MaxNestingDepthSecurity,
			config.FullSecurityScan,
		),
		resources: &processorResources{
			lastPoolReset:   0,
			lastMemoryCheck: 0,
			memoryPressure:  0,
		},
		metrics: &processorMetrics{
			operationWindow:      0, // Disabled by default for better performance
			concurrencySemaphore: make(chan struct{}, config.MaxConcurrency),
			enabled:              config.EnableMetrics,
		},
		// OPTIMIZED: Initialize hash cache for large JSON strings
		hashCache: make(map[uint64]*hashCacheEntry, hashCacheMaxSize),
	}

	// Initialize logger atomically for thread safety
	p.logger.Store(slog.Default().With("component", "json-processor"))

	// Only create metrics collector if metrics are enabled
	if config.EnableMetrics {
		p.metrics.collector = internal.NewMetricsCollector()
	}

	// Initialize cached recursiveProcessor for reuse
	p.recursiveProcessor = newRecursiveProcessor(p)

	return p, nil
}

// Close closes the processor and cleans up resources
// This method is idempotent and thread-safe
func (p *Processor) Close() error {
	p.cleanupOnce.Do(func() {
		// Mark as closing to prevent new operations
		atomic.StoreInt32(&p.state, processorStateClosing)

		// Wait for all active operations to complete with timeout
		done := make(chan struct{})
		go func() {
			p.activeOps.Wait()
			close(done)
		}()

		select {
		case <-done:
			// All operations completed normally
		case <-time.After(closeOperationTimeout):
			// Timeout waiting for operations
			// Log warning if logger is available (non-blocking)
			if logger, ok := p.logger.Load().(*slog.Logger); ok && logger != nil {
				logger.Warn("timeout waiting for active operations during close")
			}
		}

		// Drain the concurrency semaphore to release any waiting goroutines
		// Use context cancellation for clean goroutine termination
		if p.metrics != nil && p.metrics.concurrencySemaphore != nil {
			drainCtx, drainCancel := context.WithTimeout(context.Background(), semaphoreDrainTimeout)
			drainDone := make(chan struct{})
			go func() {
				defer close(drainDone)
				for {
					select {
					case <-drainCtx.Done():
						// Context cancelled - exit cleanly
						return
					case <-p.metrics.concurrencySemaphore:
						// Drained one slot
					default:
						// Semaphore is empty
						return
					}
				}
			}()

			select {
			case <-drainDone:
				// Drain completed
			case <-drainCtx.Done():
				// Timeout on drain - continue with cleanup
			}
			drainCancel() // Ensure context is cancelled to stop goroutine
		}

		// Safely close cache: cancels cleanup goroutines and clears data
		if p.cache != nil {
			p.cache.Close()
		}

		// Reset resource tracking
		if p.resources != nil {
			atomic.StoreInt32(&p.resources.memoryPressure, 0)
			atomic.StoreInt64(&p.resources.lastMemoryCheck, 0)
			atomic.StoreInt64(&p.resources.lastPoolReset, 0)
		}

		// Mark as fully closed
		atomic.StoreInt32(&p.state, processorStateClosed)
	})
	return nil
}

// IsClosed returns true if the processor has been closed
func (p *Processor) IsClosed() bool {
	return atomic.LoadInt32(&p.state) == processorStateClosed
}

// AddHook adds an operation hook to the processor.
// Hooks are called before and after each operation.
// Multiple hooks can be added and are executed in order (Before) and reverse order (After).
//
// Example:
//
//	type LoggingHook struct{}
//	func (h *LoggingHook) Before(ctx json.HookContext) error {
//	    log.Printf("before %s", ctx.Operation)
//	    return nil
//	}
//	func (h *LoggingHook) After(ctx json.HookContext, result any, err error) (any, error) {
//	    log.Printf("after %s", ctx.Operation)
//	    return result, err
//	}
//
//	p := json.MustNew()
//	p.AddHook(&LoggingHook{})
func (p *Processor) AddHook(hook Hook) {
	p.hooks = append(p.hooks, hook)
}

// ProcessBatch processes multiple operations in a single batch
func (p *Processor) ProcessBatch(operations []BatchOperation, opts ...Config) ([]BatchResult, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	_, err := p.prepareOptions(opts...)
	if err != nil {
		return nil, err
	}

	if len(operations) > p.config.MaxBatchSize {
		return nil, &JsonsError{
			Op:      "process_batch",
			Message: fmt.Sprintf("batch size %d exceeds maximum %d", len(operations), p.config.MaxBatchSize),
			Err:     ErrSizeLimit,
		}
	}

	results := make([]BatchResult, len(operations))

	for i, op := range operations {
		result := BatchResult{ID: op.ID}

		switch op.Type {
		case "get":
			result.Result, result.Error = p.Get(op.JSONStr, op.Path, opts...)
		case "set":
			result.Result, result.Error = p.Set(op.JSONStr, op.Path, op.Value, opts...)
		case "delete":
			result.Result, result.Error = p.Delete(op.JSONStr, op.Path, opts...)
		case "validate":
			valid, err := p.Valid(op.JSONStr, opts...)
			result.Result = map[string]any{"valid": valid}
			result.Error = err
		default:
			result.Error = fmt.Errorf("unknown operation type: %s", op.Type)
		}

		results[i] = result
	}

	return results, nil
}

// ClearCache clears all cached data
func (p *Processor) ClearCache() {
	if p.cache != nil {
		p.cache.Clear()
	}
}

// WarmupCache pre-loads commonly used paths into cache to improve first-access performance
func (p *Processor) WarmupCache(jsonStr string, paths []string, opts ...Config) (*WarmupResult, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	if !p.config.EnableCache {
		return nil, &JsonsError{
			Op:      "warmup_cache",
			Message: "cache is disabled, cannot warmup cache",
			Err:     ErrCacheDisabled,
		}
	}

	if len(paths) == 0 {
		return &WarmupResult{
			TotalPaths:  0,
			Successful:  0,
			Failed:      0,
			SuccessRate: 100.0,
			FailedPaths: nil,
		}, nil // Nothing to warmup
	}

	// Validate JSON input
	if err := p.validateInput(jsonStr); err != nil {
		return nil, &JsonsError{
			Op:      "warmup_cache",
			Message: "invalid JSON input for cache warmup",
			Err:     err,
		}
	}

	// Prepare options
	options, err := p.prepareOptions(opts...)
	if err != nil {
		return nil, &JsonsError{
			Op:      "warmup_cache",
			Message: "invalid options for cache warmup",
			Err:     err,
		}
	}

	// Track warmup statistics
	successCount := 0
	errorCount := 0
	var lastError error
	var failedPaths []string

	// Preload each path into cache
	for _, path := range paths {
		// Validate path
		if err := p.validatePath(path); err != nil {
			errorCount++
			failedPaths = append(failedPaths, path)
			lastError = &JsonsError{
				Op:      "warmup_cache",
				Path:    path,
				Message: fmt.Sprintf("invalid path '%s' for cache warmup: %v", path, err),
				Err:     err,
			}
			continue
		}

		// Try to get the value (this will cache it if successful)
		// Handle nil options to prevent nil pointer dereference
		var err error
		if options != nil {
			_, err = p.Get(jsonStr, path, *options)
		} else {
			_, err = p.Get(jsonStr, path)
		}
		if err != nil {
			errorCount++
			failedPaths = append(failedPaths, path)
			lastError = &JsonsError{
				Op:      "warmup_cache",
				Path:    path,
				Message: fmt.Sprintf("failed to warmup path '%s': %v", path, err),
				Err:     err,
			}
		} else {
			successCount++
		}
	}

	// Create warmup result
	successRate := 100.0
	if len(paths) > 0 {
		successRate = float64(successCount) / float64(len(paths)) * 100
	}

	result := &WarmupResult{
		TotalPaths:  len(paths),
		Successful:  successCount,
		Failed:      errorCount,
		SuccessRate: successRate,
		FailedPaths: failedPaths,
	}

	// Return error if all paths failed
	if successCount == 0 && errorCount > 0 {
		return result, &JsonsError{
			Op:      "warmup_cache",
			Message: fmt.Sprintf("cache warmup failed for all %d paths, last error: %v", len(paths), lastError),
			Err:     lastError,
		}
	}

	return result, nil
}

// hashStringToUint64 generates a fast 64-bit hash using FNV-1a.
// Delegates to internal package for consistent implementation.
// PERFORMANCE: For large strings (> 4KB), uses sampling to avoid full scan.
func hashStringToUint64(s string) uint64 {
	if len(s) > internal.LargeStringHashThreshold {
		return internal.HashStringFNV1aSampled(s)
	}
	return internal.HashStringFNV1a(s)
}

// computeHashCacheKey computes a cache lookup key for JSON strings.
// Delegates to hashStringToUint64 for consistent implementation.
// PERFORMANCE: Uses sampled FNV-1a hash for large strings.
func computeHashCacheKey(s string) uint64 {
	return hashStringToUint64(s)
}

// evictOldestHashCacheEntriesLocked removes the oldest entries from the hash cache using LRU strategy.
// PERFORMANCE: LRU eviction keeps frequently accessed entries in cache for better hit rate.
// CONCURRENCY: Must be called with hashCacheMu.Lock() held.
func (p *Processor) evictOldestHashCacheEntriesLocked(count int) {
	if count <= 0 || len(p.hashCache) == 0 {
		return
	}

	// Collect entries with their access times for LRU eviction
	type entryWithTime struct {
		key        uint64
		lastAccess int64
	}

	entries := make([]entryWithTime, 0, len(p.hashCache))
	for key, entry := range p.hashCache {
		entries = append(entries, entryWithTime{key: key, lastAccess: entry.lastAccess})
	}

	// Sort by lastAccess ascending (oldest first)
	// For small counts, use simple selection instead of full sort
	if count < len(entries)/2 {
		// Partial selection sort - find the oldest 'count' entries
		for i := 0; i < count && i < len(entries); i++ {
			oldestIdx := i
			for j := i + 1; j < len(entries); j++ {
				if entries[j].lastAccess < entries[oldestIdx].lastAccess {
					oldestIdx = j
				}
			}
			entries[i], entries[oldestIdx] = entries[oldestIdx], entries[i]
		}
	} else {
		// Full sort for larger evictions
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].lastAccess < entries[j].lastAccess
		})
	}

	// Remove the oldest entries
	for i := 0; i < count && i < len(entries); i++ {
		delete(p.hashCache, entries[i].key)
	}
}

// getOrCacheHash gets a cached hash for a JSON string, or computes and caches it
// OPTIMIZED: Caches hashes for large JSON strings to avoid repeated calculations
// Uses uint64 keys directly instead of hex strings for memory efficiency
// CONCURRENCY FIX: Uses double-checked locking to avoid TOCTOU race condition
// where multiple goroutines could compute the same hash simultaneously
func (p *Processor) getOrCacheHash(jsonStr string) uint64 {
	// For small strings, just compute directly (no caching overhead)
	if len(jsonStr) <= 4096 {
		return hashStringToUint64(jsonStr)
	}

	// For large strings, try to use cache
	// PERFORMANCE: Use strong FNV-1a hash of first/last 1KB + length for cache key
	cacheLookupKey := computeHashCacheKey(jsonStr)

	// Fast path: read lock lookup with TTL check
	p.hashCacheMu.RLock()
	if entry, ok := p.hashCache[cacheLookupKey]; ok {
		// Check if entry has expired
		if now := time.Now().UnixNano(); now > entry.expiresAt {
			p.hashCacheMu.RUnlock()
			// Entry expired, compute fresh hash (will replace in slow path)
			h := hashStringToUint64(jsonStr)
			// Try to update cache with new entry
			p.hashCacheMu.Lock()
			// Double-check after acquiring write lock
			if e, exists := p.hashCache[cacheLookupKey]; exists && e.expiresAt == entry.expiresAt {
				p.hashCache[cacheLookupKey] = &hashCacheEntry{
					hash:       h,
					lastAccess: now,
					expiresAt:  now + hashCacheTTL,
				}
			}
			p.hashCacheMu.Unlock()
			return h
		}
		hash := entry.hash
		p.hashCacheMu.RUnlock()
		return hash
	}
	p.hashCacheMu.RUnlock()

	// Slow path: compute hash outside of lock to avoid blocking other goroutines
	h := hashStringToUint64(jsonStr)
	now := time.Now().UnixNano()

	// Double-checked locking: re-verify under write lock
	// This prevents multiple goroutines from computing and storing the same hash
	p.hashCacheMu.Lock()
	// Check if another goroutine already stored this hash
	if entry, ok := p.hashCache[cacheLookupKey]; ok {
		p.hashCacheMu.Unlock()
		return entry.hash
	}

	// Cache the result with size limit
	if len(p.hashCache) >= hashCacheMaxSize {
		// Evict 25% to make room using LRU
		p.evictOldestHashCacheEntriesLocked(len(p.hashCache) / 4)
	}
	p.hashCache[cacheLookupKey] = &hashCacheEntry{
		hash:       h,
		lastAccess: now,
		expiresAt:  now + hashCacheTTL,
	}
	p.hashCacheMu.Unlock()

	return h
}

// createCacheKey creates a cache key with optimized efficiency
// Uses direct hash values instead of hex strings for better performance
// OPTIMIZED: Uses cached hash for large JSON strings
func (p *Processor) createCacheKey(operation, jsonStr, path string, options *Config) string {
	// OPTIMIZED: Use cached hash for large JSON strings
	jsonHash := p.getOrCacheHash(jsonStr)
	return p.createCacheKeyWithHash(operation, jsonHash, path, options)
}

// createCacheKeyWithHash creates a cache key using a pre-computed hash
// PERFORMANCE: Allows hash reuse across multiple cache key creations
func (p *Processor) createCacheKeyWithHash(operation string, jsonHash uint64, path string, options *Config) string {
	// Use a fixed-size array buffer for small keys to avoid allocations
	// Most cache keys are < 128 bytes
	var buf [128]byte
	var key string

	// Try to use stack-allocated buffer
	estimatedLen := len(operation) + 1 + 16 + 1 + len(path) + 16 // op:hash16:path:opts
	if estimatedLen < len(buf) && options == nil {
		// Fast path: use stack buffer
		n := copy(buf[:], operation)
		buf[n] = ':'
		n++
		n += formatUint64Hex(buf[n:], jsonHash)
		buf[n] = ':'
		n++
		n += copy(buf[n:], path)
		key = string(buf[:n])
	} else {
		// Slow path: use string builder for larger keys
		sb := p.getStringBuilder()
		defer p.putStringBuilder(sb)

		sb.Grow(estimatedLen + 32)
		sb.WriteString(operation)
		sb.WriteByte(':')
		sb.WriteString(formatUint64HexString(jsonHash))
		sb.WriteByte(':')
		sb.WriteString(path)

		// Include relevant options in the key
		if options != nil {
			if options.StrictMode {
				sb.WriteString(":s")
			}
			if options.AllowComments {
				sb.WriteString(":c")
			}
			if options.PreserveNumbers {
				sb.WriteString(":p")
			}
			if options.MaxDepth > 0 {
				sb.WriteString(":d")
				sb.WriteString(strconv.Itoa(options.MaxDepth))
			}
		}

		key = sb.String()
	}

	return key
}

// formatUint64Hex formats a uint64 as hex without allocation
func formatUint64Hex(buf []byte, v uint64) int {
	const hexChars = "0123456789abcdef"
	// Write in reverse order, then we'd need to reverse
	// Instead, write from position 15 down to 0
	for i := 15; i >= 0; i-- {
		buf[i] = hexChars[v&0xF]
		v >>= 4
	}
	return 16
}

// formatUint64HexString formats a uint64 as a hex string
func formatUint64HexString(v uint64) string {
	var buf [16]byte
	formatUint64Hex(buf[:], v)
	return string(buf[:])
}

// createSimpleCacheKey creates a simple "prefix:data" format cache key
// Uses stack-allocated buffer for small keys to avoid heap allocation
func createSimpleCacheKey(prefix, data string) string {
	totalLen := len(prefix) + 1 + len(data) // prefix + ":" + data

	// Use stack-allocated buffer for small keys (up to 256 bytes)
	const maxStackKeySize = 256
	if totalLen <= maxStackKeySize {
		var buf [maxStackKeySize]byte
		n := copy(buf[:], prefix)
		buf[n] = ':'
		n++
		n += copy(buf[n:], data)
		return string(buf[:n])
	}

	// Fall back to heap allocation for large keys
	return prefix + ":" + data
}

// getCachedPathSegments gets parsed path segments using unified cache
// PERFORMANCE: Creates cache key once and reuses for both lookup and storage
// PERFORMANCE: Returns cached segments directly (immutable after creation)
func (p *Processor) getCachedPathSegments(path string) ([]internal.PathSegment, error) {
	// Use unified cache manager
	if p.config.EnableCache {
		// PERFORMANCE: Create cache key once for both lookup and storage
		cacheKey := createSimpleCacheKey("path", path)
		if cached, ok := p.cache.Get(cacheKey); ok {
			if segments, ok := cached.([]internal.PathSegment); ok {
				// PERFORMANCE: Return cached segments directly - they are immutable after creation
				return segments, nil
			}
		}

		// Parse path
		segments, err := internal.ParsePath(path)
		if err != nil {
			return nil, err
		}

		// Cache the result using unified cache - reuse the cache key
		if atomic.LoadInt32(&p.state) == processorStateActive {
			cached := make([]internal.PathSegment, len(segments))
			copy(cached, segments)
			p.cache.Set(cacheKey, cached)
		}

		return segments, nil
	}

	// Parse path without caching
	segments, err := internal.ParsePath(path)
	if err != nil {
		return nil, err
	}

	return segments, nil
}

// getCachedResult retrieves a cached result if available
func (p *Processor) getCachedResult(key string) (any, bool) {
	if !p.config.EnableCache {
		return nil, false
	}
	return p.cache.Get(key)
}

// setCachedResult stores a result in cache with security validation
func (p *Processor) setCachedResult(key string, result any, options ...*Config) {
	if !p.config.EnableCache {
		return
	}

	// Check if caching is enabled for this operation
	if len(options) > 0 && options[0] != nil && !options[0].CacheResults {
		return
	}

	// Security validation: don't cache potentially sensitive data
	if p.containsSensitiveData(result) {
		return
	}

	// Validate cache key to prevent injection
	if !p.isValidCacheKey(key) {
		return
	}

	p.cache.Set(key, result)
}

// setCachedResultInternal stores a result in cache without sensitive data check
// PERFORMANCE: For trusted internal results (parsed JSON, navigation results) where
// security validation already happened at input. Skips expensive sensitive data scanning.
func (p *Processor) setCachedResultInternal(key string, result any) {
	if !p.config.EnableCache {
		return
	}

	// Validate cache key to prevent injection
	if !p.isValidCacheKey(key) {
		return
	}

	p.cache.Set(key, result)
}

// containsSensitiveData checks if the result contains sensitive information
// SECURITY: Delegates to securityValidator for consistent detection logic
func (p *Processor) containsSensitiveData(result any) bool {
	return p.securityValidator.ContainsSensitiveData(result)
}

// isValidCacheKey validates cache key format
// Delegates to internal package for consistent implementation
func (p *Processor) isValidCacheKey(key string) bool {
	return internal.IsValidCacheKey(key)
}

// GetConfig returns a copy of the processor configuration
func (p *Processor) GetConfig() Config {
	return *p.config.Clone()
}

// SetLogger sets a custom structured logger for the processor
func (p *Processor) SetLogger(logger *slog.Logger) {
	if logger != nil {
		p.logger.Store(logger.With("component", "json-processor"))
	} else {
		p.logger.Store(slog.Default().With("component", "json-processor"))
	}
}

// getLogger safely retrieves the current logger (thread-safe)
func (p *Processor) getLogger() *slog.Logger {
	if l, ok := p.logger.Load().(*slog.Logger); ok {
		return l
	}
	return slog.Default().With("component", "json-processor")
}

// checkClosed returns an error if the processor is closed or closing
func (p *Processor) checkClosed() error {
	state := atomic.LoadInt32(&p.state)
	if state != processorStateActive {
		if state == processorStateClosing {
			return &JsonsError{
				Op:      "check_closed",
				Message: "processor is closing",
				Err:     ErrProcessorClosed,
			}
		}
		return &JsonsError{
			Op:      "check_closed",
			Message: "processor is closed",
			Err:     ErrProcessorClosed,
		}
	}
	return nil
}

// configPool pools Config objects to reduce allocations in hot paths
// PERFORMANCE: Reduces ~6GB allocations from prepareOptions calls
var configPool = sync.Pool{
	New: func() any {
		cfg := DefaultConfig()
		return &cfg
	},
}

// getConfig gets a Config from the pool, applies defaults or provided config, and validates
func (p *Processor) getConfig(opts ...Config) (*Config, error) {
	cfg := configPool.Get().(*Config)
	if len(opts) > 0 {
		*cfg = opts[0]
	} else {
		*cfg = DefaultConfig()
	}
	if err := cfg.Validate(); err != nil {
		configPool.Put(cfg)
		return nil, err
	}
	return cfg, nil
}

// putConfig returns a Config to the pool after clearing sensitive data
func (p *Processor) putConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	// Clear sensitive fields to prevent memory leaks
	cfg.Context = nil
	configPool.Put(cfg)
}

// prepareOptions prepares and validates processor options
// Accepts Config values and returns a pointer for internal use
// PERFORMANCE: Uses pooled Config objects to reduce allocations
func (p *Processor) prepareOptions(opts ...Config) (*Config, error) {
	return p.getConfig(opts...)
}

// mergeOptionsWithOverride creates a new Config with overrides applied.
// Returns a Config value (not pointer) to prevent accidental mutation
// and encourage the caller to work with their own copy.
func mergeOptionsWithOverride(opts []Config, override func(*Config)) Config {
	var result Config
	if len(opts) > 0 {
		result = *(&opts[0]).Clone()
	} else {
		result = DefaultConfig()
	}
	override(&result)
	return result
}

// Delete removes a value from JSON at the specified path
func (p *Processor) Delete(jsonStr, path string, opts ...Config) (string, error) {
	if err := p.checkClosed(); err != nil {
		return "", err
	}

	options, err := p.prepareOptions(opts...)
	if err != nil {
		return "", err
	}

	if err := p.validateInput(jsonStr); err != nil {
		return jsonStr, err
	}

	if err := p.validatePath(path); err != nil {
		return jsonStr, err
	}

	// Parse JSON
	var data any
	err = p.Parse(jsonStr, &data, opts...)
	if err != nil {
		return jsonStr, err
	}

	// Determine cleanup options from prepared options and config
	cleanupNulls := options.CleanupNulls || p.config.CleanupNulls
	compactArrays := options.CompactArrays || p.config.CompactArrays

	// If compactArrays is enabled, automatically enable cleanupNulls
	if compactArrays {
		cleanupNulls = true
	}

	// Check if path contains array access - only then we need DeletedMarker cleanup
	needsMarkerCleanup := p.isArrayDeletePath(path)

	// Delete the value at the specified path
	err = p.deleteValueAtPath(data, path)
	if err != nil {
		// For any deletion error, return the original JSON unchanged instead of empty string
		// This includes "path not found", "property not found", and other deletion errors
		return jsonStr, &JsonsError{
			Op:      "delete",
			Path:    path,
			Message: err.Error(),
			Err:     err,
		}
	}

	// Only clean up deleted markers if the path involved array operations
	// This optimization avoids unnecessary data structure recreation for map deletions
	if needsMarkerCleanup {
		data = p.cleanupDeletedMarkers(data)
	}

	// Cleanup nulls if requested
	if cleanupNulls {
		data = p.cleanupNullValuesWithReconstruction(data, compactArrays)
	}

	// Convert back to JSON string
	resultBytes, err := json.Marshal(data)
	if err != nil {
		// Return original JSON instead of empty string when marshaling fails
		return jsonStr, &JsonsError{
			Op:      "delete",
			Path:    path,
			Message: fmt.Sprintf("failed to marshal result: %v", err),
			Err:     ErrOperationFailed,
		}
	}

	return string(resultBytes), nil
}

// isArrayDeletePath checks if the path involves array operations that require marker cleanup
func (p *Processor) isArrayDeletePath(path string) bool {
	// Check if path contains array bracket notation
	for i := 0; i < len(path); i++ {
		if path[i] == '[' {
			return true
		}
	}
	return false
}

// DeleteClean removes a value from JSON and cleans up null placeholders.
// This is the unified API for delete-with-cleanup operations.
//
// Example:
//
//	result, err := processor.DeleteClean(data, "users[0].profile")
func (p *Processor) DeleteClean(jsonStr, path string, opts ...Config) (string, error) {
	cleanupOpts := mergeOptionsWithOverride(opts, func(o *Config) {
		o.CleanupNulls = true
		o.CompactArrays = true
	})
	return p.Delete(jsonStr, path, cleanupOpts)
}

// Foreach iterates over JSON arrays or objects using this processor
func (p *Processor) Foreach(jsonStr string, fn func(key any, item *IterableValue)) {
	if err := p.checkClosed(); err != nil {
		return
	}

	data, err := p.Get(jsonStr, ".")
	if err != nil {
		return
	}

	foreachWithIterableValue(data, fn)
}

// ForeachWithPath iterates over JSON arrays or objects at a specific path using this processor
// This allows using custom processor configurations (security limits, nesting depth, etc.)
func (p *Processor) ForeachWithPath(jsonStr, path string, fn func(key any, item *IterableValue)) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	data, err := p.Get(jsonStr, path)
	if err != nil {
		return err
	}

	foreachWithIterableValue(data, fn)
	return nil
}

// ForeachWithPathAndIterator iterates over JSON at a path with path information
func (p *Processor) ForeachWithPathAndIterator(jsonStr, path string, fn func(key any, item *IterableValue, currentPath string) IteratorControl) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	data, err := p.Get(jsonStr, path)
	if err != nil {
		return err
	}

	return foreachWithPathIterableValue(data, "", fn)
}

// ForeachWithPathAndControl iterates with control over iteration flow
func (p *Processor) ForeachWithPathAndControl(jsonStr, path string, fn func(key any, value any) IteratorControl) error {
	if err := p.checkClosed(); err != nil {
		return err
	}

	data, err := p.Get(jsonStr, path)
	if err != nil {
		return err
	}

	return foreachOnValue(data, fn)
}

// ForeachReturn iterates over JSON arrays or objects and returns the JSON string
// This is useful for iteration with transformation purposes
func (p *Processor) ForeachReturn(jsonStr string, fn func(key any, item *IterableValue)) (string, error) {
	if err := p.checkClosed(); err != nil {
		return "", err
	}

	data, err := p.Get(jsonStr, ".")
	if err != nil {
		return "", err
	}

	// Execute the iteration
	foreachWithIterableValue(data, fn)

	// Return the original JSON string
	return jsonStr, nil
}

// ForeachNested recursively iterates over all nested JSON structures
// This method traverses through all nested objects and arrays
func (p *Processor) ForeachNested(jsonStr string, fn func(key any, item *IterableValue)) {
	if err := p.checkClosed(); err != nil {
		return
	}

	data, err := p.Get(jsonStr, ".")
	if err != nil {
		return
	}

	foreachNestedOnValue(data, fn)
}

// SafeGet performs a type-safe get operation with comprehensive error handling
func (p *Processor) SafeGet(jsonStr, path string) AccessResult {
	// Validate inputs
	if jsonStr == "" {
		return AccessResult{Exists: false}
	}
	if path == "" {
		return AccessResult{Exists: false}
	}

	// Perform the get operation
	value, err := p.Get(jsonStr, path)
	if err != nil {
		return AccessResult{Exists: false}
	}

	// Determine the type
	var valueType string
	if value == nil {
		valueType = "null"
	} else {
		valueType = fmt.Sprintf("%T", value)
	}

	return AccessResult{
		Value:  value,
		Exists: true,
		Type:   valueType,
	}
}

// Get retrieves a value from JSON using a path expression with performance
func (p *Processor) Get(jsonStr, path string, opts ...Config) (any, error) {
	// Check rate limiting for security
	if err := p.checkRateLimit(); err != nil {
		return nil, err
	}

	// Only track time if metrics are enabled
	var startTime time.Time
	var recordMetrics func(success bool)
	if p.metrics != nil && p.metrics.enabled && p.metrics.collector != nil {
		startTime = time.Now()
		p.metrics.collector.StartConcurrentOperation()
		recordMetrics = func(success bool) {
			p.metrics.collector.EndConcurrentOperation()
			p.metrics.collector.RecordOperation(time.Since(startTime), success, 0)
		}
	} else {
		recordMetrics = func(success bool) {}
	}

	// Increment operation counter for statistics
	p.incrementOperationCount()

	if err := p.checkClosed(); err != nil {
		p.incrementErrorCount()
		recordMetrics(false)
		return nil, err
	}

	options, err := p.prepareOptions(opts...)
	if err != nil {
		p.incrementErrorCount()
		return nil, err
	}

	// Get context from options or use background
	ctx := context.Background()
	if options.Context != nil {
		ctx = options.Context
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		p.incrementErrorCount()
		p.logError(ctx, "get", path, ctx.Err())
		return nil, ctx.Err()
	default:
	}

	// Continue with the rest of the method...
	defer func() {
		if startTime.IsZero() {
			return
		}
		duration := time.Since(startTime)
		p.logOperation(ctx, "get", path, duration)
	}()

	// PERFORMANCE: Compute hash ONCE for entire operation, reuse for all cache keys
	jsonHash := p.getOrCacheHash(jsonStr)

	// Check cache first with optimized key generation (BEFORE validation for performance)
	cacheKey := p.createCacheKeyWithHash("get", jsonHash, path, options)
	if cached, ok := p.getCachedResult(cacheKey); ok {
		// Record cache hit operation
		if p.metrics != nil && p.metrics.collector != nil {
			p.metrics.collector.RecordCacheHit()
		}
		recordMetrics(true)
		return cached, nil
	}

	// Validate input only if cache miss and validation not skipped
	// OPTIMIZED: Allow skipping validation for trusted input
	if !options.SkipValidation {
		if err := p.validateInput(jsonStr); err != nil {
			p.incrementErrorCount()
			recordMetrics(false)
			return nil, err
		}

		if err := p.validatePath(path); err != nil {
			p.incrementErrorCount()
			recordMetrics(false)
			return nil, err
		}
	}

	// Record cache miss
	if p.metrics != nil && p.metrics.collector != nil {
		p.metrics.collector.RecordCacheMiss()
	}

	// Try to get parsed data from cache first - reuse pre-computed hash
	parseCacheKey := p.createCacheKeyWithHash("parse", jsonHash, "", options)
	var data any

	if cachedData, ok := p.getCachedResult(parseCacheKey); ok {
		data = cachedData
	} else {
		// Parse JSON with error context
		parseErr := p.Parse(jsonStr, &data, opts...)
		if parseErr != nil {
			p.incrementErrorCount()
			recordMetrics(false)
			return nil, parseErr
		}

		// Cache parsed data for reuse - always cache if global cache is enabled
		// PERFORMANCE: Use setCachedResultInternal to skip sensitive data check
		// since this is trusted internal data from our parser
		if p.config.EnableCache {
			p.setCachedResultInternal(parseCacheKey, data)
		}
	}

	// Use unified recursive processor for all paths (cached instance)
	result, err := p.recursiveProcessor.ProcessRecursively(data, path, opGet, nil)
	if err != nil {
		p.incrementErrorCount()
		recordMetrics(false)
		return nil, &JsonsError{
			Op:      "get",
			Path:    path,
			Message: err.Error(),
			Err:     err,
		}
	}

	// Cache result if enabled
	p.setCachedResult(cacheKey, result, options)

	// Record successful operation
	recordMetrics(true)

	return result, nil
}

// PreParse parses a JSON string and returns a ParsedJSON object that can be reused
// for multiple Get operations. This is a performance optimization for scenarios where
// the same JSON is queried multiple times.
//
// OPTIMIZED: Pre-parsing avoids repeated JSON parsing overhead for repeated queries.
//
// Example:
//
//	parsed, err := processor.PreParse(jsonStr)
//	if err != nil { return err }
//	value1, _ := processor.GetFromParsed(parsed, "path1")
//	value2, _ := processor.GetFromParsed(parsed, "path2")
func (p *Processor) PreParse(jsonStr string, opts ...Config) (*ParsedJSON, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	options, err := p.prepareOptions(opts...)
	if err != nil {
		return nil, err
	}

	// Validate input
	if err := p.validateInput(jsonStr); err != nil {
		return nil, err
	}

	// Try to get from cache first
	parseCacheKey := p.createCacheKey("parse", jsonStr, "", options)
	var data any

	if cachedData, ok := p.getCachedResult(parseCacheKey); ok {
		data = cachedData
	} else {
		// Parse JSON
		parseErr := p.Parse(jsonStr, &data, opts...)
		if parseErr != nil {
			return nil, parseErr
		}

		// Cache parsed data - PERFORMANCE: Use internal method to skip sensitive check
		if p.config.EnableCache {
			p.setCachedResultInternal(parseCacheKey, data)
		}
	}

	return &ParsedJSON{
		data:      data,
		hash:      p.getOrCacheHash(jsonStr),
		jsonLen:   len(jsonStr),
		processor: p,
	}, nil
}

// GetFromParsed retrieves a value from a pre-parsed JSON document at the specified path.
// This is significantly faster than Get() for repeated queries on the same JSON.
//
// OPTIMIZED: Skips JSON parsing, goes directly to path navigation.
func (p *Processor) GetFromParsed(parsed *ParsedJSON, path string, opts ...Config) (any, error) {
	if parsed == nil {
		return nil, &JsonsError{
			Op:      "get_from_parsed",
			Message: "parsed JSON is nil",
			Err:     ErrOperationFailed,
		}
	}

	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	options, err := p.prepareOptions(opts...)
	if err != nil {
		return nil, err
	}

	if err := p.validatePath(path); err != nil {
		return nil, err
	}

	// Use unified recursive processor for path navigation
	result, err := p.recursiveProcessor.ProcessRecursively(parsed.data, path, opGet, nil)
	if err != nil {
		return nil, &JsonsError{
			Op:      "get_from_parsed",
			Path:    path,
			Message: err.Error(),
			Err:     err,
		}
	}

	// Cache result if enabled
	if p.config.EnableCache && options.CacheResults {
		cacheKey := p.createCacheKey("get", string(rune(parsed.hash)), path, options)
		p.setCachedResult(cacheKey, result, options)
	}

	return result, nil
}

// SetFromParsed modifies a pre-parsed JSON document at the specified path.
// Returns a new ParsedJSON with the modified data (original is not modified).
//
// OPTIMIZED: Skips JSON parsing, works directly on parsed data.
func (p *Processor) SetFromParsed(parsed *ParsedJSON, path string, value any, opts ...Config) (*ParsedJSON, error) {
	if parsed == nil {
		return nil, &JsonsError{
			Op:      "set_from_parsed",
			Message: "parsed JSON is nil",
			Err:     ErrOperationFailed,
		}
	}

	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	options, err := p.prepareOptions(opts...)
	if err != nil {
		return nil, err
	}

	if err := p.validatePath(path); err != nil {
		return nil, err
	}

	// Deep copy the data before modification
	dataCopy, err := p.deepCopyData(parsed.data)
	if err != nil {
		return nil, &JsonsError{Op: "set_from_parsed", Path: path, Err: err}
	}

	// Use unified recursive processor for path navigation and modification
	result, err := p.recursiveProcessor.ProcessRecursivelyWithOptions(dataCopy, path, opSet, value, options.CreatePaths)
	if err != nil {
		return nil, &JsonsError{
			Op:      "set_from_parsed",
			Path:    path,
			Message: err.Error(),
			Err:     err,
		}
	}

	return &ParsedJSON{
		data:      result,
		hash:      0, // New hash will be computed when needed
		jsonLen:   0, // Length unknown until serialized
		processor: p,
	}, nil
}

// GetString retrieves a string value from JSON at the specified path
func (p *Processor) GetString(jsonStr, path string, opts ...Config) (string, error) {
	return getTypedWithProcessor[string](p, jsonStr, path, opts...)
}

// GetInt retrieves an int value from JSON at the specified path
func (p *Processor) GetInt(jsonStr, path string, opts ...Config) (int, error) {
	return getTypedWithProcessor[int](p, jsonStr, path, opts...)
}

// GetFloat retrieves a float64 value from JSON at the specified path
func (p *Processor) GetFloat(jsonStr, path string, opts ...Config) (float64, error) {
	return getTypedWithProcessor[float64](p, jsonStr, path, opts...)
}

// GetBool retrieves a bool value from JSON at the specified path
func (p *Processor) GetBool(jsonStr, path string, opts ...Config) (bool, error) {
	return getTypedWithProcessor[bool](p, jsonStr, path, opts...)
}

// GetArray retrieves an array value from JSON at the specified path
func (p *Processor) GetArray(jsonStr, path string, opts ...Config) ([]any, error) {
	return getTypedWithProcessor[[]any](p, jsonStr, path, opts...)
}

// GetObject retrieves an object value from JSON at the specified path
func (p *Processor) GetObject(jsonStr, path string, opts ...Config) (map[string]any, error) {
	return getTypedWithProcessor[map[string]any](p, jsonStr, path, opts...)
}

// GetStringOr retrieves a string value from JSON at the specified path with a default fallback.
// Returns defaultValue if: path not found, value is null, or type conversion fails.
func (p *Processor) GetStringOr(jsonStr, path string, defaultValue string, opts ...Config) string {
	result, err := p.GetString(jsonStr, path, opts...)
	if err != nil {
		return defaultValue
	}
	return result
}

// GetIntOr retrieves an int value from JSON at the specified path with a default fallback.
// Returns defaultValue if: path not found, value is null, or type conversion fails.
func (p *Processor) GetIntOr(jsonStr, path string, defaultValue int, opts ...Config) int {
	result, err := p.GetInt(jsonStr, path, opts...)
	if err != nil {
		return defaultValue
	}
	return result
}

// GetFloatOr retrieves a float64 value from JSON at the specified path with a default fallback.
// Returns defaultValue if: path not found, value is null, or type conversion fails.
func (p *Processor) GetFloatOr(jsonStr, path string, defaultValue float64, opts ...Config) float64 {
	result, err := p.GetFloat(jsonStr, path, opts...)
	if err != nil {
		return defaultValue
	}
	return result
}

// GetBoolOr retrieves a bool value from JSON at the specified path with a default fallback.
// Returns defaultValue if: path not found, value is null, or type conversion fails.
func (p *Processor) GetBoolOr(jsonStr, path string, defaultValue bool, opts ...Config) bool {
	result, err := p.GetBool(jsonStr, path, opts...)
	if err != nil {
		return defaultValue
	}
	return result
}

// GetMultiple retrieves multiple values from JSON using multiple path expressions
func (p *Processor) GetMultiple(jsonStr string, paths []string, opts ...Config) (map[string]any, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	if err := p.validateInput(jsonStr); err != nil {
		return nil, err
	}

	if len(paths) == 0 {
		return make(map[string]any), nil
	}

	_, err := p.prepareOptions(opts...)
	if err != nil {
		return nil, err
	}

	// Parse JSON once for all operations
	var data any
	if err := p.Parse(jsonStr, &data, opts...); err != nil {
		return nil, err
	}

	// Sequential processing
	results := make(map[string]any, len(paths))
	for _, path := range paths {
		if err := p.validatePath(path); err != nil {
			return nil, err
		}

		// Use cached recursive processor
		result, err := p.recursiveProcessor.ProcessRecursively(data, path, opGet, nil)

		if err != nil {
			// Continue with other paths, store error as result
			results[path] = nil
		} else {
			results[path] = result
		}
	}

	return results, nil
}

// incrementOperationCount atomically increments the operation counter with rate limiting
func (p *Processor) incrementOperationCount() {
	atomic.AddInt64(&p.metrics.operationCount, 1)
}

// checkRateLimit checks if the operation rate is within acceptable limits
func (p *Processor) checkRateLimit() error {
	if p.metrics.operationWindow <= 0 {
		return nil // Rate limiting disabled
	}

	now := time.Now().UnixNano()
	lastOp := atomic.LoadInt64(&p.metrics.lastOperationTime)

	if lastOp > 0 {
		timeDiff := now - lastOp
		if timeDiff < int64(time.Second)/p.metrics.operationWindow {
			return &JsonsError{
				Op:      "rate_limit",
				Message: "operation rate limit exceeded",
				Err:     ErrOperationFailed,
			}
		}
	}

	atomic.StoreInt64(&p.metrics.lastOperationTime, now)
	return nil
}

// incrementErrorCount atomically increments the error counter with optional logging
func (p *Processor) incrementErrorCount() {
	atomic.AddInt64(&p.metrics.errorCount, 1)
}

// logError logs an error with structured logging
func (p *Processor) logError(ctx context.Context, operation, path string, err error) {
	logger := p.getLogger()
	if logger == nil {
		return
	}

	errorType := "unknown"
	var jsonErr *JsonsError
	if errors.As(err, &jsonErr) && jsonErr.Err != nil {
		errorType = jsonErr.Err.Error()
	}

	if p.metrics != nil {
		p.metrics.collector.RecordError(errorType)
	}

	sanitizedPath := sanitizePath(path)
	sanitizedError := sanitizeError(err)

	logger.ErrorContext(ctx, "JSON operation failed",
		slog.String("operation", operation),
		slog.String("path", sanitizedPath),
		slog.String("error", sanitizedError),
		slog.String("error_type", errorType),
		slog.Int64("error_count", atomic.LoadInt64(&p.metrics.errorCount)),
		slog.String("processor_id", p.getProcessorID()),
		slog.Bool("cache_enabled", p.config.EnableCache),
		slog.Int64("concurrent_ops", atomic.LoadInt64(&p.metrics.operationCount)),
	)
}

// sanitizePath removes potentially sensitive information from paths
func sanitizePath(path string) string {
	if len(path) > 100 {
		return truncateString(path, 100)
	}
	// Remove potential sensitive patterns but keep structure
	// Use case-insensitive matching for better security
	lowerPath := strings.ToLower(path)
	// Use package-level sensitivePatterns from security.go for consistency
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerPath, pattern) {
			return "[REDACTED_PATH]"
		}
	}
	return path
}

// sanitizeError removes potentially sensitive information from error messages
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	errMsg := err.Error()
	if len(errMsg) > 200 {
		return truncateString(errMsg, 200)
	}
	return errMsg
}

// truncateString efficiently truncates a string with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// logOperation logs a successful operation with structured logging and performance warnings
func (p *Processor) logOperation(ctx context.Context, operation, path string, duration time.Duration) {
	logger := p.getLogger()
	if logger == nil {
		return
	}

	// Use modern structured logging with typed attributes
	commonAttrs := []slog.Attr{
		slog.String("operation", operation),
		slog.String("path", path),
		slog.Int64("duration_ms", duration.Milliseconds()),
		slog.Int64("operation_count", atomic.LoadInt64(&p.metrics.operationCount)),
		slog.String("processor_id", p.getProcessorID()),
	}

	if duration > slowOperationThreshold {
		// Log as warning for slow operations
		attrs := append(commonAttrs, slog.Int64("threshold_ms", slowOperationThreshold.Milliseconds()))
		logger.LogAttrs(ctx, slog.LevelWarn, "Slow JSON operation detected", attrs...)
	} else {
		// Log as debug for normal operations
		logger.LogAttrs(ctx, slog.LevelDebug, "JSON operation completed", commonAttrs...)
	}
}

// getProcessorID returns a unique identifier for this processor instance
func (p *Processor) getProcessorID() string {
	// Use the processor's memory address as a unique identifier
	return fmt.Sprintf("proc_%p", p)
}

// getPathSegments gets a path segments slice from the unified resource manager
func (p *Processor) getPathSegments() []internal.PathSegment {
	return getGlobalResourceManager().GetPathSegments()
}

// putPathSegments returns a path segments slice to the unified resource manager
func (p *Processor) putPathSegments(segments []internal.PathSegment) {
	getGlobalResourceManager().PutPathSegments(segments)
}

// getStringBuilder gets a string builder from the unified resource manager
func (p *Processor) getStringBuilder() *strings.Builder {
	return getGlobalResourceManager().GetStringBuilder()
}

// putStringBuilder returns a string builder to the unified resource manager
func (p *Processor) putStringBuilder(sb *strings.Builder) {
	getGlobalResourceManager().PutStringBuilder(sb)
}

// Set sets a value in JSON at the specified path
// Returns:
//   - On success: modified JSON string and nil error
//   - On failure: original unmodified JSON string and error information
func (p *Processor) Set(jsonStr, path string, value any, opts ...Config) (string, error) {
	if err := p.checkClosed(); err != nil {
		return jsonStr, err
	}

	options, err := p.prepareOptions(opts...)
	if err != nil {
		return jsonStr, err
	}

	if err := p.validateInput(jsonStr); err != nil {
		return jsonStr, err
	}

	if err := p.validatePath(path); err != nil {
		return jsonStr, err
	}

	// Parse JSON
	var data any
	err = p.Parse(jsonStr, &data, opts...)
	if err != nil {
		return jsonStr, &JsonsError{
			Op:      "set",
			Path:    path,
			Message: fmt.Sprintf("failed to parse JSON: %v", err),
			Err:     err,
		}
	}

	// Note: We directly modify the parsed data instead of creating a deep copy.
	// This is safe because:
	// 1. If the set operation fails, we return the original JSON string (jsonStr)
	// 2. If marshaling fails, we also return the original JSON string
	// 3. The parsed data is only used within this function scope
	// This optimization reduces memory allocations significantly.

	// Determine if we should create paths
	createPaths := options.CreatePaths || p.config.CreatePaths

	// Set the value at the specified path
	err = p.setValueAtPathWithOptions(data, path, value, createPaths)
	if err != nil {
		// Return original data and detailed error information
		var setError *JsonsError
		if _, ok := err.(*rootDataTypeConversionError); ok && createPaths {
			setError = &JsonsError{
				Op:      "set",
				Path:    path,
				Message: fmt.Sprintf("root data type conversion failed: %v", err),
				Err:     err,
			}
		} else {
			setError = &JsonsError{
				Op:      "set",
				Path:    path,
				Message: fmt.Sprintf("set operation failed: %v", err),
				Err:     err,
			}
		}
		return jsonStr, setError
	}

	// Convert modified data back to JSON string
	resultBytes, err := json.Marshal(data)
	if err != nil {
		// Return original data if marshaling fails
		return jsonStr, &JsonsError{
			Op:      "set",
			Path:    path,
			Message: fmt.Sprintf("failed to marshal modified data: %v", err),
			Err:     ErrOperationFailed,
		}
	}

	return string(resultBytes), nil
}

// SetMultiple sets multiple values in JSON using a map of path-value pairs
// Returns:
//   - On success: modified JSON string and nil error
//   - On failure: original unmodified JSON string and error information
func (p *Processor) SetMultiple(jsonStr string, updates map[string]any, opts ...Config) (string, error) {
	if err := p.checkClosed(); err != nil {
		return jsonStr, err
	}

	// Validate input
	if len(updates) == 0 {
		return jsonStr, nil // No updates to apply
	}

	// Prepare options
	options, err := p.prepareOptions(opts...)
	if err != nil {
		return jsonStr, err
	}

	// Validate JSON input
	if err := p.validateInput(jsonStr); err != nil {
		return jsonStr, err
	}

	// Validate all paths before processing
	for path := range updates {
		if err := p.validatePath(path); err != nil {
			return jsonStr, &JsonsError{
				Op:      "set_multiple",
				Path:    path,
				Message: fmt.Sprintf("invalid path '%s': %v", path, err),
				Err:     err,
			}
		}
	}

	// Parse JSON
	var data any
	err = p.Parse(jsonStr, &data, opts...)
	if err != nil {
		return jsonStr, &JsonsError{
			Op:      "set_multiple",
			Message: fmt.Sprintf("failed to parse JSON: %v", err),
			Err:     err,
		}
	}

	// Create a deep copy of the data for modification attempts
	var dataCopy any
	if copyBytes, marshalErr := json.Marshal(data); marshalErr == nil {
		if unmarshalErr := json.Unmarshal(copyBytes, &dataCopy); unmarshalErr != nil {
			return jsonStr, &JsonsError{
				Op:      "set_multiple",
				Message: fmt.Sprintf("failed to create data copy: %v", unmarshalErr),
				Err:     unmarshalErr,
			}
		}
	} else {
		return jsonStr, &JsonsError{
			Op:      "set_multiple",
			Message: fmt.Sprintf("failed to create data copy: %v", marshalErr),
			Err:     marshalErr,
		}
	}

	// Determine if we should create paths
	createPaths := options.CreatePaths || p.config.CreatePaths

	// Apply all updates on the copy
	var lastError error
	successCount := 0

	for path, value := range updates {
		err := p.setValueAtPathWithOptions(dataCopy, path, value, createPaths)
		if err != nil {
			// Handle root data type conversion errors
			if _, ok := err.(*rootDataTypeConversionError); ok && createPaths {
				lastError = &JsonsError{
					Op:      "set_multiple",
					Path:    path,
					Message: fmt.Sprintf("root data type conversion failed for path '%s': %v", path, err),
					Err:     err,
				}
				if !options.ContinueOnError {
					return jsonStr, lastError
				}
			} else {
				lastError = &JsonsError{
					Op:      "set_multiple",
					Path:    path,
					Message: fmt.Sprintf("failed to set path '%s': %v", path, err),
					Err:     err,
				}
				if !options.ContinueOnError {
					return jsonStr, lastError
				}
			}
		} else {
			successCount++
		}
	}

	// If no updates were successful and we have errors, return original data and error
	if successCount == 0 && lastError != nil {
		return jsonStr, &JsonsError{
			Op:      "set_multiple",
			Message: fmt.Sprintf("all %d updates failed, last error: %v", len(updates), lastError),
			Err:     lastError,
		}
	}

	// Convert modified data back to JSON string
	resultBytes, err := json.Marshal(dataCopy)
	if err != nil {
		// Return original data if marshaling fails
		return jsonStr, &JsonsError{
			Op:      "set_multiple",
			Message: fmt.Sprintf("failed to marshal modified data: %v", err),
			Err:     ErrOperationFailed,
		}
	}

	return string(resultBytes), nil
}

// SetCreate sets a value at the specified path, creating intermediate paths as needed.
// This is the unified API for set-with-path-creation operations.
//
// Example:
//
//	result, err := processor.SetCreate(data, "users[0].profile.name", "Alice")
func (p *Processor) SetCreate(jsonStr, path string, value any, opts ...Config) (string, error) {
	addOpts := mergeOptionsWithOverride(opts, func(o *Config) {
		o.CreatePaths = true
	})
	return p.Set(jsonStr, path, value, addOpts)
}

// SetMultipleCreate sets multiple values, creating intermediate paths as needed.
// This is the unified API for batch set-with-path-creation operations.
//
// Example:
//
//	result, err := processor.SetMultipleCreate(data, map[string]any{"user.name": "Alice", "user.age": 30})
func (p *Processor) SetMultipleCreate(jsonStr string, updates map[string]any, opts ...Config) (string, error) {
	addOpts := mergeOptionsWithOverride(opts, func(o *Config) {
		o.CreatePaths = true
	})
	return p.SetMultiple(jsonStr, updates, addOpts)
}

// GetStats returns processor performance statistics
func (p *Processor) GetStats() Stats {
	cacheStats := p.cache.GetStats()

	return Stats{
		CacheSize:        cacheStats.Entries,
		CacheMemory:      cacheStats.TotalMemory,
		MaxCacheSize:     p.config.MaxCacheSize,
		HitCount:         cacheStats.HitCount,
		MissCount:        cacheStats.MissCount,
		HitRatio:         cacheStats.HitRatio,
		CacheTTL:         p.config.CacheTTL,
		CacheEnabled:     p.config.EnableCache,
		IsClosed:         p.IsClosed(),
		MemoryEfficiency: cacheStats.MemoryEfficiency,
		OperationCount:   atomic.LoadInt64(&p.metrics.operationCount),
		ErrorCount:       atomic.LoadInt64(&p.metrics.errorCount),
	}
}

// GetHealthStatus returns the current health status
func (p *Processor) GetHealthStatus() HealthStatus {
	if p.metrics == nil {
		return HealthStatus{
			Timestamp: time.Now(),
			Healthy:   false,
			Checks: map[string]CheckResult{
				"metrics": {
					Healthy: false,
					Message: "Metrics collector not initialized",
				},
			},
		}
	}

	healthChecker := internal.NewHealthChecker(p.metrics.collector, nil)
	internalStatus := healthChecker.CheckHealth()

	// Convert internal.HealthStatus to HealthStatus
	checks := make(map[string]CheckResult)
	for name, result := range internalStatus.Checks {
		checks[name] = CheckResult{
			Healthy: result.Healthy,
			Message: result.Message,
		}
	}

	return HealthStatus{
		Timestamp: internalStatus.Timestamp,
		Healthy:   internalStatus.Healthy,
		Checks:    checks,
	}
}

// processorUtils provides utility functions for JSON processing
type processorUtils struct {
	// String builder pool for efficient string operations
	stringBuilderPool *stringBuilderPool
}

// NewProcessorUtils creates a new processor utils instance
func NewProcessorUtils() *processorUtils {
	return &processorUtils{
		stringBuilderPool: newStringBuilderPool(),
	}
}

// IsArrayType checks if the data is an array type
func (u *processorUtils) IsArrayType(data any) bool {
	switch data.(type) {
	case []any:
		return true
	default:
		return false
	}
}

// IsObjectType checks if the data is an object type
func (u *processorUtils) IsObjectType(data any) bool {
	switch data.(type) {
	case map[string]any, map[any]any:
		return true
	default:
		return false
	}
}

// IsEmptyContainer checks if a container (object or array) is empty
func (u *processorUtils) IsEmptyContainer(data any) bool {
	switch v := data.(type) {
	case map[string]any:
		return len(v) == 0
	case map[any]any:
		return len(v) == 0
	case []any:
		// Check if all elements are nil
		for _, item := range v {
			if item != nil {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// stringBuilderPool provides a pool of string builders for efficient string operations
type stringBuilderPool struct {
	pool sync.Pool
}

// newStringBuilderPool creates a new string builder pool
func newStringBuilderPool() *stringBuilderPool {
	return &stringBuilderPool{
		pool: sync.Pool{
			New: func() any {
				return &strings.Builder{}
			},
		},
	}
}

// Get gets a string builder from the pool
func (p *stringBuilderPool) Get() *strings.Builder {
	return p.pool.Get().(*strings.Builder)
}

// Put returns a string builder to the pool
func (p *stringBuilderPool) Put(sb *strings.Builder) {
	sb.Reset()
	p.pool.Put(sb)
}

// ParseInt parses a string to integer with error handling
func ParseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// ParseFloat parses a string to float64 with error handling
func ParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// ParseBool parses a string to boolean with error handling
func ParseBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}

// IsNumeric checks if a string represents a numeric value
func IsNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// IsInteger checks if a string represents an integer value
func IsInteger(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

// ClampIndex clamps an index to valid bounds for an array
func ClampIndex(index, length int) int {
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

// SanitizeKey sanitizes a key for safe use in maps
func SanitizeKey(key string) string {
	// Remove any null bytes or other problematic characters
	return strings.ReplaceAll(key, "\x00", "")
}

// EscapeJSONPointer escapes special characters for JSON Pointer
func EscapeJSONPointer(s string) string {
	return internal.EscapeJSONPointer(s)
}

// UnescapeJSONPointer unescapes JSON Pointer special characters
func UnescapeJSONPointer(s string) string {
	return internal.UnescapeJSONPointer(s)
}

// IsContainer checks if the data is a container type (map or slice)
func IsContainer(data any) bool {
	switch data.(type) {
	case map[string]any, map[any]any, []any:
		return true
	default:
		return false
	}
}

// GetContainerSize returns the size of a container
func GetContainerSize(data any) int {
	switch v := data.(type) {
	case map[string]any:
		return len(v)
	case map[any]any:
		return len(v)
	case []any:
		return len(v)
	default:
		return 0
	}
}

// CreateEmptyContainer creates an empty container of the specified type
func CreateEmptyContainer(containerType string) any {
	switch containerType {
	case "object":
		return make(map[string]any)
	case "array":
		return make([]any, 0)
	default:
		return make(map[string]any) // Default to object
	}
}

// ConvertToString converts a value to string
func (u *processorUtils) ConvertToString(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ConvertToNumber converts a value to a number (float64)
func (u *processorUtils) ConvertToNumber(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", value)
	}
}

// validateInput validates JSON input string with optimized security checks
func (p *Processor) validateInput(jsonString string) error {
	return p.securityValidator.ValidateJSONInput(jsonString)
}

// validatePath validates a JSON path string with enhanced security and efficiency
func (p *Processor) validatePath(path string) error {
	// Use the cached security validator instead of creating a new one each time
	return p.securityValidator.ValidatePathInput(path)
}

// ============================================================================
// COMPILED PATH METHODS
// PERFORMANCE: Pre-parsed paths for repeated operations with zero-parse overhead
// ============================================================================

// CompilePath compiles a JSON path string into a CompiledPath for fast repeated operations
// The returned CompiledPath can be reused for multiple Get/Set/Delete operations
func (p *Processor) CompilePath(path string) (*internal.CompiledPath, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	// Use the global compiled path cache for frequently used paths
	return internal.GetGlobalCompiledPathCache().Get(path)
}

// GetCompiled retrieves a value from JSON using a pre-compiled path
// PERFORMANCE: Skips path parsing for faster repeated operations
func (p *Processor) GetCompiled(jsonStr string, cp *internal.CompiledPath) (any, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	// Parse JSON once
	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	// Navigate using compiled path
	return cp.Get(data)
}

// GetCompiledFromParsed retrieves a value from pre-parsed JSON data using a compiled path
// PERFORMANCE: No JSON parsing overhead - uses already parsed data
func (p *Processor) GetCompiledFromParsed(data any, cp *internal.CompiledPath) (any, error) {
	if err := p.checkClosed(); err != nil {
		return nil, err
	}

	return cp.Get(data)
}

// GetCompiledExists checks if a path exists in pre-parsed JSON data using a compiled path
func (p *Processor) GetCompiledExists(data any, cp *internal.CompiledPath) bool {
	return cp.Exists(data)
}
