package auth

// AuthProvider abstracts authentication across different vault backends.
//
// The standard implementation is *Authenticator, which handles HashiCorp Vault
// authentication methods (OIDC, LDAP, token, AWS IAM, AppRole). Cloud providers
// (AWS Secrets Manager, GCP Secret Manager, Azure Key Vault) use NoopAuthProvider
// since their SDKs manage credentials internally.
type AuthProvider interface {
	// Authenticate performs the provider-specific authentication flow.
	Authenticate() (*AuthResult, error)

	// StartRenewal begins background credential/token renewal.
	StartRenewal()

	// StopRenewal stops background credential/token renewal.
	StopRenewal()

	// SetOnTokenRenewed registers a callback fired when credentials are renewed.
	SetOnTokenRenewed(fn func(newToken string))

	// Token returns the current auth token. Returns empty for SDK-managed auth.
	Token() string
}

// Compile-time check that Authenticator satisfies AuthProvider.
var _ AuthProvider = (*Authenticator)(nil)
