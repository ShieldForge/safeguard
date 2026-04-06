package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"safeguard/pkg/vault"
)

// MockVaultServer creates a mock Vault HTTP server for testing
type MockVaultServer struct {
	Server  *httptest.Server
	Secrets map[string]map[string]interface{}
	Lists   map[string][]string
	Mounts  map[string]vault.MountInfo
	mu      sync.RWMutex
}

// NewMockVaultServer creates a new mock Vault server
func NewMockVaultServer() *MockVaultServer {
	mock := &MockVaultServer{
		Secrets: make(map[string]map[string]interface{}),
		Lists:   make(map[string][]string),
		Mounts: map[string]vault.MountInfo{
			"secret": {Type: "kv", Description: "KV Secrets Engine", Path: "secret"},
			"kv":     {Type: "kv", Description: "Another KV Engine", Path: "kv"},
		},
	}

	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/v1/sys/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"initialized": true,
			"sealed":      false,
		})
	})

	// Mount listing endpoint
	mux.HandleFunc("/v1/sys/mounts", func(w http.ResponseWriter, r *http.Request) {
		mock.mu.RLock()
		defer mock.mu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": mock.Mounts,
		})
	})

	// Generic handler for all mount paths
	mux.HandleFunc("/v1/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "LIST" && r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		path := r.URL.Path

		// Handle LIST requests for metadata
		if strings.Contains(path, "/metadata/") && r.Method == "LIST" {
			parts := strings.SplitN(strings.TrimPrefix(path, "/v1/"), "/metadata/", 2)
			if len(parts) < 2 {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			mount := strings.TrimSuffix(parts[0], "/")
			internalPath := parts[1]

			mock.mu.RLock()
			_, mountExists := mock.Mounts[mount]
			keys, listExists := mock.Lists[mount+"/"+internalPath]
			mock.mu.RUnlock()

			if !mountExists || !listExists {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"keys": keys,
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		// Handle READ requests for data
		if strings.Contains(path, "/data/") && r.Method == "GET" {
			parts := strings.SplitN(strings.TrimPrefix(path, "/v1/"), "/data/", 2)
			if len(parts) < 2 {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			mount := strings.TrimSuffix(parts[0], "/")
			internalPath := parts[1]

			mock.mu.RLock()
			_, mountExists := mock.Mounts[mount]
			data, dataExists := mock.Secrets[mount+"/"+internalPath]
			mock.mu.RUnlock()

			if !mountExists {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			if !dataExists {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"errors": []string{"secret not found"},
				})
				return
			}

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"data": data,
					"metadata": map[string]interface{}{
						"version": 1,
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	mock.Server = httptest.NewServer(mux)
	return mock
}

func (m *MockVaultServer) Close() {
	m.Server.Close()
}

func (m *MockVaultServer) SetSecret(path string, data map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Secrets[path] = data
}

func (m *MockVaultServer) SetList(path string, keys []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Lists[path] = keys
}

func (m *MockVaultServer) SetMount(path string, info vault.MountInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Mounts[path] = info
}

func TestNewHashiCorpClient(t *testing.T) {
	tests := []struct {
		name      string
		address   string
		token     string
		wantError bool
	}{
		{
			name:      "valid client",
			address:   "http://127.0.0.1:8200",
			token:     "test-token",
			wantError: false,
		},
		{
			name:      "empty address",
			address:   "",
			token:     "test-token",
			wantError: true,
		},
		{
			name:      "empty token",
			address:   "http://127.0.0.1:8200",
			token:     "",
			wantError: true,
		},
		{
			name:      "address with trailing slash",
			address:   "http://127.0.0.1:8200/",
			token:     "test-token",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewHashiCorpClient(tt.address, tt.token, false)
			if (err != nil) != tt.wantError {
				t.Errorf("NewHashiCorpClient() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && client == nil {
				t.Error("NewHashiCorpClient() returned nil client")
			}
		})
	}
}

func TestHashiCorpClient_Ping(t *testing.T) {
	mock := NewMockVaultServer()
	defer mock.Close()

	tests := []struct {
		name      string
		address   string
		wantError bool
	}{
		{
			name:      "successful ping",
			address:   mock.Server.URL,
			wantError: false,
		},
		{
			name:      "failed ping - invalid address",
			address:   "http://invalid-vault-address:9999",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewHashiCorpClient(tt.address, "test-token", false)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			err = client.Ping(context.Background())
			if (err != nil) != tt.wantError {
				t.Errorf("Ping() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestHashiCorpClient_List(t *testing.T) {
	mock := NewMockVaultServer()
	defer mock.Close()

	mock.SetList("secret/", []string{"app1/", "app2/", "config"})
	mock.SetList("secret/app1", []string{"database", "api"})

	client, err := NewHashiCorpClient(mock.Server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantLen   int
		wantError bool
	}{
		{
			name:      "list root (mounts)",
			path:      "",
			wantLen:   2,
			wantError: false,
		},
		{
			name:      "list secrets in mount",
			path:      "secret",
			wantLen:   3,
			wantError: false,
		},
		{
			name:      "list subdirectory",
			path:      "secret/app1",
			wantLen:   2,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.List(context.Background(), tt.path)
			if (err != nil) != tt.wantError {
				t.Errorf("List() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if len(got) != tt.wantLen {
					t.Errorf("List() returned %d items, want %d. Items: %v", len(got), tt.wantLen, got)
				}
			}
		})
	}
}

func TestHashiCorpClient_Read(t *testing.T) {
	mock := NewMockVaultServer()
	defer mock.Close()

	mock.SetSecret("secret/app1/database", map[string]interface{}{
		"username": "admin",
		"password": "secret123",
		"host":     "localhost",
	})
	mock.SetSecret("secret/app2/api", map[string]interface{}{
		"key": "api-key-123",
	})

	client, err := NewHashiCorpClient(mock.Server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		want      map[string]interface{}
		wantError bool
	}{
		{
			name: "read existing secret",
			path: "secret/app1/database",
			want: map[string]interface{}{
				"username": "admin",
				"password": "secret123",
				"host":     "localhost",
			},
			wantError: false,
		},
		{
			name: "read another secret",
			path: "secret/app2/api",
			want: map[string]interface{}{
				"key": "api-key-123",
			},
			wantError: false,
		},
		{
			name:      "read non-existent secret",
			path:      "secret/nonexistent",
			want:      nil,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.Read(context.Background(), tt.path)
			if (err != nil) != tt.wantError {
				t.Errorf("Read() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				for key, want := range tt.want {
					if got[key] != want {
						t.Errorf("Read()[%s] = %v, want %v", key, got[key], want)
					}
				}
			}
		})
	}
}

func TestHashiCorpClient_PathExists(t *testing.T) {
	mock := NewMockVaultServer()
	defer mock.Close()

	mock.SetSecret("secret/app1/database", map[string]interface{}{
		"username": "admin",
	})
	mock.SetList("secret/app1", []string{"database", "config"})

	client, err := NewHashiCorpClient(mock.Server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name       string
		path       string
		wantExists bool
		wantIsDir  bool
		wantError  bool
	}{
		{
			name:       "existing secret",
			path:       "secret/app1/database",
			wantExists: true,
			wantIsDir:  false,
			wantError:  false,
		},
		{
			name:       "existing directory",
			path:       "secret/app1",
			wantExists: true,
			wantIsDir:  true,
			wantError:  false,
		},
		{
			name:       "non-existent path",
			path:       "secret/nonexistent",
			wantExists: false,
			wantIsDir:  false,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, isDir, err := client.PathExists(context.Background(), tt.path)
			if (err != nil) != tt.wantError {
				t.Errorf("PathExists() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if exists != tt.wantExists {
				t.Errorf("PathExists() exists = %v, want %v", exists, tt.wantExists)
			}
			if isDir != tt.wantIsDir {
				t.Errorf("PathExists() isDir = %v, want %v", isDir, tt.wantIsDir)
			}
		})
	}
}

func TestHashiCorpConstructAPIPath(t *testing.T) {
	mock := NewMockVaultServer()
	defer mock.Close()

	client, _ := NewHashiCorpClient(mock.Server.URL, "test-token", false)

	tests := []struct {
		name   string
		path   string
		isList bool
		want   string
	}{
		{
			name:   "read data path with mount",
			path:   "secret/myapp/config",
			isList: false,
			want:   "/v1/secret/data/myapp/config",
		},
		{
			name:   "list metadata path with mount",
			path:   "secret/myapp",
			isList: true,
			want:   "/v1/secret/metadata/myapp?list=true",
		},
		{
			name:   "list root of mount",
			path:   "secret/",
			isList: true,
			want:   "/v1/secret/metadata/?list=true",
		},
		{
			name:   "read from different mount",
			path:   "kv/mydata",
			isList: false,
			want:   "/v1/kv/data/mydata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.constructAPIPath(context.Background(), tt.path, tt.isList)
			if got != tt.want {
				t.Errorf("constructAPIPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashiCorpClient_WithDebugMode(t *testing.T) {
	mock := NewMockVaultServer()
	defer mock.Close()

	client, err := NewHashiCorpClient(mock.Server.URL, "test-token", true)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if client == nil {
		t.Error("Client should not be nil")
	}

	client, err = NewHashiCorpClient(mock.Server.URL, "test-token", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	if client == nil {
		t.Error("Client should not be nil")
	}
}

func TestHashiCorpClient_SetToken(t *testing.T) {
	mock := NewMockVaultServer()
	defer mock.Close()

	client, err := NewHashiCorpClient(mock.Server.URL, "original-token", false)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if got := client.getToken(); got != "original-token" {
		t.Errorf("getToken() = %v, want original-token", got)
	}

	client.SetToken("new-token")

	if got := client.getToken(); got != "new-token" {
		t.Errorf("getToken() after SetToken = %v, want new-token", got)
	}
}
