package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// VaultResolver resolves secrets from a Vault-compatible HTTP key-value store.
type VaultResolver struct {
	// Address is the base URL of the Vault server.
	Address string

	// Token is the authentication token.
	Token string

	// MountPath is the KV v2 mount path (default: "secret").
	MountPath string

	// CacheTTL is how long to cache resolved secrets (default: 5 minutes).
	CacheTTL time.Duration

	client *http.Client
	mu     sync.RWMutex
	cache  map[string]cacheEntry
}

type cacheEntry struct {
	value   string
	expires time.Time
}

// NewVaultResolver creates a new Vault secret resolver.
func NewVaultResolver(address, token string) *VaultResolver {
	return &VaultResolver{
		Address:   strings.TrimRight(address, "/"),
		Token:     token,
		MountPath: "secret",
		CacheTTL:  5 * time.Minute,
		client:    &http.Client{Timeout: 10 * time.Second},
		cache:     make(map[string]cacheEntry),
	}
}

// Resolve fetches a secret from Vault.
// The ref format is "vault(path/to/secret#key)" or "vault(path/to/secret)".
// If no key is specified, the "value" key is used.
func (v *VaultResolver) Resolve(ctx context.Context, ref string) (string, error) {
	// Parse ref: vault(path#key)
	if !strings.HasPrefix(ref, "vault(") || !strings.HasSuffix(ref, ")") {
		return "", fmt.Errorf("invalid vault ref format: %s (expected vault(path#key))", ref)
	}

	inner := ref[6 : len(ref)-1] // strip vault( and )
	path, key := inner, "value"
	if idx := strings.Index(inner, "#"); idx >= 0 {
		path = inner[:idx]
		key = inner[idx+1:]
	}

	cacheKey := path + "#" + key

	// Check cache
	v.mu.RLock()
	if entry, ok := v.cache[cacheKey]; ok && time.Now().Before(entry.expires) {
		v.mu.RUnlock()
		return entry.value, nil
	}
	v.mu.RUnlock()

	// Fetch from Vault
	value, err := v.fetch(ctx, path, key)
	if err != nil {
		return "", err
	}

	// Cache the result
	v.mu.Lock()
	v.cache[cacheKey] = cacheEntry{
		value:   value,
		expires: time.Now().Add(v.CacheTTL),
	}
	v.mu.Unlock()

	return value, nil
}

func (v *VaultResolver) fetch(ctx context.Context, path, key string) (string, error) {
	// KV v2 read: GET /v1/{mount}/data/{path}
	url := fmt.Sprintf("%s/v1/%s/data/%s", v.Address, v.MountPath, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Vault-Token", v.Token)

	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("vault request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vault error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse KV v2 response
	var result struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse vault response: %w", err)
	}

	val, ok := result.Data.Data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in vault secret at %s", key, path)
	}

	s, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("vault key %q at %s is not a string", key, path)
	}

	return s, nil
}
