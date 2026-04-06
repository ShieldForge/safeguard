# safeguard Build Server

The safeguard Build Server is an HTTP service that allows you to create custom builds of the safeguard binary with hard-coded configuration values. This is useful for distributing customized versions to clients with pre-configured settings.

## Features

- 🚀 Build custom binaries with embedded default values
- 🌐 Web UI for easy configuration
- 📦 Support for multiple platforms (Windows, Linux, macOS)
- 🔒 SHA256 checksum verification
- ⬇️ Direct binary download
- 🔧 REST API for automation

## Quick Start

### Start the Build Server

```powershell
# From the project root
cd cmd/builder
go run main.go -source ../.. -output ./builds
```

Or build and run:

```powershell
go build -o build-server.exe ./cmd/builder
./build-server.exe -source . -output ./builds
```

### Command Line Options

- `-port` - HTTP server port (default: `8080`)
- `-source` - Source directory of safeguard project (default: `.`)
- `-work` - Work directory for builds (default: `./build-work`)
- `-output` - Output directory for built binaries (default: `./build-output`)

### Access the Web UI

Open your browser and navigate to:
```
http://localhost:8080
```

## Usage

### Web UI

1. Open http://localhost:8080 in your browser
2. Fill in the configuration values you want to embed:
   - **Default Vault Address** - Pre-configured Vault server URL
   - **Default Auth Method** - Authentication method (OIDC, LDAP, token, aws, approle)
   - **Conditional Fields (based on auth method):**
     - **LDAP:** Optional username and password fields (⚠️ security warning shown)
     - **Token:** Optional vault token field (⚠️ security warning shown)
   - **Default Mount Point** - Default drive/mount location
   - **Default Auth Role** - Pre-configured authentication role
   - **Default Policy Path** - Path to REGO policy files
   - **Default Mapping Config** - Path to path mapping configuration
   - **Version** - Custom version number
   - **Build Tag** - Custom tag (e.g., customer name)
   - **Target OS** - Windows, Linux, or macOS
   - **Target Architecture** - amd64, arm64, or 386
3. Click "Build Custom Binary"
4. Download the resulting binary

**⚠️ Security Note:** When selecting LDAP or Token authentication, optional credential fields appear. These allow you to embed default credentials in the binary, but this is **only recommended for testing environments**. For production, leave these fields empty and users will be prompted at runtime.

### REST API

#### POST /api/build

Build a custom binary with specified configuration.

**Request:**
```json
{
  "default_vault_addr": "https://vault.company.com",
  "default_auth_method": "oidc",
  "default_mount_point": "V:",
  "default_auth_role": "production-role",
  "default_policy_path": "/etc/safeguard/policies",
  "default_mapping_config": "/etc/safeguard/mapping.json",
  "version": "1.0.0",
  "build_tag": "acme-corp",
  "target_os": "windows",
  "target_arch": "amd64"
}
```

**Request with LDAP Credentials (Testing Only):**
```json
{
  "default_vault_addr": "https://vault.company.com",
  "default_auth_method": "ldap",
  "default_ldap_username": "service-account",
  "default_ldap_password": "password123",
  "version": "1.0.0",
  "build_tag": "test-build",
  "target_os": "windows",
  "target_arch": "amd64"
}
```

**Request with Token (Testing Only):**
```json
{
  "default_vault_addr": "https://vault.company.com",
  "default_auth_method": "token",
  "default_vault_token": "hvs.CAESIJ...",
  "version": "1.0.0",
  "build_tag": "test-build",
  "target_os": "windows",
  "target_arch": "amd64"
}
```

⚠️ **Security Warning:** Embedding credentials in binaries is not recommended for production use. Leave credential fields empty for production builds - users will be prompted at runtime.

**Response:**
```json
{
  "binary_path": "build-output/safeguard-1.0.0-windows-amd64-acme-corp.exe",
  "checksum": "a1b2c3d4e5f6...",
  "size": 45678901,
  "build_time": "2026-02-05T10:30:00Z",
  "config": { ... }
}
```

#### GET /api/download/{filename}

Download a built binary.

**Example:**
```
GET /api/download/safeguard-1.0.0-windows-amd64-acme-corp.exe
```

#### GET /api/health

Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-02-05T10:30:00Z"
}
```

## How It Works

The build server uses Go's `-ldflags -X` feature to override string variables at compile time. When you specify default values:

1. The builder creates a `go build` command with appropriate `-ldflags`
2. These flags override the default variable values in [main.go](../cli/main.go)
3. The resulting binary has your custom defaults hard-coded
4. Users can still override these with command-line flags

### Example

If you build with:
- Default Vault Address: `https://vault.company.com`
- Default Auth Method: `ldap`

The built binary will use these as defaults, but users can still run:
```
safeguard.exe -vault-addr https://other-vault.com -auth-method token
```

## Build Variables

The following variables can be customized at build time.

| Variable | JSON field | Description | Default |
|----------|------------|-------------|---------|
| `defaultVaultAddr` | `default_vault_addr` | Vault server address | `http://127.0.0.1:8200` |
| `defaultAuthMethod` | `default_auth_method` | Authentication method | `oidc` |
| `defaultVaultProvider` | `default_vault_provider` | Vault backend (`hashicorp`, `aws-secrets-manager`, `gcp-secret-manager`, `azure-keyvault`) | `hashicorp` |
| `defaultMountPoint` | `default_mount_point` | Mount point / drive letter | `V:` (Windows) |
| `defaultAuthRole` | `default_auth_role` | Auth role name | Empty |
| `defaultAuthMount` | `default_auth_mount` | Custom auth mount path | Empty |
| `defaultPolicyPath` | `default_policy_path` | Path to REGO policies | Empty |
| `defaultMappingPath` | `default_mapping_config` | Path to path-mapping JSON | Empty |
| `defaultAuditLog` | `default_audit_log` | Audit log file path | Empty |
| `defaultLogFile` | `default_log_file` | Log file path | `./logs/safeguard.log` |
| `defaultCacheEnabled` | `default_cache_enabled` | Enable response cache | `false` |
| `defaultCacheTTL` | `default_cache_ttl` | Cache TTL in seconds | `60` |
| `disableCliFlags` | `disable_cli_flags` | Hide all CLI flags from the user | `false` |
| `version` | `version` | Version string embedded in binary | `dev` |
| `buildTag` | `build_tag` | Custom build tag | Empty |

## Use Cases

### Client Distribution
Build customized binaries for each client with their Vault server and authentication settings pre-configured.

```json
{
  "default_vault_addr": "https://client-a-vault.company.com",
  "default_auth_method": "ldap",
  "build_tag": "client-a",
  "version": "1.0.0"
}
```

### Environment-Specific Builds
Create separate builds for development, staging, and production.

```json
{
  "default_vault_addr": "https://vault-prod.company.com",
  "default_policy_path": "/etc/safeguard/prod-policies",
  "build_tag": "production",
  "version": "1.0.0"
}
```

### Locked-Down Deployments
Deploy binaries with restricted settings that users can't easily change (though they can still override with flags).

```json
{
  "default_vault_addr": "https://secure-vault.company.com",
  "default_auth_method": "approle",
  "default_policy_path": "/etc/safeguard/strict-policies",
  "build_tag": "secure-deployment"
}
```

## Automation Example

### PowerShell
```powershell
$config = @{
    default_vault_addr = "https://vault.company.com"
    default_auth_method = "oidc"
    version = "1.0.0"
    build_tag = "customer-name"
    target_os = "windows"
    target_arch = "amd64"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://localhost:8080/api/build" `
    -Method Post `
    -ContentType "application/json" `
    -Body $config

Write-Host "Build complete: $($response.binary_path)"
Write-Host "Checksum: $($response.checksum)"

# Download the binary
Invoke-WebRequest -Uri "http://localhost:8080/api/download/$($response.binary_path.Split('\')[-1])" `
    -OutFile "custom-build.exe"
```

### cURL
```bash
curl -X POST http://localhost:8080/api/build \
  -H "Content-Type: application/json" \
  -d '{
    "default_vault_addr": "https://vault.company.com",
    "default_auth_method": "oidc",
    "version": "1.0.0",
    "build_tag": "customer-name",
    "target_os": "linux",
    "target_arch": "amd64"
  }'
```

## Security Considerations

1. **Access Control** - The build server should be run in a secure environment, not exposed publicly
2. **Source Protection** - Ensure the source directory is properly secured
3. **Output Directory** - Built binaries are stored in the output directory with predictable names
4. **No Secret Embedding** - Do NOT embed secrets, tokens, or passwords in the build. Only embed default configuration values.

## Troubleshooting

### Build Fails with CGO Error

This applies to **CLI builds only** (`binary_type: "cli"` or unset). Service builds (`binary_type: "service"`) use `CGO_ENABLED=0` and have no C dependency. Ensure you have the appropriate C compiler installed for CLI builds:

- **Windows**: Install MinGW-w64 or TDM-GCC
- **Linux**: Install `gcc`
- **macOS**: Install Xcode Command Line Tools

### Cross-Compilation Issues

Cross-compiling with CGO enabled requires appropriate cross-compilers:

```powershell
# Windows to Linux (example)
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"
$env:CC = "x86_64-linux-gnu-gcc"
```

## Development

To modify the build server:

1. Edit [builder.go](../../pkg/builder/builder.go) for build logic
2. Edit [main.go](main.go) for HTTP server functionality
3. Test locally before deploying

## License

Same as safeguard project.
