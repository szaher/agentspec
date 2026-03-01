package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
)

// RedisClient is the interface for Redis operations needed by the session store.
// This abstracts the actual Redis client library.
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
	// List operations for atomic message storage
	RPush(ctx context.Context, key string, values ...string) error
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	Type(ctx context.Context, key string) (string, error)
}

// RedisStore implements the session Store interface backed by Redis.
type RedisStore struct {
	client RedisClient
	prefix string
	ttl    time.Duration
}

// RedisStoreOption configures a RedisStore.
type RedisStoreOption func(*RedisStore)

// WithPrefix sets the key prefix for session keys.
func WithPrefix(prefix string) RedisStoreOption {
	return func(s *RedisStore) { s.prefix = prefix }
}

// WithTTL sets the session TTL.
func WithTTL(ttl time.Duration) RedisStoreOption {
	return func(s *RedisStore) { s.ttl = ttl }
}

// NewRedisStore creates a new Redis-backed session store.
func NewRedisStore(client RedisClient, opts ...RedisStoreOption) *RedisStore {
	s := &RedisStore{
		client: client,
		prefix: "agentspec:session:",
		ttl:    24 * time.Hour,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *RedisStore) sessionKey(id string) string {
	return s.prefix + id
}

func (s *RedisStore) messagesKey(id string) string {
	return s.prefix + id + ":messages"
}

// Create creates a new session.
func (s *RedisStore) Create(ctx context.Context, agentName string, metadata map[string]string) (*Session, error) {
	sess := &Session{
		ID:         generateSecureID(),
		AgentName:  agentName,
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
		Metadata:   metadata,
	}

	data, err := json.Marshal(sess)
	if err != nil {
		return nil, fmt.Errorf("marshal session: %w", err)
	}

	if err := s.client.Set(ctx, s.sessionKey(sess.ID), string(data), s.ttl); err != nil {
		return nil, fmt.Errorf("redis set: %w", err)
	}

	return sess, nil
}

// Get retrieves a session by ID.
func (s *RedisStore) Get(ctx context.Context, id string) (*Session, error) {
	data, err := s.client.Get(ctx, s.sessionKey(id))
	if err != nil {
		return nil, fmt.Errorf("session %q not found", id)
	}

	var sess Session
	if err := json.Unmarshal([]byte(data), &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}

	return &sess, nil
}

// Delete removes a session by ID.
func (s *RedisStore) Delete(ctx context.Context, id string) error {
	return s.client.Del(ctx, s.sessionKey(id), s.messagesKey(id))
}

// List returns all sessions, optionally filtered by agent name.
func (s *RedisStore) List(ctx context.Context, agentName string) ([]*Session, error) {
	keys, err := s.client.Keys(ctx, s.prefix+"*")
	if err != nil {
		return nil, fmt.Errorf("redis keys: %w", err)
	}

	var sessions []*Session
	for _, key := range keys {
		// Skip message keys
		if len(key) > len(":messages") && key[len(key)-len(":messages"):] == ":messages" {
			continue
		}

		data, err := s.client.Get(ctx, key)
		if err != nil {
			continue
		}

		var sess Session
		if err := json.Unmarshal([]byte(data), &sess); err != nil {
			continue
		}

		if agentName == "" || sess.AgentName == agentName {
			sessions = append(sessions, &sess)
		}
	}

	return sessions, nil
}

// Touch updates the last active timestamp.
func (s *RedisStore) Touch(ctx context.Context, id string) error {
	sess, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	sess.LastActive = time.Now()
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, s.sessionKey(id), string(data), s.ttl)
}

// SaveMessages appends conversation messages atomically using RPUSH.
func (s *RedisStore) SaveMessages(ctx context.Context, sessionID string, messages []llm.Message) error {
	if len(messages) == 0 {
		return nil
	}

	key := s.messagesKey(sessionID)

	// Marshal each message individually for RPUSH
	values := make([]string, 0, len(messages))
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			slog.Warn("failed to marshal message, skipping", "session_id", sessionID, "error", err)
			continue
		}
		values = append(values, string(data))
	}

	if len(values) == 0 {
		return nil
	}

	// Atomic append via RPUSH
	if err := s.client.RPush(ctx, key, values...); err != nil {
		return fmt.Errorf("rpush messages: %w", err)
	}

	// Refresh TTL
	if err := s.client.Expire(ctx, key, s.ttl); err != nil {
		return fmt.Errorf("expire messages key: %w", err)
	}

	return nil
}

// LoadMessages retrieves conversation messages using LRANGE.
// Transparently migrates existing string-based sessions to list format.
func (s *RedisStore) LoadMessages(ctx context.Context, sessionID string) ([]llm.Message, error) {
	key := s.messagesKey(sessionID)

	// Check key type for migration from string to list
	keyType, err := s.client.Type(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("check key type: %w", err)
	}

	// Migrate string-based sessions to list format
	if keyType == "string" {
		return s.migrateStringToList(ctx, sessionID, key)
	}

	// Key doesn't exist yet
	if keyType == "none" {
		return nil, nil
	}

	// Load from list
	elements, err := s.client.LRange(ctx, key, 0, -1)
	if err != nil {
		return nil, fmt.Errorf("lrange messages: %w", err)
	}

	messages := make([]llm.Message, 0, len(elements))
	for _, elem := range elements {
		var msg llm.Message
		if err := json.Unmarshal([]byte(elem), &msg); err != nil {
			slog.Warn("failed to unmarshal message, skipping", "session_id", sessionID, "error", err)
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// migrateStringToList migrates a string-based session key to list format.
func (s *RedisStore) migrateStringToList(ctx context.Context, sessionID, key string) ([]llm.Message, error) {
	// Read the string value
	data, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, nil // Key gone, treat as empty
	}

	var messages []llm.Message
	if err := json.Unmarshal([]byte(data), &messages); err != nil {
		slog.Warn("failed to unmarshal legacy string messages during migration", "session_id", sessionID, "error", err)
		return nil, nil
	}

	// Delete the string key
	if err := s.client.Del(ctx, key); err != nil {
		return messages, nil // Return messages even if delete fails
	}

	// RPUSH each message to the new list key
	for _, msg := range messages {
		msgData, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		if err := s.client.RPush(ctx, key, string(msgData)); err != nil {
			slog.Warn("failed to push message during migration", "session_id", sessionID, "error", err)
			break
		}
	}

	// Set TTL on new list key
	_ = s.client.Expire(ctx, key, s.ttl)

	slog.Info("migrated session messages to list format", "session_id", sessionID, "message_count", len(messages))
	return messages, nil
}
