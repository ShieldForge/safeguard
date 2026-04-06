# Setup Scripts Documentation

This document describes the Vault setup and management scripts included in the repository.

## Scripts Overview

| Script | Platform | Purpose |
|--------|----------|---------|
| `scripts/setup-vault.ps1` | Windows (PowerShell) | Launch Vault dev server and create sample secrets |
| `scripts/setup-vault.bat` | Windows (Command Prompt) | Launch Vault dev server and create sample secrets |
| `scripts/setup-vault.sh` | macOS/Linux (Bash) | Launch Vault dev server and create sample secrets |
| `scripts/stop-vault.ps1` | Windows (PowerShell) | Stop running Vault dev server |
| `scripts/stop-vault.bat` | Windows (Command Prompt) | Stop running Vault dev server |
| `scripts/stop-vault.sh` | macOS/Linux (Bash) | Stop running Vault dev server |

## setup-vault Scripts

### Features

- Checks if Vault is installed
- Starts Vault in dev mode on `http://127.0.0.1:8200`
- Sets root token to `root` for easy testing
- Creates 5 sample secrets with nested paths
- Displays helpful usage information
- Keeps running to maintain the Vault server

### Usage

**Windows:**
```powershell
.\scripts\setup-vault.ps1
```

**macOS/Linux:**
```bash
chmod +x scripts/setup-vault.sh
./scripts/setup-vault.sh
```

### Expected Output

```
=== HashiCorp Vault Local Setup ===

✓ Vault is installed: Vault v1.15.0

Starting Vault in dev mode...
Vault Address: http://127.0.0.1:8200
Vault Token: root

Waiting for Vault to start...
✓ Vault is running

=== Creating Sample Secrets ===

Creating secret: secret/database/production/credentials
Creating secret: secret/api/external/services
Creating secret: secret/app/config/production
Creating secret: secret/team/devops/ssh
Creating secret: secret/customers/acme-corp/credentials

=== Vault Setup Complete ===

Vault is running at: http://127.0.0.1:8200
Root Token: root

Sample secrets created:
  • secret/database/production/credentials
  • secret/api/external/services
  • secret/app/config/production
  • secret/team/devops/ssh
  • secret/customers/acme-corp/credentials

To view a secret, run:
  vault kv get secret/database/production/credentials

To mount the filesystem, run:
  .\safeguard.exe -auth-method token -vault-token root -debug

To stop Vault, run:
  Get-Process -Name vault | Stop-Process

Environment variables:
  $env:VAULT_ADDR = "http://127.0.0.1:8200"
  $env:VAULT_TOKEN = "root"

Press Ctrl+C to stop Vault and exit
```

### What Gets Created

The scripts create the following structure in Vault:

```
secret/
├── database/
│   └── production/
│       └── credentials (username, password, host, port, database)
├── api/
│   └── external/
│       └── services (stripe_key, sendgrid_key, aws keys)
├── app/
│   └── config/
│       └── production (environment, debug, log_level, keys)
├── team/
│   └── devops/
│       └── ssh (private_key, public_key, passphrase)
└── customers/
    └── acme-corp/
        └── credentials (account_id, api_token, webhook_url, email)
```

## stop-vault Scripts

### Features

- Gracefully stops the Vault dev server
- Cleans up process IDs and temporary files
- Provides feedback on success/failure

### Usage

**Windows:**
```powershell
.\scripts\stop-vault.ps1
```

**macOS/Linux:**
```bash
./scripts/stop-vault.sh
```

### Expected Output

```
Stopping Vault dev server...
✓ Vault server stopped
```

## Environment Variables

The scripts set/use these environment variables:

- `VAULT_ADDR=http://127.0.0.1:8200` - Vault server address
- `VAULT_TOKEN=root` - Root token for authentication

You can use these in your terminal to interact with Vault CLI:

**Windows:**
```powershell
$env:VAULT_ADDR = "http://127.0.0.1:8200"
$env:VAULT_TOKEN = "root"
vault kv get secret/database/production/credentials
```

**macOS/Linux:**
```bash
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="root"
vault kv get secret/database/production/credentials
```

## Troubleshooting

### Vault not found

**Error:**
```
✗ Vault is not installed
```

**Solution:**
- **Windows**: `choco install vault`
- **macOS**: `brew install vault`
- **Linux**: Download from https://www.vaultproject.io/downloads

### Port already in use

If port 8200 is already in use, you'll need to:

1. Stop any existing Vault process: `.\scripts\stop-vault.ps1` or `./scripts/stop-vault.sh`
2. Or kill manually:
   - Windows: `Get-Process -Name vault | Stop-Process -Force`
   - macOS/Linux: `pkill -f "vault server"`

### Permission denied (Linux/macOS)

**Error:**
```
bash: ./scripts/setup-vault.sh: Permission denied
```

**Solution:**
```bash
chmod +x scripts/setup-vault.sh scripts/stop-vault.sh
```

## Integration with safeguard

After running the setup script, you can immediately mount the filesystem:

**Windows:**
```powershell
# Terminal 1: Run setup (keeps running)
.\scripts\setup-vault.ps1

# Terminal 2: Mount filesystem
.\safeguard.exe -auth-method token -vault-token root -debug

# Terminal 3: Access secrets
cd V:
type secret\database\production\credentials.json
```

**macOS/Linux:**
```bash
# Terminal 1: Run setup (runs in background)
./scripts/setup-vault.sh

# Terminal 2: Mount filesystem
./safeguard -auth-method token -vault-token root -debug

# Terminal 3: Access secrets
cd /tmp/vault
cat secret/database/production/credentials.json
```

## Advanced Usage

### Custom Secrets

You can add your own secrets after running the setup script:

```bash
# Set environment variables
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="root"

# Add a new secret
vault kv put secret/my/custom/path \
    key1="value1" \
    key2="value2"

# Verify
vault kv get secret/my/custom/path
```

### Different Port

To run Vault on a different port, modify the scripts or start Vault manually:

```bash
vault server -dev \
    -dev-root-token-id=root \
    -dev-listen-address=127.0.0.1:9200
```

Then use:
```bash
./safeguard -vault-addr http://127.0.0.1:9200 -auth-method token -vault-token root
```

## Security Notes

⚠️ **Important**: These scripts are for **development and testing only**!

- Dev mode stores data in memory (not persistent)
- Root token is hardcoded as `root` (insecure)
- No TLS/encryption (HTTP only)
- No access controls or policies
- Automatically unsealed (no seal keys needed)

For production use:
- Use proper Vault deployment (not dev mode)
- Enable TLS
- Use proper authentication methods (OIDC, LDAP, etc.)
- Configure access policies
- Enable audit logging
- Use proper seal mechanisms (auto-unseal, Shamir, etc.)

See [SSO_INTEGRATION.md](SSO_INTEGRATION.md) for production authentication setup.
