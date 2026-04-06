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
	Register("aws-secrets-manager", newAWSSecretsManagerClient)
}

func newAWSSecretsManagerClient(cfg Config) (vault.ClientInterface, error) {
	region := cfg.Options["region"]
	if region == "" {
		region = "us-east-1"
	}
	prefix := cfg.Options["prefix"]
	log := cfg.Logger
	if log == nil {
		log = logger.New(os.Stdout, cfg.Debug)
	}
	return &awsSecretsManagerClient{
		region: region,
		prefix: prefix,
		logger: log,
	}, nil
}

// awsSecretsManagerClient is a stub implementation of vault.ClientInterface
// for AWS Secrets Manager. Replace the method bodies with real AWS SDK calls
// to use in production.
type awsSecretsManagerClient struct {
	region string
	prefix string
	logger *logger.Logger
	mu     sync.RWMutex
}

func (c *awsSecretsManagerClient) Ping(ctx context.Context) error {
	// TODO: call sts:GetCallerIdentity or secretsmanager:ListSecrets with MaxResults=1
	return fmt.Errorf("aws-secrets-manager: Ping not yet implemented — add AWS SDK call here")
}

func (c *awsSecretsManagerClient) List(ctx context.Context, path string) ([]string, error) {
	// TODO: call secretsmanager:ListSecrets, filter by prefix derived from path
	c.logger.Debug("AWS SM List", map[string]interface{}{"path": path, "region": c.region})
	return nil, fmt.Errorf("aws-secrets-manager: List not yet implemented")
}

func (c *awsSecretsManagerClient) Read(ctx context.Context, path string) (map[string]interface{}, error) {
	// TODO: call secretsmanager:GetSecretValue, parse JSON string into map
	c.logger.Debug("AWS SM Read", map[string]interface{}{"path": path, "region": c.region})
	return nil, fmt.Errorf("aws-secrets-manager: Read not yet implemented")
}

func (c *awsSecretsManagerClient) PathExists(ctx context.Context, path string) (bool, bool, error) {
	// TODO: attempt GetSecretValue; if ResourceNotFoundException → false
	return false, false, fmt.Errorf("aws-secrets-manager: PathExists not yet implemented")
}

func (c *awsSecretsManagerClient) ListMounts(ctx context.Context) (map[string]vault.MountInfo, error) {
	// AWS Secrets Manager doesn't have the concept of mounts.
	// Return a single synthetic mount so the filesystem can enumerate a root.
	name := c.prefix
	if name == "" {
		name = "secrets"
	}
	name = strings.TrimSuffix(name, "/")
	return map[string]vault.MountInfo{
		name: {Type: "aws-secrets-manager", Description: "AWS Secrets Manager (" + c.region + ")", Path: name},
	}, nil
}

func (c *awsSecretsManagerClient) RefreshMounts(ctx context.Context) error {
	return nil // no-op for AWS Secrets Manager
}

func (c *awsSecretsManagerClient) SetToken(token string) {
	// AWS uses IAM credentials, not tokens. No-op.
}
