# safeguard

A cross-platform application that mounts HashiCorp Vault secrets as a virtual filesystem using FUSE.

## Features

- **Virtual filesystem** — Mount Vault as a drive (Windows `V:`) or directory (macOS/Linux)
- **Cross-platform** — Windows (WinFsp), macOS (macFUSE), Linux (native FUSE)
- **SSO authentication** — OIDC, LDAP, token, with automatic token renewal
- **Dynamic mount discovery** — Automatically lists all available secret engines
- **Path mapping** — Map virtual paths to real files on disk (symlink-like)
- **Process monitoring** — Track which processes access secrets (PID/UID/GID)
- **Policy-based access control** — Fine-grained REGO policies by process, user, path, time
- **Audit logging** — Log all access attempts for compliance
- **Read-only** — Secrets are exposed as read-only files for security

## Quick Start

See the [Quick Start Guide](docs/QUICKSTART.md) for a full walkthrough with a local Vault instance.

```powershell
# Windows
.\scripts\setup-vault.ps1
safeguard.exe -auth-method token -vault-token root -debug
```

```bash
# macOS / Linux
./scripts/setup-vault.sh
./safeguard -auth-method token -vault-token root -debug
```

## Prerequisites

- **Go 1.25+**
- **FUSE**:
  - **Windows**: [WinFsp](https://winfsp.dev/rel/)
  - **macOS**: [macFUSE](https://osxfuse.github.io/)
  - **Linux**: `fuse` / `libfuse-dev` (usually pre-installed)
- **HashiCorp Vault** — Running instance with an auth method configured

## Installation

```bash
cd safeguard
go build -o safeguard ./cmd/cli      # Linux / macOS
go build -o safeguard.exe ./cmd/cli      # Windows
```

Or use `make build`.

See [docs/PLATFORM_GUIDE.md](docs/PLATFORM_GUIDE.md) for platform-specific FUSE installation and cross-compilation.

See [docs/BAZEL_BUILD.md](docs/BAZEL_BUILD.md) for hermetic Bazel builds.

## Usage

```bash
# Default: OIDC SSO, opens browser automatically
safeguard -mount V: -vault-addr http://127.0.0.1:8200

# LDAP (prompts for credentials if not provided)
safeguard -mount V: -auth-method ldap

# Token
safeguard -mount V: -auth-method token -vault-token hvs.CAESIJ...

# Linux / macOS
./safeguard -mount /tmp/vault -auth-method oidc -debug
```

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-mount` | `V:` (Win), `/tmp/vault` (Mac), `/mnt/vault` (Linux) | Mount point |
| `-vault-addr` | `http://127.0.0.1:8200` | Vault server address |
| `-vault-provider` | `hashicorp` | Vault backend: `hashicorp`, `aws-secrets-manager`, `gcp-secret-manager`, `azure-keyvault` |
| `-auth-method` | `oidc` | Auth method: `oidc`, `ldap`, `token`, `aws`, `approle` |
| `-auth-role` | | Role name for OIDC or AppRole |
| `-auth-mount` | (method name) | Custom auth mount path |
| `-ldap-username` | | LDAP username (prompts if omitted) |
| `-ldap-password` | | LDAP password (prompts if omitted, masked) |
| `-vault-token` | | Vault token; also reads `VAULT_TOKEN` env var |
| `-debug` | `false` | Enable debug logging |
| `-monitor` | `false` | Log PID/UID for every operation |
| `-audit-log` | | Path to audit log file |
| `-access-control` | `false` | Enable process-based access control |
| `-policy-path` | | REGO policy file or directory |
| `-allowed-pids` | | Comma-separated allowed PIDs (legacy) |
| `-allowed-uids` | | Comma-separated allowed UIDs (legacy) |
| `-mapping-config` | | Path-mapping JSON config file |
| `-log-file` | `./logs/safeguard.log` | Path to application log file (empty to disable) |
| `-log-max-size` | `100` | Max log file size in MB before rotation |
| `-log-max-backups` | `5` | Max number of rotated log files to retain |
| `-log-max-age` | `30` | Max days to retain rotated log files |
| `-log-compress` | `true` | Compress rotated log files with gzip |
| `-cache` | `false` | Enable in-memory response cache for vault operations |
| `-cache-ttl` | `300` | Cache time-to-live in seconds |

### Authentication

Authentication is provider-specific. The adapter registry (`adapter.NewAuth()`) creates the right `AuthProvider` for the selected `-vault-provider`:

| Provider | Auth methods | How it works |
|----------|-------------|-------------|
| `hashicorp` (default) | OIDC, LDAP, token, AWS IAM, AppRole | Authenticates against Vault's `/v1/auth/` API and manages token renewal |
| `aws-secrets-manager` | SDK-managed (IAM) | Uses AWS SDK default credential chain — no explicit auth needed |
| `gcp-secret-manager` | SDK-managed (ADC) | Uses Application Default Credentials — no explicit auth needed |
| `azure-keyvault` | SDK-managed (DefaultAzureCredential) | Uses Azure identity SDK — no explicit auth needed |

**HashiCorp Vault**: **OIDC** (default) opens a browser for SSO. **LDAP** and **token** auto-prompt for missing credentials (passwords are masked). Tokens are automatically renewed in the background. See [docs/SSO_INTEGRATION.md](docs/SSO_INTEGRATION.md) for provider-specific setup (Okta, Azure AD, Google, Auth0).

**Cloud providers**: No `-auth-method` flag required. Configure credentials via your cloud provider's standard mechanisms (environment variables, instance profiles, service accounts, managed identity).

## How It Works

1. **Authenticate** using the provider's auth factory (HashiCorp: OIDC/LDAP/token; cloud: SDK credentials)
2. **Discover mounts** by querying Vault's `/v1/sys/mounts` API
3. **Mount a FUSE filesystem** — the root directory lists all secret engines
4. **Serve reads** — directory listings call `LIST`, file reads call `GET` on the Vault API
5. **Enforce policies** — optional REGO evaluation on every access
6. **Renew tokens** — a background goroutine renews at half the TTL

```
V:\                         ← root shows all secret engine mounts
├── secret\
│   ├── app1\
│   │   └── database        ← cat shows "username: admin\npassword: ..."
│   └── config
├── kv\
└── custom-secrets\
```

## Process Monitoring & Access Control

```bash
# Monitor which processes access secrets
safeguard -mount V: -auth-method token -vault-token root -monitor

# Audit log to file
safeguard -mount V: -auth-method token -vault-token root -audit-log vault-audit.log

# REGO policy-based access control
safeguard -mount V: -auth-method token -vault-token root \
  -access-control -policy-path policies/path-based-access.rego
```

See [docs/PROCESS_MONITORING.md](docs/PROCESS_MONITORING.md) and [policies/README.md](policies/README.md) for full documentation.

## Response Caching

Enable in-memory caching of vault responses to provide resilience against transient network issues. When enabled, successful responses are cached and served as fallback if subsequent requests fail within the TTL window.

```bash
# Enable caching with default 5-minute TTL
safeguard -mount V: -auth-method token -vault-token root -cache

# Custom TTL (60 seconds)
safeguard -mount V: -auth-method token -vault-token root -cache -cache-ttl 60
```

Caching applies to all vault backends (HashiCorp, AWS, GCP, Azure). The cache is per-path and covers `List`, `Read`, `PathExists`, and `ListMounts` operations. `Ping` and `RefreshMounts` always go to the server. Successful responses always update the cache, so stale data is only served when the backend is unreachable.

## Path Mapping

Map virtual paths to real files on disk alongside Vault secrets:

```json
{
  "mappings": [
    {
      "virtual_path": "/config/app.json",
      "real_path": "/etc/myapp/config.json",
      "read_only": true
    }
  ]
}
```

```bash
safeguard -mount V: -auth-method token -vault-token root \
  -mapping-config mapping-config.json
```

See [docs/PATH_MAPPING.md](docs/PATH_MAPPING.md) for details.

## Project Structure

```
safeguard/
├── cmd/
│   ├── cli/
│   │   ├── main.go                  # Entry point
│   │   ├── main_test.go
│   │   └── embedded_policies.go
│   └── builder/                     # Build server (Web UI + REST API)
├── pkg/
│   ├── auth/
│   │   ├── authenticator.go         # OIDC/LDAP/token auth + token renewal
│   │   ├── authenticator_test.go
│   │   ├── provider.go              # AuthProvider interface
│   │   └── noop.go                  # NoopAuthProvider (SDK-managed auth)
│   ├── filesystem/
│   │   ├── vaultfs.go               # FUSE filesystem implementation
│   │   ├── vaultfs_test.go
│   │   ├── policy.go                # REGO policy evaluation
│   │   ├── pathmapper.go            # Virtual-to-real path mapping
│   │   ├── procinfo.go              # Process info (PID/UID resolution)
│   │   ├── procinfo_unix.go
│   │   └── procinfo_windows.go
│   ├── vault/
│   │   ├── interface.go             # ClientInterface + MountInfo (shared types)
│   │   └── adapter/
│   │       ├── registry.go          # Provider + auth registry (Register / New / NewAuth)
│   │       ├── hashicorp.go         # HashiCorp Vault adapter + auth factory
│   │       ├── awssm.go             # AWS Secrets Manager adapter (stub)
│   │       ├── gcpsm.go             # GCP Secret Manager adapter (stub)
│   │       └── azurekv.go           # Azure Key Vault adapter (stub)
│   ├── logger/
│   │   ├── logger.go                # zerolog wrapper
│   │   └── splunk_writer.go
│   └── builder/
│       └── builder.go               # Custom binary builder
├── policies/                        # Example REGO policies
├── scripts/                         # Vault setup/teardown scripts
├── examples/                        # Usage examples
├── docs/                            # Documentation
│   ├── QUICKSTART.md
│   ├── PLATFORM_GUIDE.md
│   ├── SSO_INTEGRATION.md
│   ├── TESTING.md
│   ├── BAZEL_BUILD.md
│   ├── SETUP_SCRIPTS.md
│   ├── PATH_MAPPING.md
│   ├── MOUNT_DISCOVERY.md
│   ├── PROCESS_MONITORING.md
│   ├── POLICY_QUICKSTART.md
│   └── URL_POLICY_GUIDE.md
├── go.mod
├── Makefile
└── README.md
```

## Development

```bash
# Run all tests
go test ./...

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

See [docs/TESTING.md](docs/TESTING.md) for the full testing guide.

## Security Considerations

- Always use HTTPS for production Vault instances
- Never commit tokens to source control
- The mounted drive exposes secrets as readable files — restrict access
- Use time-limited, renewable tokens
- Enable REGO policies to restrict which processes can read secrets

## Documentation

| Guide | Description |
|-------|-------------|
| [Quick Start](docs/QUICKSTART.md) | Local Vault setup + first mount |
| [Platform Guide](docs/PLATFORM_GUIDE.md) | Windows/macOS/Linux install, Docker, systemd |
| [SSO Integration](docs/SSO_INTEGRATION.md) | OIDC/LDAP provider setup |
| [Policy Quick Start](docs/POLICY_QUICKSTART.md) | REGO policy patterns |
| [URL Policy Guide](docs/URL_POLICY_GUIDE.md) | Loading policies from HTTP URLs |
| [Path Mapping](docs/PATH_MAPPING.md) | Virtual-to-real file mapping |
| [Mount Discovery](docs/MOUNT_DISCOVERY.md) | Dynamic secret engine discovery |
| [Process Monitoring](docs/PROCESS_MONITORING.md) | PID/UID tracking and audit |
| [Setup Scripts](docs/SETUP_SCRIPTS.md) | Vault dev server scripts |
| [Testing](docs/TESTING.md) | Running and writing tests |
| [Bazel Build](docs/BAZEL_BUILD.md) | Hermetic builds with Bazel |
| [Build Server](cmd/builder/README.md) | Custom binary builder (Web UI + API) |
| [Policy Examples](policies/README.md) | Example REGO policies |

