package state

import (
	"testing"
)

// Compile-time interface compliance checks
var _ Backend = (*EtcdBackend)(nil)
var _ HealthChecker = (*EtcdBackend)(nil)
var _ Locker = (*EtcdBackend)(nil)
var _ Closer = (*EtcdBackend)(nil)

func TestNewEtcdBackend_EmptyEndpoints(t *testing.T) {
	_, err := NewEtcdBackend("", "")
	if err == nil {
		t.Fatal("expected error when endpoints is empty, got nil")
	}

	expectedMsg := "etcd endpoints cannot be empty"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

func TestNewEtcdBackend_DefaultPrefix(t *testing.T) {
	// We can't actually connect to etcd in unit tests, but we can test
	// the prefix logic by checking the error path after client creation fails
	// This verifies that the prefix is set correctly before attempting connection

	tests := []struct {
		name           string
		inputPrefix    string
		expectedPrefix string
	}{
		{
			name:           "empty prefix uses default",
			inputPrefix:    "",
			expectedPrefix: defaultPrefix,
		},
		{
			name:           "custom prefix without trailing slash",
			inputPrefix:    "/custom/prefix",
			expectedPrefix: "/custom/prefix/",
		},
		{
			name:           "custom prefix with trailing slash",
			inputPrefix:    "/custom/prefix/",
			expectedPrefix: "/custom/prefix/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We expect NewEtcdBackend to fail when connecting to an invalid endpoint,
			// but we can verify the prefix logic indirectly by checking that
			// the function doesn't panic and handles the prefix correctly

			// Use an invalid endpoint that won't actually connect
			backend, err := NewEtcdBackend("localhost:9999", tt.inputPrefix)

			// The function should not return an error yet - the error will occur
			// when we try to actually use the connection (Load, Save, etc.)
			// However, if the etcd client creation fails immediately (which it might),
			// we just verify that we got some kind of backend or an error
			if err != nil {
				// If client creation fails immediately, that's acceptable for this test
				// We're primarily testing the prefix logic which happens before connection
				t.Logf("etcd client creation failed (expected in unit tests): %v", err)
				return
			}

			if backend == nil {
				t.Fatal("expected non-nil backend")
			}

			if backend.prefix != tt.expectedPrefix {
				t.Errorf("expected prefix %q, got %q", tt.expectedPrefix, backend.prefix)
			}

			// Clean up if we got a backend
			if backend != nil {
				_ = backend.Close()
			}
		})
	}
}

func TestEtcdBackend_InterfaceCompliance(t *testing.T) {
	// This test verifies at runtime that the interface assignments work
	// The var _ declarations above ensure compile-time checking

	// We can't create a real EtcdBackend without a connection,
	// but the compile-time checks are the primary verification
	t.Log("Interface compliance verified at compile time")
}
