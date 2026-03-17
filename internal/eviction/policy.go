package eviction

import (
	"fmt"
	"time"
)

// Policy configures when and how stale entries are removed from in-memory stores.
type Policy struct {
	// MaxEntries is the maximum number of entries before forced eviction.
	// Must be > 0. Default: 10000.
	MaxEntries int

	// TTL is the time-to-live for entries since last access.
	// Must be > 0. Default: 10m.
	TTL time.Duration

	// EvictionInterval is how often the background eviction goroutine runs.
	// Must be > 0 and < TTL. Default: 5m.
	EvictionInterval time.Duration
}

// DefaultPolicy returns the default eviction policy.
func DefaultPolicy() Policy {
	return Policy{
		MaxEntries:       10000,
		TTL:              10 * time.Minute,
		EvictionInterval: 5 * time.Minute,
	}
}

// Validate checks that all fields have valid values.
func (p Policy) Validate() error {
	if p.MaxEntries <= 0 {
		return fmt.Errorf("eviction: MaxEntries must be > 0, got %d", p.MaxEntries)
	}
	if p.TTL <= 0 {
		return fmt.Errorf("eviction: TTL must be > 0, got %v", p.TTL)
	}
	if p.EvictionInterval <= 0 {
		return fmt.Errorf("eviction: EvictionInterval must be > 0, got %v", p.EvictionInterval)
	}
	if p.EvictionInterval >= p.TTL {
		return fmt.Errorf("eviction: EvictionInterval (%v) must be < TTL (%v)", p.EvictionInterval, p.TTL)
	}
	return nil
}
