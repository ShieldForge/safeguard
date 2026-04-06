package adapter

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"safeguard/pkg/logger"
	"safeguard/pkg/vault"
)

func init() {
	Register("azure-keyvault", newAzureKeyVaultClient)
}

func newAzureKeyVaultClient(cfg Config) (vault.ClientInterface, error) {
	vaultName := cfg.Options["vault-name"]
	if vaultName == "" {
		return nil, fmt.Errorf("azure-keyvault: \"vault-name\" option is required")
	}
	vaultURL := fmt.Sprintf("https://%s.vault.azure.net", vaultName)
	if override := cfg.Options["vault-url"]; override != "" {
		vaultURL = override
	}
	log := cfg.Logger
	if log == nil {
		log = logger.New(os.Stdout, cfg.Debug)
	}
	return &azureKeyVaultClient{
		vaultName: vaultName,
		vaultURL:  vaultURL,
		logger:    log,
	}, nil
}

// azureKeyVaultClient is a stub implementation of vault.ClientInterface
// for Azure Key Vault. Replace the method bodies with real
// github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets SDK calls
// to use in production.
type azureKeyVaultClient struct {
	vaultName string
	vaultURL  string
	logger    *logger.Logger
	mu        sync.RWMutex
}

func (c *azureKeyVaultClient) Ping(ctx context.Context) error {
	// TODO: call azsecrets.Client.ListSecrets with MaxResults=1 to verify connectivity
	return fmt.Errorf("azure-keyvault: Ping not yet implemented — add Azure SDK call here")
}

func (c *azureKeyVaultClient) List(ctx context.Context, path string) ([]string, error) {
	// TODO: call azsecrets.Client.ListSecrets, filter by prefix derived from path
	c.logger.Debug("Azure KV List", map[string]interface{}{"path": path, "vault": c.vaultName})
	return nil, fmt.Errorf("azure-keyvault: List not yet implemented")
}

func (c *azureKeyVaultClient) Read(ctx context.Context, path string) (map[string]interface{}, error) {
	// TODO: call azsecrets.Client.GetSecret, parse value into map
	c.logger.Debug("Azure KV Read", map[string]interface{}{"path": path, "vault": c.vaultName})
	return nil, fmt.Errorf("azure-keyvault: Read not yet implemented")
}

func (c *azureKeyVaultClient) PathExists(ctx context.Context, path string) (bool, bool, error) {
	// TODO: attempt GetSecret; if ResourceNotFoundError → false
	return false, false, fmt.Errorf("azure-keyvault: PathExists not yet implemented")
}

func (c *azureKeyVaultClient) ListMounts(ctx context.Context) (map[string]vault.MountInfo, error) {
	// Azure Key Vault doesn't have mounts.
	// Return a single synthetic mount so the filesystem can enumerate a root.
	name := strings.TrimSuffix(c.vaultName, "/")
	return map[string]vault.MountInfo{
		name: {Type: "azure-keyvault", Description: "Azure Key Vault (" + c.vaultName + ")", Path: name},
	}, nil
}

func (c *azureKeyVaultClient) RefreshMounts(ctx context.Context) error {
	return nil // no-op
}

func (c *azureKeyVaultClient) SetToken(token string) {
	// Azure uses DefaultAzureCredential / managed identity, not tokens. No-op.
}
