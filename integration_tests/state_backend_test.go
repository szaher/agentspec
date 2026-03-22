package integration_tests

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/szaher/agentspec/internal/state"
)

// generateEntries creates n test entries for backend parity testing.
func generateEntries(n int) []state.Entry {
	entries := make([]state.Entry, n)
	now := time.Now().Truncate(time.Second)
	for i := 0; i < n; i++ {
		entries[i] = state.Entry{
			FQN:         fmt.Sprintf("resource/test-%03d", i),
			Hash:        fmt.Sprintf("hash-%03d", i),
			Status:      state.StatusApplied,
			LastApplied: now,
			Adapter:     "test",
		}
	}
	return entries
}

// TestBackendParityLocal verifies save/load/get/list parity with the local backend.
// Other backends (etcd, postgres, s3, kubernetes) require external infrastructure
// and should be tested in CI with appropriate services running.
func TestBackendParityLocal(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "parity.state.json")

	backend, err := state.New("local", map[string]string{"path": statePath})
	if err != nil {
		t.Fatalf("New(local) failed: %v", err)
	}

	entries := generateEntries(100)

	// Save
	if err := backend.Save(entries); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loaded, err := backend.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded) != 100 {
		t.Fatalf("Load returned %d entries, want 100", len(loaded))
	}

	// Verify all entries present with correct data
	fqnMap := make(map[string]state.Entry)
	for _, e := range loaded {
		fqnMap[e.FQN] = e
	}

	for _, orig := range entries {
		got, ok := fqnMap[orig.FQN]
		if !ok {
			t.Errorf("missing entry %q after load", orig.FQN)
			continue
		}
		if got.Hash != orig.Hash {
			t.Errorf("entry %q: hash = %q, want %q", orig.FQN, got.Hash, orig.Hash)
		}
		if got.Status != orig.Status {
			t.Errorf("entry %q: status = %q, want %q", orig.FQN, got.Status, orig.Status)
		}
		if got.Adapter != orig.Adapter {
			t.Errorf("entry %q: adapter = %q, want %q", orig.FQN, got.Adapter, orig.Adapter)
		}
	}

	// Get individual entries
	for _, fqn := range []string{"resource/test-000", "resource/test-050", "resource/test-099"} {
		entry, err := backend.Get(fqn)
		if err != nil {
			t.Errorf("Get(%q) failed: %v", fqn, err)
			continue
		}
		if entry == nil {
			t.Errorf("Get(%q) returned nil", fqn)
			continue
		}
		if entry.FQN != fqn {
			t.Errorf("Get(%q) returned FQN %q", fqn, entry.FQN)
		}
	}

	// Get nonexistent
	entry, err := backend.Get("resource/nonexistent")
	if err != nil {
		t.Errorf("Get(nonexistent) should not error: %v", err)
	}
	if entry != nil {
		t.Errorf("Get(nonexistent) should return nil")
	}

	// List all
	all, err := backend.List(nil)
	if err != nil {
		t.Fatalf("List(nil) failed: %v", err)
	}
	if len(all) != 100 {
		t.Errorf("List(nil) returned %d, want 100", len(all))
	}

	// List by status
	applied := state.StatusApplied
	filtered, err := backend.List(&applied)
	if err != nil {
		t.Fatalf("List(applied) failed: %v", err)
	}
	if len(filtered) != 100 {
		t.Errorf("List(applied) returned %d, want 100", len(filtered))
	}

	failed := state.StatusFailed
	failedEntries, err := backend.List(&failed)
	if err != nil {
		t.Fatalf("List(failed) failed: %v", err)
	}
	if len(failedEntries) != 0 {
		t.Errorf("List(failed) returned %d, want 0", len(failedEntries))
	}
}

// TestBackendRegistryAvailability verifies all expected backends are registered.
func TestBackendRegistryAvailability(t *testing.T) {
	available := state.Available()
	expected := []string{"etcd", "kubernetes", "local", "postgres", "s3"}

	if len(available) < len(expected) {
		t.Fatalf("Available() = %v, want at least %v", available, expected)
	}

	avSet := make(map[string]bool)
	for _, name := range available {
		avSet[name] = true
	}

	for _, name := range expected {
		if !avSet[name] {
			t.Errorf("backend %q not registered; available = %v", name, available)
		}
	}
}

// TestConcurrentWriteNoCorruption verifies two goroutines writing to the same
// local backend simultaneously don't corrupt the state file.
func TestConcurrentWriteNoCorruption(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "concurrent.state.json")

	const numWriters = 5
	const entriesPerWriter = 20
	var wg sync.WaitGroup
	errCh := make(chan error, numWriters)

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()

			backend := state.NewLocalBackend(statePath).WithLockConfig(state.LockConfig{
				LockTimeout: 30 * time.Second,
			})

			if err := backend.LockWithContext(t.Context()); err != nil {
				errCh <- fmt.Errorf("writer %d lock: %w", writerID, err)
				return
			}
			defer func() { _ = backend.Unlock() }()

			// Load current state
			current, err := backend.Load()
			if err != nil {
				errCh <- fmt.Errorf("writer %d load: %w", writerID, err)
				return
			}

			// Add entries
			for j := 0; j < entriesPerWriter; j++ {
				current = append(current, state.Entry{
					FQN:         fmt.Sprintf("resource/writer-%d-entry-%d", writerID, j),
					Hash:        fmt.Sprintf("hash-%d-%d", writerID, j),
					Status:      state.StatusApplied,
					LastApplied: time.Now(),
					Adapter:     "test",
				})
			}

			if err := backend.Save(current); err != nil {
				errCh <- fmt.Errorf("writer %d save: %w", writerID, err)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent write error: %v", err)
	}

	// Verify final state is valid and contains all entries
	backend := state.NewLocalBackend(statePath)
	entries, err := backend.Load()
	if err != nil {
		t.Fatalf("final Load failed: %v", err)
	}

	expectedTotal := numWriters * entriesPerWriter
	if len(entries) != expectedTotal {
		t.Errorf("expected %d entries, got %d (data lost in concurrent writes)", expectedTotal, len(entries))
	}

	// Verify no duplicate FQNs
	fqns := make(map[string]bool)
	for _, e := range entries {
		if fqns[e.FQN] {
			t.Errorf("duplicate FQN: %s", e.FQN)
		}
		fqns[e.FQN] = true
	}
}

// TestMigrationLocalToLocal verifies state migration between two local backends.
func TestMigrationLocalToLocal(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.state.json")
	dstPath := filepath.Join(dir, "dst.state.json")

	src := state.NewLocalBackend(srcPath)
	dst := state.NewLocalBackend(dstPath)

	// Populate source
	entries := generateEntries(50)
	if err := src.Save(entries); err != nil {
		t.Fatalf("Save to source failed: %v", err)
	}

	// Migrate
	result, err := state.Migrate(src, dst, false)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if result.Migrated != 50 {
		t.Errorf("Migrated = %d, want 50", result.Migrated)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}

	// Verify destination has all entries
	dstEntries, err := dst.Load()
	if err != nil {
		t.Fatalf("Load from destination failed: %v", err)
	}
	if len(dstEntries) != 50 {
		t.Errorf("destination has %d entries, want 50", len(dstEntries))
	}

	// Verify source is unmodified
	srcEntries, err := src.Load()
	if err != nil {
		t.Fatalf("Load from source failed: %v", err)
	}
	if len(srcEntries) != 50 {
		t.Errorf("source has %d entries, want 50 (source was modified!)", len(srcEntries))
	}
}

// TestMigrationDryRun verifies dry-run mode doesn't write to destination.
func TestMigrationDryRun(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.state.json")
	dstPath := filepath.Join(dir, "dst.state.json")

	src := state.NewLocalBackend(srcPath)
	dst := state.NewLocalBackend(dstPath)

	// Populate source
	entries := generateEntries(10)
	if err := src.Save(entries); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Dry-run migrate
	result, err := state.Migrate(src, dst, true)
	if err != nil {
		t.Fatalf("Migrate dry-run failed: %v", err)
	}

	if result.Migrated != 10 {
		t.Errorf("Migrated = %d, want 10", result.Migrated)
	}

	// Verify destination is empty
	dstEntries, err := dst.Load()
	if err != nil {
		t.Fatalf("Load from destination failed: %v", err)
	}
	if len(dstEntries) != 0 {
		t.Errorf("destination has %d entries after dry-run, want 0", len(dstEntries))
	}
}
