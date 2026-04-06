# Testing Guide

## Overview

This project includes comprehensive test coverage for all major components:
- **Vault Client** - HTTP client for interacting with Vault API
- **Authentication** - SSO, OIDC, LDAP, and token authentication
- **Filesystem** - FUSE filesystem implementation
- **Main Application** - Platform-specific defaults and utilities

## Running Tests

### Quick Test
```bash
# Run all tests
go test ./...

# Run with verbose output
go test ./... -v

# Run specific package
go test ./pkg/vault/...
go test ./pkg/auth
go test ./pkg/filesystem
```

### Coverage Report
```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
```

### Using Makefile
```bash
# Run tests
make test

# Run tests with verbose output
make test-verbose

# Generate coverage report
make test-coverage

# Run with race detector
make test-race
```

## Test Structure

### Vault Client Tests (`pkg/vault/adapter/hashicorp_test.go`)

**Mock Server**: `MockVaultServer` simulates Vault HTTP API
- Health check endpoint (`/v1/sys/health`)
- LIST endpoint (`/v1/secret/metadata/`)
- READ endpoint (`/v1/secret/data/`)

**Test Coverage**:
- ✅ Client creation with various parameters
- ✅ Health check (ping)
- ✅ Listing secrets at various paths
- ✅ Reading secrets with KV v2 format
- ✅ Path existence checking
- ✅ API path construction
- ✅ Debug mode
- ✅ Error handling

**Example**:
```go
mock := NewMockVaultServer()
defer mock.Close()

mock.SetSecret("app1/database", map[string]interface{}{
    "username": "admin",
    "password": "secret123",
})

client, _ := NewHashiCorpClient(mock.Server.URL, "test-token", false)
data, err := client.Read("app1/database")
```

### Authentication Tests (`pkg/auth/authenticator_test.go`)

**Mock Server**: `MockAuthServer` simulates Vault authentication endpoints
- OIDC auth URL endpoint
- OIDC callback endpoint
- LDAP login endpoint

**Test Coverage**:
- ✅ Authenticator creation with various methods
- ✅ Token authentication
- ✅ LDAP authentication (valid/invalid credentials)
- ✅ OIDC auth flow (start and complete)
- ✅ State extraction from URLs
- ✅ Unsupported auth methods
- ✅ Token lookup and renewal
- ✅ Background renewal start/stop
- ✅ OnTokenRenewed callback
- ✅ canReauthenticate per method
- ✅ Debug mode
- ✅ HTTP client timeout configuration

**Example**:
```go
mock := NewMockAuthServer()
defer mock.Close()

config := &AuthConfig{
    Method:    AuthMethodLDAP,
    VaultAddr: mock.Server.URL,
    Username:  "testuser",
    Password:  "testpass",
}
auth := NewAuthenticator(config)
token, err := auth.GetToken()
```

### Adapter Registry Tests (`pkg/vault/adapter/registry_test.go`)

Tests the provider and auth factory registry that powers the multi-provider model.

**Test Coverage**:
- ✅ All 4 providers registered and creatable (hashicorp, aws-secrets-manager, gcp-secret-manager, azure-keyvault)
- ✅ Required-option validation (GCP `project`, Azure `vault-name`)
- ✅ Unknown provider returns error
- ✅ Provider list is sorted alphabetically
- ✅ `NewAuth()` returns HashiCorp `Authenticator` for hashicorp provider
- ✅ `NewAuth()` returns `NoopAuthProvider` for cloud providers (AWS, GCP, Azure)
- ✅ `NoopAuthProvider` methods are safe no-ops (Authenticate, StartRenewal, StopRenewal, Token)

**Example**:
```go
// Create an auth provider via the adapter registry
ap, err := NewAuth(Config{
    Provider: "hashicorp",
    Address:  "http://127.0.0.1:8200",
    Token:    "test-token",
    Auth:     AuthConfig{Method: "token"},
})

// Cloud providers return a NoopAuthProvider (SDK manages auth)
ap, err := NewAuth(Config{Provider: "aws-secrets-manager"})
result, _ := ap.Authenticate() // returns empty AuthResult, no error
ap.Token()                     // returns ""
```

### Filesystem Tests (`pkg/filesystem/vaultfs_test.go`)

**Mock Client**: `MockVaultClient` implements the Vault client interface
- In-memory storage for secrets and lists
- No HTTP overhead

**Test Coverage**:
- ✅ VaultFS creation
- ✅ Path normalization
- ✅ Secret data formatting
- ✅ Mount options (platform-specific)
- ✅ Getattr (root, files, directories, non-existent)
- ✅ Open (existing/non-existent)
- ✅ Read (with/without offset)
- ✅ Readdir (populated/empty directories)

**Example**:
```go
mockClient := NewMockVaultClient()
mockClient.SetSecret("app/config", map[string]interface{}{
    "key": "value",
})

fs := NewVaultFS(mockClient, false)
stat := &fuse.Stat_t{}
result := fs.Getattr("/app/config", stat, 0)
```

### Main Application Tests (`cmd/cli/main_test.go`)

**Test Coverage**:
- ✅ Default mount point detection per OS
- ✅ Platform-specific behavior

## Mock Servers

### MockVaultServer

Simulates HashiCorp Vault HTTP API for testing:

```go
mock := NewMockVaultServer()
defer mock.Close()

// Add test data
mock.SetSecret("path/to/secret", map[string]interface{}{
    "key1": "value1",
    "key2": "value2",
})
mock.SetList("path", []string{"item1/", "item2"})

// Use in tests
client, _ := NewHashiCorpClient(mock.Server.URL, "test-token", false)
```

### MockAuthServer

Simulates Vault authentication endpoints:

```go
mock := NewMockAuthServer()
defer mock.Close()

// Configure valid credentials
mock.ValidUsers["testuser"] = "testpass"

// Test authentication
config := &AuthConfig{
    Method:    AuthMethodLDAP,
    VaultAddr: mock.Server.URL,
    Username:  "testuser",
    Password:  "testpass",
}
```

### MockVaultClient

Lightweight in-memory mock for filesystem tests:

```go
mockClient := NewMockVaultClient()
mockClient.SetSecret("app/db", map[string]interface{}{
    "host": "localhost",
    "port": 5432,
})
mockClient.SetList("app", []string{"db", "api"})
```

## Writing New Tests

### Test Naming Convention
- `Test<FunctionName>` - Test a specific function
- `Test<Type>_<Method>` - Test a method on a type
- Table-driven tests for multiple scenarios

### Example Test Template

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        want      string
        wantError bool
    }{
        {
            name:      "valid input",
            input:     "test",
            want:      "TEST",
            wantError: false,
        },
        {
            name:      "empty input",
            input:     "",
            want:      "",
            wantError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := MyFunction(tt.input)
            if (err != nil) != tt.wantError {
                t.Errorf("MyFunction() error = %v, wantError %v", err, tt.wantError)
                return
            }
            if got != tt.want {
                t.Errorf("MyFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Test Coverage Report

Current coverage (as of last run):

```
safeguard/cmd/cli              100.0%
safeguard/pkg/auth              85.2%
safeguard/pkg/filesystem        78.9%
safeguard/pkg/vault             92.1%
```

### Viewing Coverage

```bash
# Generate and open HTML report
make test-coverage

# Or manually:
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      - name: Run tests
        run: go test ./... -v -race -coverprofile=coverage.out
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

## Integration Tests

Full integration tests (requiring actual FUSE mounting) should be run separately:

```bash
# These require elevated privileges and platform-specific setup
go test ./... -tags=integration

# Or create separate test files
# filename: *_integration_test.go
```

## Benchmarking

Run performance benchmarks:

```bash
# Run all benchmarks
go test ./... -bench=. -benchmem

# Run specific benchmark
go test ./pkg/vault -bench=BenchmarkClientRead -benchmem

# Compare benchmarks
go test ./... -bench=. -benchmem > old.txt
# Make changes
go test ./... -bench=. -benchmem > new.txt
benchcmp old.txt new.txt
```

## Debugging Tests

```bash
# Run single test
go test ./pkg/vault -run TestClient_Read

# Run with verbose output
go test ./pkg/vault -v -run TestClient_Read

# Debug with delve
dlv test ./pkg/vault -- -test.run TestClient_Read
```

## Best Practices

1. **Use Table-Driven Tests** - Test multiple scenarios efficiently
2. **Mock External Dependencies** - Use mock servers/clients
3. **Test Error Cases** - Don't just test happy paths
4. **Use Subtests** - Organize related test cases
5. **Clean Up Resources** - Always defer cleanup (mock.Close())
6. **Test Edge Cases** - Empty strings, nil values, large inputs
7. **Keep Tests Fast** - Use mocks instead of real services
8. **Write Descriptive Names** - Test names should explain what they test

## Common Issues

### Tests Timeout
```bash
# Increase timeout
go test ./... -timeout 30s
```

### Race Conditions
```bash
# Enable race detector
go test ./... -race
```

### Flaky Tests
- Use deterministic test data
- Avoid time.Sleep in tests
- Mock time-dependent operations
- Use proper synchronization

## Contributing Tests

When adding new features:
1. Write tests first (TDD approach)
2. Ensure at least 80% coverage for new code
3. Include both success and error cases
4. Update this guide if adding new test patterns
