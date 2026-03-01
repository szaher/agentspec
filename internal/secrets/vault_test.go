package secrets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewVaultResolver_Defaults(t *testing.T) {
	v := NewVaultResolver("https://vault.example.com", "test-token")

	if v.MountPath != "secret" {
		t.Errorf("MountPath = %q, want %q", v.MountPath, "secret")
	}
	if v.CacheTTL != 5*time.Minute {
		t.Errorf("CacheTTL = %v, want %v", v.CacheTTL, 5*time.Minute)
	}
	if v.Address != "https://vault.example.com" {
		t.Errorf("Address = %q, want %q", v.Address, "https://vault.example.com")
	}
	if v.Token != "test-token" {
		t.Errorf("Token = %q, want %q", v.Token, "test-token")
	}
}

func TestNewVaultResolver_TrailingSlashTrimmed(t *testing.T) {
	v := NewVaultResolver("https://vault.example.com/", "tok")
	if v.Address != "https://vault.example.com" {
		t.Errorf("Address = %q, want trailing slash trimmed", v.Address)
	}
}

func TestVaultResolver_Resolve_ValidRefWithKey(t *testing.T) {
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		// Verify the request path: /v1/secret/data/myapp/config
		if want := "/v1/secret/data/myapp/config"; r.URL.Path != want {
			t.Errorf("request path = %q, want %q", r.URL.Path, want)
		}
		// Verify the token header
		if got := r.Header.Get("X-Vault-Token"); got != "test-token" {
			t.Errorf("X-Vault-Token = %q, want %q", got, "test-token")
		}

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"mykey": "myvalue",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	v := NewVaultResolver(srv.URL, "test-token")
	got, err := v.Resolve(context.Background(), "vault(myapp/config#mykey)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "myvalue" {
		t.Errorf("got %q, want %q", got, "myvalue")
	}
}

func TestVaultResolver_Resolve_RefWithoutKey_DefaultsToValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"value": "default-key-value",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	v := NewVaultResolver(srv.URL, "tok")
	got, err := v.Resolve(context.Background(), "vault(myapp/secret)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "default-key-value" {
		t.Errorf("got %q, want %q", got, "default-key-value")
	}
}

func TestVaultResolver_Resolve_MalformedRef(t *testing.T) {
	v := NewVaultResolver("https://vault.example.com", "tok")
	_, err := v.Resolve(context.Background(), "notavault(path)")
	if err == nil {
		t.Fatal("expected error for malformed ref, got nil")
	}
	if want := "invalid vault ref format"; !contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}

func TestVaultResolver_Resolve_CacheHit(t *testing.T) {
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)

		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"data": map[string]interface{}{
					"apikey": "cached-secret",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	v := NewVaultResolver(srv.URL, "tok")
	v.CacheTTL = 1 * time.Minute

	ctx := context.Background()

	// First call: should hit the server
	got1, err := v.Resolve(ctx, "vault(app/keys#apikey)")
	if err != nil {
		t.Fatalf("first resolve: unexpected error: %v", err)
	}
	if got1 != "cached-secret" {
		t.Errorf("first resolve: got %q, want %q", got1, "cached-secret")
	}

	// Second call: should return cached value, no new HTTP request
	got2, err := v.Resolve(ctx, "vault(app/keys#apikey)")
	if err != nil {
		t.Fatalf("second resolve: unexpected error: %v", err)
	}
	if got2 != "cached-secret" {
		t.Errorf("second resolve: got %q, want %q", got2, "cached-secret")
	}

	if count := requestCount.Load(); count != 1 {
		t.Errorf("expected 1 HTTP request, got %d", count)
	}
}

func TestVaultResolver_Resolve_VaultReturns404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":["secret not found"]}`))
	}))
	defer srv.Close()

	v := NewVaultResolver(srv.URL, "tok")
	_, err := v.Resolve(context.Background(), "vault(nonexistent/path#key)")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	if want := "vault error (status 404)"; !contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err.Error(), want)
	}
}
