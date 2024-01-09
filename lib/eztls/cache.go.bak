package eztls

/*
 * cache.go
 * In memory autocert.Cache
 * By J. Stuart McMurray
 * Created 20231209
 * Last Modified 20231209
 */

import (
	"context"
	"fmt"
	"os"
	"sync"

	"golang.org/x/crypto/acme/autocert"
)

// MemCache is an in-memory implementation of autocert.Cache.
type MemCache struct{ m *sync.Map }

// NewMemCache returns a new MemCache, ready for use.
func NewMemCache() *MemCache { return &MemCache{m: new(sync.Map)} }

// Get returns a certificate data for the specified key.
// If there's no such key, Get returns ErrCacheMiss.
func (m *MemCache) Get(_ context.Context, key string) ([]byte, error) {
	v, ok := m.m.Load(key)
	if !ok {
		return nil, autocert.ErrCacheMiss
	}
	return v.([]byte), nil
}

// Put stores the data in the cache under the specified key.
// Underlying implementations may use any data storage format,
// as long as the reverse operation, Get, results in the original data.
// Put always returns nil.
func (m *MemCache) Put(_ context.Context, key string, data []byte) error {
	m.m.Store(key, data)
	return nil
}

// Delete removes a certificate data from the cache under the specified key.
// If there's no such key in the cache, Delete returns nil.  Delete will always
// return nil.
func (m *MemCache) Delete(_ context.Context, key string) error {
	m.m.Delete(key)
	return nil
}

// dirCache makes sure the directory exists and returns an autocert.DirCache
// which uses it.  This beats trying to figure out why LE cert provisioning
// is failing.
func dirCache(dir string) (autocert.Cache, error) {
	/* Make sure the directory exists. */
	if err := os.MkdirAll(dir, 0700); nil != err {
		return nil, fmt.Errorf("making directory: %w", err)
	}

	/* Use it. */
	return autocert.DirCache(dir), nil
}
