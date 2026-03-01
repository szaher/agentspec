package session

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMemoryStoreCreate(t *testing.T) {
	store := NewMemoryStore(0)
	ctx := context.Background()

	meta := map[string]string{"env": "test", "version": "1"}
	sess, err := store.Create(ctx, "my-agent", meta)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	if !strings.HasPrefix(sess.ID, "sess_") {
		t.Errorf("session ID %q does not have \"sess_\" prefix", sess.ID)
	}

	if sess.AgentName != "my-agent" {
		t.Errorf("AgentName = %q, want %q", sess.AgentName, "my-agent")
	}

	if sess.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}

	if sess.LastActive.IsZero() {
		t.Error("LastActive is zero")
	}

	if sess.Metadata["env"] != "test" {
		t.Errorf("Metadata[\"env\"] = %q, want %q", sess.Metadata["env"], "test")
	}

	if sess.Metadata["version"] != "1" {
		t.Errorf("Metadata[\"version\"] = %q, want %q", sess.Metadata["version"], "1")
	}
}

func TestMemoryStoreGet(t *testing.T) {
	store := NewMemoryStore(0)
	ctx := context.Background()

	sess, err := store.Create(ctx, "agent-x", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}

	if got.ID != sess.ID {
		t.Errorf("Get ID = %q, want %q", got.ID, sess.ID)
	}

	if got.AgentName != "agent-x" {
		t.Errorf("Get AgentName = %q, want %q", got.AgentName, "agent-x")
	}

	// Get with unknown ID should return an error containing "not found".
	_, err = store.Get(ctx, "sess_nonexistent")
	if err == nil {
		t.Fatal("Get with unknown ID should return an error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not contain \"not found\"", err.Error())
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	store := NewMemoryStore(0)
	ctx := context.Background()

	sess, err := store.Create(ctx, "agent-d", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	if err := store.Delete(ctx, sess.ID); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	_, err = store.Get(ctx, sess.ID)
	if err == nil {
		t.Fatal("Get after Delete should return an error")
	}
}

func TestMemoryStoreList(t *testing.T) {
	store := NewMemoryStore(0)
	ctx := context.Background()

	// Create 2 sessions for "agent-a" and 1 for "agent-b".
	if _, err := store.Create(ctx, "agent-a", nil); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if _, err := store.Create(ctx, "agent-a", nil); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if _, err := store.Create(ctx, "agent-b", nil); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	listA, err := store.List(ctx, "agent-a")
	if err != nil {
		t.Fatalf("List(\"agent-a\") returned unexpected error: %v", err)
	}
	if len(listA) != 2 {
		t.Errorf("List(\"agent-a\") returned %d sessions, want 2", len(listA))
	}

	listB, err := store.List(ctx, "agent-b")
	if err != nil {
		t.Fatalf("List(\"agent-b\") returned unexpected error: %v", err)
	}
	if len(listB) != 1 {
		t.Errorf("List(\"agent-b\") returned %d sessions, want 1", len(listB))
	}

	listAll, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("List(\"\") returned unexpected error: %v", err)
	}
	if len(listAll) != 3 {
		t.Errorf("List(\"\") returned %d sessions, want 3", len(listAll))
	}
}

func TestMemoryStoreTouch(t *testing.T) {
	store := NewMemoryStore(0)
	ctx := context.Background()

	sess, err := store.Create(ctx, "agent-t", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	originalLastActive := sess.LastActive

	// Sleep briefly so the clock advances.
	time.Sleep(10 * time.Millisecond)

	if err := store.Touch(ctx, sess.ID); err != nil {
		t.Fatalf("Touch returned unexpected error: %v", err)
	}

	got, err := store.Get(ctx, sess.ID)
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}

	if !got.LastActive.After(originalLastActive) {
		t.Errorf("LastActive was not updated: original=%v, after touch=%v", originalLastActive, got.LastActive)
	}

	// Touch with unknown ID should return an error.
	err = store.Touch(ctx, "sess_unknown")
	if err == nil {
		t.Fatal("Touch with unknown ID should return an error")
	}
}

func TestMemoryStoreExpiry(t *testing.T) {
	store := NewMemoryStore(1 * time.Millisecond)
	ctx := context.Background()

	sess, err := store.Create(ctx, "agent-e", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	// Wait for the session to expire.
	time.Sleep(5 * time.Millisecond)

	_, err = store.Get(ctx, sess.ID)
	if err == nil {
		t.Fatal("Get should return error for expired session")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("error %q does not contain \"expired\"", err.Error())
	}

	list, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List returned %d sessions for expired store, want 0", len(list))
	}
}

func TestGenerateSecureID(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		id := generateSecureID()

		if !strings.HasPrefix(id, "sess_") {
			t.Errorf("generateSecureID() = %q, missing \"sess_\" prefix", id)
		}

		if len(id) <= 5 {
			t.Errorf("generateSecureID() = %q, length %d is too short", id, len(id))
		}

		if _, exists := seen[id]; exists {
			t.Errorf("generateSecureID() produced duplicate ID %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestMemoryStoreConcurrency(t *testing.T) {
	store := NewMemoryStore(0)
	ctx := context.Background()

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := store.Create(ctx, "concurrent-agent", nil)
			if err != nil {
				t.Errorf("Create returned unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()

	list, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(list) != goroutines {
		t.Errorf("List returned %d sessions, want %d", len(list), goroutines)
	}
}
