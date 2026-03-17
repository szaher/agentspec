package memory

import "testing"

func TestLRUNewEmpty(t *testing.T) {
	lru := NewLRU()
	if lru.Len() != 0 {
		t.Fatalf("expected Len() == 0 for new LRU, got %d", lru.Len())
	}
}

func TestLRUPromoteNew(t *testing.T) {
	lru := NewLRU()

	lru.Promote("s1")
	if lru.Len() != 1 {
		t.Fatalf("expected Len() == 1 after promoting one new session, got %d", lru.Len())
	}

	lru.Promote("s2")
	if lru.Len() != 2 {
		t.Fatalf("expected Len() == 2 after promoting two new sessions, got %d", lru.Len())
	}
}

func TestLRUPromoteExisting(t *testing.T) {
	lru := NewLRU()

	lru.Promote("s1")
	lru.Promote("s2")
	if lru.Len() != 2 {
		t.Fatalf("expected Len() == 2, got %d", lru.Len())
	}

	// Promote an existing session; length must not change.
	lru.Promote("s1")
	if lru.Len() != 2 {
		t.Fatalf("expected Len() == 2 after re-promoting existing session, got %d", lru.Len())
	}
}

func TestLRUEvictOrder(t *testing.T) {
	t.Run("basic FIFO order", func(t *testing.T) {
		lru := NewLRU()
		lru.Promote("A")
		lru.Promote("B")
		lru.Promote("C")

		// Order (front→back): C, B, A. Evict should return the back element: A.
		got := lru.Evict()
		if got != "A" {
			t.Fatalf("expected Evict() == %q, got %q", "A", got)
		}
	})

	t.Run("promote changes eviction order", func(t *testing.T) {
		lru := NewLRU()
		lru.Promote("A")
		lru.Promote("B")
		lru.Promote("C")

		// Order (front→back): C, B, A. Now promote A to front.
		lru.Promote("A")
		// Order (front→back): A, C, B. Evict should return B.
		got := lru.Evict()
		if got != "B" {
			t.Fatalf("expected Evict() == %q after promoting A, got %q", "B", got)
		}

		// Next eviction should return C.
		got = lru.Evict()
		if got != "C" {
			t.Fatalf("expected Evict() == %q, got %q", "C", got)
		}

		// Next eviction should return A.
		got = lru.Evict()
		if got != "A" {
			t.Fatalf("expected Evict() == %q, got %q", "A", got)
		}
	})
}

func TestLRUEvictEmpty(t *testing.T) {
	lru := NewLRU()
	got := lru.Evict()
	if got != "" {
		t.Fatalf("expected Evict() on empty tracker to return %q, got %q", "", got)
	}
}

func TestLRURemove(t *testing.T) {
	lru := NewLRU()
	lru.Promote("s1")
	lru.Promote("s2")
	lru.Promote("s3")

	// Remove a tracked session.
	lru.Remove("s2")
	if lru.Len() != 2 {
		t.Fatalf("expected Len() == 2 after removing one session, got %d", lru.Len())
	}

	// Verify s2 is gone: eviction order should be s1, s3 (front→back: s3, s1).
	got := lru.Evict()
	if got != "s1" {
		t.Fatalf("expected Evict() == %q after removing s2, got %q", "s1", got)
	}
	got = lru.Evict()
	if got != "s3" {
		t.Fatalf("expected Evict() == %q, got %q", "s3", got)
	}

	// Remove non-existent session is a no-op.
	lru.Remove("does-not-exist")
	if lru.Len() != 0 {
		t.Fatalf("expected Len() == 0 after removing non-existent session from empty tracker, got %d", lru.Len())
	}
}

func TestLRUPromoteAfterEvict(t *testing.T) {
	lru := NewLRU()
	lru.Promote("A")
	lru.Promote("B")
	lru.Promote("C")

	// Evict the least recently used (A).
	evicted := lru.Evict()
	if evicted != "A" {
		t.Fatalf("expected Evict() == %q, got %q", "A", evicted)
	}
	if lru.Len() != 2 {
		t.Fatalf("expected Len() == 2 after eviction, got %d", lru.Len())
	}

	// Promote a new session D.
	lru.Promote("D")
	if lru.Len() != 3 {
		t.Fatalf("expected Len() == 3 after promoting D, got %d", lru.Len())
	}

	// Order (front→back): D, C, B. Evict should return B.
	got := lru.Evict()
	if got != "B" {
		t.Fatalf("expected Evict() == %q, got %q", "B", got)
	}

	// Evict next: should return C.
	got = lru.Evict()
	if got != "C" {
		t.Fatalf("expected Evict() == %q, got %q", "C", got)
	}

	// Evict next: should return D.
	got = lru.Evict()
	if got != "D" {
		t.Fatalf("expected Evict() == %q, got %q", "D", got)
	}

	// Tracker should be empty now.
	if lru.Len() != 0 {
		t.Fatalf("expected Len() == 0 after evicting all sessions, got %d", lru.Len())
	}
}

func TestLRULen(t *testing.T) {
	lru := NewLRU()

	// Starts at zero.
	if lru.Len() != 0 {
		t.Fatalf("expected Len() == 0, got %d", lru.Len())
	}

	// Additions increase Len.
	lru.Promote("s1")
	if lru.Len() != 1 {
		t.Fatalf("expected Len() == 1, got %d", lru.Len())
	}
	lru.Promote("s2")
	if lru.Len() != 2 {
		t.Fatalf("expected Len() == 2, got %d", lru.Len())
	}
	lru.Promote("s3")
	if lru.Len() != 3 {
		t.Fatalf("expected Len() == 3, got %d", lru.Len())
	}

	// Promotion of existing does not change Len.
	lru.Promote("s2")
	if lru.Len() != 3 {
		t.Fatalf("expected Len() == 3 after re-promoting, got %d", lru.Len())
	}

	// Eviction decreases Len.
	lru.Evict()
	if lru.Len() != 2 {
		t.Fatalf("expected Len() == 2 after eviction, got %d", lru.Len())
	}

	// Removal decreases Len.
	lru.Remove("s2")
	if lru.Len() != 1 {
		t.Fatalf("expected Len() == 1 after removal, got %d", lru.Len())
	}

	// Removing non-existent does not change Len.
	lru.Remove("nonexistent")
	if lru.Len() != 1 {
		t.Fatalf("expected Len() == 1 after removing non-existent, got %d", lru.Len())
	}

	// Final eviction brings to zero.
	lru.Evict()
	if lru.Len() != 0 {
		t.Fatalf("expected Len() == 0 after final eviction, got %d", lru.Len())
	}

	// Evict on empty does not go negative.
	lru.Evict()
	if lru.Len() != 0 {
		t.Fatalf("expected Len() == 0 after evicting from empty tracker, got %d", lru.Len())
	}
}
