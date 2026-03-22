package state

import (
	"context"
	"testing"
)

// TestS3BackendInterfaceCompliance verifies S3Backend implements required interfaces.
func TestS3BackendInterfaceCompliance(t *testing.T) {
	// This test verifies that S3Backend implements the interfaces at compile time.
	// We don't instantiate because that requires AWS credentials/connectivity.
	var _ Backend = (*S3Backend)(nil)
	var _ HealthChecker = (*S3Backend)(nil)
	var _ Closer = (*S3Backend)(nil)
}

// TestNewS3BackendEmptyBucket verifies that NewS3Backend returns an error when bucket is empty.
func TestNewS3BackendEmptyBucket(t *testing.T) {
	ctx := context.Background()
	_, err := NewS3Backend(ctx, "", "us-east-1", "", "")
	if err == nil {
		t.Fatal("expected error when bucket is empty, got nil")
	}
	if err.Error() != "bucket is required" {
		t.Errorf("expected error 'bucket is required', got: %v", err)
	}
}
