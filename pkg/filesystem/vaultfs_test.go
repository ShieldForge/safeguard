package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"safeguard/pkg/logger"
	"safeguard/pkg/vault"

	"github.com/winfsp/cgofuse/fuse"
)

// MockVaultClient implements a mock Vault client for testing
type MockVaultClient struct {
	secrets map[string]map[string]interface{}
	lists   map[string][]string
	mounts  map[string]vault.MountInfo
}

func NewMockVaultClient() *MockVaultClient {
	return &MockVaultClient{
		secrets: make(map[string]map[string]interface{}),
		lists:   make(map[string][]string),
		mounts: map[string]vault.MountInfo{
			"secret": {Type: "kv", Description: "KV Secrets Engine", Path: "secret"},
		},
	}
}

func (m *MockVaultClient) SetSecret(path string, data map[string]interface{}) {
	m.secrets[path] = data
}

func (m *MockVaultClient) SetList(path string, items []string) {
	m.lists[path] = items
}

func (m *MockVaultClient) SetMount(path string, info vault.MountInfo) {
	m.mounts[path] = info
}

func (m *MockVaultClient) List(ctx context.Context, path string) ([]string, error) {
	// If path is empty, list mounts
	if path == "" || path == "/" {
		var result []string
		for mountPath := range m.mounts {
			result = append(result, mountPath+"/")
		}
		return result, nil
	}

	if items, exists := m.lists[path]; exists {
		return items, nil
	}
	return []string{}, nil
}

func (m *MockVaultClient) Read(ctx context.Context, path string) (map[string]interface{}, error) {
	if data, exists := m.secrets[path]; exists {
		return data, nil
	}
	return nil, fuse.Error(-fuse.ENOENT)
}

func (m *MockVaultClient) PathExists(ctx context.Context, path string) (bool, bool, error) {
	// Check if it's a directory (has list entries)
	if items, exists := m.lists[path]; exists && len(items) > 0 {
		return true, true, nil
	}

	// Check if it's a secret
	if _, exists := m.secrets[path]; exists {
		return true, false, nil
	}

	return false, false, nil
}

func (m *MockVaultClient) Ping(ctx context.Context) error {
	return nil
}

func (m *MockVaultClient) ListMounts(ctx context.Context) (map[string]vault.MountInfo, error) {
	return m.mounts, nil
}

func (m *MockVaultClient) RefreshMounts(ctx context.Context) error {
	return nil
}

func (m *MockVaultClient) SetToken(token string) {
	// no-op for mock
}

func TestNewVaultFS(t *testing.T) {
	mockClient := NewMockVaultClient()

	tests := []struct {
		name  string
		debug bool
	}{
		{
			name:  "with debug",
			debug: true,
		},
		{
			name:  "without debug",
			debug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewVaultFS(mockClient, tt.debug)
			if fs == nil {
				t.Error("NewVaultFS() returned nil")
			}
			if fs.debug != tt.debug {
				t.Errorf("VaultFS.debug = %v, want %v", fs.debug, tt.debug)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "root path",
			path: "/",
			want: "",
		},
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "simple path",
			path: "/app/config",
			want: "app/config",
		},
		{
			name: "path with trailing slash",
			path: "/app/config/",
			want: "app/config",
		},
		{
			name: "path without leading slash",
			path: "app/config",
			want: "app/config",
		},
		{
			name: "windows-style path",
			path: "\\app\\config",
			want: "app/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePath(tt.path)
			if got != tt.want {
				t.Errorf("normalizePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatSecretData(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		want string
	}{
		{
			name: "simple secret",
			data: map[string]interface{}{
				"username": "admin",
				"password": "secret123",
			},
			want: "username: admin\npassword: secret123\n",
		},
		{
			name: "single key",
			data: map[string]interface{}{
				"key": "value",
			},
			want: "key: value\n",
		},
		{
			name: "empty secret",
			data: map[string]interface{}{},
			want: "",
		},
		{
			name: "numeric values",
			data: map[string]interface{}{
				"port": 5432,
				"ssl":  true,
			},
			want: "port: 5432\nssl: true\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(formatSecretData(tt.data))
			// Note: map iteration order is not guaranteed, so we check for key presence
			for key := range tt.data {
				if !containsKey(got, key) {
					t.Errorf("formatSecretData() missing key %v", key)
				}
			}
		})
	}
}

func containsKey(s, key string) bool {
	return len(s) > 0 && (s == "" || len(key) == 0 || stringContains(s, key+":"))
}

func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || indexOfString(s, substr) >= 0)
}

func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestGetMountOptions(t *testing.T) {
	tests := []struct {
		name          string
		debug         bool
		wantDebugFlag bool
	}{
		{
			name:          "with debug",
			debug:         true,
			wantDebugFlag: true,
		},
		{
			name:          "without debug",
			debug:         false,
			wantDebugFlag: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := getMountOptions(tt.debug)
			if len(options) == 0 {
				t.Error("getMountOptions() returned empty options")
			}

			hasDebugFlag := false
			for _, opt := range options {
				if opt == "-d" {
					hasDebugFlag = true
					break
				}
			}

			if hasDebugFlag != tt.wantDebugFlag {
				t.Errorf("getMountOptions() debug flag = %v, want %v", hasDebugFlag, tt.wantDebugFlag)
			}
		})
	}
}

// Integration-style tests would require actual FUSE mounting which needs elevated privileges
// and is platform-specific. These are better done as separate integration tests.

func TestVaultFS_Getattr_Root(t *testing.T) {
	mockClient := NewMockVaultClient()
	fs := &VaultFS{
		client: mockClient,
		debug:  false,
	}

	stat := &fuse.Stat_t{}
	result := fs.Getattr("/", stat, 0)

	if result != 0 {
		t.Errorf("Getattr() for root returned %v, want 0", result)
	}

	if stat.Mode&fuse.S_IFDIR == 0 {
		t.Error("Root should be a directory")
	}
}

func TestVaultFS_Getattr_Secret(t *testing.T) {
	mockClient := NewMockVaultClient()
	mockClient.SetSecret("app/config", map[string]interface{}{
		"key": "value",
	})

	fs := &VaultFS{
		client: mockClient,
		debug:  false,
	}

	stat := &fuse.Stat_t{}
	result := fs.Getattr("/app/config", stat, 0)

	if result != 0 {
		t.Errorf("Getattr() for secret returned %v, want 0", result)
	}

	if stat.Mode&fuse.S_IFREG == 0 {
		t.Error("Secret should be a regular file")
	}
}

func TestVaultFS_Getattr_NonExistent(t *testing.T) {
	mockClient := NewMockVaultClient()
	fs := &VaultFS{
		client: mockClient,
		debug:  false,
	}

	stat := &fuse.Stat_t{}
	result := fs.Getattr("/nonexistent", stat, 0)

	if result != -fuse.ENOENT {
		t.Errorf("Getattr() for nonexistent returned %v, want ENOENT", result)
	}
}

func TestVaultFS_Open_ExistingSecret(t *testing.T) {
	mockClient := NewMockVaultClient()
	mockClient.SetSecret("app/config", map[string]interface{}{
		"key": "value",
	})

	fs := &VaultFS{
		client: mockClient,
		debug:  false,
	}

	result, _ := fs.Open("/app/config", 0)
	if result != 0 {
		t.Errorf("Open() for existing secret returned %v, want 0", result)
	}
}

func TestVaultFS_Open_NonExistent(t *testing.T) {
	mockClient := NewMockVaultClient()
	fs := &VaultFS{
		client: mockClient,
		debug:  false,
	}

	result, _ := fs.Open("/nonexistent", 0)
	if result != -fuse.ENOENT {
		t.Errorf("Open() for nonexistent returned %v, want ENOENT", result)
	}
}

func TestVaultFS_Read(t *testing.T) {
	mockClient := NewMockVaultClient()
	mockClient.SetSecret("app/config", map[string]interface{}{
		"username": "admin",
		"password": "secret",
	})

	fs := &VaultFS{
		client: mockClient,
		debug:  false,
	}

	buff := make([]byte, 1024)
	result := fs.Read("/app/config", buff, 0, 0)

	if result <= 0 {
		t.Errorf("Read() returned %v, expected positive value", result)
	}

	content := string(buff[:result])
	if !stringContains(content, "username") {
		t.Error("Read() content missing 'username' field")
	}
}

func TestVaultFS_Read_WithOffset(t *testing.T) {
	mockClient := NewMockVaultClient()
	mockClient.SetSecret("app/config", map[string]interface{}{
		"key": "value",
	})

	fs := &VaultFS{
		client: mockClient,
		debug:  false,
	}

	buff := make([]byte, 1024)

	// Read from offset beyond content
	result := fs.Read("/app/config", buff, 10000, 0)
	if result != 0 {
		t.Errorf("Read() with large offset returned %v, want 0", result)
	}
}

func TestVaultFS_Readdir(t *testing.T) {
	mockClient := NewMockVaultClient()
	mockClient.SetList("", []string{"app1/", "app2/", "config"})

	fs := &VaultFS{
		client: mockClient,
		debug:  false,
	}

	entries := []string{}
	fill := func(name string, stat *fuse.Stat_t, ofst int64) bool {
		entries = append(entries, name)
		return true
	}

	result := fs.Readdir("/", fill, 0, 0)
	if result != 0 {
		t.Errorf("Readdir() returned %v, want 0", result)
	}

	// Should have . and .. plus the actual entries
	if len(entries) < 2 {
		t.Errorf("Readdir() returned %v entries, want at least 2 (. and ..)", len(entries))
	}

	hasCurrentDir := false
	hasParentDir := false
	for _, entry := range entries {
		if entry == "." {
			hasCurrentDir = true
		}
		if entry == ".." {
			hasParentDir = true
		}
	}

	if !hasCurrentDir {
		t.Error("Readdir() missing '.' entry")
	}
	if !hasParentDir {
		t.Error("Readdir() missing '..' entry")
	}
}

func TestVaultFS_Readdir_NonExistent(t *testing.T) {
	mockClient := NewMockVaultClient()
	fs := &VaultFS{
		client: mockClient,
		debug:  false,
	}

	entries := []string{}
	fill := func(name string, stat *fuse.Stat_t, ofst int64) bool {
		entries = append(entries, name)
		return true
	}

	result := fs.Readdir("/nonexistent", fill, 0, 0)

	// Empty directory should still return success with . and ..
	if result != 0 {
		t.Errorf("Readdir() for nonexistent returned %v, want 0 (success)", result)
	}

	// Should still have . and .. even if empty
	if len(entries) != 2 {
		t.Errorf("Readdir() for empty dir returned %v entries, want 2 (. and ..)", len(entries))
	}
}

func TestVaultFS_Readdir_RootWithMappings(t *testing.T) {
	// Create temporary test files
	tmpFile1, err := os.CreateTemp("", "test1-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file 1: %v", err)
	}
	defer os.Remove(tmpFile1.Name())
	tmpFile1.Close()

	tmpDir, err := os.MkdirTemp("", "testdir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create mock Vault client with some mounts (root level listing returns mounts)
	mockClient := NewMockVaultClient()
	// The default mock has "secret" mount, let's add another
	mockClient.SetMount("admin", vault.MountInfo{Type: "kv", Description: "Admin Secrets", Path: "admin"})

	// Create path mapper with some mappings
	pm := NewPathMapper(true)
	mappings := []PathMapping{
		{
			VirtualPath: "/config/app.txt",
			RealPath:    tmpFile1.Name(),
			ReadOnly:    true,
		},
		{
			VirtualPath: "/data",
			RealPath:    tmpDir,
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	// Create VaultFS with both client and path mapper
	fs := &VaultFS{
		client:     mockClient,
		debug:      false,
		pathMapper: pm,
		logger:     logger.New(os.Stdout, false),
	}

	entries := []string{}
	fill := func(name string, stat *fuse.Stat_t, ofst int64) bool {
		entries = append(entries, name)
		return true
	}

	result := fs.Readdir("/", fill, 0, 0)
	if result != 0 {
		t.Errorf("Readdir() returned %v, want 0", result)
	}

	// Should have . and .. plus:
	// - config (from mappings)
	// - data (from mappings)
	// - secret (from Vault mounts)
	// - admin (from Vault mounts)
	// = 6 total entries

	expectedEntries := map[string]bool{
		".":      true,
		"..":     true,
		"config": true,
		"data":   true,
		"secret": true,
		"admin":  true,
	}

	if len(entries) != len(expectedEntries) {
		t.Errorf("Readdir() returned %v entries, want %v", len(entries), len(expectedEntries))
		t.Logf("Got entries: %v", entries)
	}

	for _, entry := range entries {
		if !expectedEntries[entry] {
			t.Errorf("Unexpected entry: %q", entry)
		}
	}

	for expected := range expectedEntries {
		found := false
		for _, entry := range entries {
			if entry == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected entry: %q", expected)
		}
	}
}

func TestVaultFS_Readdir_RootWithOverlap(t *testing.T) {
	// Create temporary test files
	tmpFile1, err := os.CreateTemp("", "test1-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file 1: %v", err)
	}
	defer os.Remove(tmpFile1.Name())
	tmpFile1.Close()

	// Create mock Vault client with a mount that overlaps with mapping name
	mockClient := NewMockVaultClient()
	// The default mock has "secret" mount
	// Let's add a "config" mount to test overlap
	mockClient.SetMount("config", vault.MountInfo{Type: "kv", Description: "Config Secrets", Path: "config"})

	// Create path mapper with mapping that has same name as Vault mount
	pm := NewPathMapper(true)
	mappings := []PathMapping{
		{
			VirtualPath: "/config/app.txt",
			RealPath:    tmpFile1.Name(),
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	// Create VaultFS with both client and path mapper
	fs := &VaultFS{
		client:     mockClient,
		debug:      false,
		pathMapper: pm,
		logger:     logger.New(os.Stdout, false),
	}

	entries := []string{}
	fill := func(name string, stat *fuse.Stat_t, ofst int64) bool {
		entries = append(entries, name)
		return true
	}

	result := fs.Readdir("/", fill, 0, 0)
	if result != 0 {
		t.Errorf("Readdir() returned %v, want 0", result)
	}

	// Should have . and .. plus:
	// - config (from mappings, should appear only once even though it's in Vault too)
	// - secret (from Vault mounts)
	// = 4 total entries

	expectedEntries := map[string]bool{
		".":      true,
		"..":     true,
		"config": true,
		"secret": true,
	}

	if len(entries) != len(expectedEntries) {
		t.Errorf("Readdir() returned %v entries, want %v", len(entries), len(expectedEntries))
		t.Logf("Got entries: %v", entries)
	}

	// Ensure no duplicates
	seenEntries := make(map[string]int)
	for _, entry := range entries {
		seenEntries[entry]++
	}

	for entry, count := range seenEntries {
		if count > 1 {
			t.Errorf("Entry %q appears %d times, should appear only once", entry, count)
		}
	}
}

func TestVaultFS_Readdir_RootLevelMapping(t *testing.T) {
	// Create temporary directory for root-level mapping
	tmpDir, err := os.MkdirTemp("", "rootfs-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some files in the temp directory
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)
	subdir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subdir, 0755)
	os.WriteFile(filepath.Join(subdir, "nested.txt"), []byte("nested"), 0644)

	// Create mock Vault client
	mockClient := NewMockVaultClient()
	// The default mock has "secret" mount

	// Create path mapper with root-level mapping
	pm := NewPathMapper(true)
	mappings := []PathMapping{
		{
			VirtualPath: "/",
			RealPath:    tmpDir,
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	// Create VaultFS
	fs := &VaultFS{
		client:     mockClient,
		debug:      false,
		pathMapper: pm,
		logger:     logger.New(os.Stdout, false),
	}

	entries := []string{}
	fill := func(name string, stat *fuse.Stat_t, ofst int64) bool {
		entries = append(entries, name)
		return true
	}

	// List root directory - should show real directory contents from mapping
	result := fs.Readdir("/", fill, 0, 0)
	if result != 0 {
		t.Errorf("Readdir() returned %v, want 0", result)
	}

	// When "/" is mapped, it should list the contents of the mapped directory
	// plus Vault secrets
	expectedToContain := []string{".", "..", "file1.txt", "file2.txt", "subdir", "secret"}

	for _, expected := range expectedToContain {
		found := false
		for _, entry := range entries {
			if entry == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected entry: %q, got entries: %v", expected, entries)
		}
	}
}
