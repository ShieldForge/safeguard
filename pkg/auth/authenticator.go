package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"safeguard/pkg/logger"
	"strings"
	"sync"
	"time"

	"github.com/pkg/browser"
)

// AuthMethod represents the authentication method to use
type AuthMethod string

const (
	AuthMethodOIDC    AuthMethod = "oidc"
	AuthMethodLDAP    AuthMethod = "ldap"
	AuthMethodToken   AuthMethod = "token"
	AuthMethodAWS     AuthMethod = "aws"
	AuthMethodAppRole AuthMethod = "approle"

	defaultCallbackPort = 8250
)

// AuthConfig holds configuration for authentication
type AuthConfig struct {
	Method       AuthMethod
	VaultAddr    string
	Token        string // Used for token auth method
	Username     string // Used for LDAP
	Password     string // Used for LDAP
	Role         string // Used for OIDC/AppRole
	MountPath    string // Auth mount path (default: method name)
	CallbackPort int    // OIDC callback port (default: 8250)
	Debug        bool
	Logger       *logger.Logger // Optional shared logger; if nil a default stdout logger is created
}

// AuthResult holds the result of an authentication including token metadata.
type AuthResult struct {
	Token         string
	LeaseDuration int    // Token TTL in seconds
	Renewable     bool   // Whether the token can be renewed
	Accessor      string // Token accessor
}

// Authenticator handles authentication with Vault
type Authenticator struct {
	config     *AuthConfig
	httpClient *http.Client
	logger     *logger.Logger

	// Token renewal state
	mu             sync.Mutex
	result         *AuthResult
	renewDone      chan struct{}
	renewing       bool
	onTokenRenewed func(newToken string)
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(config *AuthConfig) *Authenticator {
	if config.MountPath == "" {
		config.MountPath = string(config.Method)
	}
	if config.CallbackPort == 0 {
		config.CallbackPort = defaultCallbackPort
	}

	return &Authenticator{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: func() *logger.Logger {
			if config.Logger != nil {
				return config.Logger
			}
			return logger.New(os.Stdout, config.Debug)
		}(),
	}
}

// SetOnTokenRenewed sets a callback that fires whenever the token is renewed.
func (a *Authenticator) SetOnTokenRenewed(fn func(newToken string)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onTokenRenewed = fn
}

// callbackPort returns the configured OIDC callback port.
func (a *Authenticator) callbackPort() int {
	if a.config.CallbackPort > 0 {
		return a.config.CallbackPort
	}
	return defaultCallbackPort
}

// GetToken authenticates and returns a Vault token.
// This delegates to Authenticate() for the actual auth flow.
func (a *Authenticator) GetToken() (string, error) {
	result, err := a.Authenticate()
	if err != nil {
		return "", err
	}
	return result.Token, nil
}

// Authenticate authenticates and returns an AuthResult with token metadata.
func (a *Authenticator) Authenticate() (*AuthResult, error) {
	switch a.config.Method {
	case AuthMethodToken:
		if a.config.Token == "" {
			return nil, fmt.Errorf("token is required for token auth method")
		}
		// For pre-provided tokens, look up metadata from Vault
		result := &AuthResult{Token: a.config.Token}
		if info, err := a.LookupToken(a.config.Token); err == nil {
			result.LeaseDuration = info.LeaseDuration
			result.Renewable = info.Renewable
			result.Accessor = info.Accessor
		}
		a.mu.Lock()
		a.result = result
		a.mu.Unlock()
		return result, nil

	case AuthMethodOIDC:
		return a.oidcAuthenticate()
	case AuthMethodLDAP:
		return a.ldapAuthenticate()
	case AuthMethodAWS:
		return nil, fmt.Errorf("AWS auth not yet implemented")
	case AuthMethodAppRole:
		return nil, fmt.Errorf("AppRole auth not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported auth method: %s", a.config.Method)
	}
}

// oidcAuthenticate performs OIDC auth and returns full AuthResult.
func (a *Authenticator) oidcAuthenticate() (*AuthResult, error) {
	a.logger.Debug("Starting OIDC authentication flow", nil)

	authURL, state, err := a.startOIDCAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to start OIDC auth: %w", err)
	}

	a.logger.Debug("Opening browser for authentication", map[string]interface{}{
		"auth_url": authURL,
	})
	logger.Info("Opening browser for SSO authentication", nil)
	logger.Info("If browser doesn't open automatically, visit the URL", map[string]interface{}{
		"auth_url": authURL,
	})
	if err := browser.OpenURL(authURL); err != nil {
		a.logger.Error("Failed to open browser automatically", map[string]interface{}{
			"error": err.Error(),
		})
	}

	result, err := a.waitForOIDCCallback(state)
	if err != nil {
		return nil, fmt.Errorf("OIDC callback failed: %w", err)
	}

	a.logger.Debug("OIDC authentication successful", nil)
	a.mu.Lock()
	a.result = result
	a.mu.Unlock()
	return result, nil
}

// OIDCFlowResult is the outcome of a headless OIDC flow started by StartOIDCFlow.
type OIDCFlowResult struct {
	Result *AuthResult
	Err    error
}

// StartOIDCFlow begins an OIDC authentication flow without opening a browser.
// It contacts Vault to obtain the authorisation URL, starts the local callback
// listener on the configured callback port (default 8250), and returns immediately.
// The caller is responsible for directing the user to authURL; the returned channel
// delivers the result once the browser completes the IdP login and the callback
// is received.
func (a *Authenticator) StartOIDCFlow() (authURL string, done <-chan OIDCFlowResult, err error) {
	url, state, err := a.startOIDCAuth()
	if err != nil {
		return "", nil, fmt.Errorf("start OIDC auth: %w", err)
	}

	ch := make(chan OIDCFlowResult, 1)
	go func() {
		result, err := a.waitForOIDCCallback(state)
		if err != nil {
			ch <- OIDCFlowResult{Err: err}
			return
		}
		a.mu.Lock()
		a.result = result
		a.mu.Unlock()
		ch <- OIDCFlowResult{Result: result}
	}()
	return url, ch, nil
}

// startOIDCAuth initiates the OIDC auth flow
func (a *Authenticator) startOIDCAuth() (string, string, error) {
	apiPath := fmt.Sprintf("%s/v1/auth/%s/oidc/auth_url",
		strings.TrimSuffix(a.config.VaultAddr, "/"),
		a.config.MountPath)

	payload := map[string]string{
		"redirect_uri": fmt.Sprintf("http://localhost:%d/oidc/callback", a.callbackPort()),
	}

	if a.config.Role != "" {
		payload["role"] = a.config.Role
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal OIDC auth request: %w", err)
	}
	req, err := http.NewRequest("POST", apiPath, strings.NewReader(string(body)))
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to get auth URL (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data struct {
			AuthURL string `json:"auth_url"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	// Extract state from auth URL for validation
	state := extractStateFromURL(result.Data.AuthURL)

	return result.Data.AuthURL, state, nil
}

// waitForOIDCCallback starts a local server and waits for the OIDC callback.
// Uses a dedicated ServeMux to avoid polluting http.DefaultServeMux.
func (a *Authenticator) waitForOIDCCallback(expectedState string) (*AuthResult, error) {
	resultChan := make(chan *AuthResult, 1)
	errorChan := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", a.callbackPort()),
		Handler: mux,
	}

	mux.HandleFunc("/oidc/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")

		if state == "" || code == "" {
			errorChan <- fmt.Errorf("missing state or code in callback")
			http.Error(w, "Authentication failed", http.StatusBadRequest)
			return
		}

		authResult, err := a.completeOIDCAuth(state, code)
		if err != nil {
			errorChan <- err
			http.Error(w, "Authentication failed", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<html>
			<head><title>Authentication Successful</title></head>
			<body>
				<h1>Authentication Successful</h1>
				<p>You can close this window and return to the application.</p>
				<script>window.close();</script>
			</body>
			</html>
		`)

		resultChan <- authResult
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorChan <- err
		}
	}()

	select {
	case result := <-resultChan:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return result, nil
	case err := <-errorChan:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return nil, err
	case <-time.After(5 * time.Minute):
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
		return nil, fmt.Errorf("authentication timeout")
	}
}

// completeOIDCAuth completes the OIDC authentication flow and returns full AuthResult.
func (a *Authenticator) completeOIDCAuth(state, code string) (*AuthResult, error) {
	apiPath := fmt.Sprintf("%s/v1/auth/%s/oidc/callback",
		strings.TrimSuffix(a.config.VaultAddr, "/"),
		a.config.MountPath)

	payload := map[string]string{
		"state": state,
		"code":  code,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OIDC callback request: %w", err)
	}
	req, err := http.NewRequest("POST", apiPath, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OIDC callback failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Auth struct {
			ClientToken   string `json:"client_token"`
			Accessor      string `json:"accessor"`
			LeaseDuration int    `json:"lease_duration"`
			Renewable     bool   `json:"renewable"`
		} `json:"auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &AuthResult{
		Token:         result.Auth.ClientToken,
		LeaseDuration: result.Auth.LeaseDuration,
		Renewable:     result.Auth.Renewable,
		Accessor:      result.Auth.Accessor,
	}, nil
}

// ldapAuthenticate performs LDAP auth and returns full AuthResult.
func (a *Authenticator) ldapAuthenticate() (*AuthResult, error) {
	if a.config.Username == "" || a.config.Password == "" {
		return nil, fmt.Errorf("username and password are required for LDAP auth")
	}

	apiPath := fmt.Sprintf("%s/v1/auth/%s/login/%s",
		strings.TrimSuffix(a.config.VaultAddr, "/"),
		a.config.MountPath,
		a.config.Username)

	payload := map[string]string{
		"password": a.config.Password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LDAP request: %w", err)
	}
	req, err := http.NewRequest("POST", apiPath, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("LDAP authentication failed with status %d", resp.StatusCode)
	}

	var result struct {
		Auth struct {
			ClientToken   string `json:"client_token"`
			Accessor      string `json:"accessor"`
			LeaseDuration int    `json:"lease_duration"`
			Renewable     bool   `json:"renewable"`
		} `json:"auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	authResult := &AuthResult{
		Token:         result.Auth.ClientToken,
		LeaseDuration: result.Auth.LeaseDuration,
		Renewable:     result.Auth.Renewable,
		Accessor:      result.Auth.Accessor,
	}
	a.mu.Lock()
	a.result = authResult
	a.mu.Unlock()
	return authResult, nil
}

// LookupToken retrieves metadata about the given token from Vault.
func (a *Authenticator) LookupToken(token string) (*AuthResult, error) {
	apiPath := fmt.Sprintf("%s/v1/auth/token/lookup-self",
		strings.TrimSuffix(a.config.VaultAddr, "/"))

	req, err := http.NewRequest("GET", apiPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Vault-Token", token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token lookup failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Data struct {
			Accessor  string `json:"accessor"`
			TTL       int    `json:"ttl"`
			Renewable bool   `json:"renewable"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &AuthResult{
		Token:         token,
		LeaseDuration: result.Data.TTL,
		Renewable:     result.Data.Renewable,
		Accessor:      result.Data.Accessor,
	}, nil
}

// RenewToken renews the current token and returns updated metadata.
func (a *Authenticator) RenewToken() (*AuthResult, error) {
	a.mu.Lock()
	current := a.result
	a.mu.Unlock()

	if current == nil {
		return nil, fmt.Errorf("no token to renew; authenticate first")
	}
	if !current.Renewable {
		return nil, fmt.Errorf("token is not renewable")
	}

	apiPath := fmt.Sprintf("%s/v1/auth/token/renew-self",
		strings.TrimSuffix(a.config.VaultAddr, "/"))

	req, err := http.NewRequest("POST", apiPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Vault-Token", current.Token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token renewal failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Auth struct {
			ClientToken   string `json:"client_token"`
			Accessor      string `json:"accessor"`
			LeaseDuration int    `json:"lease_duration"`
			Renewable     bool   `json:"renewable"`
		} `json:"auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	renewed := &AuthResult{
		Token:         result.Auth.ClientToken,
		LeaseDuration: result.Auth.LeaseDuration,
		Renewable:     result.Auth.Renewable,
		Accessor:      result.Auth.Accessor,
	}

	a.mu.Lock()
	a.result = renewed
	callback := a.onTokenRenewed
	a.mu.Unlock()

	if callback != nil {
		callback(renewed.Token)
	}

	a.logger.Debug("Token renewed successfully", map[string]interface{}{
		"lease_duration": renewed.LeaseDuration,
		"renewable":      renewed.Renewable,
	})

	return renewed, nil
}

// canReauthenticate reports whether the configured auth method supports
// non-interactive re-authentication (i.e., headless credential replay).
// OIDC and LDAP require user interaction, so we don't auto-reauthenticate those.
func (a *Authenticator) canReauthenticate() bool {
	switch a.config.Method {
	case AuthMethodToken, AuthMethodAWS, AuthMethodAppRole:
		return true
	default:
		return false
	}
}

// StartRenewal begins a background goroutine that renews the token before it expires.
// It renews at roughly half the remaining TTL. Call StopRenewal to stop.
func (a *Authenticator) StartRenewal() {
	a.mu.Lock()
	if a.renewing {
		a.mu.Unlock()
		return
	}
	a.renewDone = make(chan struct{})
	a.renewing = true
	result := a.result
	a.mu.Unlock()

	if result == nil || !result.Renewable || result.LeaseDuration <= 0 {
		a.logger.Debug("Token is not renewable or has no TTL; skipping background renewal", nil)
		a.mu.Lock()
		a.renewing = false
		a.mu.Unlock()
		return
	}

	go a.renewLoop()
}

// StopRenewal stops the background token renewal goroutine.
func (a *Authenticator) StopRenewal() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.renewing {
		close(a.renewDone)
		a.renewing = false
	}
}

func (a *Authenticator) renewLoop() {
	for {
		a.mu.Lock()
		result := a.result
		a.mu.Unlock()

		if result == nil || !result.Renewable || result.LeaseDuration <= 0 {
			a.logger.Debug("Token no longer renewable; stopping renewal loop", nil)
			return
		}

		// Renew at half the TTL, with a minimum of 10 seconds
		sleepDuration := time.Duration(result.LeaseDuration) * time.Second / 2
		if sleepDuration < 10*time.Second {
			sleepDuration = 10 * time.Second
		}

		select {
		case <-a.renewDone:
			return
		case <-time.After(sleepDuration):
			renewed, err := a.RenewToken()
			if err != nil {
				a.logger.Error("Token renewal failed", map[string]interface{}{
					"error": err.Error(),
				})
				// Only attempt re-authentication for non-interactive methods
				if !a.canReauthenticate() {
					a.logger.Error("Cannot re-authenticate automatically for interactive auth method", map[string]interface{}{
						"method": string(a.config.Method),
					})
					return
				}
				newResult, reAuthErr := a.Authenticate()
				if reAuthErr != nil {
					a.logger.Error("Re-authentication also failed", map[string]interface{}{
						"error": reAuthErr.Error(),
					})
					return
				}
				a.logger.Info("Re-authenticated after renewal failure", map[string]interface{}{
					"lease_duration": newResult.LeaseDuration,
				})
				continue
			}
			a.logger.Debug("Background token renewal succeeded", map[string]interface{}{
				"lease_duration": renewed.LeaseDuration,
			})
		}
	}
}

// Token returns the current token. Safe for concurrent use.
func (a *Authenticator) Token() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.result != nil {
		return a.result.Token
	}
	return ""
}

// extractStateFromURL extracts the state parameter from a URL using net/url.
func extractStateFromURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return u.Query().Get("state")
}
