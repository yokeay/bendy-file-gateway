package storage

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// Manager manages multiple storage drivers and routes operations to them.
type Manager struct {
	mu       sync.RWMutex
	backends map[string]*driverRef
}

type driverRef struct {
	Driver Driver
	Config map[string]string
}

var mgr = &Manager{
	backends: map[string]*driverRef{},
}

// Init initializes the storage manager.
func Init() {
	// Called at startup; actual backends are loaded lazily
}

// GetManager returns the global Manager.
func GetManager() *Manager {
	return mgr
}

// AddBackend registers a driver instance by backend ID.
func (m *Manager) AddBackend(backendID, driverName string, cfg map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	d, err := Create(driverName, cfg)
	if err != nil {
		return err
	}

	m.backends[backendID] = &driverRef{Driver: d, Config: cfg}
	return nil
}

// RemoveBackend removes a backend.
func (m *Manager) RemoveBackend(backendID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.backends, backendID)
}

// Put stores a file on the specified backend.
func (m *Manager) Put(ctx context.Context, backendID, key string, body io.Reader, opts UploadOptions) (FileInfo, error) {
	d, err := m.getDriver(backendID)
	if err != nil {
		return FileInfo{}, err
	}
	return d.Put(ctx, key, body, opts)
}

// Get retrieves a file from the specified backend.
func (m *Manager) Get(ctx context.Context, backendID, key string, opts DownloadOptions) (io.ReadCloser, FileInfo, error) {
	d, err := m.getDriver(backendID)
	if err != nil {
		return nil, FileInfo{}, err
	}
	return d.Get(ctx, key, opts)
}

// Head returns file metadata from the specified backend.
func (m *Manager) Head(ctx context.Context, backendID, key string) (FileInfo, error) {
	d, err := m.getDriver(backendID)
	if err != nil {
		return FileInfo{}, err
	}
	return d.Head(ctx, key)
}

// Delete removes a file from the specified backend.
func (m *Manager) Delete(ctx context.Context, backendID, key string) error {
	d, err := m.getDriver(backendID)
	if err != nil {
		return err
	}
	return d.Delete(ctx, key)
}

// Ping checks connectivity to a backend.
func (m *Manager) Ping(ctx context.Context, backendID string) error {
	d, err := m.getDriver(backendID)
	if err != nil {
		return err
	}
	return d.Ping(ctx)
}

func (m *Manager) getDriver(backendID string) (Driver, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ref, ok := m.backends[backendID]
	if !ok {
		return nil, fmt.Errorf("backend not found: %s", backendID)
	}
	return ref.Driver, nil
}
