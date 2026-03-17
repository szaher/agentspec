package integration_tests

import (
	"context"
	"testing"
	"time"

	"github.com/szaher/agentspec/internal/session"
)

// TestMemoryStoreBackgroundEviction verifies that the background eviction
// goroutine removes expired sessions automatically. It creates a MemoryStore
// with a very short expiry and eviction interval, adds sessions, and then
// waits long enough for the background ticker to fire and evict them.
func TestMemoryStoreBackgroundEviction(t *testing.T) {
	const (
		expiry           = 100 * time.Millisecond
		evictionInterval = 50 * time.Millisecond
		sessionCount     = 5
	)

	store := session.NewMemoryStore(expiry, evictionInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store.Start(ctx)

	// Create several sessions.
	// A small sleep between creations avoids ID collisions from the
	// nanosecond-based generateID() used internally by MemoryStore.
	ids := make([]string, 0, sessionCount)
	for i := 0; i < sessionCount; i++ {
		sess, err := store.Create(ctx, "test-agent", map[string]string{
			"index": string(rune('0' + i)),
		})
		if err != nil {
			t.Fatalf("Create session %d: %v", i, err)
		}
		ids = append(ids, sess.ID)
		if i < sessionCount-1 {
			time.Sleep(1 * time.Millisecond)
		}
	}

	// Verify all sessions exist before expiry.
	for i, id := range ids {
		if _, err := store.Get(ctx, id); err != nil {
			t.Fatalf("Get session %d before expiry: %v", i, err)
		}
	}

	// Wait for expiry + at least one eviction cycle.
	time.Sleep(expiry + evictionInterval*3)

	// All sessions should have been evicted by the background goroutine.
	for i, id := range ids {
		_, err := store.Get(ctx, id)
		if err == nil {
			t.Errorf("session %d (%s): expected error after background eviction, but Get succeeded", i, id)
		}
	}

	// List should also return no sessions.
	remaining, err := store.List(ctx, "test-agent")
	if err != nil {
		t.Fatalf("List after eviction: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("expected 0 sessions after background eviction, got %d", len(remaining))
	}
}

// TestMemoryStoreListLazyEviction verifies that List() lazily evicts expired
// sessions even when the background eviction goroutine has not yet run.
// It uses a very long eviction interval (1 hour) to ensure the background
// cleanup does not trigger, and instead relies on List() to filter out and
// delete expired sessions.
func TestMemoryStoreListLazyEviction(t *testing.T) {
	const (
		expiry           = 100 * time.Millisecond
		evictionInterval = 1 * time.Hour // intentionally long to prevent background cleanup
		sessionCount     = 5
	)

	store := session.NewMemoryStore(expiry, evictionInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the store (background eviction won't fire within this test).
	store.Start(ctx)

	// Create several sessions for the same agent.
	// A small sleep between creations avoids ID collisions from the
	// nanosecond-based generateID() used internally by MemoryStore.
	ids := make([]string, 0, sessionCount)
	for i := 0; i < sessionCount; i++ {
		sess, err := store.Create(ctx, "lazy-agent", nil)
		if err != nil {
			t.Fatalf("Create session %d: %v", i, err)
		}
		ids = append(ids, sess.ID)
		if i < sessionCount-1 {
			time.Sleep(1 * time.Millisecond)
		}
	}

	// Verify all sessions are listed before expiry.
	listed, err := store.List(ctx, "lazy-agent")
	if err != nil {
		t.Fatalf("List before expiry: %v", err)
	}
	if len(listed) != sessionCount {
		t.Fatalf("expected %d sessions before expiry, got %d", sessionCount, len(listed))
	}

	// Wait for sessions to expire.
	time.Sleep(expiry + 50*time.Millisecond)

	// List should return no sessions (lazy eviction filters them out).
	listed, err = store.List(ctx, "lazy-agent")
	if err != nil {
		t.Fatalf("List after expiry: %v", err)
	}
	if len(listed) != 0 {
		t.Errorf("expected 0 sessions from List after lazy eviction, got %d", len(listed))
	}

	// Confirm that Get also returns errors for all sessions, verifying
	// that the lazy eviction in List() actually deleted them.
	for i, id := range ids {
		_, err := store.Get(ctx, id)
		if err == nil {
			t.Errorf("session %d (%s): expected error after lazy eviction via List, but Get succeeded", i, id)
		}
	}

	// A second List call with empty agent name should also return nothing,
	// confirming the sessions were truly deleted (not just filtered).
	all, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("List (unfiltered) after lazy eviction: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 sessions in unfiltered List after lazy eviction, got %d", len(all))
	}
}
