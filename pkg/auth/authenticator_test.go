package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockAuthServer creates a mock Vault authentication server
type MockAuthServer struct {
	Server         *httptest.Server
	ValidTokens    map[string]bool
	ValidUsers     map[string]string // username -> password
	OIDCState      string
	OIDCCode       string
	CallbackCalled bool
	RenewCount     int
}

// NewMockAuthServer creates a new mock authentication server
func NewMockAuthServer() *MockAuthServer {
	mock := &MockAuthServer{
		ValidTokens: map[string]bool{
			"valid-token": true,
		},
		ValidUsers: map[string]string{
			"testuser": "testpass",
		},
		OIDCState: "test-state-123",
		OIDCCode:  "test-code-456",
	}

	mux := http.NewServeMux()

	// OIDC auth URL endpoint
	mux.HandleFunc("/v1/auth/oidc/oidc/auth_url", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"auth_url": "http://localhost:8250/oidc/login?state=" + mock.OIDCState,
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// OIDC callback endpoint
	mux.HandleFunc("/v1/auth/oidc/oidc/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var payload map[string]string
		json.NewDecoder(r.Body).Decode(&payload)

		if payload["state"] != mock.OIDCState || payload["code"] != mock.OIDCCode {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []string{"invalid state or code"},
			})
			return
		}

		mock.CallbackCalled = true
		response := map[string]interface{}{
			"auth": map[string]interface{}{
				"client_token":   "oidc-generated-token",
				"accessor":       "oidc-accessor",
				"lease_duration": 3600,
				"renewable":      true,
				"policies":       []string{"default"},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// LDAP login endpoint
	mux.HandleFunc("/v1/auth/ldap/login/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Extract username from path
		username := strings.TrimPrefix(r.URL.Path, "/v1/auth/ldap/login/")

		var payload map[string]string
		json.NewDecoder(r.Body).Decode(&payload)

		expectedPass, exists := mock.ValidUsers[username]
		if !exists || payload["password"] != expectedPass {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []string{"invalid credentials"},
			})
			return
		}

		response := map[string]interface{}{
			"auth": map[string]interface{}{
				"client_token":   "ldap-generated-token",
				"accessor":       "ldap-accessor",
				"lease_duration": 3600,
				"renewable":      true,
				"policies":       []string{"default"},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// Token lookup-self endpoint
	mux.HandleFunc("/v1/auth/token/lookup-self", func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Vault-Token")
		if token == "" || !mock.ValidTokens[token] {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []string{"permission denied"},
			})
			return
		}
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"accessor":  "test-accessor",
				"ttl":       1800,
				"renewable": true,
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// Token renew-self endpoint
	mux.HandleFunc("/v1/auth/token/renew-self", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		token := r.Header.Get("X-Vault-Token")
		if token == "" || !mock.ValidTokens[token] {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"errors": []string{"permission denied"},
			})
			return
		}
		mock.RenewCount++
		response := map[string]interface{}{
			"auth": map[string]interface{}{
				"client_token":   token,
				"accessor":       "renewed-accessor",
				"lease_duration": 3600,
				"renewable":      true,
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	mock.Server = httptest.NewServer(mux)
	return mock
}

// Close shuts down the mock server
func (m *MockAuthServer) Close() {
	m.Server.Close()
}

func TestNewAuthenticator(t *testing.T) {
	tests := []struct {
		name   string
		config *AuthConfig
	}{
		{
			name: "token auth",
			config: &AuthConfig{
				Method:    AuthMethodToken,
				VaultAddr: "http://localhost:8200",
				Token:     "test-token",
			},
		},
		{
			name: "oidc auth with custom mount",
			config: &AuthConfig{
				Method:    AuthMethodOIDC,
				VaultAddr: "http://localhost:8200",
				MountPath: "oidc-custom",
			},
		},
		{
			name: "ldap auth",
			config: &AuthConfig{
				Method:    AuthMethodLDAP,
				VaultAddr: "http://localhost:8200",
				Username:  "testuser",
				Password:  "testpass",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuthenticator(tt.config)
			if auth == nil {
				t.Error("NewAuthenticator() returned nil")
			}
			if auth.config.Method != tt.config.Method {
				t.Errorf("Method = %v, want %v", auth.config.Method, tt.config.Method)
			}
			// Check default mount path is set
			if tt.config.MountPath == "" && auth.config.MountPath != string(tt.config.Method) {
				t.Errorf("MountPath not set to default, got %v", auth.config.MountPath)
			}
		})
	}
}

func TestAuthenticator_TokenAuth(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantToken string
		wantError bool
	}{
		{
			name:      "valid token",
			token:     "test-token-123",
			wantToken: "test-token-123",
			wantError: false,
		},
		{
			name:      "empty token",
			token:     "",
			wantToken: "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AuthConfig{
				Method:    AuthMethodToken,
				VaultAddr: "http://localhost:8200",
				Token:     tt.token,
			}
			auth := NewAuthenticator(config)

			token, err := auth.GetToken()
			if (err != nil) != tt.wantError {
				t.Errorf("GetToken() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if token != tt.wantToken {
				t.Errorf("GetToken() = %v, want %v", token, tt.wantToken)
			}
		})
	}
}

func TestAuthenticator_LDAPAuth(t *testing.T) {
	mock := NewMockAuthServer()
	defer mock.Close()

	tests := []struct {
		name      string
		username  string
		password  string
		wantError bool
	}{
		{
			name:      "valid credentials",
			username:  "testuser",
			password:  "testpass",
			wantError: false,
		},
		{
			name:      "invalid password",
			username:  "testuser",
			password:  "wrongpass",
			wantError: true,
		},
		{
			name:      "invalid username",
			username:  "nonexistent",
			password:  "testpass",
			wantError: true,
		},
		{
			name:      "empty credentials",
			username:  "",
			password:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AuthConfig{
				Method:    AuthMethodLDAP,
				VaultAddr: mock.Server.URL,
				Username:  tt.username,
				Password:  tt.password,
			}
			auth := NewAuthenticator(config)

			token, err := auth.GetToken()
			if (err != nil) != tt.wantError {
				t.Errorf("GetToken() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError && token == "" {
				t.Error("GetToken() returned empty token")
			}
		})
	}
}

func TestAuthenticator_StartOIDCAuth(t *testing.T) {
	mock := NewMockAuthServer()
	defer mock.Close()

	config := &AuthConfig{
		Method:    AuthMethodOIDC,
		VaultAddr: mock.Server.URL,
		Role:      "default",
	}
	auth := NewAuthenticator(config)

	authURL, state, err := auth.startOIDCAuth()
	if err != nil {
		t.Fatalf("startOIDCAuth() error = %v", err)
	}

	if authURL == "" {
		t.Error("startOIDCAuth() returned empty auth URL")
	}

	if state == "" {
		t.Error("startOIDCAuth() returned empty state")
	}

	if !strings.Contains(authURL, "state=") {
		t.Error("Auth URL does not contain state parameter")
	}
}

func TestAuthenticator_CompleteOIDCAuth(t *testing.T) {
	mock := NewMockAuthServer()
	defer mock.Close()

	config := &AuthConfig{
		Method:    AuthMethodOIDC,
		VaultAddr: mock.Server.URL,
	}
	auth := NewAuthenticator(config)

	tests := []struct {
		name      string
		state     string
		code      string
		wantError bool
	}{
		{
			name:      "valid callback",
			state:     mock.OIDCState,
			code:      mock.OIDCCode,
			wantError: false,
		},
		{
			name:      "invalid state",
			state:     "invalid-state",
			code:      mock.OIDCCode,
			wantError: true,
		},
		{
			name:      "invalid code",
			state:     mock.OIDCState,
			code:      "invalid-code",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := auth.completeOIDCAuth(tt.state, tt.code)
			if (err != nil) != tt.wantError {
				t.Errorf("completeOIDCAuth() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if !tt.wantError {
				if result == nil {
					t.Fatal("completeOIDCAuth() returned nil result")
				}
				if result.Token == "" {
					t.Error("completeOIDCAuth() returned empty token")
				}
				if result.LeaseDuration != 3600 {
					t.Errorf("LeaseDuration = %v, want 3600", result.LeaseDuration)
				}
				if !result.Renewable {
					t.Error("Expected token to be renewable")
				}
			}
		})
	}
}

func TestExtractStateFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "url with state",
			url:  "http://localhost:8250/oidc/login?state=abc123&other=param",
			want: "abc123",
		},
		{
			name: "url with state at end",
			url:  "http://localhost:8250/oidc/login?other=param&state=xyz789",
			want: "xyz789",
		},
		{
			name: "url without state",
			url:  "http://localhost:8250/oidc/login?other=param",
			want: "",
		},
		{
			name: "empty url",
			url:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractStateFromURL(tt.url)
			if got != tt.want {
				t.Errorf("extractStateFromURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthenticator_UnsupportedMethod(t *testing.T) {
	config := &AuthConfig{
		Method:    "unsupported",
		VaultAddr: "http://localhost:8200",
	}
	auth := NewAuthenticator(config)

	_, err := auth.GetToken()
	if err == nil {
		t.Error("GetToken() expected error for unsupported method, got nil")
	}
}

func TestAuthenticator_AWSAuth(t *testing.T) {
	config := &AuthConfig{
		Method:    AuthMethodAWS,
		VaultAddr: "http://localhost:8200",
	}
	auth := NewAuthenticator(config)

	_, err := auth.GetToken()
	if err == nil {
		t.Error("GetToken() expected error for AWS auth (not implemented), got nil")
	}
}

func TestAuthenticator_AppRoleAuth(t *testing.T) {
	config := &AuthConfig{
		Method:    AuthMethodAppRole,
		VaultAddr: "http://localhost:8200",
	}
	auth := NewAuthenticator(config)

	_, err := auth.GetToken()
	if err == nil {
		t.Error("GetToken() expected error for AppRole auth (not implemented), got nil")
	}
}

func TestAuthenticator_WithDebugMode(t *testing.T) {
	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: "http://localhost:8200",
		Token:     "test-token",
		Debug:     true,
	}
	auth := NewAuthenticator(config)

	if !auth.config.Debug {
		t.Error("Debug mode not enabled in authenticator")
	}
}

func TestAuthenticator_HTTPClientTimeout(t *testing.T) {
	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: "http://localhost:8200",
		Token:     "test-token",
	}
	auth := NewAuthenticator(config)

	if auth.httpClient.Timeout != 30*time.Second {
		t.Errorf("HTTP client timeout = %v, want 30s", auth.httpClient.Timeout)
	}
}

func TestAuthenticator_Authenticate_Token(t *testing.T) {
	mock := NewMockAuthServer()
	defer mock.Close()

	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: mock.Server.URL,
		Token:     "valid-token",
	}
	auth := NewAuthenticator(config)

	result, err := auth.Authenticate()
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if result.Token != "valid-token" {
		t.Errorf("Token = %v, want valid-token", result.Token)
	}
	if !result.Renewable {
		t.Error("Expected token to be renewable")
	}
	if result.LeaseDuration != 1800 {
		t.Errorf("LeaseDuration = %v, want 1800", result.LeaseDuration)
	}
}

func TestAuthenticator_Authenticate_TokenEmpty(t *testing.T) {
	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: "http://localhost:8200",
		Token:     "",
	}
	auth := NewAuthenticator(config)

	_, err := auth.Authenticate()
	if err == nil {
		t.Error("Authenticate() expected error for empty token, got nil")
	}
}

func TestAuthenticator_Authenticate_LDAP(t *testing.T) {
	mock := NewMockAuthServer()
	defer mock.Close()

	config := &AuthConfig{
		Method:    AuthMethodLDAP,
		VaultAddr: mock.Server.URL,
		Username:  "testuser",
		Password:  "testpass",
	}
	auth := NewAuthenticator(config)

	result, err := auth.Authenticate()
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if result.Token != "ldap-generated-token" {
		t.Errorf("Token = %v, want ldap-generated-token", result.Token)
	}
	if !result.Renewable {
		t.Error("Expected token to be renewable")
	}
	if result.LeaseDuration != 3600 {
		t.Errorf("LeaseDuration = %v, want 3600", result.LeaseDuration)
	}
	if result.Accessor != "ldap-accessor" {
		t.Errorf("Accessor = %v, want ldap-accessor", result.Accessor)
	}
}

func TestAuthenticator_LookupToken(t *testing.T) {
	mock := NewMockAuthServer()
	defer mock.Close()

	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: mock.Server.URL,
		Token:     "valid-token",
	}
	auth := NewAuthenticator(config)

	result, err := auth.LookupToken("valid-token")
	if err != nil {
		t.Fatalf("LookupToken() error = %v", err)
	}
	if result.LeaseDuration != 1800 {
		t.Errorf("LeaseDuration = %v, want 1800", result.LeaseDuration)
	}
	if !result.Renewable {
		t.Error("Expected token to be renewable")
	}
	if result.Accessor != "test-accessor" {
		t.Errorf("Accessor = %v, want test-accessor", result.Accessor)
	}
}

func TestAuthenticator_LookupToken_Invalid(t *testing.T) {
	mock := NewMockAuthServer()
	defer mock.Close()

	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: mock.Server.URL,
	}
	auth := NewAuthenticator(config)

	_, err := auth.LookupToken("invalid-token")
	if err == nil {
		t.Error("LookupToken() expected error for invalid token, got nil")
	}
}

func TestAuthenticator_RenewToken(t *testing.T) {
	mock := NewMockAuthServer()
	defer mock.Close()

	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: mock.Server.URL,
		Token:     "valid-token",
	}
	auth := NewAuthenticator(config)

	// Must authenticate first to populate result
	_, err := auth.Authenticate()
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	renewed, err := auth.RenewToken()
	if err != nil {
		t.Fatalf("RenewToken() error = %v", err)
	}
	if renewed.Token != "valid-token" {
		t.Errorf("Token = %v, want valid-token", renewed.Token)
	}
	if renewed.LeaseDuration != 3600 {
		t.Errorf("LeaseDuration = %v, want 3600", renewed.LeaseDuration)
	}
	if !renewed.Renewable {
		t.Error("Expected renewed token to be renewable")
	}
	if mock.RenewCount != 1 {
		t.Errorf("RenewCount = %v, want 1", mock.RenewCount)
	}
}

func TestAuthenticator_RenewToken_NoAuth(t *testing.T) {
	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: "http://localhost:8200",
		Token:     "test-token",
	}
	auth := NewAuthenticator(config)

	_, err := auth.RenewToken()
	if err == nil {
		t.Error("RenewToken() expected error when no auth result, got nil")
	}
}

func TestAuthenticator_RenewToken_NotRenewable(t *testing.T) {
	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: "http://localhost:8200",
		Token:     "test-token",
	}
	auth := NewAuthenticator(config)

	// Set a non-renewable result
	auth.result = &AuthResult{
		Token:         "test-token",
		LeaseDuration: 3600,
		Renewable:     false,
	}

	_, err := auth.RenewToken()
	if err == nil {
		t.Error("RenewToken() expected error for non-renewable token, got nil")
	}
}

func TestAuthenticator_StartStopRenewal(t *testing.T) {
	mock := NewMockAuthServer()
	defer mock.Close()

	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: mock.Server.URL,
		Token:     "valid-token",
	}
	auth := NewAuthenticator(config)

	_, err := auth.Authenticate()
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Start renewal
	auth.StartRenewal()

	// Verify goroutine is running
	auth.mu.Lock()
	renewing := auth.renewing
	auth.mu.Unlock()
	if !renewing {
		t.Error("Expected renewing to be true")
	}

	// Starting again should be a no-op
	auth.StartRenewal()

	// Stop renewal
	auth.StopRenewal()

	auth.mu.Lock()
	renewing = auth.renewing
	auth.mu.Unlock()
	if renewing {
		t.Error("Expected renewing to be false after StopRenewal")
	}

	// Stopping again should be safe
	auth.StopRenewal()
}

func TestAuthenticator_StartRenewal_NotRenewable(t *testing.T) {
	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: "http://localhost:8200",
		Token:     "test-token",
	}
	auth := NewAuthenticator(config)
	auth.result = &AuthResult{
		Token:         "test-token",
		LeaseDuration: 3600,
		Renewable:     false,
	}

	auth.StartRenewal()

	// Should not start renewal for non-renewable token
	time.Sleep(50 * time.Millisecond)
	auth.mu.Lock()
	renewing := auth.renewing
	auth.mu.Unlock()
	if renewing {
		t.Error("Expected renewing to be false for non-renewable token")
	}
}

func TestAuthenticator_Token(t *testing.T) {
	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: "http://localhost:8200",
		Token:     "my-token",
	}
	auth := NewAuthenticator(config)

	// Before auth, Token() returns empty
	if got := auth.Token(); got != "" {
		t.Errorf("Token() before auth = %v, want empty", got)
	}

	auth.result = &AuthResult{Token: "my-token"}
	if got := auth.Token(); got != "my-token" {
		t.Errorf("Token() = %v, want my-token", got)
	}
}

func TestAuthenticator_CanReauthenticate(t *testing.T) {
	tests := []struct {
		method AuthMethod
		want   bool
	}{
		{AuthMethodToken, true},
		{AuthMethodAWS, true},
		{AuthMethodAppRole, true},
		{AuthMethodOIDC, false},
		{AuthMethodLDAP, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			auth := NewAuthenticator(&AuthConfig{Method: tt.method, VaultAddr: "http://localhost:8200"})
			if got := auth.canReauthenticate(); got != tt.want {
				t.Errorf("canReauthenticate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthenticator_CallbackPort(t *testing.T) {
	// Default port
	auth := NewAuthenticator(&AuthConfig{Method: AuthMethodOIDC, VaultAddr: "http://localhost:8200"})
	if got := auth.callbackPort(); got != 8250 {
		t.Errorf("callbackPort() = %v, want 8250", got)
	}

	// Custom port
	auth = NewAuthenticator(&AuthConfig{
		Method:       AuthMethodOIDC,
		VaultAddr:    "http://localhost:8200",
		CallbackPort: 9090,
	})
	if got := auth.callbackPort(); got != 9090 {
		t.Errorf("callbackPort() = %v, want 9090", got)
	}
}

func TestAuthenticator_OnTokenRenewed(t *testing.T) {
	mock := NewMockAuthServer()
	defer mock.Close()

	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: mock.Server.URL,
		Token:     "valid-token",
	}
	auth := NewAuthenticator(config)

	var renewedToken string
	auth.SetOnTokenRenewed(func(newToken string) {
		renewedToken = newToken
	})

	// Authenticate first
	_, err := auth.Authenticate()
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Renew should trigger callback
	_, err = auth.RenewToken()
	if err != nil {
		t.Fatalf("RenewToken() error = %v", err)
	}

	if renewedToken != "valid-token" {
		t.Errorf("OnTokenRenewed callback got token %q, want %q", renewedToken, "valid-token")
	}
}

func TestAuthenticator_RenewToken_InvalidToken(t *testing.T) {
	mock := NewMockAuthServer()
	defer mock.Close()

	config := &AuthConfig{
		Method:    AuthMethodToken,
		VaultAddr: mock.Server.URL,
		Token:     "valid-token",
	}
	auth := NewAuthenticator(config)

	// Set a result with an invalid token (not in ValidTokens)
	auth.result = &AuthResult{
		Token:         "invalid-token-for-renewal",
		LeaseDuration: 3600,
		Renewable:     true,
	}

	_, err := auth.RenewToken()
	if err == nil {
		t.Error("RenewToken() expected error for invalid token, got nil")
	}
}
