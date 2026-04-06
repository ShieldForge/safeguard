package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"safeguard/pkg/auth"
	"safeguard/pkg/logger"
	"safeguard/pkg/vault"
)

func init() {
	Register("hashicorp", newHashiCorpClient)
	RegisterAuth("hashicorp", newHashiCorpAuth)
}

func newHashiCorpClient(cfg Config) (vault.ClientInterface, error) {
	return NewHashiCorpClientWithLogger(cfg.Address, cfg.Token, cfg.Debug, cfg.Logger)
}

func newHashiCorpAuth(cfg Config) (auth.AuthProvider, error) {
	authCfg := &auth.AuthConfig{
		Method:       auth.AuthMethod(cfg.Auth.Method),
		VaultAddr:    cfg.Address,
		Token:        cfg.Token,
		Username:     cfg.Auth.Username,
		Password:     cfg.Auth.Password,
		Role:         cfg.Auth.Role,
		MountPath:    cfg.Auth.MountPath,
		CallbackPort: cfg.Auth.CallbackPort,
		Debug:        cfg.Debug,
		Logger:       cfg.Logger,
	}
	return auth.NewAuthenticator(authCfg), nil
}

// HashiCorpClient represents a HashiCorp Vault client for HTTP API operations.
//
// HashiCorpClient maintains an authenticated connection to Vault and provides methods
// for common secret operations. It caches mount information to optimize path
// resolution and supports both KV v1 and KV v2 secret engines.
type HashiCorpClient struct {
	address    string
	token      string
	httpClient *http.Client
	mounts     map[string]vault.MountInfo
	logger     *logger.Logger
	mu         sync.RWMutex
}

// Ensure HashiCorpClient implements ClientInterface
var _ vault.ClientInterface = (*HashiCorpClient)(nil)

// NewHashiCorpClient creates a new HashiCorp Vault client with the specified address and token.
//
// Parameters:
//   - address: The Vault server URL (e.g., "http://127.0.0.1:8200")
//   - token: A valid Vault authentication token
//   - debug: Enable debug logging for all Vault operations
//
// Returns an error if address or token is empty.
func NewHashiCorpClient(address, token string, debug bool) (*HashiCorpClient, error) {
	return NewHashiCorpClientWithLogger(address, token, debug, nil)
}

// NewHashiCorpClientWithLogger creates a new HashiCorp Vault client with an optional shared logger.
// If log is nil, a default stdout logger is created.
func NewHashiCorpClientWithLogger(address, token string, debug bool, log *logger.Logger) (*HashiCorpClient, error) {
	if address == "" {
		return nil, fmt.Errorf("vault address is required")
	}
	if token == "" {
		return nil, fmt.Errorf("vault token is required")
	}

	if log == nil {
		log = logger.New(os.Stdout, debug)
	}

	client := &HashiCorpClient{
		address: strings.TrimSuffix(address, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		mounts: make(map[string]vault.MountInfo),
		logger: log,
	}

	return client, nil
}

// SetToken updates the authentication token used for Vault requests.
// This is safe for concurrent use.
func (c *HashiCorpClient) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// getToken returns the current token. Safe for concurrent use.
func (c *HashiCorpClient) getToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// ListMounts retrieves all mounted secret engines from Vault.
//
// This method queries Vault's /sys/mounts endpoint to get information about
// all mounted secret engines. The results are cached internally.
func (c *HashiCorpClient) ListMounts(ctx context.Context) (map[string]vault.MountInfo, error) {
	c.logger.Debug("Vault operation", map[string]interface{}{
		"operation": "ListMounts",
	})

	req, err := http.NewRequestWithContext(ctx, "GET", c.address+"/v1/sys/mounts", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Vault-Token", c.getToken())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list mounts failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	mounts := make(map[string]vault.MountInfo)
	for path, info := range result.Data {
		cleanPath := strings.TrimSuffix(path, "/")
		mounts[cleanPath] = vault.MountInfo{
			Type:        info.Type,
			Description: info.Description,
			Path:        cleanPath,
		}
	}

	c.mounts = mounts

	return mounts, nil
}

// RefreshMounts updates the cached mount information by calling ListMounts.
func (c *HashiCorpClient) RefreshMounts(ctx context.Context) error {
	_, err := c.ListMounts(ctx)
	return err
}

// GetMountForPath determines which secret engine mount a given path belongs to.
func (c *HashiCorpClient) GetMountForPath(ctx context.Context, path string) (string, string, error) {
	if len(c.mounts) == 0 {
		if err := c.RefreshMounts(ctx); err != nil {
			return "", "", err
		}
	}

	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return "", "", nil
	}

	parts := strings.Split(path, "/")
	mountPath := parts[0]

	if _, exists := c.mounts[mountPath]; exists {
		remainingPath := strings.Join(parts[1:], "/")
		return mountPath, remainingPath, nil
	}

	return "", "", fmt.Errorf("no mount found for path: %s", path)
}

// Ping tests the connection to Vault by calling the health endpoint.
func (c *HashiCorpClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.address+"/v1/sys/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 600 {
		return nil
	}

	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// List lists secrets and subdirectories at a given path in Vault.
func (c *HashiCorpClient) List(ctx context.Context, path string) ([]string, error) {
	c.logger.Debug("Vault operation", map[string]interface{}{
		"operation": "List",
		"path":      path,
	})

	if path == "" || path == "/" {
		mounts, err := c.ListMounts(ctx)
		if err != nil {
			return nil, err
		}
		var result []string
		for mountPath, info := range mounts {
			if info.Type == "kv" || info.Type == "generic" {
				result = append(result, mountPath+"/")
			}
		}
		return result, nil
	}

	apiPath := c.constructAPIPath(ctx, path, true)

	req, err := http.NewRequestWithContext(ctx, "LIST", c.address+apiPath, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Vault-Token", c.getToken())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return []string{}, nil
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vault list failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Keys []string `json:"keys"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Data.Keys, nil
}

// Read reads a secret from Vault at the specified path.
func (c *HashiCorpClient) Read(ctx context.Context, path string) (map[string]interface{}, error) {
	c.logger.Debug("Vault operation", map[string]interface{}{
		"operation": "Read",
		"path":      path,
	})

	apiPath := c.constructAPIPath(ctx, path, false)

	req, err := http.NewRequestWithContext(ctx, "GET", c.address+apiPath, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Vault-Token", c.getToken())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("secret not found")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vault read failed with status %d: %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var v2Result struct {
		Data struct {
			Data     map[string]interface{} `json:"data"`
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &v2Result); err != nil {
		return nil, err
	}
	if v2Result.Data.Data != nil {
		return v2Result.Data.Data, nil
	}

	var v1Result struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &v1Result); err != nil {
		return nil, err
	}
	if v1Result.Data != nil {
		return v1Result.Data, nil
	}

	return nil, fmt.Errorf("failed to parse secret data")
}

// PathExists checks if a path exists in Vault and whether it's a directory or secret.
func (c *HashiCorpClient) PathExists(ctx context.Context, path string) (exists bool, isDir bool, err error) {
	c.logger.Debug("Vault operation", map[string]interface{}{
		"operation": "PathExists",
		"path":      path,
	})

	entries, err := c.List(ctx, path)
	if err == nil && len(entries) > 0 {
		return true, true, nil
	}

	_, err = c.Read(ctx, path)
	if err == nil {
		return true, false, nil
	}

	if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404") {
		return false, false, nil
	}

	return false, false, err
}

// constructAPIPath builds the appropriate Vault API endpoint for a given path.
func (c *HashiCorpClient) constructAPIPath(ctx context.Context, path string, isList bool) string {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return ""
	}

	mountPath, internalPath, err := c.GetMountForPath(ctx, path)
	if err != nil {
		parts := strings.SplitN(path, "/", 2)
		mountPath = parts[0]
		if len(parts) > 1 {
			internalPath = parts[1]
		} else {
			internalPath = ""
		}
	}

	if isList {
		return fmt.Sprintf("/v1/%s/metadata/%s?list=true", mountPath, internalPath)
	}
	return fmt.Sprintf("/v1/%s/data/%s", mountPath, internalPath)
}
