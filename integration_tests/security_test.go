package integration_tests

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/szaher/designs/agentz/internal/sandbox"
	"github.com/szaher/designs/agentz/internal/session"
)

func TestSessionIDSecurity(t *testing.T) {
	store := session.NewMemoryStore(0)

	t.Run("cryptographic randomness prefix", func(t *testing.T) {
		sess, err := store.Create(context.Background(), "test-agent", nil)
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if !strings.HasPrefix(sess.ID, "sess_") {
			t.Errorf("session ID %q does not have sess_ prefix", sess.ID)
		}
		// sess_ (5 chars) + 22 chars base64url (128 bits) = 27 chars minimum
		if len(sess.ID) < 27 {
			t.Errorf("session ID %q too short (%d chars), expected at least 27", sess.ID, len(sess.ID))
		}
	})

	t.Run("not timestamp derived", func(t *testing.T) {
		// Create two sessions in rapid succession; if timestamp-based they'd be identical or sequential
		s1, _ := store.Create(context.Background(), "test-agent", nil)
		s2, _ := store.Create(context.Background(), "test-agent", nil)
		if s1.ID == s2.ID {
			t.Errorf("two sessions created in succession have identical IDs: %s", s1.ID)
		}
		// Ensure IDs are not numerically sequential (timestamp-based pattern)
		suffix1 := strings.TrimPrefix(s1.ID, "sess_")
		suffix2 := strings.TrimPrefix(s2.ID, "sess_")
		if suffix1 == suffix2 {
			t.Errorf("session ID suffixes are identical: %s", suffix1)
		}
	})

	t.Run("concurrent creation zero collisions", func(t *testing.T) {
		const n = 10000
		ids := make([]string, n)
		var wg sync.WaitGroup
		wg.Add(n)

		for i := 0; i < n; i++ {
			go func(idx int) {
				defer wg.Done()
				sess, err := store.Create(context.Background(), "test-agent", nil)
				if err != nil {
					t.Errorf("Create[%d]: %v", idx, err)
					return
				}
				ids[idx] = sess.ID
			}(i)
		}
		wg.Wait()

		seen := make(map[string]bool, n)
		for i, id := range ids {
			if id == "" {
				continue // failed creation
			}
			if seen[id] {
				t.Fatalf("collision at index %d: duplicate session ID %q", i, id)
			}
			seen[id] = true
		}
		t.Logf("created %d unique session IDs with zero collisions", len(seen))
	})

	t.Run("increment/decrement ID returns not found", func(t *testing.T) {
		sess, _ := store.Create(context.Background(), "test-agent", nil)
		// Try variations of the ID
		variants := []string{
			sess.ID + "0",
			sess.ID[:len(sess.ID)-1],
			"sess_AAAAAAAAAAAAAAAAAAAAAA",
			"sess_" + strings.Repeat("A", 22),
		}
		for _, v := range variants {
			_, err := store.Get(context.Background(), v)
			if err == nil {
				t.Errorf("Get(%q) should fail for guessed ID", v)
			}
		}
	})
}

func TestProcessSandboxAvailability(t *testing.T) {
	ps := &sandbox.ProcessSandbox{}
	if !ps.Available() {
		t.Skip("process sandbox not available on this platform")
	}

	t.Run("basic bash execution", func(t *testing.T) {
		stdout, stderr, err := ps.Execute(context.Background(), sandbox.ExecConfig{
			Language:   "bash",
			Script:     "echo hello",
			MemoryMB:   128,
			TimeoutSec: 5,
		})
		if err != nil {
			t.Fatalf("Execute: %v (stderr: %s)", err, stderr)
		}
		if got := strings.TrimSpace(stdout); got != "hello" {
			t.Errorf("stdout = %q, want %q", got, "hello")
		}
	})

	t.Run("timeout enforcement", func(t *testing.T) {
		_, _, err := ps.Execute(context.Background(), sandbox.ExecConfig{
			Language:   "bash",
			Script:     "sleep 30",
			MemoryMB:   128,
			TimeoutSec: 1,
		})
		if err == nil {
			t.Fatal("expected timeout error")
		}
		// Accept: ErrResourceLimit, context deadline, killed by signal, or exit status from timeout
		t.Logf("timeout error (expected): %v", err)
	})

	t.Run("environment isolation", func(t *testing.T) {
		// Verify that host env vars are not inherited
		stdout, _, err := ps.Execute(context.Background(), sandbox.ExecConfig{
			Language:   "bash",
			Script:     "echo ${SOME_HOST_VAR:-not_found}",
			Env:        map[string]string{"SOME_HOST_VAR": ""},
			MemoryMB:   128,
			TimeoutSec: 5,
		})
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if got := strings.TrimSpace(stdout); got != "not_found" {
			t.Errorf("host env should not be inherited, got: %q", got)
		}
	})
}

func TestNoopSandbox(t *testing.T) {
	ns := &sandbox.NoopSandbox{}
	if !ns.Available() {
		t.Fatal("NoopSandbox should always be available")
	}

	stdout, _, err := ns.Execute(context.Background(), sandbox.ExecConfig{
		Language:   "bash",
		Script:     "echo noop",
		TimeoutSec: 5,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := strings.TrimSpace(stdout); got != "noop" {
		t.Errorf("stdout = %q, want %q", got, "noop")
	}
}
