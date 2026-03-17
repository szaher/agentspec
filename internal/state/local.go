package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"sync/atomic"
	"syscall"
	"time"
)

// LocalBackend implements Backend using a local JSON file.
type LocalBackend struct {
	Path     string
	lockFile *os.File

	// Cache fields (T020)
	cachedEntries []Entry           // cached copy of all entries
	index         map[string]*Entry // FQN → Entry pointer for O(1) lookup
	cacheModTime  time.Time         // file mtime when cache was populated
	hits          uint64            // cache hit counter
	misses        uint64            // cache miss counter
	logger        *slog.Logger      // structured logger
	getCalls      atomic.Uint64     // total Get() calls for throttled logging
}

// NewLocalBackend creates a new local JSON state backend.
func NewLocalBackend(path string) *LocalBackend {
	return &LocalBackend{Path: path}
}

// SetLogger sets the structured logger for cache diagnostics.
func (b *LocalBackend) SetLogger(l *slog.Logger) {
	b.logger = l
}

// stateFile is the on-disk JSON structure.
type stateFile struct {
	Version string  `json:"version"`
	Entries []Entry `json:"entries"`
}

// Load reads all state entries from the JSON file.
// If the cache is warm and the file has not been modified, it returns
// a copy of the cached entries without touching disk (T021).
func (b *LocalBackend) Load() ([]Entry, error) {
	// Stat the file to get current mtime.
	info, err := os.Stat(b.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// File does not exist — clear cache and return nil.
			b.invalidateCache()
			return nil, nil
		}
		return nil, err
	}

	modTime := info.ModTime()

	// Cache hit: entries are cached and mtime has not changed.
	if b.cachedEntries != nil && modTime.Equal(b.cacheModTime) {
		return b.copyEntries(), nil
	}

	// Cache miss: read from disk.
	data, err := os.ReadFile(b.Path)
	if err != nil {
		return nil, err
	}
	var sf stateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}

	// Populate cache and build index.
	b.cachedEntries = sf.Entries
	b.buildIndex()
	b.cacheModTime = modTime

	return b.copyEntries(), nil
}

// Save writes all state entries to the JSON file with sorted keys.
// After writing, the cache is invalidated so the next read picks up
// the fresh data (T023).
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
	if err := os.WriteFile(b.Path, data, 0644); err != nil {
		return err
	}

	// Invalidate cache after successful write (T023).
	b.invalidateCache()
	return nil
}

// Get retrieves a single entry by FQN using O(1) index lookup (T022).
func (b *LocalBackend) Get(fqn string) (*Entry, error) {
	if err := b.ensureCache(); err != nil {
		return nil, err
	}

	entry, ok := b.index[fqn]
	if ok {
		b.hits++
		b.emitCacheLog()
		// Return a copy so callers cannot mutate cached data.
		cp := *entry
		return &cp, nil
	}

	b.misses++
	b.emitCacheLog()
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

// Lock acquires an exclusive file lock on the state file.
// Returns an error if the lock is already held by another process.
func (b *LocalBackend) Lock() error {
	lockPath := b.Path + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}

	// Try non-blocking exclusive lock
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("state file is locked by another process (concurrent apply not allowed)")
	}

	b.lockFile = f
	return nil
}

// Unlock releases the file lock.
func (b *LocalBackend) Unlock() error {
	if b.lockFile == nil {
		return nil
	}
	err := syscall.Flock(int(b.lockFile.Fd()), syscall.LOCK_UN)
	_ = b.lockFile.Close()
	b.lockFile = nil
	_ = os.Remove(b.Path + ".lock")
	return err
}

// CacheStats returns the current hit and miss counters.
func (b *LocalBackend) CacheStats() (hits, misses uint64) {
	return b.hits, b.misses
}

// --- internal helpers ---

// ensureCache guarantees the cache is warm by calling Load if needed.
func (b *LocalBackend) ensureCache() error {
	if b.cachedEntries != nil {
		// Cache may be stale — Load will check mtime.
		_, err := b.Load()
		return err
	}
	_, err := b.Load()
	return err
}

// buildIndex constructs the FQN → *Entry index from cachedEntries.
func (b *LocalBackend) buildIndex() {
	b.index = make(map[string]*Entry, len(b.cachedEntries))
	for i := range b.cachedEntries {
		b.index[b.cachedEntries[i].FQN] = &b.cachedEntries[i]
	}
}

// invalidateCache clears all cached state (T023).
func (b *LocalBackend) invalidateCache() {
	b.cachedEntries = nil
	b.index = nil
	b.cacheModTime = time.Time{}
}

// copyEntries returns a shallow copy of the cached entries slice
// so callers cannot mutate the cache.
func (b *LocalBackend) copyEntries() []Entry {
	if b.cachedEntries == nil {
		return nil
	}
	cp := make([]Entry, len(b.cachedEntries))
	copy(cp, b.cachedEntries)
	return cp
}

// emitCacheLog logs cache statistics every 100th Get() call (T024).
func (b *LocalBackend) emitCacheLog() {
	if b.logger == nil {
		return
	}
	n := b.getCalls.Add(1)
	if n%100 == 0 {
		b.logger.Info("state cache",
			"hits", b.hits,
			"misses", b.misses,
		)
	}
}
