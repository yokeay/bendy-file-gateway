package drivers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/bendy/file-gateway/internal/storage"
)

var memStore = struct {
	mu    sync.RWMutex
	blobs map[string]memBlob
}{
	blobs: map[string]memBlob{},
}

type memBlob struct {
	data        []byte
	contentType string
	metadata    map[string]string
	createdAt   time.Time
}

type memoryDriver struct {
	name string
}

func (d *memoryDriver) Name() string { return d.name }

func (d *memoryDriver) Put(_ context.Context, key string, body io.Reader, opts storage.UploadOptions) (storage.FileInfo, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return storage.FileInfo{}, err
	}

	hash := sha256.Sum256(data)
	etag := fmt.Sprintf("%x", hash)

	memStore.mu.Lock()
	memStore.blobs[key] = memBlob{
		data:        data,
		contentType: opts.ContentType,
		metadata:    opts.Metadata,
		createdAt:   time.Now(),
	}
	memStore.mu.Unlock()

	return storage.FileInfo{
		Key:          key,
		Size:         int64(len(data)),
		ContentType:  opts.ContentType,
		ETag:         etag,
		LastModified: time.Now(),
		Metadata:     opts.Metadata,
	}, nil
}

func (d *memoryDriver) Get(_ context.Context, key string, opts storage.DownloadOptions) (io.ReadCloser, storage.FileInfo, error) {
	memStore.mu.RLock()
	blob, ok := memStore.blobs[key]
	memStore.mu.RUnlock()
	if !ok {
		return nil, storage.FileInfo{}, fmt.Errorf("file not found: %s", key)
	}

	data := blob.data
	if opts.RangeStart > 0 || opts.RangeEnd > 0 {
		start := opts.RangeStart
		end := int64(len(data))
		if opts.RangeEnd > 0 && opts.RangeEnd < end {
			end = opts.RangeEnd
		}
		if start >= int64(len(data)) {
			return nil, storage.FileInfo{}, fmt.Errorf("range not satisfiable")
		}
		data = data[start:end]
	}

	return io.NopCloser(bytes.NewReader(data)), storage.FileInfo{
		Key:          key,
		Size:         int64(len(blob.data)),
		ContentType:  blob.contentType,
		LastModified: blob.createdAt,
		Metadata:     blob.metadata,
	}, nil
}

func (d *memoryDriver) Head(_ context.Context, key string) (storage.FileInfo, error) {
	memStore.mu.RLock()
	blob, ok := memStore.blobs[key]
	memStore.mu.RUnlock()
	if !ok {
		return storage.FileInfo{}, fmt.Errorf("file not found: %s", key)
	}
	return storage.FileInfo{
		Key:          key,
		Size:         int64(len(blob.data)),
		ContentType:  blob.contentType,
		LastModified: blob.createdAt,
		Metadata:     blob.metadata,
	}, nil
}

func (d *memoryDriver) Delete(_ context.Context, key string) error {
	memStore.mu.Lock()
	delete(memStore.blobs, key)
	memStore.mu.Unlock()
	return nil
}

func (d *memoryDriver) List(_ context.Context, prefix string, limit int, continuationToken string) ([]storage.FileInfo, string, error) {
	memStore.mu.RLock()
	defer memStore.mu.RUnlock()

	var infos []storage.FileInfo
	skip := continuationToken != ""

	for key, blob := range memStore.blobs {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		if skip {
			if key == continuationToken {
				skip = false
			}
			continue
		}
		infos = append(infos, storage.FileInfo{
			Key:          key,
			Size:         int64(len(blob.data)),
			ContentType:  blob.contentType,
			LastModified: blob.createdAt,
			Metadata:     blob.metadata,
		})
		if limit > 0 && len(infos) >= limit {
			break
		}
	}

	var nextToken string
	// Return the last key as continuation token if there might be more
	if limit > 0 && len(infos) == limit {
		nextToken = infos[len(infos)-1].Key
	}

	return infos, nextToken, nil
}

func (d *memoryDriver) Ping(_ context.Context) error { return nil }

func init() {
	storage.Register("memory", func(cfg map[string]string) (storage.Driver, error) {
		return &memoryDriver{name: "memory"}, nil
	})
}
