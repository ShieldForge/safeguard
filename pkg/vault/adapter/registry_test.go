package adapter

import (
	"testing"
)

func TestListProviders(t *testing.T) {
	providers := ListProviders()
	if len(providers) < 4 {
		t.Fatalf("expected at least 4 providers, got %d: %v", len(providers), providers)
	}

	want := map[string]bool{"hashicorp": false, "aws-secrets-manager": false, "gcp-secret-manager": false, "azure-keyvault": false}
	for _, p := range providers {
		if _, ok := want[p]; ok {
			want[p] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("expected provider %q to be registered", name)
		}
	}
}

func TestNewHashiCorp(t *testing.T) {
	client, err := New(Config{
		Provider: "hashicorp",
		Address:  "http://127.0.0.1:8200",
		Token:    "test-token",
		Debug:    false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewAWSSecretsManager(t *testing.T) {
	client, err := New(Config{
		Provider: "aws-secrets-manager",
		Options:  map[string]string{"region": "eu-west-1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewGCPSecretManager(t *testing.T) {
	client, err := New(Config{
		Provider: "gcp-secret-manager",
		Options:  map[string]string{"project": "my-project"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewGCPSecretManagerMissingProject(t *testing.T) {
	_, err := New(Config{
		Provider: "gcp-secret-manager",
		Options:  map[string]string{},
	})
	if err == nil {
		t.Fatal("expected error when project is missing")
	}
}

func TestNewAzureKeyVault(t *testing.T) {
	client, err := New(Config{
		Provider: "azure-keyvault",
		Options:  map[string]string{"vault-name": "my-vault"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNewAzureKeyVaultMissingName(t *testing.T) {
	_, err := New(Config{
		Provider: "azure-keyvault",
		Options:  map[string]string{},
	})
	if err == nil {
		t.Fatal("expected error when vault-name is missing")
	}
}

func TestNewUnknownProvider(t *testing.T) {
	_, err := New(Config{Provider: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestListProvidersSorted(t *testing.T) {
	providers := ListProviders()
	for i := 1; i < len(providers); i++ {
		if providers[i] < providers[i-1] {
			t.Errorf("providers not sorted: %v", providers)
			break
		}
	}
}

func TestNewAuthHashiCorp(t *testing.T) {
	ap, err := NewAuth(Config{
		Provider: "hashicorp",
		Address:  "http://127.0.0.1:8200",
		Token:    "test-token",
		Auth: AuthConfig{
			Method: "token",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ap == nil {
		t.Fatal("expected non-nil auth provider")
	}
}

func TestNewAuthCloudProviderReturnsNoop(t *testing.T) {
	tests := []struct {
		name     string
		provider string
	}{
		{"aws", "aws-secrets-manager"},
		{"gcp", "gcp-secret-manager"},
		{"azure", "azure-keyvault"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ap, err := NewAuth(Config{Provider: tt.provider})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ap == nil {
				t.Fatal("expected non-nil auth provider")
			}

			// Noop should authenticate without error and return empty token
			result, err := ap.Authenticate()
			if err != nil {
				t.Fatalf("Authenticate() error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil auth result")
			}
			if ap.Token() != "" {
				t.Errorf("expected empty token for noop provider, got %q", ap.Token())
			}

			// Renewal methods should not panic
			ap.StartRenewal()
			ap.StopRenewal()
			ap.SetOnTokenRenewed(func(string) {})
		})
	}
}

func TestNewAuthUnknownProviderReturnsNoop(t *testing.T) {
	ap, err := NewAuth(Config{Provider: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ap == nil {
		t.Fatal("expected non-nil noop auth provider")
	}
	if ap.Token() != "" {
		t.Errorf("expected empty token for noop, got %q", ap.Token())
	}
}
