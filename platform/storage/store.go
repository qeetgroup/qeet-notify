package storage

import "context"

// ObjectStore is the interface every object-storage backend must satisfy.
// The default implementation targets AWS S3 / MinIO; swap for GCS or Azure Blob
// by providing an alternate implementation.
type ObjectStore interface {
	// Put uploads data under key in the configured bucket.
	Put(ctx context.Context, key string, data []byte, contentType string) error
	// Get retrieves the object at key.
	Get(ctx context.Context, key string) ([]byte, error)
	// Delete removes the object at key.
	Delete(ctx context.Context, key string) error
	// PresignGet returns a time-limited URL for direct download of key.
	PresignGet(ctx context.Context, key string, ttlSeconds int) (string, error)
}
