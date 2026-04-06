package vault

import (
	"context"
	"sync"
	"time"
)

// cacheEntry holds a cached value with its expiration time.
type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

func (e *cacheEntry) expired() bool {
	return time.Now().After(e.expiresAt)
}

// CachingClient wraps a ClientInterface and caches List, Read, and
// PathExists responses in memory. When the underlying client returns an
// error (e.g. network failure) and a non-expired cache entry exists, the
// cached value is returned instead of the error. This provides resilience
// against transient connectivity issues.
//
// Cache entries expire after the configured TTL. Successful responses
// always refresh the cache.
type CachingClient struct {
	inner ClientInterface
	ttl   time.Duration

	mu    sync.RWMutex
	store map[string]*cacheEntry
}

// NewCachingClient creates a CachingClient that wraps inner. The ttl
// parameter controls how long cached responses remain valid.
func NewCachingClient(inner ClientInterface, ttl time.Duration) *CachingClient {
	return &CachingClient{
		inner: inner,
		ttl:   ttl,
		store: make(map[string]*cacheEntry),
	}
}

// get returns a non-expired cache entry or nil.
func (c *CachingClient) get(key string) *cacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if e, ok := c.store[key]; ok && !e.expired() {
		return e
	}
	return nil
}

// set writes a value into the cache with the configured TTL.
func (c *CachingClient) set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Ping delegates to the inner client without caching.
func (c *CachingClient) Ping(ctx context.Context) error {
	return c.inner.Ping(ctx)
}

// List retrieves keys at path. On error, returns a cached result if available.
func (c *CachingClient) List(ctx context.Context, path string) ([]string, error) {
	key := "list:" + path
	result, err := c.inner.List(ctx, path)
	if err == nil {
		c.set(key, result)
		return result, nil
	}
	if e := c.get(key); e != nil {
		return e.value.([]string), nil
	}
	return nil, err
}

// Read retrieves secret data at path. On error, returns a cached result if available.
func (c *CachingClient) Read(ctx context.Context, path string) (map[string]interface{}, error) {
	key := "read:" + path
	result, err := c.inner.Read(ctx, path)
	if err == nil {
		c.set(key, result)
		return result, nil
	}
	if e := c.get(key); e != nil {
		return e.value.(map[string]interface{}), nil
	}
	return nil, err
}

// PathExists checks path existence. On error, returns a cached result if available.
func (c *CachingClient) PathExists(ctx context.Context, path string) (bool, bool, error) {
	type pathExistsResult struct {
		exists bool
		isDir  bool
	}
	key := "pathexists:" + path
	exists, isDir, err := c.inner.PathExists(ctx, path)
	if err == nil {
		c.set(key, pathExistsResult{exists: exists, isDir: isDir})
		return exists, isDir, nil
	}
	if e := c.get(key); e != nil {
		r := e.value.(pathExistsResult)
		return r.exists, r.isDir, nil
	}
	return false, false, err
}

// ListMounts delegates to the inner client. On error, returns a cached result if available.
func (c *CachingClient) ListMounts(ctx context.Context) (map[string]MountInfo, error) {
	key := "listmounts"
	result, err := c.inner.ListMounts(ctx)
	if err == nil {
		c.set(key, result)
		return result, nil
	}
	if e := c.get(key); e != nil {
		return e.value.(map[string]MountInfo), nil
	}
	return nil, err
}

// RefreshMounts delegates to the inner client (no caching on this operation).
func (c *CachingClient) RefreshMounts(ctx context.Context) error {
	return c.inner.RefreshMounts(ctx)
}

// SetToken delegates to the inner client.
func (c *CachingClient) SetToken(token string) {
	c.inner.SetToken(token)
}
