package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

// ErrStateCorrupted is returned when the state file and its backup are both corrupted.
type ErrStateCorrupted struct {
	Path       string
	BackupUsed bool
	Err        error
}

func (e *ErrStateCorrupted) Error() string {
	if e.BackupUsed {
		return fmt.Sprintf("state file %q and backup are both corrupted: %v", e.Path, e.Err)
	}
	return fmt.Sprintf("state file %q is corrupted: %v", e.Path, e.Err)
}

func (e *ErrStateCorrupted) Unwrap() error { return e.Err }

// ErrStateLocked is returned when a lock cannot be acquired within the timeout.
type ErrStateLocked struct {
	HolderPID int
	Hostname  string
	LockedAt  time.Time
}

func (e *ErrStateLocked) Error() string {
	return fmt.Sprintf("state file is locked by PID %d on %s since %s",
		e.HolderPID, e.Hostname, e.LockedAt.Format(time.RFC3339))
}

// LockConfig configures lock behavior.
type LockConfig struct {
	LockTimeout    time.Duration // How long to wait for a lock (default 30s)
	StaleThreshold time.Duration // Age after which a lock is considered stale (default 5m)
}

// DefaultLockConfig returns the default lock configuration.
func DefaultLockConfig() LockConfig {
	return LockConfig{
		LockTimeout:    30 * time.Second,
		StaleThreshold: 5 * time.Minute,
	}
}

// lockInfo is the JSON content written to the lock file for stale detection.
type lockInfo struct {
	PID      int       `json:"pid"`
	Created  time.Time `json:"created"`
	Hostname string    `json:"hostname"`
}

// LocalBackend implements Backend using a local JSON file.
type LocalBackend struct {
	Path       string
	lockFile   *os.File
	lockConfig LockConfig
	lockTime   time.Time // when lock was acquired, for held duration logging
}

// NewLocalBackend creates a new local JSON state backend.
func NewLocalBackend(path string) *LocalBackend {
	return &LocalBackend{
		Path:       path,
		lockConfig: DefaultLockConfig(),
	}
}

// WithLockConfig sets the lock configuration.
func (b *LocalBackend) WithLockConfig(cfg LockConfig) *LocalBackend {
	if cfg.LockTimeout > 0 {
		b.lockConfig.LockTimeout = cfg.LockTimeout
	}
	if cfg.StaleThreshold > 0 {
		b.lockConfig.StaleThreshold = cfg.StaleThreshold
	}
	return b
}

// stateFile is the on-disk JSON structure.
type stateFile struct {
	Version string  `json:"version"`
	Entries []Entry `json:"entries"`
}

// Load reads all state entries from the JSON file.
// If the state file is corrupted, it attempts recovery from the .bak backup.
func (b *LocalBackend) Load() ([]Entry, error) {
	data, err := os.ReadFile(b.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Try backup if main file doesn't exist
			return b.loadFromBackup()
		}
		return nil, err
	}

	var sf stateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		slog.Error("state file corrupted", "path", b.Path, "json_error", err)
		return b.recoverFromBackup(err)
	}
	return sf.Entries, nil
}

// loadFromBackup attempts to load from the .bak file when the main file is missing.
func (b *LocalBackend) loadFromBackup() ([]Entry, error) {
	bakPath := b.Path + ".bak"
	data, err := os.ReadFile(bakPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil // No state file and no backup — fresh start
		}
		return nil, err
	}

	var sf stateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, &ErrStateCorrupted{Path: b.Path, BackupUsed: true, Err: err}
	}

	// Restore backup to main path
	slog.Error("state file missing, restored from backup", "path", b.Path, "backup_path", bakPath)
	if writeErr := os.WriteFile(b.Path, data, 0644); writeErr != nil {
		slog.Error("failed to restore backup to state path", "path", b.Path, "error", writeErr)
	}
	return sf.Entries, nil
}

// recoverFromBackup attempts to load from the .bak file after detecting corruption.
func (b *LocalBackend) recoverFromBackup(originalErr error) ([]Entry, error) {
	bakPath := b.Path + ".bak"
	data, err := os.ReadFile(bakPath)
	if err != nil {
		slog.Error("backup also unavailable", "path", b.Path, "backup_path", bakPath)
		return nil, &ErrStateCorrupted{Path: b.Path, BackupUsed: true, Err: originalErr}
	}

	var sf stateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		slog.Error("backup also corrupted", "path", b.Path, "backup_path", bakPath, "json_error", err)
		return nil, &ErrStateCorrupted{Path: b.Path, BackupUsed: true, Err: originalErr}
	}

	// Restore backup to main path
	slog.Error("state file corrupted, falling back to backup", "path", b.Path, "backup_path", bakPath)
	if writeErr := os.WriteFile(b.Path, data, 0644); writeErr != nil {
		slog.Error("failed to restore backup to state path", "path", b.Path, "error", writeErr)
	}
	return sf.Entries, nil
}

// Save writes all state entries atomically using temp-file → fsync → rename.
// Creates a .bak backup of the previous state before replacing.
func (b *LocalBackend) Save(entries []Entry) error {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].FQN < entries[j].FQN
	})
	sf := stateFile{
		Version: "1.0",
		Entries: entries,
	}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(b.Path)

	// Step 1: Write to temp file in same directory
	tmp, err := os.CreateTemp(dir, ".state-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Clean up temp file on any error
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup
		}
	}()

	// Step 2: Write data and fsync
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("fsync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Step 3: Backup current state file (if it exists)
	bakPath := b.Path + ".bak"
	if _, err := os.Stat(b.Path); err == nil {
		if err := os.Rename(b.Path, bakPath); err != nil {
			return fmt.Errorf("create backup: %w", err)
		}
		slog.Info("state backup created", "path", b.Path, "backup_path", bakPath)
	}

	// Step 4: Rename temp to state (atomic)
	if err := os.Rename(tmpPath, b.Path); err != nil {
		// Try to restore backup
		if restoreErr := os.Rename(bakPath, b.Path); restoreErr != nil {
			slog.Error("failed to restore backup after rename failure", "path", b.Path, "error", restoreErr)
		}
		return fmt.Errorf("rename temp to state: %w", err)
	}

	success = true
	return nil
}

// Get retrieves a single entry by FQN.
func (b *LocalBackend) Get(fqn string) (*Entry, error) {
	entries, err := b.Load()
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].FQN == fqn {
			return &entries[i], nil
		}
	}
	return nil, nil
}

// List returns all entries, optionally filtered by status.
func (b *LocalBackend) List(status *Status) ([]Entry, error) {
	entries, err := b.Load()
	if err != nil {
		return nil, err
	}
	if status == nil {
		return entries, nil
	}
	var filtered []Entry
	for _, e := range entries {
		if e.Status == *status {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}

// LockWithContext acquires an exclusive file lock with context support and stale detection.
func (b *LocalBackend) LockWithContext(ctx context.Context) error {
	lockPath := b.Path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}

	pid := os.Getpid()

	// Try non-blocking exclusive lock first
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		// Lock acquired immediately
		b.lockFile = f
		b.lockTime = time.Now()
		b.writeLockInfo(f, pid)
		slog.Info("lock acquired", "path", b.Path, "pid", pid)
		return nil
	}

	// Lock is held — check for stale lock
	holder := b.readLockInfo(lockPath)
	if holder != nil {
		slog.Info("lock wait started", "path", b.Path, "holder_pid", holder.PID, "holder_hostname", holder.Hostname)

		// Check if holder PID is dead
		if holder.PID > 0 && !isProcessAlive(holder.PID) {
			slog.Warn("stale lock broken (dead process)", "path", b.Path, "stale_pid", holder.PID, "stale_age", time.Since(holder.Created), "stale_hostname", holder.Hostname)
			_ = f.Close()
			_ = os.Remove(lockPath)
			return b.LockWithContext(ctx) // retry
		}

		// Check if lock age exceeds stale threshold
		if !holder.Created.IsZero() && time.Since(holder.Created) > b.lockConfig.StaleThreshold {
			slog.Warn("stale lock broken (age exceeded threshold)", "path", b.Path, "stale_pid", holder.PID, "stale_age", time.Since(holder.Created), "stale_hostname", holder.Hostname)
			_ = f.Close()
			_ = os.Remove(lockPath)
			return b.LockWithContext(ctx) // retry
		}
	}

	// Wait with timeout
	_ = f.Close()

	deadline := time.Now().Add(b.lockConfig.LockTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Error("lock timeout (context cancelled)", "path", b.Path, "wait_duration", b.lockConfig.LockTimeout)
			if holder != nil {
				return &ErrStateLocked{HolderPID: holder.PID, Hostname: holder.Hostname, LockedAt: holder.Created}
			}
			return fmt.Errorf("lock acquisition cancelled: %w", ctx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				slog.Error("lock timeout", "path", b.Path, "wait_duration", b.lockConfig.LockTimeout)
				if holder != nil {
					return &ErrStateLocked{HolderPID: holder.PID, Hostname: holder.Hostname, LockedAt: holder.Created}
				}
				return fmt.Errorf("lock acquisition timed out after %s", b.lockConfig.LockTimeout)
			}

			f2, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				continue
			}
			err = syscall.Flock(int(f2.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			if err == nil {
				b.lockFile = f2
				b.lockTime = time.Now()
				b.writeLockInfo(f2, pid)
				slog.Info("lock acquired", "path", b.Path, "pid", pid)
				return nil
			}
			_ = f2.Close()
		}
	}
}

// Lock acquires an exclusive file lock (backward-compatible, no context).
func (b *LocalBackend) Lock() error {
	return b.LockWithContext(context.Background())
}

// Unlock releases the file lock.
func (b *LocalBackend) Unlock() error {
	if b.lockFile == nil {
		return nil
	}

	heldDuration := time.Since(b.lockTime)
	pid := os.Getpid()

	err := syscall.Flock(int(b.lockFile.Fd()), syscall.LOCK_UN)
	_ = b.lockFile.Close()
	b.lockFile = nil
	_ = os.Remove(b.Path + ".lock")

	slog.Info("lock released", "path", b.Path, "pid", pid, "held_duration", heldDuration)
	return err
}

// writeLockInfo writes PID, timestamp, and hostname to the lock file.
func (b *LocalBackend) writeLockInfo(f *os.File, pid int) {
	hostname, _ := os.Hostname()
	info := lockInfo{
		PID:      pid,
		Created:  time.Now(),
		Hostname: hostname,
	}
	data, err := json.Marshal(info)
	if err != nil {
		return
	}
	_ = f.Truncate(0)
	_, _ = f.Seek(0, 0)
	_, _ = f.Write(data)
	_ = f.Sync()
}

// readLockInfo reads lock holder info from the lock file.
func (b *LocalBackend) readLockInfo(lockPath string) *lockInfo {
	data, err := os.ReadFile(lockPath)
	if err != nil || len(data) == 0 {
		return nil
	}
	var info lockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil
	}
	return &info
}

// isProcessAlive checks if a process with the given PID is still running.
func isProcessAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
