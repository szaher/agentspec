package integration_tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/szaher/designs/agentz/internal/state"
)

func TestStateFileLocking(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "test.state.json")

	backend := state.NewLocalBackend(stateFile)

	// First lock should succeed
	if err := backend.Lock(); err != nil {
		t.Fatalf("first lock failed: %v", err)
	}

	// Second lock from same process should fail (LOCK_NB)
	backend2 := state.NewLocalBackend(stateFile)
	err := backend2.Lock()
	if err == nil {
		t.Fatal("expected second lock to fail")
		_ = backend2.Unlock()
	}

	// Unlock should succeed
	if err := backend.Unlock(); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}

	// Lock again after unlock should succeed
	if err := backend2.Lock(); err != nil {
		t.Fatalf("lock after unlock failed: %v", err)
	}
	_ = backend2.Unlock()
}

func TestStateIdempotentSave(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "idempotent.state.json")

	backend := state.NewLocalBackend(stateFile)

	// First save
	entries := []state.Entry{
		{FQN: "pkg/agent/helper", Hash: "abc123", Status: state.StatusApplied, Adapter: "process"},
		{FQN: "pkg/prompt/system", Hash: "def456", Status: state.StatusApplied, Adapter: "process"},
	}
	if err := backend.Save(entries); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	// Load and verify
	loaded, err := backend.Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded))
	}

	// Save same entries again (idempotent)
	if err := backend.Save(entries); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	// Load and verify same entries
	loaded2, err := backend.Load()
	if err != nil {
		t.Fatalf("second load failed: %v", err)
	}
	if len(loaded2) != 2 {
		t.Fatalf("expected 2 entries after idempotent save, got %d", len(loaded2))
	}

	// Entries should be sorted by FQN
	if loaded2[0].FQN != "pkg/agent/helper" {
		t.Errorf("expected sorted first entry 'pkg/agent/helper', got %q", loaded2[0].FQN)
	}
	if loaded2[1].FQN != "pkg/prompt/system" {
		t.Errorf("expected sorted second entry 'pkg/prompt/system', got %q", loaded2[1].FQN)
	}
}

func TestStateGetEntry(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "get.state.json")

	backend := state.NewLocalBackend(stateFile)

	entries := []state.Entry{
		{FQN: "pkg/agent/helper", Hash: "abc123", Status: state.StatusApplied},
	}
	if err := backend.Save(entries); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	entry, err := backend.Get("pkg/agent/helper")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}
	if entry.Hash != "abc123" {
		t.Errorf("expected hash 'abc123', got %q", entry.Hash)
	}

	// Get non-existent entry
	missing, err := backend.Get("pkg/agent/missing")
	if err != nil {
		t.Fatalf("get missing: %v", err)
	}
	if missing != nil {
		t.Error("expected nil for missing entry")
	}
}

func TestStateFileNotExist(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "nonexistent.state.json")

	backend := state.NewLocalBackend(stateFile)

	// Load from non-existent file should return empty
	entries, err := backend.Load()
	if err != nil {
		t.Fatalf("load non-existent: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestStateLockFileCleanup(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "cleanup.state.json")

	backend := state.NewLocalBackend(stateFile)
	if err := backend.Lock(); err != nil {
		t.Fatalf("lock failed: %v", err)
	}

	lockPath := stateFile + ".lock"
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("lock file should exist while locked")
	}

	if err := backend.Unlock(); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}

	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("lock file should be cleaned up after unlock")
	}
}
