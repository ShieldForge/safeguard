# Quick Start Guide

This guide shows you how to quickly get started with safeguard using a local Vault instance.

## Prerequisites

You need to install HashiCorp Vault:

- **Windows**: `choco install vault` or download from https://www.vaultproject.io/downloads
- **macOS**: `brew install vault`
- **Linux**: See https://www.vaultproject.io/downloads

## Setup Scripts

The repository includes convenient setup scripts to launch a local Vault dev server with sample secrets.

### Windows (PowerShell)

```powershell
# Start Vault dev server with sample secrets
.\scripts\setup-vault.ps1
```

This will:
1. Check if Vault is installed
2. Start Vault in dev mode on http://127.0.0.1:8200
3. Create sample nested secrets for testing
4. Keep running until you press Ctrl+C

### macOS / Linux (Bash)

```bash
# Make script executable
chmod +x scripts/setup-vault.sh

# Start Vault dev server with sample secrets
./scripts/setup-vault.sh
```

## Sample Secrets

The setup script creates the following nested paths with secrets:

### 1. Database Credentials
**Path**: `secret/database/production/credentials`

```json
{
  "username": "admin",
  "password": "SuperSecret123!",
  "host": "db.example.com",
  "port": "5432",
  "database": "myapp_prod"
}
```

### 2. External API Services
**Path**: `secret/api/external/services`

```json
{
  "stripe_key": "sk_test_51234567890",
  "sendgrid_key": "SG.1234567890abcdef",
  "aws_access_key": "AKIAIOSFODNN7EXAMPLE",
  "aws_secret_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
}
```

### 3. Application Configuration
**Path**: `secret/app/config/production`

```json
{
  "environment": "production",
  "debug": "false",
  "log_level": "info",
  "secret_key": "a1b2c3d4e5f6g7h8i9j0",
  "encryption_key": "0j9i8h7g6f5e4d3c2b1a"
}
```

### 4. Team SSH Credentials
**Path**: `secret/team/devops/ssh`

```json
{
  "private_key": "-----BEGIN RSA PRIVATE KEY-----\n(placeholder)\n-----END RSA PRIVATE KEY-----",
  "public_key": "ssh-rsa AAAAB3... devops@example.com",
  "passphrase": "MySSHPassphrase123"
}
```

### 5. Customer Credentials
**Path**: `secret/customers/acme-corp/credentials`

```json
{
  "account_id": "ACME-12345",
  "api_token": "acme_token_abcdef123456",
  "webhook_url": "https://acme.example.com/webhooks",
  "contact_email": "admin@acme-corp.com"
}
```

## Using the Mounted Filesystem

### Windows

```powershell
# In a new terminal, mount the filesystem
.\safeguard.exe -auth-method token -vault-token root -debug

# In another terminal, browse the secrets
cd V:
dir

# View directory structure
dir secret\database
dir secret\api\external

# Read a secret
type secret\database\production\credentials.json

# Pretty print with PowerShell
Get-Content secret\database\production\credentials.json | ConvertFrom-Json | ConvertTo-Json
```

### macOS / Linux

```bash
# In a new terminal, mount the filesystem
./safeguard -auth-method token -vault-token root -debug

# In another terminal, browse the secrets
cd /tmp/vault
ls -la

# View directory structure
ls -la secret/database
ls -la secret/api/external

# Read a secret
cat secret/database/production/credentials.json

# Pretty print with jq
cat secret/database/production/credentials.json | jq .
```

## Stopping Vault

### Windows

```powershell
.\scripts\stop-vault.ps1
```

or press `Ctrl+C` in the terminal running `scripts\setup-vault.ps1`

### macOS / Linux

```bash
./scripts/stop-vault.sh
```

## Environment Variables

The setup script uses these default values:

```bash
VAULT_ADDR=http://127.0.0.1:8200
VAULT_TOKEN=root
```

You can use these to interact with Vault using the CLI:

```bash
# Windows PowerShell
$env:VAULT_ADDR = "http://127.0.0.1:8200"
$env:VAULT_TOKEN = "root"
vault kv list secret/

# macOS / Linux
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="root"
vault kv list secret/
```

## Next Steps

- Try the [Process Monitoring](PROCESS_MONITORING.md) features
- Set up [Policy-based Access Control](POLICY_QUICKSTART.md)
- Configure [SSO Integration](SSO_INTEGRATION.md) for production use
- Review [Platform-specific features](PLATFORM_GUIDE.md)
