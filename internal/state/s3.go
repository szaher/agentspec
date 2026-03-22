package state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Backend implements Backend, HealthChecker, and Closer using S3-compatible object storage.
type S3Backend struct {
	client *s3.Client
	bucket string
	key    string
	mu     sync.RWMutex
	etag   *string // For optimistic concurrency control
}

// stateDocument is the JSON structure stored in S3.
type stateDocument struct {
	Entries []Entry `json:"entries"`
}

// NewS3Backend creates an S3Backend client.
// If endpoint is non-empty, uses it for S3-compatible stores (e.g., MinIO).
// If prefix is empty, defaults to root. Key is always "{prefix}state.json".
func NewS3Backend(ctx context.Context, bucket, region, prefix, endpoint string) (*S3Backend, error) {
	if bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}

	// Load default AWS config
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	var client *s3.Client
	if endpoint != "" {
		// Use custom endpoint for S3-compatible stores
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}

	// Construct key from prefix
	key := prefix + "state.json"
	if prefix == "" {
		key = "state.json"
	}

	return &S3Backend{
		client: client,
		bucket: bucket,
		key:    key,
	}, nil
}

// Load reads all state entries from S3.
func (s *S3Backend) Load() ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key),
	})
	if err != nil {
		// If object doesn't exist, return empty list
		return []Entry{}, nil
	}
	defer func() { _ = result.Body.Close() }()

	// Store ETag for future saves
	s.etag = result.ETag

	body, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object: %w", err)
	}

	var doc stateDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state document: %w", err)
	}

	return doc.Entries, nil
}

// Save writes all state entries to S3 as a single JSON object.
func (s *S3Backend) Save(entries []Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc := stateDocument{Entries: entries}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state document: %w", err)
	}

	ctx := context.Background()
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key),
		Body:   bytes.NewReader(data),
	}

	// Use ETag for optimistic concurrency if we have one
	if s.etag != nil {
		input.IfMatch = s.etag
	}

	result, err := s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put S3 object: %w", err)
	}

	// Update stored ETag
	s.etag = result.ETag

	return nil
}

// Get retrieves a single entry by FQN.
func (s *S3Backend) Get(fqn string) (*Entry, error) {
	entries, err := s.Load()
	if err != nil {
		return nil, err
	}

	for i := range entries {
		if entries[i].FQN == fqn {
			return &entries[i], nil
		}
	}

	return nil, fmt.Errorf("entry not found: %s", fqn)
}

// List returns all entries, optionally filtered by status.
func (s *S3Backend) List(status *Status) ([]Entry, error) {
	entries, err := s.Load()
	if err != nil {
		return nil, err
	}

	if status == nil {
		return entries, nil
	}

	var filtered []Entry
	for _, e := range entries {
		if e.Status == *status {
			filtered = append(filtered, e)
		}
	}

	return filtered, nil
}

// Ping verifies S3 bucket access via HeadBucket.
func (s *S3Backend) Ping(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to access S3 bucket: %w", err)
	}
	return nil
}

// Close is a no-op for S3 (client doesn't need cleanup).
func (s *S3Backend) Close() error {
	return nil
}
