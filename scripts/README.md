# Scripts

This directory contains helper scripts for setting up and managing a local Vault instance for development and testing.

## Available Scripts

### Setup Scripts

- **`setup-vault.ps1`** - Windows PowerShell script to launch Vault dev server with sample secrets
- **`setup-vault.sh`** - macOS/Linux bash script to launch Vault dev server with sample secrets

### Cleanup Scripts

- **`stop-vault.ps1`** - Windows PowerShell script to stop the Vault dev server
- **`stop-vault.sh`** - macOS/Linux bash script to stop the Vault dev server

## Quick Usage

### Windows

```powershell
# Start Vault with sample data
.\scripts\setup-vault.ps1

# Stop Vault
.\scripts\stop-vault.ps1
```

### macOS / Linux

```bash
# Make scripts executable (first time only)
chmod +x scripts/*.sh

# Start Vault with sample data
./scripts/setup-vault.sh

# Stop Vault
./scripts/stop-vault.sh
```

## Documentation

For detailed information about these scripts, see:

- [QUICKSTART.md](../docs/QUICKSTART.md) - Quick start guide with examples
- [SETUP_SCRIPTS.md](../docs/SETUP_SCRIPTS.md) - Comprehensive script documentation

## What the Setup Scripts Do

1. Check if Vault is installed
2. Start Vault in dev mode on `http://127.0.0.1:8200`
3. Create 5 sample secrets with nested paths:
   - `secret/database/production/credentials`
   - `secret/api/external/services`
   - `secret/app/config/production`
   - `secret/team/devops/ssh`
   - `secret/customers/acme-corp/credentials`

## Environment

The scripts use these default values:

- **Vault Address**: `http://127.0.0.1:8200`
- **Root Token**: `root`

⚠️ **Note**: These scripts are for development/testing only. Do not use in production!
