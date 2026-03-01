package state

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestNewLocalBackend(t *testing.T) {
	path := "/tmp/test-state.json"
	b := NewLocalBackend(path)

	if b.Path != path {
		t.Errorf("Path = %q, want %q", b.Path, path)
	}

	defaults := DefaultLockConfig()
	if b.lockConfig.LockTimeout != defaults.LockTimeout {
		t.Errorf("LockTimeout = %v, want %v", b.lockConfig.LockTimeout, defaults.LockTimeout)
	}
	if b.lockConfig.StaleThreshold != defaults.StaleThreshold {
		t.Errorf("StaleThreshold = %v, want %v", b.lockConfig.StaleThreshold, defaults.StaleThreshold)
	}
}

func TestWithLockConfig(t *testing.T) {
	t.Run("set only LockTimeout", func(t *testing.T) {
		b := NewLocalBackend("/tmp/test.json")
		originalStale := b.lockConfig.StaleThreshold

		b.WithLockConfig(LockConfig{LockTimeout: 10 * time.Second})

		if b.lockConfig.LockTimeout != 10*time.Second {
			t.Errorf("LockTimeout = %v, want %v", b.lockConfig.LockTimeout, 10*time.Second)
		}
		if b.lockConfig.StaleThreshold != originalStale {
			t.Errorf("StaleThreshold changed to %v, want %v", b.lockConfig.StaleThreshold, originalStale)
		}
	})

	t.Run("set only StaleThreshold", func(t *testing.T) {
		b := NewLocalBackend("/tmp/test.json")
		originalTimeout := b.lockConfig.LockTimeout

		b.WithLockConfig(LockConfig{StaleThreshold: 10 * time.Minute})

		if b.lockConfig.StaleThreshold != 10*time.Minute {
			t.Errorf("StaleThreshold = %v, want %v", b.lockConfig.StaleThreshold, 10*time.Minute)
		}
		if b.lockConfig.LockTimeout != originalTimeout {
			t.Errorf("LockTimeout changed to %v, want %v", b.lockConfig.LockTimeout, originalTimeout)
		}
	})

	t.Run("set both", func(t *testing.T) {
		b := NewLocalBackend("/tmp/test.json")

		b.WithLockConfig(LockConfig{
			LockTimeout:    15 * time.Second,
			StaleThreshold: 8 * time.Minute,
		})

		if b.lockConfig.LockTimeout != 15*time.Second {
			t.Errorf("LockTimeout = %v, want %v", b.lockConfig.LockTimeout, 15*time.Second)
		}
		if b.lockConfig.StaleThreshold != 8*time.Minute {
			t.Errorf("StaleThreshold = %v, want %v", b.lockConfig.StaleThreshold, 8*time.Minute)
		}
	})

	t.Run("returns same backend for chaining", func(t *testing.T) {
		b := NewLocalBackend("/tmp/test.json")
		got := b.WithLockConfig(LockConfig{LockTimeout: 5 * time.Second})
		if got != b {
			t.Error("WithLockConfig did not return the same *LocalBackend")
		}
	})
}

func TestDefaultLockConfig(t *testing.T) {
	cfg := DefaultLockConfig()

	if cfg.LockTimeout != 30*time.Second {
		t.Errorf("LockTimeout = %v, want %v", cfg.LockTimeout, 30*time.Second)
	}
	if cfg.StaleThreshold != 5*time.Minute {
		t.Errorf("StaleThreshold = %v, want %v", cfg.StaleThreshold, 5*time.Minute)
	}
}

func TestLocalBackendSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	b := NewLocalBackend(statePath)

	now := time.Now().Truncate(time.Millisecond)
	entries := []Entry{
		{FQN: "z.resource", Hash: "aaa", Status: StatusApplied, LastApplied: now, Adapter: "adapter1"},
		{FQN: "a.resource", Hash: "bbb", Status: StatusFailed, LastApplied: now, Adapter: "adapter2", Error: "timeout"},
		{FQN: "m.resource", Hash: "ccc", Status: StatusApplied, LastApplied: now, Adapter: "adapter3"},
	}

	// First save
	if err := b.Save(entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := b.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("Load returned %d entries, want 3", len(loaded))
	}

	// Verify sorted by FQN
	if loaded[0].FQN != "a.resource" {
		t.Errorf("entries[0].FQN = %q, want %q", loaded[0].FQN, "a.resource")
	}
	if loaded[1].FQN != "m.resource" {
		t.Errorf("entries[1].FQN = %q, want %q", loaded[1].FQN, "m.resource")
	}
	if loaded[2].FQN != "z.resource" {
		t.Errorf("entries[2].FQN = %q, want %q", loaded[2].FQN, "z.resource")
	}

	// Verify entry data round-trips
	for _, e := range loaded {
		if e.Adapter == "" {
			t.Errorf("entry %q: Adapter is empty", e.FQN)
		}
		if e.Hash == "" {
			t.Errorf("entry %q: Hash is empty", e.FQN)
		}
	}

	// Second save should create .bak
	entries[0].Hash = "updated"
	if err := b.Save(entries); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	bakPath := statePath + ".bak"
	if _, err := os.Stat(bakPath); os.IsNotExist(err) {
		t.Error(".bak file was not created after second save")
	}

	// Verify backup contains the previous state
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	var bakState stateFile
	if err := json.Unmarshal(bakData, &bakState); err != nil {
		t.Fatalf("unmarshal backup: %v", err)
	}
	// The backup should have the original hash for the first entry (sorted: a.resource)
	for _, e := range bakState.Entries {
		if e.FQN == "a.resource" && e.Hash != "bbb" {
			t.Errorf("backup entry a.resource hash = %q, want %q", e.Hash, "bbb")
		}
	}
}

func TestLocalBackendGet(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	b := NewLocalBackend(statePath)

	now := time.Now()
	entries := []Entry{
		{FQN: "alpha", Hash: "h1", Status: StatusApplied, LastApplied: now, Adapter: "a1"},
		{FQN: "beta", Hash: "h2", Status: StatusFailed, LastApplied: now, Adapter: "a2"},
		{FQN: "gamma", Hash: "h3", Status: StatusApplied, LastApplied: now, Adapter: "a3"},
	}

	if err := b.Save(entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Get existing entry
	got, err := b.Get("beta")
	if err != nil {
		t.Fatalf("Get(beta): %v", err)
	}
	if got == nil {
		t.Fatal("Get(beta) returned nil, want entry")
	}
	if got.FQN != "beta" {
		t.Errorf("Get(beta).FQN = %q, want %q", got.FQN, "beta")
	}
	if got.Hash != "h2" {
		t.Errorf("Get(beta).Hash = %q, want %q", got.Hash, "h2")
	}
	if got.Status != StatusFailed {
		t.Errorf("Get(beta).Status = %q, want %q", got.Status, StatusFailed)
	}

	// Get non-existent entry
	got, err = b.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get(nonexistent): %v", err)
	}
	if got != nil {
		t.Errorf("Get(nonexistent) = %+v, want nil", got)
	}
}

func TestLocalBackendList(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	b := NewLocalBackend(statePath)

	now := time.Now()
	entries := []Entry{
		{FQN: "res1", Hash: "h1", Status: StatusApplied, LastApplied: now, Adapter: "a1"},
		{FQN: "res2", Hash: "h2", Status: StatusApplied, LastApplied: now, Adapter: "a2"},
		{FQN: "res3", Hash: "h3", Status: StatusFailed, LastApplied: now, Adapter: "a3", Error: "broken"},
	}

	if err := b.Save(entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// List all (nil status filter)
	all, err := b.List(nil)
	if err != nil {
		t.Fatalf("List(nil): %v", err)
	}
	if len(all) != 3 {
		t.Errorf("List(nil) returned %d entries, want 3", len(all))
	}

	// List applied only
	applied := StatusApplied
	appliedEntries, err := b.List(&applied)
	if err != nil {
		t.Fatalf("List(applied): %v", err)
	}
	if len(appliedEntries) != 2 {
		t.Errorf("List(applied) returned %d entries, want 2", len(appliedEntries))
	}
	for _, e := range appliedEntries {
		if e.Status != StatusApplied {
			t.Errorf("List(applied) returned entry with status %q", e.Status)
		}
	}

	// List failed only
	failed := StatusFailed
	failedEntries, err := b.List(&failed)
	if err != nil {
		t.Fatalf("List(failed): %v", err)
	}
	if len(failedEntries) != 1 {
		t.Errorf("List(failed) returned %d entries, want 1", len(failedEntries))
	}
	if failedEntries[0].FQN != "res3" {
		t.Errorf("List(failed)[0].FQN = %q, want %q", failedEntries[0].FQN, "res3")
	}
}

func TestLocalBackendLoadEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	b := NewLocalBackend(statePath)

	// Load from non-existent file should return empty slice and nil error
	entries, err := b.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if entries != nil {
		t.Errorf("Load returned %v, want nil", entries)
	}
}

func TestLocalBackendLoadCorrupted(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")

	// Write invalid JSON to the state file
	if err := os.WriteFile(statePath, []byte("{not valid json!!!"), 0644); err != nil {
		t.Fatalf("write corrupted file: %v", err)
	}

	b := NewLocalBackend(statePath)

	// No backup exists, so Load should return ErrStateCorrupted
	_, err := b.Load()
	if err == nil {
		t.Fatal("Load returned nil error for corrupted state file without backup")
	}

	var corrupted *ErrStateCorrupted
	if !errors.As(err, &corrupted) {
		t.Fatalf("error type = %T, want *ErrStateCorrupted", err)
	}
	if corrupted.Path != statePath {
		t.Errorf("ErrStateCorrupted.Path = %q, want %q", corrupted.Path, statePath)
	}
	if !corrupted.BackupUsed {
		t.Error("ErrStateCorrupted.BackupUsed = false, want true (backup was attempted)")
	}
}

func TestLocalBackendLoadCorruptedWithBackup(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	bakPath := statePath + ".bak"

	// Write invalid JSON to the state file
	if err := os.WriteFile(statePath, []byte("{corrupted}"), 0644); err != nil {
		t.Fatalf("write corrupted file: %v", err)
	}

	// Write valid backup
	now := time.Now()
	backupEntries := stateFile{
		Version: "1.0",
		Entries: []Entry{
			{FQN: "from.backup", Hash: "backup-hash", Status: StatusApplied, LastApplied: now, Adapter: "a1"},
		},
	}
	bakData, _ := json.Marshal(backupEntries)
	if err := os.WriteFile(bakPath, bakData, 0644); err != nil {
		t.Fatalf("write backup file: %v", err)
	}

	b := NewLocalBackend(statePath)

	// Load should recover from backup
	entries, err := b.Load()
	if err != nil {
		t.Fatalf("Load with valid backup: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Load returned %d entries, want 1", len(entries))
	}
	if entries[0].FQN != "from.backup" {
		t.Errorf("entries[0].FQN = %q, want %q", entries[0].FQN, "from.backup")
	}
}

func TestErrStateCorruptedError(t *testing.T) {
	inner := errors.New("unexpected EOF")

	t.Run("BackupUsed=false", func(t *testing.T) {
		e := &ErrStateCorrupted{
			Path:       "/tmp/state.json",
			BackupUsed: false,
			Err:        inner,
		}
		want := `state file "/tmp/state.json" is corrupted: unexpected EOF`
		if got := e.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("BackupUsed=true", func(t *testing.T) {
		e := &ErrStateCorrupted{
			Path:       "/tmp/state.json",
			BackupUsed: true,
			Err:        inner,
		}
		want := `state file "/tmp/state.json" and backup are both corrupted: unexpected EOF`
		if got := e.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		e := &ErrStateCorrupted{Path: "/tmp/state.json", Err: inner}
		if got := e.Unwrap(); got != inner {
			t.Errorf("Unwrap() = %v, want %v", got, inner)
		}
	})
}

func TestErrStateLockedError(t *testing.T) {
	lockedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	e := &ErrStateLocked{
		HolderPID: 12345,
		Hostname:  "build-host",
		LockedAt:  lockedAt,
	}

	got := e.Error()
	want := "state file is locked by PID 12345 on build-host since 2026-03-01T12:00:00Z"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Lock / Unlock tests
// ---------------------------------------------------------------------------

func TestLocalBackendLockUnlock(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	b := NewLocalBackend(statePath)

	// Lock should succeed on a fresh backend.
	if err := b.Lock(); err != nil {
		t.Fatalf("Lock: %v", err)
	}

	// Lock file should exist while held.
	lockPath := statePath + ".lock"
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file should exist while lock is held: %v", err)
	}

	// Unlock should succeed.
	if err := b.Unlock(); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	// Lock file should be cleaned up after unlock.
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Errorf("lock file should be removed after Unlock, got err=%v", err)
	}
}

func TestLocalBackendLockWithContext(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	b := NewLocalBackend(statePath)

	ctx := context.Background()
	if err := b.LockWithContext(ctx); err != nil {
		t.Fatalf("LockWithContext: %v", err)
	}
	if err := b.Unlock(); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
}

func TestLocalBackendDoubleLock(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")

	// First backend acquires the lock.
	b1 := NewLocalBackend(statePath)
	if err := b1.Lock(); err != nil {
		t.Fatalf("b1.Lock: %v", err)
	}

	// Second backend (different file descriptor) tries to lock the same path
	// with a very short timeout so it times out quickly.
	b2 := NewLocalBackend(statePath)
	b2.WithLockConfig(LockConfig{LockTimeout: 100 * time.Millisecond})

	var wg sync.WaitGroup
	var lockErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		lockErr = b2.Lock()
	}()
	wg.Wait()

	// The second lock attempt should fail because the first backend holds it.
	if lockErr == nil {
		t.Error("second Lock should have failed due to contention, but got nil")
		// Clean up in case it somehow succeeded.
		_ = b2.Unlock()
	}

	// Release the first lock.
	if err := b1.Unlock(); err != nil {
		t.Fatalf("b1.Unlock: %v", err)
	}
}

// ---------------------------------------------------------------------------
// isProcessAlive tests
// ---------------------------------------------------------------------------

func TestIsProcessAlive(t *testing.T) {
	// Current process should be alive.
	if !isProcessAlive(os.Getpid()) {
		t.Error("isProcessAlive(os.Getpid()) = false, want true")
	}

	// A very large PID that almost certainly does not exist.
	if isProcessAlive(999999999) {
		t.Error("isProcessAlive(999999999) = true, want false")
	}
}

// ---------------------------------------------------------------------------
// writeLockInfo / readLockInfo tests
// ---------------------------------------------------------------------------

func TestWriteAndReadLockInfo(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	f, err := os.Create(lockPath)
	if err != nil {
		t.Fatalf("create lock file: %v", err)
	}

	b := NewLocalBackend(filepath.Join(tmpDir, "state.json"))
	pid := os.Getpid()
	before := time.Now()
	b.writeLockInfo(f, pid)
	f.Close()

	info := b.readLockInfo(lockPath)
	if info == nil {
		t.Fatal("readLockInfo returned nil, want non-nil")
	}

	if info.PID != pid {
		t.Errorf("PID = %d, want %d", info.PID, pid)
	}
	if info.Hostname == "" {
		t.Error("Hostname is empty, want non-empty")
	}
	if info.Created.Before(before) {
		t.Errorf("Created = %v, want >= %v", info.Created, before)
	}
	if time.Since(info.Created) > 5*time.Second {
		t.Errorf("Created is too old: %v (more than 5s ago)", info.Created)
	}
}

func TestReadLockInfoMissing(t *testing.T) {
	b := NewLocalBackend("/tmp/nonexistent-state.json")
	info := b.readLockInfo("/tmp/this-lock-file-does-not-exist-12345.lock")
	if info != nil {
		t.Errorf("readLockInfo on missing file = %+v, want nil", info)
	}
}

func TestReadLockInfoInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "invalid.lock")

	if err := os.WriteFile(lockPath, []byte("{not valid json!!!}"), 0644); err != nil {
		t.Fatalf("write invalid lock file: %v", err)
	}

	b := NewLocalBackend(filepath.Join(tmpDir, "state.json"))
	info := b.readLockInfo(lockPath)
	if info != nil {
		t.Errorf("readLockInfo on invalid JSON = %+v, want nil", info)
	}
}

// ---------------------------------------------------------------------------
// Save coverage: backup creation and sorting
// ---------------------------------------------------------------------------

func TestLocalBackendSaveCreatesBackup(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	b := NewLocalBackend(statePath)
	bakPath := statePath + ".bak"
	now := time.Now().Truncate(time.Millisecond)

	// First save: entries with known hashes.
	firstEntries := []Entry{
		{FQN: "alpha", Hash: "first-hash", Status: StatusApplied, LastApplied: now, Adapter: "a1"},
	}
	if err := b.Save(firstEntries); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	// No backup yet after the very first save.
	if _, err := os.Stat(bakPath); !os.IsNotExist(err) {
		t.Fatalf("backup should not exist after first save, got err=%v", err)
	}

	// Second save: different entries.
	secondEntries := []Entry{
		{FQN: "alpha", Hash: "second-hash", Status: StatusApplied, LastApplied: now, Adapter: "a1"},
	}
	if err := b.Save(secondEntries); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	// Backup should now exist and contain the first save's data.
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	var bakState stateFile
	if err := json.Unmarshal(bakData, &bakState); err != nil {
		t.Fatalf("unmarshal backup: %v", err)
	}
	if len(bakState.Entries) != 1 {
		t.Fatalf("backup has %d entries, want 1", len(bakState.Entries))
	}
	if bakState.Entries[0].Hash != "first-hash" {
		t.Errorf("backup entry hash = %q, want %q", bakState.Entries[0].Hash, "first-hash")
	}
}

func TestLocalBackendSaveSortsEntries(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	b := NewLocalBackend(statePath)
	now := time.Now()

	// Save entries in reverse alphabetical order.
	entries := []Entry{
		{FQN: "z.thing", Hash: "hz", Status: StatusApplied, LastApplied: now, Adapter: "az"},
		{FQN: "b.thing", Hash: "hb", Status: StatusApplied, LastApplied: now, Adapter: "ab"},
		{FQN: "a.thing", Hash: "ha", Status: StatusApplied, LastApplied: now, Adapter: "aa"},
	}
	if err := b.Save(entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := b.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("Load returned %d entries, want 3", len(loaded))
	}

	expected := []string{"a.thing", "b.thing", "z.thing"}
	for i, want := range expected {
		if loaded[i].FQN != want {
			t.Errorf("loaded[%d].FQN = %q, want %q", i, loaded[i].FQN, want)
		}
	}
}

// ---------------------------------------------------------------------------
// loadFromBackup coverage
// ---------------------------------------------------------------------------

func TestLocalBackendLoadFromBackup(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	bakPath := statePath + ".bak"
	b := NewLocalBackend(statePath)
	now := time.Now().Truncate(time.Millisecond)

	// Create a state file with known content.
	entries := []Entry{
		{FQN: "recovered.resource", Hash: "rhash", Status: StatusApplied, LastApplied: now, Adapter: "ra"},
	}
	if err := b.Save(entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Simulate main file missing by renaming it to .bak.
	if err := os.Rename(statePath, bakPath); err != nil {
		t.Fatalf("rename to .bak: %v", err)
	}

	// Verify main file is gone.
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("state file should not exist after rename, got err=%v", err)
	}

	// Load should recover from backup.
	loaded, err := b.Load()
	if err != nil {
		t.Fatalf("Load from backup: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("Load returned %d entries, want 1", len(loaded))
	}
	if loaded[0].FQN != "recovered.resource" {
		t.Errorf("loaded[0].FQN = %q, want %q", loaded[0].FQN, "recovered.resource")
	}
	if loaded[0].Hash != "rhash" {
		t.Errorf("loaded[0].Hash = %q, want %q", loaded[0].Hash, "rhash")
	}

	// Main file should be restored after loading from backup.
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("state file should be restored after loading from backup")
	}
}

// ---------------------------------------------------------------------------
// LockWithContext: stale lock detection — dead process
// ---------------------------------------------------------------------------

func TestLocalBackendLockWithContext_StaleLockDeadProcess(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	lockPath := statePath + ".lock"

	// Create a lock file manually with a PID that does not exist.
	deadPID := 999999999
	info := lockInfo{
		PID:      deadPID,
		Created:  time.Now(),
		Hostname: "test-host",
	}
	infoData, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal lockInfo: %v", err)
	}

	// Write the lock file and acquire an flock on it so LockWithContext sees contention.
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("create lock file: %v", err)
	}
	if _, err := f.Write(infoData); err != nil {
		t.Fatalf("write lock info: %v", err)
	}
	_ = f.Sync()

	// Hold an flock so the first non-blocking attempt fails.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("flock: %v", err)
	}
	// Release the flock but keep the file so readLockInfo can read the dead PID.
	// The code removes the lock file and retries; on retry it should succeed.
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()

	// Now create a backend and try to acquire the lock.
	// Since the PID is dead, the code should break the stale lock and succeed.
	b := NewLocalBackend(statePath)
	b.WithLockConfig(LockConfig{LockTimeout: 2 * time.Second})

	ctx := context.Background()
	if err := b.LockWithContext(ctx); err != nil {
		t.Fatalf("LockWithContext should succeed after breaking stale lock (dead process): %v", err)
	}

	// Verify lock is held by checking lock file contains our PID.
	readInfo := b.readLockInfo(lockPath)
	if readInfo == nil {
		t.Fatal("readLockInfo returned nil after lock acquisition")
	}
	if readInfo.PID != os.Getpid() {
		t.Errorf("lock PID = %d, want %d", readInfo.PID, os.Getpid())
	}

	if err := b.Unlock(); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
}

// ---------------------------------------------------------------------------
// LockWithContext: stale lock detection — age exceeded
// ---------------------------------------------------------------------------

func TestLocalBackendLockWithContext_StaleLockAgeExceeded(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	lockPath := statePath + ".lock"

	// Create a lock file with our own PID (alive) but with a creation time far in the past.
	info := lockInfo{
		PID:      os.Getpid(),
		Created:  time.Now().Add(-1 * time.Hour),
		Hostname: "test-host",
	}
	infoData, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal lockInfo: %v", err)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("create lock file: %v", err)
	}
	if _, err := f.Write(infoData); err != nil {
		t.Fatalf("write lock info: %v", err)
	}
	_ = f.Sync()

	// Hold an flock so the non-blocking attempt fails, then release so retry succeeds.
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("flock: %v", err)
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	_ = f.Close()

	// Use a very short StaleThreshold so the age check triggers.
	b := NewLocalBackend(statePath)
	b.WithLockConfig(LockConfig{
		LockTimeout:    2 * time.Second,
		StaleThreshold: 1 * time.Millisecond,
	})

	ctx := context.Background()
	if err := b.LockWithContext(ctx); err != nil {
		t.Fatalf("LockWithContext should succeed after breaking stale lock (age exceeded): %v", err)
	}

	if err := b.Unlock(); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
}

// ---------------------------------------------------------------------------
// LockWithContext: context cancelled
// ---------------------------------------------------------------------------

func TestLocalBackendLockWithContext_ContextCancelled(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")

	// First backend holds the lock.
	b1 := NewLocalBackend(statePath)
	if err := b1.Lock(); err != nil {
		t.Fatalf("b1.Lock: %v", err)
	}
	defer func() { _ = b1.Unlock() }()

	// Second backend tries to lock with a very short context deadline.
	b2 := NewLocalBackend(statePath)
	b2.WithLockConfig(LockConfig{LockTimeout: 5 * time.Second}) // longer than context

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	err := b2.LockWithContext(ctx)
	if err == nil {
		_ = b2.Unlock()
		t.Fatal("LockWithContext should have failed due to context cancellation")
	}

	// Should get either ErrStateLocked or a context error.
	var locked *ErrStateLocked
	if !errors.As(err, &locked) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("unexpected error type: %T (%v), want *ErrStateLocked or context.DeadlineExceeded", err, err)
	}
}

// ---------------------------------------------------------------------------
// Save: atomic write verification
// ---------------------------------------------------------------------------

func TestLocalBackendSaveAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	b := NewLocalBackend(statePath)

	now := time.Now()
	entries := []Entry{
		{FQN: "atomic.test", Hash: "ahash", Status: StatusApplied, LastApplied: now, Adapter: "a1"},
	}

	if err := b.Save(entries); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify the state file exists and contains valid JSON.
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	var sf stateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		t.Fatalf("state file is not valid JSON: %v", err)
	}
	if len(sf.Entries) != 1 || sf.Entries[0].FQN != "atomic.test" {
		t.Errorf("state file entries = %+v, want single entry with FQN=atomic.test", sf.Entries)
	}

	// Verify no temp files are left behind.
	dirEntries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, de := range dirEntries {
		if strings.HasSuffix(de.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", de.Name())
		}
	}
}

// ---------------------------------------------------------------------------
// Save: read-only directory error path
// ---------------------------------------------------------------------------

func TestLocalBackendSaveToReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not effective on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("test cannot run as root (chmod restrictions bypassed)")
	}

	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer func() { _ = os.Chmod(readOnlyDir, 0755) }()

	statePath := filepath.Join(readOnlyDir, ".agentspec.state.json")
	b := NewLocalBackend(statePath)

	now := time.Now()
	entries := []Entry{
		{FQN: "readonly.test", Hash: "rhash", Status: StatusApplied, LastApplied: now, Adapter: "a1"},
	}

	err := b.Save(entries)
	if err == nil {
		t.Fatal("Save to read-only dir should return error")
	}
	if !strings.Contains(err.Error(), "create temp file") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "create temp file")
	}
}

// ---------------------------------------------------------------------------
// recoverFromBackup: valid backup restores main file
// ---------------------------------------------------------------------------

func TestLocalBackendRecoverFromBackup_ValidBackup(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	bakPath := statePath + ".bak"

	// Write invalid JSON to the main state file.
	if err := os.WriteFile(statePath, []byte("{corrupted!!!}"), 0644); err != nil {
		t.Fatalf("write corrupted file: %v", err)
	}

	// Write a valid backup.
	now := time.Now().Truncate(time.Millisecond)
	backup := stateFile{
		Version: "1.0",
		Entries: []Entry{
			{FQN: "recovered.resource", Hash: "rhash", Status: StatusApplied, LastApplied: now, Adapter: "a1"},
		},
	}
	bakData, _ := json.Marshal(backup)
	if err := os.WriteFile(bakPath, bakData, 0644); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	b := NewLocalBackend(statePath)
	entries, err := b.Load()
	if err != nil {
		t.Fatalf("Load should recover from backup: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Load returned %d entries, want 1", len(entries))
	}
	if entries[0].FQN != "recovered.resource" {
		t.Errorf("entries[0].FQN = %q, want %q", entries[0].FQN, "recovered.resource")
	}

	// Verify the main file was restored from the backup.
	restoredData, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read restored state file: %v", err)
	}
	var sf stateFile
	if err := json.Unmarshal(restoredData, &sf); err != nil {
		t.Fatalf("restored state file is not valid JSON: %v", err)
	}
	if len(sf.Entries) != 1 || sf.Entries[0].FQN != "recovered.resource" {
		t.Errorf("restored state = %+v, want single entry with FQN=recovered.resource", sf.Entries)
	}
}

// ---------------------------------------------------------------------------
// recoverFromBackup: both main and backup are corrupted
// ---------------------------------------------------------------------------

func TestLocalBackendRecoverFromBackup_BothCorrupted(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	bakPath := statePath + ".bak"

	// Write invalid JSON to both the main state file and the backup.
	if err := os.WriteFile(statePath, []byte("{main corrupted!!!}"), 0644); err != nil {
		t.Fatalf("write corrupted main: %v", err)
	}
	if err := os.WriteFile(bakPath, []byte("{backup corrupted!!!}"), 0644); err != nil {
		t.Fatalf("write corrupted backup: %v", err)
	}

	b := NewLocalBackend(statePath)
	_, err := b.Load()
	if err == nil {
		t.Fatal("Load should return error when both files are corrupted")
	}

	var corrupted *ErrStateCorrupted
	if !errors.As(err, &corrupted) {
		t.Fatalf("error type = %T, want *ErrStateCorrupted", err)
	}
	if !corrupted.BackupUsed {
		t.Error("ErrStateCorrupted.BackupUsed = false, want true")
	}
	if corrupted.Path != statePath {
		t.Errorf("ErrStateCorrupted.Path = %q, want %q", corrupted.Path, statePath)
	}
}

// ---------------------------------------------------------------------------
// loadFromBackup: backup file is corrupted (no main file)
// ---------------------------------------------------------------------------

func TestLocalBackendLoadFromBackup_BackupError(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")
	bakPath := statePath + ".bak"

	// No main file exists. Write invalid JSON to the backup.
	if err := os.WriteFile(bakPath, []byte("{backup corrupted!!!}"), 0644); err != nil {
		t.Fatalf("write corrupted backup: %v", err)
	}

	b := NewLocalBackend(statePath)
	_, err := b.Load()
	if err == nil {
		t.Fatal("Load should return error when backup is corrupted and main file is missing")
	}

	var corrupted *ErrStateCorrupted
	if !errors.As(err, &corrupted) {
		t.Fatalf("error type = %T, want *ErrStateCorrupted", err)
	}
	if !corrupted.BackupUsed {
		t.Error("ErrStateCorrupted.BackupUsed = false, want true")
	}
}
