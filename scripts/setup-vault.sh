#!/bin/bash
# Setup script for local Vault instance with default secrets
# This script launches Vault in dev mode and populates it with sample secrets

set -e

echo "=== HashiCorp Vault Local Setup ==="
echo ""

# Check if Vault is installed
if ! command -v vault &> /dev/null; then
    echo "✗ Vault is not installed"
    echo ""
    echo "Please install Vault from: https://www.vaultproject.io/downloads"
    echo "Or use your package manager:"
    echo "  - macOS: brew install vault"
    echo "  - Linux: See https://www.vaultproject.io/downloads"
    exit 1
fi

echo "✓ Vault is installed: $(vault version)"

# Kill any existing Vault dev servers
pkill -f "vault server -dev" || true

# Set environment variables
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="root"

echo ""
echo "Starting Vault in dev mode..."
echo "Vault Address: $VAULT_ADDR"
echo "Vault Token: $VAULT_TOKEN"
echo ""

# Start Vault in dev mode in the background
vault server -dev -dev-root-token-id=root -dev-listen-address=127.0.0.1:8200 > /tmp/vault-dev.log 2>&1 &
VAULT_PID=$!

# Wait for Vault to start
echo "Waiting for Vault to start..."
sleep 3

# Verify Vault is running
if ! vault status &> /dev/null; then
    echo "✗ Failed to start Vault"
    echo "Check logs at: /tmp/vault-dev.log"
    exit 1
fi

echo "✓ Vault is running (PID: $VAULT_PID)"

echo ""
echo "=== Creating Sample Secrets ==="
echo ""

# Database credentials
echo "Creating secret: secret/database/production/credentials"
vault kv put secret/database/production/credentials \
    username="admin" \
    password="SuperSecret123!" \
    host="db.example.com" \
    port="5432" \
    database="myapp_prod"

# API keys
echo "Creating secret: secret/api/external/services"
vault kv put secret/api/external/services \
    stripe_key="sk_test_51234567890" \
    sendgrid_key="SG.1234567890abcdef" \
    aws_access_key="AKIAIOSFODNN7EXAMPLE" \
    aws_secret_key="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

# Application configuration
echo "Creating secret: secret/app/config/production"
vault kv put secret/app/config/production \
    environment="production" \
    debug="false" \
    log_level="info" \
    secret_key="a1b2c3d4e5f6g7h8i9j0" \
    encryption_key="0j9i8h7g6f5e4d3c2b1a"

# Team credentials
echo "Creating secret: secret/team/devops/ssh"
vault kv put secret/team/devops/ssh \
    private_key="-----BEGIN RSA PRIVATE KEY-----\n(placeholder)\n-----END RSA PRIVATE KEY-----" \
    public_key="ssh-rsa AAAAB3... devops@example.com" \
    passphrase="MySSHPassphrase123"

# Customer data
echo "Creating secret: secret/customers/acme-corp/credentials"
vault kv put secret/customers/acme-corp/credentials \
    account_id="ACME-12345" \
    api_token="acme_token_abcdef123456" \
    webhook_url="https://acme.example.com/webhooks" \
    contact_email="admin@acme-corp.com"

echo ""
echo "=== Vault Setup Complete ==="
echo ""
echo "Vault is running at: $VAULT_ADDR"
echo "Root Token: $VAULT_TOKEN"
echo "Process ID: $VAULT_PID"
echo ""
echo "Sample secrets created:"
echo "  • secret/database/production/credentials"
echo "  • secret/api/external/services"
echo "  • secret/app/config/production"
echo "  • secret/team/devops/ssh"
echo "  • secret/customers/acme-corp/credentials"
echo ""
echo "To view a secret, run:"
echo "  vault kv get secret/database/production/credentials"
echo ""
echo "To mount the filesystem, run:"
echo "  ./safeguard -auth-method token -vault-token root -debug"
echo ""
echo "To stop Vault, run:"
echo "  kill $VAULT_PID"
echo "  # or: pkill -f 'vault server -dev'"
echo ""
echo "Environment variables:"
echo "  export VAULT_ADDR=\"http://127.0.0.1:8200\""
echo "  export VAULT_TOKEN=\"root\""
echo ""
echo "Vault server is running in the background."
echo "Logs are available at: /tmp/vault-dev.log"
echo ""

# Save PID to file for easy cleanup
echo $VAULT_PID > /tmp/vault-dev.pid
echo "PID saved to: /tmp/vault-dev.pid"