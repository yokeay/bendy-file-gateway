package storage

import (
	"context"
	"io"
	"time"
)

// FileInfo represents metadata about a stored file.
type FileInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
	Metadata     map[string]string
}

// UploadOptions configures an upload operation.
type UploadOptions struct {
	ContentType string
	Metadata    map[string]string
}

// DownloadOptions configures a download operation.
type DownloadOptions struct {
	RangeStart int64
	RangeEnd   int64
}

// Driver is the interface that every storage backend must implement.
type Driver interface {
	// Name returns the driver identifier (e.g., "s3", "aliyun_oss").
	Name() string

	// Put stores data and returns the storage key and metadata.
	Put(ctx context.Context, key string, body io.Reader, opts UploadOptions) (FileInfo, error)

	// Get retrieves file content. Caller must close the returned ReadCloser.
	Get(ctx context.Context, key string, opts DownloadOptions) (io.ReadCloser, FileInfo, error)

	// Head returns file metadata without downloading the body.
	Head(ctx context.Context, key string) (FileInfo, error)

	// Delete removes a file. Should return nil if file does not exist.
	Delete(ctx context.Context, key string) error

	// List returns files matching a prefix, up to the specified limit.
	List(ctx context.Context, prefix string, limit int, continuationToken string) ([]FileInfo, string, error)

	// Ping checks connectivity to the backend.
	Ping(ctx context.Context) error
}

// Factory creates a Driver from configuration.
type Factory func(cfg map[string]string) (Driver, error)
