package integration_tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/szaher/designs/agentz/internal/state"
)

func testEntries() []state.Entry {
	return []state.Entry{
		{
			FQN:         "agent/test-agent",
			Hash:        "abc123",
			Status:      state.StatusApplied,
			LastApplied: time.Now().Truncate(time.Second),
			Adapter:     "process",
		},
		{
			FQN:         "tool/test-tool",
			Hash:        "def456",
			Status:      state.StatusApplied,
			LastApplied: time.Now().Truncate(time.Second),
			Adapter:     "process",
		},
	}
}

func TestStateAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "test.state.json")
	backend := state.NewLocalBackend(statePath)

	entries := testEntries()
	if err := backend.Save(entries); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify state file exists and contains valid JSON
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("state file is not valid JSON: %v", err)
	}

	// Verify entries are sorted by FQN
	loaded, err := backend.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded))
	}
	if loaded[0].FQN != "agent/test-agent" {
		t.Errorf("entries not sorted: first FQN = %q", loaded[0].FQN)
	}
}

func TestStateBackupCreation(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "test.state.json")
	bakPath := statePath + ".bak"
	backend := state.NewLocalBackend(statePath)

	// First save — no backup expected (no previous file)
	entries := testEntries()
	if err := backend.Save(entries); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}
	if _, err := os.Stat(bakPath); !os.IsNotExist(err) {
		t.Fatal("backup should not exist after first save")
	}

	// Second save — backup should be created
	entries[0].Hash = "updated123"
	if err := backend.Save(entries); err != nil {
		t.Fatalf("second Save failed: %v", err)
	}

	if _, err := os.Stat(bakPath); err != nil {
		t.Fatalf("backup file should exist after second save: %v", err)
	}

	// Verify backup contains valid JSON
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("ReadFile backup failed: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(bakData, &raw); err != nil {
		t.Fatalf("backup is not valid JSON: %v", err)
	}
}

func TestStateCorruptionRecovery(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "test.state.json")
	bakPath := statePath + ".bak"
	backend := state.NewLocalBackend(statePath)

	// Save valid state twice to create a backup
	entries := testEntries()
	if err := backend.Save(entries); err != nil {
		t.Fatalf("first Save failed: %v", err)
	}
	entries[0].Hash = "second-save"
	if err := backend.Save(entries); err != nil {
		t.Fatalf("second Save failed: %v", err)
	}

	// Verify backup exists
	if _, err := os.Stat(bakPath); err != nil {
		t.Fatalf("backup should exist: %v", err)
	}

	// Corrupt the main state file
	if err := os.WriteFile(statePath, []byte("corrupted{"), 0644); err != nil {
		t.Fatalf("failed to corrupt state file: %v", err)
	}

	// Load should detect corruption and fall back to backup
	loaded, err := backend.Load()
	if err != nil {
		t.Fatalf("Load should succeed via backup fallback, got error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 entries from backup, got %d", len(loaded))
	}

	// Verify the state file was restored from backup
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("ReadFile state failed: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("restored state file should be valid JSON: %v", err)
	}
}

func TestStateBothCorruptedError(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "test.state.json")
	bakPath := statePath + ".bak"
	backend := state.NewLocalBackend(statePath)

	// Create corrupted state and corrupted backup
	if err := os.WriteFile(statePath, []byte("corrupted{"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bakPath, []byte("also-corrupted{"), 0644); err != nil {
		t.Fatal(err)
	}

	// Load should return ErrStateCorrupted
	_, err := backend.Load()
	if err == nil {
		t.Fatal("expected error when both state and backup are corrupted")
	}
	var corruptErr *state.ErrStateCorrupted
	if !errors.As(err, &corruptErr) {
		t.Fatalf("expected ErrStateCorrupted, got %T: %v", err, err)
	}
	if !corruptErr.BackupUsed {
		t.Error("expected BackupUsed=true")
	}
}

func TestStateDeletedDuringRuntime(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "test.state.json")
	backend := state.NewLocalBackend(statePath)

	// Save creates the file
	entries := testEntries()
	if err := backend.Save(entries); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Delete the state file
	if err := os.Remove(statePath); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Next Save should create a new file (no backup to rename, so just creates)
	newEntries := []state.Entry{entries[0]}
	if err := backend.Save(newEntries); err != nil {
		t.Fatalf("Save after delete failed: %v", err)
	}

	// Verify new file exists and is valid
	loaded, err := backend.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded))
	}
}

func TestStateNoTempFileLeftOnError(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "test.state.json")
	backend := state.NewLocalBackend(statePath)

	// Normal save should not leave temp files
	entries := testEntries()
	if err := backend.Save(entries); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Check no .tmp files remain
	matches, _ := filepath.Glob(filepath.Join(dir, ".state-*.tmp"))
	if len(matches) > 0 {
		t.Errorf("temp files left behind: %v", matches)
	}
}

func TestStateLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "nonexistent.state.json")
	backend := state.NewLocalBackend(statePath)

	// Load on missing file should return nil, nil (fresh start)
	entries, err := backend.Load()
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
	if entries != nil {
		t.Errorf("expected nil entries for missing file, got %d entries", len(entries))
	}
}

// --- US2: Concurrent Apply Serialization Tests ---

func TestConcurrentLockWait(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "test.state.json")

	backend1 := state.NewLocalBackend(statePath)
	backend2 := state.NewLocalBackend(statePath).WithLockConfig(state.LockConfig{
		LockTimeout: 5 * time.Second,
	})

	ctx := context.Background()

	// First lock should succeed
	if err := backend1.LockWithContext(ctx); err != nil {
		t.Fatalf("first lock failed: %v", err)
	}

	// Second lock should wait and then succeed after first unlocks
	done := make(chan error, 1)
	go func() {
		done <- backend2.LockWithContext(ctx)
	}()

	// Release first lock after a short delay
	time.Sleep(200 * time.Millisecond)
	if err := backend1.Unlock(); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second lock should have succeeded after first unlock: %v", err)
		}
		_ = backend2.Unlock()
	case <-time.After(5 * time.Second):
		t.Fatal("second lock timed out")
	}
}

func TestConcurrentLockTimeout(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "test.state.json")

	backend1 := state.NewLocalBackend(statePath)
	backend2 := state.NewLocalBackend(statePath).WithLockConfig(state.LockConfig{
		LockTimeout: 500 * time.Millisecond,
	})

	ctx := context.Background()

	// Acquire first lock
	if err := backend1.LockWithContext(ctx); err != nil {
		t.Fatalf("first lock failed: %v", err)
	}
	defer func() { _ = backend1.Unlock() }()

	// Second lock should time out
	err := backend2.LockWithContext(ctx)
	if err == nil {
		t.Fatal("expected timeout error")
		_ = backend2.Unlock()
	}

	var lockedErr *state.ErrStateLocked
	if !errors.As(err, &lockedErr) {
		t.Fatalf("expected ErrStateLocked, got %T: %v", err, err)
	}
}

func TestStaleLockDetection(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "test.state.json")
	lockPath := statePath + ".lock"

	// Create a lock file with a dead PID
	lockData := fmt.Sprintf(`{"pid": 99999, "created": %q, "hostname": "test-host"}`,
		time.Now().Add(-10*time.Minute).Format(time.RFC3339))
	if err := os.WriteFile(lockPath, []byte(lockData), 0644); err != nil {
		t.Fatalf("failed to write stale lock: %v", err)
	}

	backend := state.NewLocalBackend(statePath).WithLockConfig(state.LockConfig{
		LockTimeout:    2 * time.Second,
		StaleThreshold: 5 * time.Minute,
	})

	ctx := context.Background()

	// Should detect stale lock (dead PID) and acquire
	err := backend.LockWithContext(ctx)
	if err != nil {
		t.Fatalf("should have broken stale lock: %v", err)
	}
	defer func() { _ = backend.Unlock() }()
}

func TestConcurrentSavesWithLocking(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "test.state.json")

	const numWorkers = 10
	var wg sync.WaitGroup
	errCh := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			backend := state.NewLocalBackend(statePath).WithLockConfig(state.LockConfig{
				LockTimeout: 30 * time.Second,
			})

			ctx := context.Background()

			if err := backend.LockWithContext(ctx); err != nil {
				errCh <- fmt.Errorf("worker %d lock: %w", id, err)
				return
			}
			defer func() { _ = backend.Unlock() }()

			// Load, modify, save
			entries, err := backend.Load()
			if err != nil {
				errCh <- fmt.Errorf("worker %d load: %w", id, err)
				return
			}

			entries = append(entries, state.Entry{
				FQN:         fmt.Sprintf("resource/worker-%d", id),
				Hash:        fmt.Sprintf("hash-%d", id),
				Status:      state.StatusApplied,
				LastApplied: time.Now(),
				Adapter:     "test",
			})

			if err := backend.Save(entries); err != nil {
				errCh <- fmt.Errorf("worker %d save: %w", id, err)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent save error: %v", err)
	}

	// Verify all resources are tracked
	backend := state.NewLocalBackend(statePath)
	entries, err := backend.Load()
	if err != nil {
		t.Fatalf("final Load failed: %v", err)
	}
	if len(entries) != numWorkers {
		t.Errorf("expected %d entries, got %d", numWorkers, len(entries))
	}

	// Verify state file is valid JSON
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("state file is not valid JSON after concurrent saves: %v", err)
	}
}

func TestLockInfoPersisted(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "test.state.json")
	lockPath := statePath + ".lock"

	backend := state.NewLocalBackend(statePath)
	ctx := context.Background()

	if err := backend.LockWithContext(ctx); err != nil {
		t.Fatalf("lock failed: %v", err)
	}

	// Verify lock file contains valid JSON with PID
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ReadFile lock failed: %v", err)
	}

	var info struct {
		PID      int    `json:"pid"`
		Hostname string `json:"hostname"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		t.Fatalf("lock file is not valid JSON: %v", err)
	}
	if info.PID != os.Getpid() {
		t.Errorf("lock PID = %d, want %d", info.PID, os.Getpid())
	}

	_ = backend.Unlock()
}
