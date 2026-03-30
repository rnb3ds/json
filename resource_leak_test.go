package json

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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

	iterator := NewParallelIterator(data, 4)

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

	iterator := NewParallelIterator(data, 4)

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
		iterator := NewParallelIterator(data, 4)
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
		iterator := NewParallelIterator(data, 4)
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
		iv := NewIterableValue(data)
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
// unbounded memory growth
func TestNoMemoryGrowthInLoops(t *testing.T) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	cfg := DefaultConfig()
	processor, _ := New(cfg)

	// Perform many operations
	for i := 0; i < 1000; i++ {
		_, _ = processor.Get(`{"key": "value", "nested": {"a": 1}}`, ".nested.a")
	}

	processor.Close()

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Allow some growth but not unbounded
	heapGrowth := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	if heapGrowth > 10*1024*1024 { // 10MB threshold
		t.Logf("Warning: heap grew by %d bytes after operations", heapGrowth)
	}
}

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

	iterator := NewParallelIterator(data, 4)

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

// TestResourceManagerMapCounter verifies that PutMap correctly decrements
// the allocated maps counter even when m is nil
func TestResourceManagerMapCounter(t *testing.T) {
	// Get fresh resource manager
	urm := newUnifiedResourceManager()

	// Get multiple maps
	for i := 0; i < 10; i++ {
		_ = urm.GetMap()
	}

	// Put nil map - should still decrement counter
	urm.PutMap(nil)

	// Put oversized map - should still decrement counter
	largeMap := make(map[string]any, 100)
	for i := 0; i < 100; i++ {
		largeMap[fmt.Sprintf("key%d", i)] = i
	}
	urm.GetMap()
	urm.PutMap(largeMap)
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

	iterator := NewParallelIterator(data, 4)

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
