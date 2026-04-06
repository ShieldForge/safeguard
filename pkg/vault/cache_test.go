package vault

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockClient is a minimal ClientInterface for testing the caching layer.
type mockClient struct {
	listFunc       func(ctx context.Context, path string) ([]string, error)
	readFunc       func(ctx context.Context, path string) (map[string]interface{}, error)
	pathExistsFunc func(ctx context.Context, path string) (bool, bool, error)
	listMountsFunc func(ctx context.Context) (map[string]MountInfo, error)
	pingFunc       func(ctx context.Context) error
	token          string
}

func (m *mockClient) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

func (m *mockClient) List(ctx context.Context, path string) ([]string, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, path)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockClient) Read(ctx context.Context, path string) (map[string]interface{}, error) {
	if m.readFunc != nil {
		return m.readFunc(ctx, path)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockClient) PathExists(ctx context.Context, path string) (bool, bool, error) {
	if m.pathExistsFunc != nil {
		return m.pathExistsFunc(ctx, path)
	}
	return false, false, fmt.Errorf("not implemented")
}

func (m *mockClient) ListMounts(ctx context.Context) (map[string]MountInfo, error) {
	if m.listMountsFunc != nil {
		return m.listMountsFunc(ctx)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockClient) RefreshMounts(ctx context.Context) error {
	return nil
}

func (m *mockClient) SetToken(token string) {
	m.token = token
}

func TestCachingClient_ReadCachesSuccess(t *testing.T) {
	calls := 0
	inner := &mockClient{
		readFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
			calls++
			return map[string]interface{}{"key": "value"}, nil
		},
	}

	cc := NewCachingClient(inner, 5*time.Second)
	ctx := context.Background()

	// First call hits inner
	data, err := cc.Read(ctx, "secret/app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["key"] != "value" {
		t.Fatalf("unexpected data: %v", data)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}

	// Make inner fail — should return cached value
	inner.readFunc = func(ctx context.Context, path string) (map[string]interface{}, error) {
		calls++
		return nil, fmt.Errorf("network error")
	}

	data, err = cc.Read(ctx, "secret/app")
	if err != nil {
		t.Fatalf("expected cached result, got error: %v", err)
	}
	if data["key"] != "value" {
		t.Fatalf("unexpected cached data: %v", data)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestCachingClient_ReadCacheExpires(t *testing.T) {
	calls := 0
	inner := &mockClient{
		readFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
			calls++
			return map[string]interface{}{"key": "value"}, nil
		},
	}

	cc := NewCachingClient(inner, 50*time.Millisecond)
	ctx := context.Background()

	_, err := cc.Read(ctx, "secret/app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Make inner fail — cache expired, so error should propagate
	inner.readFunc = func(ctx context.Context, path string) (map[string]interface{}, error) {
		calls++
		return nil, fmt.Errorf("network error")
	}

	_, err = cc.Read(ctx, "secret/app")
	if err == nil {
		t.Fatal("expected error after cache expiry, got nil")
	}
}

func TestCachingClient_ListCachesSuccess(t *testing.T) {
	calls := 0
	inner := &mockClient{
		listFunc: func(ctx context.Context, path string) ([]string, error) {
			calls++
			return []string{"a/", "b"}, nil
		},
	}

	cc := NewCachingClient(inner, 5*time.Second)
	ctx := context.Background()

	list, err := cc.List(ctx, "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}

	// Make inner fail
	inner.listFunc = func(ctx context.Context, path string) ([]string, error) {
		calls++
		return nil, fmt.Errorf("network error")
	}

	list, err = cc.List(ctx, "secret")
	if err != nil {
		t.Fatalf("expected cached result, got error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 cached entries, got %d", len(list))
	}
}

func TestCachingClient_PathExistsCachesSuccess(t *testing.T) {
	calls := 0
	inner := &mockClient{
		pathExistsFunc: func(ctx context.Context, path string) (bool, bool, error) {
			calls++
			return true, true, nil
		},
	}

	cc := NewCachingClient(inner, 5*time.Second)
	ctx := context.Background()

	exists, isDir, err := cc.PathExists(ctx, "secret/app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists || !isDir {
		t.Fatalf("expected exists=true isDir=true, got %v %v", exists, isDir)
	}

	// Make inner fail
	inner.pathExistsFunc = func(ctx context.Context, path string) (bool, bool, error) {
		calls++
		return false, false, fmt.Errorf("network error")
	}

	exists, isDir, err = cc.PathExists(ctx, "secret/app")
	if err != nil {
		t.Fatalf("expected cached result, got error: %v", err)
	}
	if !exists || !isDir {
		t.Fatalf("expected cached exists=true isDir=true, got %v %v", exists, isDir)
	}
}

func TestCachingClient_ListMountsCachesSuccess(t *testing.T) {
	calls := 0
	inner := &mockClient{
		listMountsFunc: func(ctx context.Context) (map[string]MountInfo, error) {
			calls++
			return map[string]MountInfo{
				"secret": {Type: "kv", Path: "secret"},
			}, nil
		},
	}

	cc := NewCachingClient(inner, 5*time.Second)
	ctx := context.Background()

	mounts, err := cc.ListMounts(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(mounts))
	}

	// Make inner fail
	inner.listMountsFunc = func(ctx context.Context) (map[string]MountInfo, error) {
		calls++
		return nil, fmt.Errorf("network error")
	}

	mounts, err = cc.ListMounts(ctx)
	if err != nil {
		t.Fatalf("expected cached result, got error: %v", err)
	}
	if len(mounts) != 1 {
		t.Fatalf("expected 1 cached mount, got %d", len(mounts))
	}
}

func TestCachingClient_SetTokenDelegates(t *testing.T) {
	inner := &mockClient{}
	cc := NewCachingClient(inner, 5*time.Second)

	cc.SetToken("new-token")
	if inner.token != "new-token" {
		t.Fatalf("expected token 'new-token', got %q", inner.token)
	}
}

func TestCachingClient_PingDelegates(t *testing.T) {
	pingErr := fmt.Errorf("ping failed")
	inner := &mockClient{
		pingFunc: func(ctx context.Context) error {
			return pingErr
		},
	}
	cc := NewCachingClient(inner, 5*time.Second)
	err := cc.Ping(context.Background())
	if err != pingErr {
		t.Fatalf("expected ping error, got %v", err)
	}
}

func TestCachingClient_DifferentPathsSeparatelyCached(t *testing.T) {
	inner := &mockClient{
		readFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
			return map[string]interface{}{"path": path}, nil
		},
	}

	cc := NewCachingClient(inner, 5*time.Second)
	ctx := context.Background()

	d1, _ := cc.Read(ctx, "secret/a")
	d2, _ := cc.Read(ctx, "secret/b")

	if d1["path"] != "secret/a" {
		t.Fatalf("expected path secret/a, got %v", d1["path"])
	}
	if d2["path"] != "secret/b" {
		t.Fatalf("expected path secret/b, got %v", d2["path"])
	}

	// Fail inner — each path should return its own cached value
	inner.readFunc = func(ctx context.Context, path string) (map[string]interface{}, error) {
		return nil, fmt.Errorf("network error")
	}

	d1, err := cc.Read(ctx, "secret/a")
	if err != nil || d1["path"] != "secret/a" {
		t.Fatalf("expected cached secret/a, got %v err=%v", d1, err)
	}
	d2, err = cc.Read(ctx, "secret/b")
	if err != nil || d2["path"] != "secret/b" {
		t.Fatalf("expected cached secret/b, got %v err=%v", d2, err)
	}
}

func TestCachingClient_SuccessRefreshesCache(t *testing.T) {
	version := 1
	inner := &mockClient{
		readFunc: func(ctx context.Context, path string) (map[string]interface{}, error) {
			v := version
			version++
			return map[string]interface{}{"version": v}, nil
		},
	}

	cc := NewCachingClient(inner, 5*time.Second)
	ctx := context.Background()

	d1, _ := cc.Read(ctx, "secret/app")
	if d1["version"] != 1 {
		t.Fatalf("expected version 1, got %v", d1["version"])
	}

	// Second successful call updates the cache
	d2, _ := cc.Read(ctx, "secret/app")
	if d2["version"] != 2 {
		t.Fatalf("expected version 2, got %v", d2["version"])
	}

	// Make inner fail — should return the latest cached value (version 2)
	inner.readFunc = func(ctx context.Context, path string) (map[string]interface{}, error) {
		return nil, fmt.Errorf("network error")
	}

	d3, err := cc.Read(ctx, "secret/app")
	if err != nil {
		t.Fatalf("expected cached result, got error: %v", err)
	}
	if d3["version"] != 2 {
		t.Fatalf("expected cached version 2, got %v", d3["version"])
	}
}
