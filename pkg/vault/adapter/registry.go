package adapter

import (
	"fmt"
	"sort"
	"sync"

	"safeguard/pkg/auth"
	"safeguard/pkg/logger"
	"safeguard/pkg/vault"
)

// Config holds provider-agnostic configuration for creating a vault client.
type Config struct {
	// Provider is the registered name of the vault backend (e.g., "hashicorp", "aws-secrets-manager").
	Provider string

	// Address is the service endpoint URL (e.g., "http://127.0.0.1:8200" for HashiCorp Vault).
	Address string

	// Token is the authentication token or credential.
	Token string

	// Debug enables verbose logging in the provider.
	Debug bool

	// Options holds provider-specific key-value configuration.
	Options map[string]string

	// Auth holds authentication configuration for providers that need explicit auth.
	Auth AuthConfig

	// Logger is an optional shared logger. If nil, each provider creates its own stdout logger.
	Logger *logger.Logger
}

// AuthConfig holds authentication configuration passed through the adapter.
// Provider-specific auth factories convert these fields as needed.
type AuthConfig struct {
	Method       string
	Username     string
	Password     string
	Role         string
	MountPath    string
	CallbackPort int
}

// ProviderFactory is a function that creates a ClientInterface from a Config.
// Each vault backend registers one of these via Register().
type ProviderFactory func(cfg Config) (vault.ClientInterface, error)

// AuthFactory creates an AuthProvider for a given provider configuration.
// Each vault backend can register one via RegisterAuth().
type AuthFactory func(cfg Config) (auth.AuthProvider, error)

var (
	mu            sync.RWMutex
	providers     = make(map[string]ProviderFactory)
	authProviders = make(map[string]AuthFactory)
)

// Register adds a named provider factory to the global registry.
// It is typically called from an init() function in each provider package.
// Panics if a provider with the same name is already registered.
func Register(name string, factory ProviderFactory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := providers[name]; exists {
		panic(fmt.Sprintf("vault adapter: provider %q already registered", name))
	}
	providers[name] = factory
}

// RegisterAuth adds a named authentication factory to the global registry.
// It is typically called from an init() function alongside Register().
// Providers that don't register an auth factory will use NoopAuthProvider
// (appropriate for SDK-managed auth like AWS IAM, GCP ADC, Azure DefaultAzureCredential).
func RegisterAuth(name string, factory AuthFactory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := authProviders[name]; exists {
		panic(fmt.Sprintf("vault adapter: auth provider %q already registered", name))
	}
	authProviders[name] = factory
}

// New creates a ClientInterface using the provider specified in cfg.Provider.
// Returns an error if the provider name is not registered.
func New(cfg Config) (vault.ClientInterface, error) {
	mu.RLock()
	factory, ok := providers[cfg.Provider]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown vault provider %q (registered: %v)", cfg.Provider, ListProviders())
	}
	return factory(cfg)
}

// NewAuth creates an AuthProvider for the provider specified in cfg.Provider.
// If no auth factory is registered for the provider, returns a NoopAuthProvider
// (appropriate for providers like AWS, GCP, Azure where the SDK manages auth).
func NewAuth(cfg Config) (auth.AuthProvider, error) {
	mu.RLock()
	factory, ok := authProviders[cfg.Provider]
	mu.RUnlock()
	if !ok {
		return auth.NewNoopAuthProvider(cfg.Provider), nil
	}
	return factory(cfg)
}

// ListProviders returns the names of all registered providers in sorted order.
func ListProviders() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
