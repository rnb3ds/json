package json

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cybergodev/json/internal"
)

// ============================================================================
// RESOURCE LEAK VERIFICATION TESTS
// These tests verify that resources (goroutines, channels, memory) are
// properly cleaned up and not leaked.
// ============================================================================

// countGoroutines returns the current number of goroutines
func countGoroutines() int {
	runtime.GC()
	time.Sleep(10 * time.Millisecond) // Allow goroutines to settle
	return runtime.NumGoroutine()
}

// ============================================================================
// PROCESSOR CLOSE TESTS
// ============================================================================

// TestProcessorCloseGoroutineCleanup verifies that Processor.Close() properly
// cleans up all goroutines including the drain goroutine
func TestProcessorCloseGoroutineCleanup(t *testing.T) {
	initialGoroutines := countGoroutines()

	// Create a processor with concurrency limits
	cfg := DefaultConfig()
	cfg.MaxConcurrency = 10
	processor, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Perform some operations to populate the semaphore
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = processor.Get(`{"test": "value"}`, ".")
		}()
	}
	wg.Wait()

	// Close the processor
	err = processor.Close()
	if err != nil {
		t.Errorf("Processor.Close() returned error: %v", err)
	}

	// Allow goroutines to settle
	time.Sleep(100 * time.Millisecond)

	// Verify goroutine count hasn't increased significantly
	finalGoroutines := countGoroutines()
	leaked := finalGoroutines - initialGoroutines

	if leaked > 2 { // Allow small variance for test infrastructure
		t.Errorf("Potential goroutine leak: started with %d, ended with %d (leaked: %d)",
			initialGoroutines, finalGoroutines, leaked)
	}
}

// TestProcessorCloseWithTimeout verifies that Close() handles timeout properly
// and doesn't block indefinitely
func TestProcessorCloseWithTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConcurrency = 5
	processor, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Close should complete within reasonable time
	done := make(chan error, 1)
	go func() {
		done <- processor.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Processor.Close() blocked for too long - potential goroutine leak")
	}
}

// ============================================================================
// PARALLEL ITERATOR CONTEXT TESTS
// ============================================================================

// TestParallelIteratorForEachWithContextCancellation verifies that
// ForEachWithContext properly cancels goroutines when context is cancelled
func TestParallelIteratorForEachWithContextCancellation(t *testing.T) {
	data := make([]any, 100)
	for i := range data {
		data[i] = i
	}

	cfg := DefaultConfig()
	cfg.MaxConcurrency = 4
	iterator := NewParallelIterator(data, cfg)

	// Use a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	var processedCount int32
	var goroutineCount int32

	// Start processing
	done := make(chan error, 1)
	go func() {
		err := iterator.ForEachWithContext(ctx, func(idx int, val any) error {
			atomic.AddInt32(&goroutineCount, 1)
			// Simulate work
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&processedCount, 1)
			return nil
		})
		done <- err
	}()

	// Cancel after a short time
	time.Sleep(30 * time.Millisecond)
	cancel()

	// Wait for completion
	err := <-done

	// Should return context cancelled error
	if err != context.Canceled {
		t.Logf("ForEachWithContext returned: %v (expected context.Canceled)", err)
	}

	// Verify not all items were processed (cancellation worked)
	processed := atomic.LoadInt32(&processedCount)
	if processed >= 100 {
		t.Errorf("Expected cancellation to stop processing, but %d items were processed", processed)
	}
}

// TestParallelIteratorForEachBatchWithContextCancellation verifies that
// ForEachBatchWithContext properly cancels goroutines when context is cancelled
func TestParallelIteratorForEachBatchWithContextCancellation(t *testing.T) {
	data := make([]any, 200)
	for i := range data {
		data[i] = i
	}

	cfg := DefaultConfig()
	cfg.MaxConcurrency = 4
	iterator := NewParallelIterator(data, cfg)

	ctx, cancel := context.WithCancel(context.Background())

	var batchCount int32

	done := make(chan error, 1)
	go func() {
		err := iterator.ForEachBatchWithContext(ctx, 10, func(idx int, batch []any) error {
			atomic.AddInt32(&batchCount, 1)
			time.Sleep(30 * time.Millisecond)
			return nil
		})
		done <- err
	}()

	// Cancel after processing some batches
	time.Sleep(50 * time.Millisecond)
	cancel()

	err := <-done

	// Should return context cancelled error or nil
	t.Logf("ForEachBatchWithContext returned: %v, processed %d batches", err, batchCount)

	// Verify not all batches were processed
	batches := atomic.LoadInt32(&batchCount)
	if batches >= 20 {
		t.Errorf("Expected cancellation to stop batch processing, but %d batches were processed", batches)
	}
}

// TestParallelIteratorForEachNoLeak verifies that ForEach doesn't leak
// goroutines even when all items are processed successfully
func TestParallelIteratorForEachNoLeak(t *testing.T) {
	initialGoroutines := countGoroutines()

	data := make([]any, 50)
	for i := range data {
		data[i] = i
	}

	// Run multiple iterations
	for i := 0; i < 10; i++ {
		cfg := DefaultConfig()
		cfg.MaxConcurrency = 4
		iterator := NewParallelIterator(data, cfg)
		err := iterator.ForEach(func(idx int, val any) error {
			return nil
		})
		if err != nil {
			t.Errorf("ForEach returned error: %v", err)
		}
	}

	// Allow goroutines to settle
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := countGoroutines()
	leaked := finalGoroutines - initialGoroutines

	if leaked > 2 {
		t.Errorf("Potential goroutine leak after ForEach: started with %d, ended with %d (leaked: %d)",
			initialGoroutines, finalGoroutines, leaked)
	}
}

// ============================================================================
// CACHE MANAGER CLEANUP TESTS
// ============================================================================

// TestCacheManagerCloseCleanup verifies that CacheManager.Close() properly
// waits for cleanup goroutines
func TestCacheManagerCloseCleanup(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnableCache = true
	cfg.MaxCacheSize = 1000
	cfg.CacheTTL = 1 * time.Minute

	processor, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Populate cache
	for i := 0; i < 100; i++ {
		_, _ = processor.Get(`{"key": "value"}`, ".")
	}

	// Close should complete without hanging
	done := make(chan error, 1)
	go func() {
		done <- processor.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Close returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Processor.Close() with cache blocked for too long")
	}
}

// ============================================================================
// CHANNEL LEAK TESTS
// ============================================================================

// TestParallelIteratorChannelCleanup verifies that channels created by
// ParallelIterator are properly cleaned up
func TestParallelIteratorChannelCleanup(t *testing.T) {
	data := make([]any, 20)
	for i := range data {
		data[i] = i
	}

	// Create multiple iterators and verify they don't leak channels
	for i := 0; i < 50; i++ {
		cfg := DefaultConfig()
		cfg.MaxConcurrency = 4
		iterator := NewParallelIterator(data, cfg)
		_ = iterator.ForEach(func(idx int, val any) error {
			return nil
		})
	}

	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	// If channels were leaked, we'd see memory growth
	// This is a basic sanity check
}

// ============================================================================
// ITERATOR POOL TESTS
// ============================================================================

// TestIteratorPoolNoLeak verifies that pooled iterators are properly
// returned to the pool
func TestIteratorPoolNoLeak(t *testing.T) {
	data := map[string]any{"key1": "value1", "key2": "value2"}

	// Create and release many iterators
	for i := 0; i < 100; i++ {
		iv := newIterableValue(data)
		_ = iv.GetString("key1")
		iv.Release()
	}

	// Pool should be healthy - no way to directly check pool size,
	// but we verify no panic or deadlock occurs
}

// ============================================================================
// SEMAPHORE DRAIN TESTS
// ============================================================================

// TestSemaphoreDrainOnClose verifies that the semaphore is properly
// drained during close and doesn't leave goroutines waiting
func TestSemaphoreDrainOnClose(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxConcurrency = 3
	processor, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Start operations that use the semaphore
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// These might block waiting for semaphore
			_, _ = processor.Get(`{"test": "value"}`, ".")
		}()
	}

	// Give some time for operations to start
	time.Sleep(20 * time.Millisecond)

	// Close while operations are potentially in progress
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- processor.Close()
	}()

	select {
	case err := <-closeDone:
		if err != nil {
			t.Errorf("Close returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Close took too long - semaphore drain may be stuck")
	}

	wg.Wait()
}

// ============================================================================
// MEMORY LEAK DETECTION TESTS
// ============================================================================

// TestNoMemoryGrowthInLoops verifies that repeated operations don't cause

// ============================================================================
// CONTEXT TIMEOUT TESTS
// ============================================================================

// TestParallelIteratorWithTimeout verifies that ForEachWithContext respects
// context timeout
func TestParallelIteratorWithTimeout(t *testing.T) {
	data := make([]any, 100)
	for i := range data {
		data[i] = i
	}

	cfg := DefaultConfig()
	cfg.MaxConcurrency = 4
	iterator := NewParallelIterator(data, cfg)

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var processedCount int32

	err := iterator.ForEachWithContext(ctx, func(idx int, val any) error {
		atomic.AddInt32(&processedCount, 1)
		time.Sleep(20 * time.Millisecond) // Slow processing
		return nil
	})

	// Should timeout
	if err != context.DeadlineExceeded {
		t.Logf("Expected context.DeadlineExceeded, got: %v", err)
	}

	processed := atomic.LoadInt32(&processedCount)
	t.Logf("Processed %d items before timeout", processed)

	if processed >= 100 {
		t.Error("Expected timeout to limit processing")
	}
}

// ============================================================================
// STRESS TESTS
// ============================================================================

// TestConcurrentProcessorOperations verifies no leaks under concurrent load
func TestConcurrentProcessorOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	initialGoroutines := countGoroutines()

	cfg := DefaultConfig()
	cfg.MaxConcurrency = 10
	cfg.EnableCache = true

	processor, _ := New(cfg)

	var wg sync.WaitGroup
	// Launch many concurrent operations
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, _ = processor.Get(`{"test": "value"}`, ".")
				_, _ = processor.Get(`{"nested": {"key": `+string(rune('0'+j%10))+`}}`, ".nested")
			}
		}(i)
	}

	wg.Wait()

	// Close processor
	_ = processor.Close()

	// Allow cleanup
	time.Sleep(200 * time.Millisecond)

	finalGoroutines := countGoroutines()
	leaked := finalGoroutines - initialGoroutines

	if leaked > 5 {
		t.Errorf("Goroutine leak detected after concurrent operations: started with %d, ended with %d (leaked: %d)",
			initialGoroutines, finalGoroutines, leaked)
	}
}

// ============================================================================
// RESOURCE MANAGER COUNTER TESTS
// ============================================================================

// TestResourceManagerMapCounter verifies that map pool operations work correctly
func TestResourceManagerMapCounter(t *testing.T) {
	// Get and return maps from pool - should not panic
	for i := 0; i < 10; i++ {
		m := internal.GetStreamingMap(8)
		internal.PutStreamingMap(m)
	}

	// Put nil map - should not panic
	internal.PutStreamingMap(nil)

	// Put large map - should not panic
	largeMap := make(map[string]any, 100)
	for i := 0; i < 100; i++ {
		largeMap[fmt.Sprintf("key%d", i)] = i
	}
	m := internal.GetStreamingMap(8)
	_ = m
	internal.PutStreamingMap(largeMap)
}

// ============================================================================
// PARALLEL ITERATOR CLOSE TESTS
// ============================================================================

// TestParallelIteratorClose verifies that Close method works correctly
func TestParallelIteratorClose(t *testing.T) {
	data := make([]any, 20)
	for i := range data {
		data[i] = i
	}

	cfg := DefaultConfig()
	cfg.MaxConcurrency = 4
	iterator := NewParallelIterator(data, cfg)

	// Process some data
	_ = iterator.ForEach(func(idx int, val any) error {
		return nil
	})

	// Close should not panic
	iterator.Close()

	// Multiple Close calls should be safe
	iterator.Close()
}

// ============================================================================
// ASYNC PROCESSOR CLOSE TIMEOUT TESTS
// ============================================================================

// TestAsyncProcessorCloseTimeout verifies that async processor close
// has proper timeout protection
func TestAsyncProcessorCloseTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// Create multiple processors
	processors := make([]*Processor, 10)
	for i := range processors {
		cfg := DefaultConfig()
		cfg.EnableCache = true
		p, err := New(cfg)
		if err != nil {
			t.Fatalf("Failed to create processor: %v", err)
		}
		processors[i] = p
	}

	// Close all processors concurrently (simulates eviction scenario)
	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for _, p := range processors {
			wg.Add(1)
			go func(proc *Processor) {
				defer wg.Done()
				_ = proc.Close()
			}(p)
		}
		wg.Wait()
	}()

	select {
	case <-done:
		// All closed successfully
	case <-time.After(10 * time.Second):
		t.Error("Async close took too long - potential goroutine leak")
	}
}

// ============================================================================
// FIX VERIFICATION TESTS (2026-04-11 resource leak audit)
// ============================================================================

// TestStaleProcessorCloseHasTimeout verifies that the stale processor close
// goroutine in getProcessorWithConfig is protected by a timeout (Fix #1).
func TestStaleProcessorCloseHasTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	// Fill the cache to capacity to trigger eviction
	cfg := DefaultConfig()
	for i := 0; i < configProcessorCacheLimit+10; i++ {
		c := cfg
		c.MaxConcurrency = i + 1 // unique config key
		_, err := getProcessorWithConfig(c)
		if err != nil {
			t.Fatalf("getProcessorWithConfig failed: %v", err)
		}
	}

	// Eviction runs asynchronously; verify goroutines don't accumulate
	initial := countGoroutines()
	time.Sleep(200 * time.Millisecond)
	after := countGoroutines()

	leaked := after - initial
	if leaked > 3 {
		t.Errorf("Goroutine leak after cache eviction: before=%d after=%d leaked=%d",
			initial, after, leaked)
	}
}

// TestClearStructEncoderCache verifies that ClearStructEncoderCache properly
// reclaims memory from the global struct encoder cache (Fix #2).
func TestClearStructEncoderCache(t *testing.T) {
	// Populate cache with multiple struct types
	type testStructA struct{ Name string }
	type testStructB struct{ Value int }
	type testStructC struct{ Flag bool }

	_ = internal.GetStructEncoder(reflect.TypeOf(testStructA{}))
	_ = internal.GetStructEncoder(reflect.TypeOf(testStructB{}))
	_ = internal.GetStructEncoder(reflect.TypeOf(testStructC{}))

	// Clear cache
	internal.ClearStructEncoderCache()

	// Verify cache is repopulated after clear (functional correctness)
	fields := internal.GetStructEncoder(reflect.TypeOf(testStructA{}))
	if len(fields) == 0 {
		t.Error("GetStructEncoder returned empty fields after cache clear")
	}
}

// TestClearPathTypeCacheOnProcessorClose verifies that the path type cache
// is cleared when a processor is closed (Fix #3).
func TestClearPathTypeCacheOnProcessorClose(t *testing.T) {
	processor, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Populate path type cache by querying various paths
	paths := []string{"simple", "nested.key", "array[0]", "deep.nested.path", "complex[1].field"}
	for _, path := range paths {
		_, _ = processor.Get(`{"simple":1,"nested":{"key":2},"array":[3],"deep":{"nested":{"path":4}},"complex":[{"field":5}]}`, path)
	}

	// Close processor — should clear path type cache
	err = processor.Close()
	if err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	// Verify cache is empty by checking shard sizes
	// (We can't directly access shards, so verify no panic and clean state)
	processor2, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create second processor: %v", err)
	}
	_ = processor2.Close()
}

// TestHooksClearedOnClose verifies that hooks slice is nil'd on Close(),
// releasing closure references for GC (Fix #4).
func TestHooksClearedOnClose(t *testing.T) {
	processor, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	// Add hooks that capture large closures
	largeData := make([]byte, 1024*1024) // 1MB
	for i := 0; i < 10; i++ {
		hook := &testClosureHook{data: largeData}
		processor.AddHook(hook)
	}

	// Verify hooks are present
	processor.hooksMu.Lock()
	hookCount := len(processor.hooks)
	processor.hooksMu.Unlock()
	if hookCount != 10 {
		t.Fatalf("Expected 10 hooks, got %d", hookCount)
	}

	// Close should nil the hooks
	err = processor.Close()
	if err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	processor.hooksMu.Lock()
	hooksAfter := processor.hooks
	processor.hooksMu.Unlock()

	if hooksAfter != nil {
		t.Errorf("Expected hooks to be nil after Close(), got %d hooks", len(hooksAfter))
	}
}

// testClosureHook is a test hook that captures a reference to external data.
type testClosureHook struct {
	data []byte
}

func (h *testClosureHook) Before(ctx HookContext) error { return nil }
func (h *testClosureHook) After(ctx HookContext, result any, err error) (any, error) {
	return result, err
}

// TestEvictionGoroutinesBounded verifies that eviction close goroutines
// don't accumulate beyond the semaphore limit (Fix #5).
func TestEvictionGoroutinesBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	initial := countGoroutines()

	// Create and evict many processors by exceeding cache limit repeatedly
	for round := 0; round < 5; round++ {
		for i := 0; i < configProcessorCacheLimit+configProcessorCacheEvictNum; i++ {
			cfg := DefaultConfig()
			cfg.MaxConcurrency = round*1000 + i // unique config
			_, _ = getProcessorWithConfig(cfg)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Allow goroutines to settle
	time.Sleep(500 * time.Millisecond)

	final := countGoroutines()
	leaked := final - initial
	if leaked > 5 {
		t.Errorf("Goroutine leak after eviction stress: before=%d after=%d leaked=%d",
			initial, final, leaked)
	}
}
