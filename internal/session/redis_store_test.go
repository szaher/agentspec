package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
)

// mockRedisClient is an in-memory implementation of RedisClient for testing.
type mockRedisClient struct {
	mu    sync.Mutex
	data  map[string]string   // for string keys
	lists map[string][]string // for list keys
	types map[string]string   // track key types explicitly
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		data:  make(map[string]string),
		lists: make(map[string][]string),
		types: make(map[string]string),
	}
}

func (m *mockRedisClient) Get(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, ok := m.data[key]
	if !ok {
		return "", fmt.Errorf("key not found")
	}
	return val, nil
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	m.types[key] = "string"
	return nil
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.data, key)
		delete(m.lists, key)
		delete(m.types, key)
	}
	return nil
}

func (m *mockRedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simple prefix match: strip trailing '*' and match prefix.
	prefix := strings.TrimSuffix(pattern, "*")

	var result []string
	seen := make(map[string]struct{})

	for k := range m.data {
		if strings.HasPrefix(k, prefix) {
			if _, ok := seen[k]; !ok {
				result = append(result, k)
				seen[k] = struct{}{}
			}
		}
	}
	for k := range m.lists {
		if strings.HasPrefix(k, prefix) {
			if _, ok := seen[k]; !ok {
				result = append(result, k)
				seen[k] = struct{}{}
			}
		}
	}

	return result, nil
}

func (m *mockRedisClient) RPush(ctx context.Context, key string, values ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lists[key] = append(m.lists[key], values...)
	m.types[key] = "list"
	return nil
}

func (m *mockRedisClient) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
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

func (m *mockRedisClient) Expire(ctx context.Context, key string, ttl time.Duration) error {
	// No-op for mock; we don't implement actual expiry.
	return nil
}

func (m *mockRedisClient) Type(ctx context.Context, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t, ok := m.types[key]; ok {
		return t, nil
	}
	if _, ok := m.data[key]; ok {
		return "string", nil
	}
	if _, ok := m.lists[key]; ok {
		return "list", nil
	}
	return "none", nil
}

func TestNewRedisStore_Defaults(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)

	if store.prefix != "agentspec:session:" {
		t.Errorf("default prefix = %q, want %q", store.prefix, "agentspec:session:")
	}

	if store.ttl != 24*time.Hour {
		t.Errorf("default ttl = %v, want %v", store.ttl, 24*time.Hour)
	}
}

func TestNewRedisStore_WithOptions(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client,
		WithPrefix("custom:prefix:"),
		WithTTL(2*time.Hour),
	)

	if store.prefix != "custom:prefix:" {
		t.Errorf("prefix = %q, want %q", store.prefix, "custom:prefix:")
	}

	if store.ttl != 2*time.Hour {
		t.Errorf("ttl = %v, want %v", store.ttl, 2*time.Hour)
	}
}

func TestRedisStore_Create(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
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

	// Verify the session was stored in the mock client.
	key := "agentspec:session:" + sess.ID
	raw, err := client.Get(ctx, key)
	if err != nil {
		t.Fatalf("session key %q not found in mock client: %v", key, err)
	}

	var stored Session
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatalf("failed to unmarshal stored session: %v", err)
	}

	if stored.ID != sess.ID {
		t.Errorf("stored session ID = %q, want %q", stored.ID, sess.ID)
	}

	if stored.AgentName != "my-agent" {
		t.Errorf("stored session AgentName = %q, want %q", stored.AgentName, "my-agent")
	}
}

func TestRedisStore_Get(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
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

func TestRedisStore_Delete(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
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

func TestRedisStore_List(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
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

func TestRedisStore_Touch(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
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
		t.Errorf("LastActive was not updated: original=%v, after touch=%v",
			originalLastActive, got.LastActive)
	}

	// Touch with unknown ID should return an error.
	err = store.Touch(ctx, "sess_unknown")
	if err == nil {
		t.Fatal("Touch with unknown ID should return an error")
	}
}

func TestRedisStore_SaveMessages(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
	ctx := context.Background()

	sess, err := store.Create(ctx, "agent-msg", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
		{Role: llm.RoleAssistant, Content: "Hi there!"},
	}

	if err := store.SaveMessages(ctx, sess.ID, messages); err != nil {
		t.Fatalf("SaveMessages returned unexpected error: %v", err)
	}

	// Verify messages are stored in the list.
	key := "agentspec:session:" + sess.ID + ":messages"
	client.mu.Lock()
	storedList, ok := client.lists[key]
	client.mu.Unlock()

	if !ok {
		t.Fatalf("messages key %q not found in mock client lists", key)
	}

	if len(storedList) != 2 {
		t.Fatalf("stored list has %d elements, want 2", len(storedList))
	}

	var msg0 llm.Message
	if err := json.Unmarshal([]byte(storedList[0]), &msg0); err != nil {
		t.Fatalf("failed to unmarshal stored message[0]: %v", err)
	}
	if msg0.Role != llm.RoleUser || msg0.Content != "Hello" {
		t.Errorf("stored message[0] = {Role:%q, Content:%q}, want {Role:%q, Content:%q}",
			msg0.Role, msg0.Content, llm.RoleUser, "Hello")
	}

	var msg1 llm.Message
	if err := json.Unmarshal([]byte(storedList[1]), &msg1); err != nil {
		t.Fatalf("failed to unmarshal stored message[1]: %v", err)
	}
	if msg1.Role != llm.RoleAssistant || msg1.Content != "Hi there!" {
		t.Errorf("stored message[1] = {Role:%q, Content:%q}, want {Role:%q, Content:%q}",
			msg1.Role, msg1.Content, llm.RoleAssistant, "Hi there!")
	}
}

func TestRedisStore_LoadMessages_List(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
	ctx := context.Background()

	sess, err := store.Create(ctx, "agent-load", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	// Manually push messages via RPush to simulate stored list data.
	key := "agentspec:session:" + sess.ID + ":messages"
	msg1, _ := json.Marshal(llm.Message{Role: llm.RoleUser, Content: "Question?"})
	msg2, _ := json.Marshal(llm.Message{Role: llm.RoleAssistant, Content: "Answer."})
	if err := client.RPush(ctx, key, string(msg1), string(msg2)); err != nil {
		t.Fatalf("RPush returned unexpected error: %v", err)
	}

	loaded, err := store.LoadMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadMessages returned unexpected error: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("LoadMessages returned %d messages, want 2", len(loaded))
	}

	if loaded[0].Role != llm.RoleUser || loaded[0].Content != "Question?" {
		t.Errorf("loaded[0] = {Role:%q, Content:%q}, want {Role:%q, Content:%q}",
			loaded[0].Role, loaded[0].Content, llm.RoleUser, "Question?")
	}

	if loaded[1].Role != llm.RoleAssistant || loaded[1].Content != "Answer." {
		t.Errorf("loaded[1] = {Role:%q, Content:%q}, want {Role:%q, Content:%q}",
			loaded[1].Role, loaded[1].Content, llm.RoleAssistant, "Answer.")
	}
}

func TestRedisStore_LoadMessages_None(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
	ctx := context.Background()

	// LoadMessages with no stored messages should return nil.
	loaded, err := store.LoadMessages(ctx, "nonexistent-session")
	if err != nil {
		t.Fatalf("LoadMessages returned unexpected error: %v", err)
	}

	if loaded != nil {
		t.Errorf("LoadMessages returned %v, want nil", loaded)
	}
}

func TestRedisStore_SaveMessagesEmpty(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
	ctx := context.Background()

	// SaveMessages with empty slice should return no error and write no data.
	if err := store.SaveMessages(ctx, "some-session", []llm.Message{}); err != nil {
		t.Fatalf("SaveMessages with empty slice returned unexpected error: %v", err)
	}

	key := "agentspec:session:some-session:messages"
	client.mu.Lock()
	_, exists := client.lists[key]
	client.mu.Unlock()

	if exists {
		t.Error("SaveMessages with empty slice should not create a list entry")
	}
}

func TestRedisStore_ListSkipsMessageKeys(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
	ctx := context.Background()

	// Create a session.
	sess, err := store.Create(ctx, "agent-skip", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	// Save messages (this creates a ":messages" key in the lists map).
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "Hello"},
	}
	if err := store.SaveMessages(ctx, sess.ID, messages); err != nil {
		t.Fatalf("SaveMessages returned unexpected error: %v", err)
	}

	// List should return only the session, not the messages key.
	listed, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}

	if len(listed) != 1 {
		t.Errorf("List returned %d sessions, want 1", len(listed))
	}

	if len(listed) > 0 && listed[0].ID != sess.ID {
		t.Errorf("List returned session ID %q, want %q", listed[0].ID, sess.ID)
	}
}

func TestRedisStore_LoadMessages_StringMigration(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
	ctx := context.Background()

	sess, err := store.Create(ctx, "agent-migrate", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	key := "agentspec:session:" + sess.ID + ":messages"

	// Simulate legacy string-based storage: store a JSON array of messages
	// as a plain string value and mark the key type as "string".
	legacyMessages := []llm.Message{
		{Role: llm.RoleUser, Content: "legacy question"},
		{Role: llm.RoleAssistant, Content: "legacy answer"},
		{Role: llm.RoleUser, Content: "follow-up"},
	}
	legacyData, err := json.Marshal(legacyMessages)
	if err != nil {
		t.Fatalf("failed to marshal legacy messages: %v", err)
	}

	client.mu.Lock()
	client.data[key] = string(legacyData)
	client.types[key] = "string"
	client.mu.Unlock()

	// LoadMessages should detect the string type and migrate to list format.
	loaded, err := store.LoadMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadMessages returned unexpected error: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("LoadMessages returned %d messages, want 3", len(loaded))
	}

	if loaded[0].Role != llm.RoleUser || loaded[0].Content != "legacy question" {
		t.Errorf("loaded[0] = {Role:%q, Content:%q}, want {Role:%q, Content:%q}",
			loaded[0].Role, loaded[0].Content, llm.RoleUser, "legacy question")
	}

	if loaded[1].Role != llm.RoleAssistant || loaded[1].Content != "legacy answer" {
		t.Errorf("loaded[1] = {Role:%q, Content:%q}, want {Role:%q, Content:%q}",
			loaded[1].Role, loaded[1].Content, llm.RoleAssistant, "legacy answer")
	}

	if loaded[2].Role != llm.RoleUser || loaded[2].Content != "follow-up" {
		t.Errorf("loaded[2] = {Role:%q, Content:%q}, want {Role:%q, Content:%q}",
			loaded[2].Role, loaded[2].Content, llm.RoleUser, "follow-up")
	}

	// Verify the old string key was deleted.
	client.mu.Lock()
	_, stringExists := client.data[key]
	client.mu.Unlock()
	if stringExists {
		t.Error("legacy string key should have been deleted after migration")
	}

	// Verify messages were migrated to list format (RPush'd).
	client.mu.Lock()
	listData, listExists := client.lists[key]
	client.mu.Unlock()
	if !listExists {
		t.Fatal("messages should have been migrated to list format")
	}
	if len(listData) != 3 {
		t.Errorf("migrated list has %d elements, want 3", len(listData))
	}
}

func TestRedisStore_LoadMessages_LRange(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
	ctx := context.Background()

	sess, err := store.Create(ctx, "agent-lrange", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	// Store messages individually via RPush (simulating how SaveMessages works).
	key := "agentspec:session:" + sess.ID + ":messages"
	msg1, _ := json.Marshal(llm.Message{Role: llm.RoleUser, Content: "first"})
	msg2, _ := json.Marshal(llm.Message{Role: llm.RoleAssistant, Content: "second"})
	msg3, _ := json.Marshal(llm.Message{Role: llm.RoleUser, Content: "third"})

	if err := client.RPush(ctx, key, string(msg1)); err != nil {
		t.Fatalf("RPush returned unexpected error: %v", err)
	}
	if err := client.RPush(ctx, key, string(msg2)); err != nil {
		t.Fatalf("RPush returned unexpected error: %v", err)
	}
	if err := client.RPush(ctx, key, string(msg3)); err != nil {
		t.Fatalf("RPush returned unexpected error: %v", err)
	}

	loaded, err := store.LoadMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadMessages returned unexpected error: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("LoadMessages returned %d messages, want 3", len(loaded))
	}

	expected := []struct {
		role    llm.Role
		content string
	}{
		{llm.RoleUser, "first"},
		{llm.RoleAssistant, "second"},
		{llm.RoleUser, "third"},
	}

	for i, exp := range expected {
		if loaded[i].Role != exp.role || loaded[i].Content != exp.content {
			t.Errorf("loaded[%d] = {Role:%q, Content:%q}, want {Role:%q, Content:%q}",
				i, loaded[i].Role, loaded[i].Content, exp.role, exp.content)
		}
	}
}

func TestRedisStore_SaveMessages_MarshalAndStore(t *testing.T) {
	client := newMockRedisClient()
	store := NewRedisStore(client)
	ctx := context.Background()

	sess, err := store.Create(ctx, "agent-marshal", nil)
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "You are a helpful assistant."},
		{Role: llm.RoleAssistant, Content: "What is 2+2?"},
		{Role: llm.RoleUser, Content: "4"},
	}

	if err := store.SaveMessages(ctx, sess.ID, messages); err != nil {
		t.Fatalf("SaveMessages returned unexpected error: %v", err)
	}

	// Verify all 3 messages are stored in the list.
	key := "agentspec:session:" + sess.ID + ":messages"
	client.mu.Lock()
	storedList := client.lists[key]
	client.mu.Unlock()

	if len(storedList) != 3 {
		t.Fatalf("stored list has %d elements, want 3", len(storedList))
	}

	// Verify round-trip: load them back and compare.
	loaded, err := store.LoadMessages(ctx, sess.ID)
	if err != nil {
		t.Fatalf("LoadMessages returned unexpected error: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("LoadMessages returned %d messages, want 3", len(loaded))
	}

	for i, orig := range messages {
		if loaded[i].Role != orig.Role || loaded[i].Content != orig.Content {
			t.Errorf("loaded[%d] = {Role:%q, Content:%q}, want {Role:%q, Content:%q}",
				i, loaded[i].Role, loaded[i].Content, orig.Role, orig.Content)
		}
	}
}
