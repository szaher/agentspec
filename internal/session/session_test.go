package session

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
)

// mockMemoryStore implements memory.Store for testing.
type mockMemoryStore struct {
	messages map[string][]llm.Message
}

func newMockMemoryStore() *mockMemoryStore {
	return &mockMemoryStore{messages: make(map[string][]llm.Message)}
}

func (m *mockMemoryStore) Load(_ context.Context, sessionID string) ([]llm.Message, error) {
	return m.messages[sessionID], nil
}

func (m *mockMemoryStore) Save(_ context.Context, sessionID string, msgs []llm.Message) error {
	m.messages[sessionID] = append(m.messages[sessionID], msgs...)
	return nil
}

func (m *mockMemoryStore) Clear(_ context.Context, sessionID string) error {
	delete(m.messages, sessionID)
	return nil
}

func TestManagerCreate(t *testing.T) {
	store := NewMemoryStore(0)
	mem := newMockMemoryStore()
	mgr := NewManager(store, mem)
	ctx := context.Background()

	sess, err := mgr.Create(ctx, "test-agent", map[string]string{"key": "val"})
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	if !strings.HasPrefix(sess.ID, "sess_") {
		t.Errorf("session ID %q does not have \"sess_\" prefix", sess.ID)
	}

	if sess.AgentName != "test-agent" {
		t.Errorf("AgentName = %q, want %q", sess.AgentName, "test-agent")
	}
}

func TestManagerSaveAndLoadMessages(t *testing.T) {
	store := NewMemoryStore(0)
	mem := newMockMemoryStore()
	mgr := NewManager(store, mem)
	ctx := context.Background()

	sess, err := mgr.Create(ctx, "chat-agent", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
		{Role: llm.RoleAssistant, Content: "Hi there!"},
	}

	if err := mgr.SaveMessages(ctx, sess.ID, msgs); err != nil {
		t.Fatalf("SaveMessages returned unexpected error: %v", err)
	}

	loaded, err := mgr.LoadMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadMessages returned unexpected error: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("LoadMessages returned %d messages, want 2", len(loaded))
	}

	if loaded[0].Role != llm.RoleUser || loaded[0].Content != "Hello" {
		t.Errorf("message[0] = {Role: %q, Content: %q}, want {Role: %q, Content: %q}",
			loaded[0].Role, loaded[0].Content, llm.RoleUser, "Hello")
	}

	if loaded[1].Role != llm.RoleAssistant || loaded[1].Content != "Hi there!" {
		t.Errorf("message[1] = {Role: %q, Content: %q}, want {Role: %q, Content: %q}",
			loaded[1].Role, loaded[1].Content, llm.RoleAssistant, "Hi there!")
	}
}

func TestManagerLoadMessagesInvalidSession(t *testing.T) {
	store := NewMemoryStore(0)
	mem := newMockMemoryStore()
	mgr := NewManager(store, mem)
	ctx := context.Background()

	_, err := mgr.LoadMessages(ctx, "sess_does_not_exist")
	if err == nil {
		t.Fatal("LoadMessages with non-existent session ID should return an error")
	}
}

func TestManagerClose(t *testing.T) {
	store := NewMemoryStore(0)
	mem := newMockMemoryStore()
	mgr := NewManager(store, mem)
	ctx := context.Background()

	sess, err := mgr.Create(ctx, "close-agent", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	// Save some messages so we can verify they get cleared.
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "test message"},
	}
	if err := mgr.SaveMessages(ctx, sess.ID, msgs); err != nil {
		t.Fatalf("SaveMessages returned unexpected error: %v", err)
	}

	if err := mgr.Close(ctx, sess.ID); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}

	// Get should return an error because the session was deleted.
	_, err = mgr.Get(ctx, sess.ID)
	if err == nil {
		t.Fatal("Get after Close should return an error")
	}

	// Messages should be cleared.
	loaded, err := mem.Load(ctx, sess.ID)
	if err != nil {
		t.Fatalf("mem.Load returned unexpected error: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("messages after Close: got %d, want 0", len(loaded))
	}
}

func TestManagerList(t *testing.T) {
	store := NewMemoryStore(5 * time.Minute)
	mem := newMockMemoryStore()
	mgr := NewManager(store, mem)
	ctx := context.Background()

	if _, err := mgr.Create(ctx, "agent-a", nil); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if _, err := mgr.Create(ctx, "agent-a", nil); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if _, err := mgr.Create(ctx, "agent-b", nil); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	listA, err := mgr.List(ctx, "agent-a")
	if err != nil {
		t.Fatalf("List(\"agent-a\") returned unexpected error: %v", err)
	}
	if len(listA) != 2 {
		t.Errorf("List(\"agent-a\") returned %d sessions, want 2", len(listA))
	}

	listB, err := mgr.List(ctx, "agent-b")
	if err != nil {
		t.Fatalf("List(\"agent-b\") returned unexpected error: %v", err)
	}
	if len(listB) != 1 {
		t.Errorf("List(\"agent-b\") returned %d sessions, want 1", len(listB))
	}
}

func TestManagerSaveMessages_TouchError(t *testing.T) {
	store := NewMemoryStore(0)
	mem := newMockMemoryStore()
	mgr := NewManager(store, mem)
	ctx := context.Background()

	// Create a session, then delete it so Touch will fail.
	sess, err := mgr.Create(ctx, "touch-err-agent", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	// Delete the session from the store directly, making Touch fail.
	if err := store.Delete(ctx, sess.ID); err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	// SaveMessages should fail because Touch cannot find the session.
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}
	err = mgr.SaveMessages(ctx, sess.ID, msgs)
	if err == nil {
		t.Fatal("SaveMessages should return an error when Touch fails")
	}
	if !strings.Contains(err.Error(), "touch session") {
		t.Errorf("error %q does not contain \"touch session\"", err.Error())
	}
}

// failClearMemoryStore embeds mockMemoryStore and overrides Clear to return an error.
type failClearMemoryStore struct {
	mockMemoryStore
}

func (m *failClearMemoryStore) Clear(_ context.Context, _ string) error {
	return fmt.Errorf("redis connection refused")
}

func TestManagerClose_ClearError(t *testing.T) {
	store := NewMemoryStore(0)
	mem := &failClearMemoryStore{
		mockMemoryStore: mockMemoryStore{messages: make(map[string][]llm.Message)},
	}
	mgr := NewManager(store, mem)
	ctx := context.Background()

	sess, err := mgr.Create(ctx, "clear-err-agent", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	// Save some messages so the session has data.
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "test"},
	}
	if err := mgr.SaveMessages(ctx, sess.ID, msgs); err != nil {
		t.Fatalf("SaveMessages returned unexpected error: %v", err)
	}

	// Close should fail because Clear returns an error.
	err = mgr.Close(ctx, sess.ID)
	if err == nil {
		t.Fatal("Close should return an error when Clear fails")
	}
	if !strings.Contains(err.Error(), "clear memory") {
		t.Errorf("error %q does not contain \"clear memory\"", err.Error())
	}
}
