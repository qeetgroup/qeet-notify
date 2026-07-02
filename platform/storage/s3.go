package storage

import "context"

// S3Config holds the credentials and addressing for an S3-compatible object store.
// Set Endpoint to a MinIO URL (e.g. "http://localhost:9000") for local development;
// leave it empty for AWS S3.
//
// Add github.com/aws/aws-sdk-go-v2/service/s3 to go.mod, then fill S3Store
// with a real aws s3.Client implementation.
type S3Config struct {
	Bucket   string
	Endpoint string // MinIO endpoint or "" for AWS
	Region   string
	// Credentials are picked up from the environment via aws config.LoadDefaultConfig.
}

// S3Store is a placeholder ObjectStore backed by S3 / MinIO.
// TODO: replace stub methods with aws-sdk-go-v2/service/s3 calls.
type S3Store struct {
	cfg S3Config
}

// NewS3Store creates an S3Store from cfg.
func NewS3Store(cfg S3Config) (*S3Store, error) {
	return &S3Store{cfg: cfg}, nil
}

func (s *S3Store) Put(_ context.Context, _ string, _ []byte, _ string) error { return nil }
func (s *S3Store) Get(_ context.Context, _ string) ([]byte, error)            { return nil, nil }
func (s *S3Store) Delete(_ context.Context, _ string) error                   { return nil }
func (s *S3Store) PresignGet(_ context.Context, _ string, _ int) (string, error) {
	return "", nil
}
