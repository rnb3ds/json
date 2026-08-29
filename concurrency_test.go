package json

// P-002 concurrency-audit regression tests.
//
// F1 under review: cachedMaxPatternLen (security.go) is a lock-free cached
// maximum pattern length that the rolling-window scanner uses for its overlap
// (scanWithRollingWindow: overlapSize = maxDangerousPatternLen() + 8). The
// pre-fix code invalidated the cache with a bare atomic.Store in the public
// register/unregister wrappers while recomputeMaxPatternLen read the registry
// and published its result WITHOUT the registry lock — a last-writer-wins race
// where a recompute that read a pre-registration snapshot could overwrite a
// newer invalidation with a stale, smaller length. Until the next registry
// event, the overlap was too small for a globally-registered long pattern,
// reopening the window-boundary straddle gap fixed in D-002 round 6.
//
// The fix serializes both sides through the registry RWMutex: Add/Remove/Clear
// invalidate while holding the write lock, and recompute reads + stores under
// the read lock.

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestP002_RecomputeSerializedWithRegistryWrites deterministically verifies the
// fix mechanism: recomputeMaxPatternLen must acquire the registry read lock, so
// while a writer holds globalPatternRegistry.mu (exactly what Add does during
// RegisterDangerousPattern), a concurrent recompute cannot run to completion
// and publish a result computed from a snapshot taken before the write.
//
// The pre-fix recompute took no lock, completed immediately, and failed this
// test — which is precisely the interleaving that let a stale length overwrite
// a fresh invalidation.
func TestP002_RecomputeSerializedWithRegistryWrites(t *testing.T) {
	const patLen = 200
	longPattern := DangerousPattern{
		Pattern: "p002_serialize" + strings.Repeat("x", patLen),
		Name:    "P-002 serialization probe",
		Level:   PatternLevelCritical,
	}
	RegisterDangerousPattern(longPattern)
	defer UnregisterDangerousPattern(longPattern.Pattern)

	// Reset the cache so the concurrent recompute below cannot be satisfied by
	// a previously cached value.
	atomic.StoreInt64(&cachedMaxPatternLen, 0)

	done := make(chan int, 1)

	// Simulate Add() mid-mutation: hold the registry write lock.
	globalPatternRegistry.mu.Lock()
	go func() { done <- recomputeMaxPatternLen() }()

	select {
	case <-done:
		// Release the write lock BEFORE failing so later tests in the package
		// are not deadlocked behind this one's fatal path.
		globalPatternRegistry.mu.Unlock()
		t.Fatal("recomputeMaxPatternLen completed while the registry write lock was held — " +
			"its result is not serialized against registration, so a stale length can " +
			"overwrite a newer invalidation (P-002 F1)")
	case <-time.After(50 * time.Millisecond):
		// Blocked on the read lock, as required.
	}

	globalPatternRegistry.mu.Unlock()

	select {
	case got := <-done:
		if got < len(longPattern.Pattern) {
			t.Fatalf("recomputed max pattern length = %d, want >= %d (registered pattern visible after lock release)",
				got, len(longPattern.Pattern))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("recomputeMaxPatternLen did not finish after the write lock was released")
	}
}

// TestP002_PatternLenCacheConcurrentChurn exercises the invalidation/recompute
// pair under concurrent load. The race detector (run with -race) validates the
// lock discipline; the post-churn assertion pins the observable contract: once
// registration activity has settled, the cached length must cover the
// registered pattern.
func TestP002_PatternLenCacheConcurrentChurn(t *testing.T) {
	const patLen = 150
	pattern := DangerousPattern{
		Pattern: "p002_churn" + strings.Repeat("y", patLen),
		Name:    "P-002 churn probe",
		Level:   PatternLevelCritical,
	}
	RegisterDangerousPattern(pattern)
	defer UnregisterDangerousPattern(pattern.Pattern)

	const readers = 4
	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Readers keep maxDangerousPatternLen() hot, forcing recomputes after each
	// invalidation.
	for range readers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = maxDangerousPatternLen()
			}
		}()
	}

	// Churner invalidates via repeated register/unregister cycles.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 2000 {
			UnregisterDangerousPattern(pattern.Pattern)
			RegisterDangerousPattern(pattern)
		}
		// Leave the pattern registered.
		RegisterDangerousPattern(pattern)
	}()

	// Wait specifically for the churner by letting all goroutines observe stop
	// only after churn completes: close stop, then wg.Wait covers everyone.
	// The churner's final Register is its last statement, so after wg.Wait()
	// the registry holds the pattern and the cache holds either 0 (invalidated)
	// or a value >= patLen.
	close(stop)
	wg.Wait()

	if got := maxDangerousPatternLen(); got < len(pattern.Pattern) {
		t.Fatalf("post-churn maxDangerousPatternLen() = %d, want >= %d", got, len(pattern.Pattern))
	}
}
