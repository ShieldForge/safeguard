package auth

import "testing"

func TestNewNoopAuthProvider(t *testing.T) {
	p := NewNoopAuthProvider("aws-secrets-manager")
	if p == nil {
		t.Fatal("NewNoopAuthProvider() returned nil")
	}
	if p.providerName != "aws-secrets-manager" {
		t.Errorf("providerName = %v, want aws-secrets-manager", p.providerName)
	}
}

func TestNoopAuthProvider_Authenticate(t *testing.T) {
	p := NewNoopAuthProvider("test")
	result, err := p.Authenticate()
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if result == nil {
		t.Fatal("Authenticate() returned nil result")
	}
}

func TestNoopAuthProvider_Token(t *testing.T) {
	p := NewNoopAuthProvider("test")
	if got := p.Token(); got != "" {
		t.Errorf("Token() = %v, want empty string", got)
	}
}

func TestNoopAuthProvider_RenewalMethods(t *testing.T) {
	p := NewNoopAuthProvider("test")
	// These should not panic
	p.StartRenewal()
	p.StopRenewal()
	p.SetOnTokenRenewed(func(s string) {})
}

func TestNoopAuthProvider_ImplementsAuthProvider(t *testing.T) {
	// Compile-time check is already in noop.go, but verify at runtime too
	var _ AuthProvider = (*NoopAuthProvider)(nil)
}
