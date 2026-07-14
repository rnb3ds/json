package json

import "testing"

// ============================================================================
// Boundary tests for api.go low-coverage paths: config->processor cache
// eviction and stale-entry handling.
// ============================================================================

// --- getProcessorWithConfig cache eviction (api.go:1269/1356) ---

func TestGetProcessorWithConfig_CacheEviction(t *testing.T) {
	// configProcessorCacheLimit is 64; flooding with 80 distinct configs
	// forces the eviction path. Evicted processors are closed asynchronously.
	for i := 0; i < 80; i++ {
		cfg := DefaultConfig()
		cfg.MaxCacheSize = 100 + i // distinct config -> distinct cache key
		p, err := getProcessorWithConfig(cfg)
		if err != nil || p == nil {
			t.Fatalf("getProcessorWithConfig(%d) err=%v p=%v", i, err, p)
		}
	}
}

// --- getProcessorWithConfig stale cached entry (api.go:1281) ---

func TestGetProcessorWithConfig_StaleCache(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxCacheSize = 99991 // distinct value so no other test collides
	p1, err := getProcessorWithConfig(cfg)
	if err != nil || p1 == nil {
		t.Fatalf("first get err=%v p=%v", err, p1)
	}
	p1.Close() // cached entry is now stale (closed processor)

	// Second request for the same config must detect staleness and return a
	// fresh, usable processor rather than the closed one.
	p2, err := getProcessorWithConfig(cfg)
	if err != nil || p2 == nil {
		t.Fatalf("expected fresh processor after stale entry, got err=%v p=%v", err, p2)
	}
}
