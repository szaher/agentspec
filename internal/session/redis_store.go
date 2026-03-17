package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
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
	Scan(ctx context.Context, cursor uint64, pattern string, count int64) (keys []string, nextCursor uint64, err error)
	RPush(ctx context.Context, key string, values ...string) error
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
}

// RedisStore implements the session Store interface backed by Redis.
type RedisStore struct {
	client RedisClient
	prefix string
	ttl    time.Duration
	logger *slog.Logger
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

// SetLogger sets the structured logger for diagnostics.
func (s *RedisStore) SetLogger(logger *slog.Logger) {
	s.logger = logger
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
		ID:         generateSessionID(),
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
// Uses cursor-based SCAN instead of KEYS for production safety.
func (s *RedisStore) List(ctx context.Context, agentName string) ([]*Session, error) {
	var keys []string
	var cursor uint64
	for {
		batch, next, err := s.client.Scan(ctx, cursor, s.prefix+"*", 100)
		if err != nil {
			return nil, fmt.Errorf("redis scan: %w", err)
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}

	var sessions []*Session
	for _, key := range keys {
		// Skip message keys
		if strings.HasSuffix(key, ":messages") {
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

	if s.logger != nil {
		s.logger.Info("redis session list",
			slog.Int("count", len(sessions)),
		)
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

// SaveMessages appends conversation messages to the session's message list
// using RPush for O(1) append performance. Each message is individually
// JSON-serialized and pushed.
func (s *RedisStore) SaveMessages(ctx context.Context, sessionID string, messages []llm.Message) error {
	if len(messages) == 0 {
		return nil
	}

	key := s.messagesKey(sessionID)
	values := make([]string, 0, len(messages))
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("marshal message: %w", err)
		}
		values = append(values, string(data))
	}

	return s.client.RPush(ctx, key, values...)
}

// LoadMessages loads conversation messages for a session using LRange.
// If the key holds a legacy String-type value (from the old Set-based storage),
// it falls back to Get(), parses the JSON array, deletes the old key, and
// re-stores each message via RPush for transparent migration.
func (s *RedisStore) LoadMessages(ctx context.Context, sessionID string) ([]llm.Message, error) {
	key := s.messagesKey(sessionID)

	// Try LRange first (expected path for new-format data).
	elements, err := s.client.LRange(ctx, key, 0, -1)
	if err == nil {
		// LRange succeeded — key is a list (or does not exist, returning empty).
		if len(elements) == 0 {
			return nil, nil
		}
		messages := make([]llm.Message, 0, len(elements))
		for _, elem := range elements {
			var msg llm.Message
			if err := json.Unmarshal([]byte(elem), &msg); err != nil {
				return nil, fmt.Errorf("unmarshal message element: %w", err)
			}
			messages = append(messages, msg)
		}
		return messages, nil
	}

	// LRange failed — key may hold a legacy string value (WRONGTYPE error).
	// Fall back to Get() for migration.
	data, getErr := s.client.Get(ctx, key)
	if getErr != nil {
		// Key does not exist at all.
		return nil, nil
	}

	var messages []llm.Message
	if err := json.Unmarshal([]byte(data), &messages); err != nil {
		return nil, fmt.Errorf("unmarshal legacy messages: %w", err)
	}

	// Migrate: delete the old string key and re-store as a list.
	_ = s.client.Del(ctx, key)
	if len(messages) > 0 {
		values := make([]string, 0, len(messages))
		for _, msg := range messages {
			d, err := json.Marshal(msg)
			if err != nil {
				return messages, nil // Return what we have; migration is best-effort.
			}
			values = append(values, string(d))
		}
		_ = s.client.RPush(ctx, key, values...)
	}

	return messages, nil
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}
