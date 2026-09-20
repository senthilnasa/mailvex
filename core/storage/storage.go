package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	ErrNotFound = errors.New("object not found")
)

type Storage interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// MemoryStorage is an in-memory thread-safe storage implementation for tests & dev
type MemoryStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string][]byte),
	}
}

func (m *MemoryStorage) Put(ctx context.Context, key string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = data
	return nil
}

func (m *MemoryStorage) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.data[key]
	if !ok {
		return nil, ErrNotFound
	}
	return val, nil
}

func (m *MemoryStorage) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

// FSStorage stores objects on local filesystem (for single-server & homelab profiles)
type FSStorage struct {
	baseDir string
}

func NewFSStorage(baseDir string) (*FSStorage, error) {
	if err := os.MkdirAll(baseDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to initialize storage directory: %w", err)
	}
	return &FSStorage{baseDir: baseDir}, nil
}

func (fs *FSStorage) Put(ctx context.Context, key string, data []byte) error {
	fullPath := filepath.Join(fs.baseDir, filepath.Clean(key))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
		return err
	}
	return os.WriteFile(fullPath, data, 0600)
}

func (fs *FSStorage) Get(ctx context.Context, key string) ([]byte, error) {
	fullPath := filepath.Join(fs.baseDir, filepath.Clean(key))
	data, err := os.ReadFile(fullPath)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	return data, err
}

func (fs *FSStorage) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(fs.baseDir, filepath.Clean(key))
	err := os.Remove(fullPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
