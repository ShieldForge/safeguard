// Package vault defines the shared types and interface for vault backend adapters.
//
// Each adapter (HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager,
// Azure Key Vault, etc.) lives under pkg/vault/adapter and implements
// ClientInterface. Use adapter.New(cfg) to create a client by provider name.
//
// Example usage:
//
//	cfg := adapter.Config{Provider: "hashicorp", Address: "http://127.0.0.1:8200", Token: token}
//	client, err := adapter.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Test connection
//	if err := client.Ping(ctx); err != nil {
//	    log.Fatal("Vault not reachable:", err)
//	}
//
//	// List secrets at a path
//	secrets, err := client.List(ctx, "secret/myapp")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Read a specific secret
//	data, err := client.Read(ctx, "secret/myapp/config")
//	if err != nil {
//	    log.Fatal(err)
//	}
package vault

import "context"

// ClientInterface defines the interface for Vault client operations.
//
// This interface abstracts Vault interactions to enable testing and alternative
// implementations. It covers the essential operations needed for a FUSE filesystem
// backed by Vault:
//   - Ping: Test connectivity to Vault
//   - List: Enumerate secrets at a given path
//   - Read: Retrieve secret data from a path
//   - PathExists: Check if a path exists and whether it's a directory or secret
//   - ListMounts: Enumerate all mounted secret engines
//   - RefreshMounts: Update cached mount information
//   - SetToken: Update the authentication token (e.g., after renewal)
//
// The standard implementation is adapter.HashiCorpClient; other adapters
// (AWS, GCP, Azure) also implement this interface.
type ClientInterface interface {
	// Ping tests connectivity to Vault by calling the health endpoint.
	// Returns nil if Vault is reachable, or an error if the connection fails.
	Ping(ctx context.Context) error

	// List retrieves a list of keys at the specified path.
	// For an empty or "/" path, it returns all mounted secret engines.
	// For other paths, it returns the keys (secrets and subdirectories) at that location.
	// Directory names are returned with a trailing "/" suffix.
	List(ctx context.Context, path string) ([]string, error)

	// Read retrieves the secret data at the specified path.
	// Returns a map containing the secret's key-value pairs.
	// Supports both KV v1 and KV v2 secret engines automatically.
	Read(ctx context.Context, path string) (map[string]interface{}, error)

	// PathExists checks if a path exists in Vault and determines whether it's
	// a directory (contains sub-paths) or a secret (leaf node).
	//
	// Returns:
	//   - exists: true if the path exists, false otherwise
	//   - isDir: true if the path is a directory, false if it's a secret
	//   - err: any error encountered during the check
	PathExists(ctx context.Context, path string) (bool, bool, error)

	// ListMounts retrieves all mounted secret engines from Vault.
	ListMounts(ctx context.Context) (map[string]MountInfo, error)

	// RefreshMounts updates cached mount information by calling ListMounts.
	RefreshMounts(ctx context.Context) error

	// SetToken updates the authentication token used for Vault requests.
	SetToken(token string)
}

// MountInfo contains information about a Vault secret engine mount.
//
// Each mounted secret engine in Vault has a unique path and type.
type MountInfo struct {
	Type        string
	Description string
	Path        string
}
