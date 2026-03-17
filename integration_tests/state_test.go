package integration_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/szaher/designs/agentz/internal/state"
)

// makeEntries generates n state entries with sequential FQNs.
func makeEntries(n int) []state.Entry {
	entries := make([]state.Entry, n)
	now := time.Now()
	for i := 0; i < n; i++ {
		entries[i] = state.Entry{
			FQN:         fmt.Sprintf("agent.test-%03d", i),
			Hash:        fmt.Sprintf("sha256:%064d", i),
			Status:      state.StatusApplied,
			LastApplied: now,
			Adapter:     "mock",
		}
	}
	return entries
}

// TestStateCacheWarmVsCold verifies that the LocalBackend cache works
// correctly for cold reads (first Load from disk) and warm hits
// (subsequent Load and Get calls served from the in-memory cache).
// T024A: State cache benchmark integration test.
func TestStateCacheWarmVsCold(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, ".agentspec.state.json")
	backend := state.NewLocalBackend(stateFile)

	entries := makeEntries(100)

	// Save 100 entries to disk.
	if err := backend.Save(entries); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Cold read: first Load reads from disk.
	coldEntries, err := backend.Load()
	if err != nil {
		t.Fatalf("cold Load failed: %v", err)
	}
	if len(coldEntries) != 100 {
		t.Fatalf("cold Load returned %d entries, want 100", len(coldEntries))
	}

	// Warm read: second Load should hit the cache.
	warmEntries, err := backend.Load()
	if err != nil {
		t.Fatalf("warm Load failed: %v", err)
	}
	if len(warmEntries) != 100 {
		t.Fatalf("warm Load returned %d entries, want 100", len(warmEntries))
	}

	// Verify Get() works for specific FQNs (cache hit path).
	for _, fqn := range []string{"agent.test-000", "agent.test-050", "agent.test-099"} {
		entry, err := backend.Get(fqn)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", fqn, err)
		}
		if entry == nil {
			t.Fatalf("Get(%q) returned nil, want entry", fqn)
		}
		if entry.FQN != fqn {
			t.Errorf("Get(%q) returned FQN %q", fqn, entry.FQN)
		}
	}

	// Verify cache stats show hits > 0.
	hits, _ := backend.CacheStats()
	if hits == 0 {
		t.Errorf("CacheStats hits = 0, want > 0 after Get calls")
	}
	t.Logf("CacheStats: hits=%d", hits)
}

// TestStateCacheInvalidateOnSave verifies that saving new entries
// invalidates the cache so the next Load returns the fresh data.
// T024A: Ensures Save() properly busts the cache (T023 invariant).
func TestStateCacheInvalidateOnSave(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, ".agentspec.state.json")
	backend := state.NewLocalBackend(stateFile)

	// Phase 1: Save initial entries and warm the cache.
	initial := makeEntries(100)
	if err := backend.Save(initial); err != nil {
		t.Fatalf("Save(initial) failed: %v", err)
	}
	loaded, err := backend.Load()
	if err != nil {
		t.Fatalf("Load(initial) failed: %v", err)
	}
	if len(loaded) != 100 {
		t.Fatalf("Load(initial) returned %d entries, want 100", len(loaded))
	}

	// Phase 2: Save a different set of entries (fewer, different FQNs).
	updated := []state.Entry{
		{
			FQN:         "agent.updated-001",
			Hash:        "sha256:updated1",
			Status:      state.StatusApplied,
			LastApplied: time.Now(),
			Adapter:     "mock",
		},
		{
			FQN:         "agent.updated-002",
			Hash:        "sha256:updated2",
			Status:      state.StatusApplied,
			LastApplied: time.Now(),
			Adapter:     "mock",
		},
	}
	if err := backend.Save(updated); err != nil {
		t.Fatalf("Save(updated) failed: %v", err)
	}

	// Phase 3: Load again and verify cache was invalidated.
	reloaded, err := backend.Load()
	if err != nil {
		t.Fatalf("Load(updated) failed: %v", err)
	}
	if len(reloaded) != 2 {
		t.Fatalf("Load(updated) returned %d entries, want 2 (cache not invalidated?)", len(reloaded))
	}

	// Verify the new entries are present.
	fqns := map[string]bool{}
	for _, e := range reloaded {
		fqns[e.FQN] = true
	}
	for _, want := range []string{"agent.updated-001", "agent.updated-002"} {
		if !fqns[want] {
			t.Errorf("expected FQN %q in reloaded entries, not found", want)
		}
	}

	// Verify old entries are gone.
	entry, err := backend.Get("agent.test-000")
	if err != nil {
		t.Fatalf("Get(old FQN) failed: %v", err)
	}
	if entry != nil {
		t.Errorf("Get(old FQN) returned entry, want nil after cache invalidation")
	}
}

// BenchmarkStateCacheGet measures the performance of Get() calls against
// a warm cache. With the O(1) index lookup (T022), Get should be very fast.
func BenchmarkStateCacheGet(b *testing.B) {
	tmpDir := b.TempDir()
	stateFile := filepath.Join(tmpDir, ".agentspec.state.json")
	backend := state.NewLocalBackend(stateFile)

	entries := makeEntries(100)
	if err := backend.Save(entries); err != nil {
		b.Fatalf("Save failed: %v", err)
	}

	// Warm the cache.
	if _, err := backend.Load(); err != nil {
		b.Fatalf("Load (warm) failed: %v", err)
	}

	// Build a list of FQNs to look up in round-robin.
	fqns := make([]string, len(entries))
	for i, e := range entries {
		fqns[i] = e.FQN
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fqn := fqns[i%len(fqns)]
		entry, err := backend.Get(fqn)
		if err != nil {
			b.Fatalf("Get(%q) failed: %v", fqn, err)
		}
		if entry == nil {
			b.Fatalf("Get(%q) returned nil", fqn)
		}
	}
	b.StopTimer()

	hits, misses := backend.CacheStats()
	b.Logf("CacheStats after benchmark: hits=%d misses=%d", hits, misses)
}

// Ensure unused imports are referenced.
var _ = os.Remove
