package auth

// NoopAuthProvider is an AuthProvider for vault backends where the SDK
// manages authentication internally (e.g., AWS IAM, GCP ADC, Azure
// DefaultAzureCredential). All methods are no-ops.
type NoopAuthProvider struct {
	providerName string
}

// Compile-time check that NoopAuthProvider satisfies AuthProvider.
var _ AuthProvider = (*NoopAuthProvider)(nil)

// NewNoopAuthProvider creates a NoopAuthProvider for the given provider name.
func NewNoopAuthProvider(providerName string) *NoopAuthProvider {
	return &NoopAuthProvider{providerName: providerName}
}

func (n *NoopAuthProvider) Authenticate() (*AuthResult, error) {
	return &AuthResult{}, nil
}

func (n *NoopAuthProvider) StartRenewal()                     {}
func (n *NoopAuthProvider) StopRenewal()                      {}
func (n *NoopAuthProvider) SetOnTokenRenewed(fn func(string)) {}
func (n *NoopAuthProvider) Token() string                     { return "" }
