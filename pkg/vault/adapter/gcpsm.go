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
	Register("gcp-secret-manager", newGCPSecretManagerClient)
}

func newGCPSecretManagerClient(cfg Config) (vault.ClientInterface, error) {
	project := cfg.Options["project"]
	if project == "" {
		return nil, fmt.Errorf("gcp-secret-manager: \"project\" option is required")
	}
	prefix := cfg.Options["prefix"]
	log := cfg.Logger
	if log == nil {
		log = logger.New(os.Stdout, cfg.Debug)
	}
	return &gcpSecretManagerClient{
		project: project,
		prefix:  prefix,
		logger:  log,
	}, nil
}

// gcpSecretManagerClient is a stub implementation of vault.ClientInterface
// for Google Cloud Secret Manager. Replace the method bodies with real
// cloud.google.com/go/secretmanager SDK calls to use in production.
type gcpSecretManagerClient struct {
	project string
	prefix  string
	logger  *logger.Logger
	mu      sync.RWMutex
}

func (c *gcpSecretManagerClient) Ping(ctx context.Context) error {
	// TODO: call secretmanager.ListSecrets with PageSize=1 to verify connectivity
	return fmt.Errorf("gcp-secret-manager: Ping not yet implemented — add GCP SDK call here")
}

func (c *gcpSecretManagerClient) List(ctx context.Context, path string) ([]string, error) {
	// TODO: call secretmanager.ListSecrets, filter by prefix derived from path
	c.logger.Debug("GCP SM List", map[string]interface{}{"path": path, "project": c.project})
	return nil, fmt.Errorf("gcp-secret-manager: List not yet implemented")
}

func (c *gcpSecretManagerClient) Read(ctx context.Context, path string) (map[string]interface{}, error) {
	// TODO: call secretmanager.AccessSecretVersion, parse JSON payload into map
	c.logger.Debug("GCP SM Read", map[string]interface{}{"path": path, "project": c.project})
	return nil, fmt.Errorf("gcp-secret-manager: Read not yet implemented")
}

func (c *gcpSecretManagerClient) PathExists(ctx context.Context, path string) (bool, bool, error) {
	// TODO: attempt GetSecret; if NotFound → false
	return false, false, fmt.Errorf("gcp-secret-manager: PathExists not yet implemented")
}

func (c *gcpSecretManagerClient) ListMounts(ctx context.Context) (map[string]vault.MountInfo, error) {
	// GCP Secret Manager doesn't have mounts.
	// Return a single synthetic mount so the filesystem can enumerate a root.
	name := c.prefix
	if name == "" {
		name = "secrets"
	}
	name = strings.TrimSuffix(name, "/")
	return map[string]vault.MountInfo{
		name: {Type: "gcp-secret-manager", Description: "GCP Secret Manager (project: " + c.project + ")", Path: name},
	}, nil
}

func (c *gcpSecretManagerClient) RefreshMounts(ctx context.Context) error {
	return nil // no-op
}

func (c *gcpSecretManagerClient) SetToken(token string) {
	// GCP uses ADC / service account credentials, not tokens. No-op.
}
