package state

import (
	"errors"
	"testing"
)

func TestAvailable(t *testing.T) {
	available := Available()

	// Must contain at least "local" (registered in init())
	found := false
	for _, name := range available {
		if name == "local" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Available() missing 'local' backend; got %v", available)
	}

	// Should be sorted
	if len(available) > 1 {
		for i := 1; i < len(available); i++ {
			if available[i-1] >= available[i] {
				t.Errorf("Available() not sorted: %v", available)
				break
			}
		}
	}

	// Should contain all init-registered backends
	expectedBackends := []string{"local", "etcd", "postgres", "s3", "kubernetes"}
	for _, expected := range expectedBackends {
		found := false
		for _, name := range available {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Available() missing expected backend %q; got %v", expected, available)
		}
	}
}

func TestNewLocal(t *testing.T) {
	// Create with default path
	backend, err := New("local", nil)
	if err != nil {
		t.Fatalf("New('local', nil) failed: %v", err)
	}
	if backend == nil {
		t.Fatal("New('local', nil) returned nil backend")
	}

	// Create with custom path
	backend, err = New("local", map[string]string{"path": "/tmp/test.json"})
	if err != nil {
		t.Fatalf("New('local', custom path) failed: %v", err)
	}
	if backend == nil {
		t.Fatal("New('local', custom path) returned nil backend")
	}
}

func TestNewUnknownType(t *testing.T) {
	backend, err := New("unknown-type", nil)
	if err == nil {
		t.Fatal("New('unknown-type') should return error, got nil")
	}
	if backend != nil {
		t.Errorf("New('unknown-type') should return nil backend, got %v", backend)
	}

	// Error message should mention available types
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error message should not be empty")
	}

	// Should contain "unknown-type" and mention available types
	if !contains(errMsg, "unknown-type") {
		t.Errorf("Error message should mention the unknown type; got: %s", errMsg)
	}
	if !contains(errMsg, "available") {
		t.Errorf("Error message should mention available types; got: %s", errMsg)
	}
}

func TestRegisterCustomBackend(t *testing.T) {
	// Register a custom backend
	customBackendCalled := false
	Register("test-custom", func(props map[string]string) (Backend, error) {
		customBackendCalled = true
		if props["fail"] == "true" {
			return nil, errors.New("custom factory error")
		}
		// Return a mock backend (local backend is fine for testing)
		return NewLocalBackend(".test.json"), nil
	})

	// Verify it appears in Available()
	available := Available()
	found := false
	for _, name := range available {
		if name == "test-custom" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Custom backend 'test-custom' not found in Available(); got %v", available)
	}

	// Create backend using custom factory
	backend, err := New("test-custom", nil)
	if err != nil {
		t.Fatalf("New('test-custom') failed: %v", err)
	}
	if backend == nil {
		t.Fatal("New('test-custom') returned nil backend")
	}
	if !customBackendCalled {
		t.Error("Custom factory was not called")
	}

	// Test factory error handling
	customBackendCalled = false
	backend, err = New("test-custom", map[string]string{"fail": "true"})
	if err == nil {
		t.Fatal("Expected error from custom factory, got nil")
	}
	if backend != nil {
		t.Errorf("Expected nil backend on factory error, got %v", backend)
	}
	if !customBackendCalled {
		t.Error("Custom factory was not called on error case")
	}
}

func TestRegisterOverwrite(t *testing.T) {
	// Register a backend
	callCount := 0
	Register("test-overwrite", func(props map[string]string) (Backend, error) {
		callCount++
		return NewLocalBackend(".test1.json"), nil
	})

	// Overwrite with a different factory
	Register("test-overwrite", func(props map[string]string) (Backend, error) {
		callCount += 10
		return NewLocalBackend(".test2.json"), nil
	})

	// Create backend - should use the second factory
	_, err := New("test-overwrite", nil)
	if err != nil {
		t.Fatalf("New('test-overwrite') failed: %v", err)
	}

	// Should have called the second factory (adds 10), not the first (adds 1)
	if callCount != 10 {
		t.Errorf("Expected callCount=10 (second factory), got %d", callCount)
	}
}

func TestNewWithNilProps(t *testing.T) {
	// Ensure nil props map is handled gracefully
	backend, err := New("local", nil)
	if err != nil {
		t.Fatalf("New('local', nil) should succeed, got error: %v", err)
	}
	if backend == nil {
		t.Fatal("New('local', nil) returned nil backend")
	}
}

func TestNewWithEmptyProps(t *testing.T) {
	// Ensure empty props map is handled gracefully
	backend, err := New("local", map[string]string{})
	if err != nil {
		t.Fatalf("New('local', empty props) should succeed, got error: %v", err)
	}
	if backend == nil {
		t.Fatal("New('local', empty props) returned nil backend")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
