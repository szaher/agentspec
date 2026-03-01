package session

import (
	"context"
	"encoding/json"
	"fmt"
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

// SaveMessages saves conversation messages for a session.
func (s *RedisStore) SaveMessages(ctx context.Context, sessionID string, messages []llm.Message) error {
	// Load existing messages
	existing, _ := s.LoadMessages(ctx, sessionID)
	all := append(existing, messages...)

	data, err := json.Marshal(all)
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}

	return s.client.Set(ctx, s.messagesKey(sessionID), string(data), s.ttl)
}

// LoadMessages loads conversation messages for a session.
func (s *RedisStore) LoadMessages(ctx context.Context, sessionID string) ([]llm.Message, error) {
	data, err := s.client.Get(ctx, s.messagesKey(sessionID))
	if err != nil {
		return nil, nil // No messages yet
	}

	var messages []llm.Message
	if err := json.Unmarshal([]byte(data), &messages); err != nil {
		return nil, fmt.Errorf("unmarshal messages: %w", err)
	}

	return messages, nil
}
