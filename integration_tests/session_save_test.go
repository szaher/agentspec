package integration_tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/session"
)

// mockRedisClient implements session.RedisClient with in-memory list support.
type mockRedisClient struct {
	mu      sync.Mutex
	strings map[string]string
	lists   map[string][]string
	ttls    map[string]time.Duration
	failGet bool // simulate Redis failure on Get
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		strings: make(map[string]string),
		lists:   make(map[string][]string),
		ttls:    make(map[string]time.Duration),
	}
}

func (m *mockRedisClient) Get(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failGet {
		return "", fmt.Errorf("redis connection error")
	}
	v, ok := m.strings[key]
	if !ok {
		return "", fmt.Errorf("key not found: %s", key)
	}
	return v, nil
}

func (m *mockRedisClient) Set(_ context.Context, key string, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strings[key] = value
	m.ttls[key] = ttl
	return nil
}

func (m *mockRedisClient) Del(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.strings, k)
		delete(m.lists, k)
		delete(m.ttls, k)
	}
	return nil
}

func (m *mockRedisClient) Keys(_ context.Context, pattern string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var keys []string
	for k := range m.strings {
		keys = append(keys, k)
	}
	for k := range m.lists {
		keys = append(keys, k)
	}
	return keys, nil
}

func (m *mockRedisClient) RPush(_ context.Context, key string, values ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lists[key] = append(m.lists[key], values...)
	return nil
}

func (m *mockRedisClient) LRange(_ context.Context, key string, start, stop int64) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list, ok := m.lists[key]
	if !ok {
		return nil, nil
	}
	length := int64(len(list))
	if start < 0 {
		start = length + start
	}
	if stop < 0 {
		stop = length + stop
	}
	if start < 0 {
		start = 0
	}
	if stop >= length {
		stop = length - 1
	}
	if start > stop {
		return nil, nil
	}
	return list[start : stop+1], nil
}

func (m *mockRedisClient) Expire(_ context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ttls[key] = ttl
	return nil
}

func (m *mockRedisClient) Type(_ context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.lists[key]; ok {
		return "list", nil
	}
	if _, ok := m.strings[key]; ok {
		return "string", nil
	}
	return "none", nil
}

func TestSessionSaveMessages(t *testing.T) {
	client := newMockRedisClient()
	store := session.NewRedisStore(client)
	ctx := context.Background()

	// Create a session first
	sess, err := store.Create(ctx, "test-agent", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Save messages
	msgs := []llm.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}
	if err := store.SaveMessages(ctx, sess.ID, msgs); err != nil {
		t.Fatalf("SaveMessages failed: %v", err)
	}

	// Load messages
	loaded, err := store.LoadMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadMessages failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(loaded))
	}
	if loaded[0].Content != "Hello" {
		t.Errorf("first message content = %q, want %q", loaded[0].Content, "Hello")
	}
	if loaded[1].Content != "Hi there!" {
		t.Errorf("second message content = %q, want %q", loaded[1].Content, "Hi there!")
	}
}

func TestSessionConcurrentSaves(t *testing.T) {
	client := newMockRedisClient()
	store := session.NewRedisStore(client)
	ctx := context.Background()

	// Create a session
	sess, err := store.Create(ctx, "test-agent", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	const numWorkers = 100
	var wg sync.WaitGroup
	errCh := make(chan error, numWorkers)

	// Send 100 messages concurrently, one per goroutine
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := llm.Message{
				Role:    "user",
				Content: fmt.Sprintf("message-%d", id),
			}
			if err := store.SaveMessages(ctx, sess.ID, []llm.Message{msg}); err != nil {
				errCh <- fmt.Errorf("worker %d: %w", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent save error: %v", err)
	}

	// Verify all 100 messages are present
	loaded, err := store.LoadMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadMessages failed: %v", err)
	}
	if len(loaded) != numWorkers {
		t.Errorf("expected %d messages, got %d (lost %d messages)",
			numWorkers, len(loaded), numWorkers-len(loaded))
	}
}

func TestSessionLoadMessagesEmptySession(t *testing.T) {
	client := newMockRedisClient()
	store := session.NewRedisStore(client)
	ctx := context.Background()

	// Load from a non-existent session
	loaded, err := store.LoadMessages(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("LoadMessages should not error for missing session: %v", err)
	}
	if loaded != nil {
		t.Errorf("expected nil messages for empty session, got %d", len(loaded))
	}
}

func TestSessionMigrationFromString(t *testing.T) {
	client := newMockRedisClient()
	store := session.NewRedisStore(client)
	ctx := context.Background()

	// Create a session
	sess, err := store.Create(ctx, "test-agent", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Simulate old-style string-based message storage
	oldMsgs := `[{"role":"user","content":"old message 1"},{"role":"assistant","content":"old message 2"}]`
	msgKey := "agentspec:session:" + sess.ID + ":messages"
	if err := client.Set(ctx, msgKey, oldMsgs, 24*time.Hour); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// LoadMessages should detect string key and migrate to list
	loaded, err := store.LoadMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadMessages after migration failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 messages after migration, got %d", len(loaded))
	}
	if loaded[0].Content != "old message 1" {
		t.Errorf("migrated message 1 content = %q, want %q", loaded[0].Content, "old message 1")
	}

	// Verify key is now a list (subsequent Load should use LRANGE)
	keyType, _ := client.Type(ctx, msgKey)
	if keyType != "list" {
		t.Errorf("key type after migration = %q, want %q", keyType, "list")
	}

	// Save new messages should append to the list
	newMsgs := []llm.Message{{Role: "user", Content: "new message"}}
	if err := store.SaveMessages(ctx, sess.ID, newMsgs); err != nil {
		t.Fatalf("SaveMessages after migration failed: %v", err)
	}

	loaded, err = store.LoadMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadMessages after append failed: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected 3 messages after migration + append, got %d", len(loaded))
	}
}

func TestSessionSaveAppendNotReplace(t *testing.T) {
	client := newMockRedisClient()
	store := session.NewRedisStore(client)
	ctx := context.Background()

	sess, err := store.Create(ctx, "test-agent", nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Save first batch
	batch1 := []llm.Message{{Role: "user", Content: "msg1"}}
	if err := store.SaveMessages(ctx, sess.ID, batch1); err != nil {
		t.Fatalf("first SaveMessages failed: %v", err)
	}

	// Save second batch
	batch2 := []llm.Message{{Role: "assistant", Content: "msg2"}}
	if err := store.SaveMessages(ctx, sess.ID, batch2); err != nil {
		t.Fatalf("second SaveMessages failed: %v", err)
	}

	// Both messages should be present
	loaded, err := store.LoadMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadMessages failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 messages (append), got %d", len(loaded))
	}
}
