#!/usr/bin/env pwsh
# Setup script for local Vault instance with default secrets
# This script launches Vault in dev mode and populates it with sample secrets

Write-Host "=== HashiCorp Vault Local Setup ===" -ForegroundColor Cyan
Write-Host ""

# Check if Vault is installed
try {
    $vaultVersion = vault version 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Vault not found"
    }
    Write-Host "✓ Vault is installed: $vaultVersion" -ForegroundColor Green
}
catch {
    Write-Host "✗ Vault is not installed" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please install Vault from: https://www.vaultproject.io/downloads" -ForegroundColor Yellow
    Write-Host "Or use: choco install vault" -ForegroundColor Yellow
    exit 1
}

Write-Host ""
Write-Host "Starting Vault in dev mode..." -ForegroundColor Cyan
Write-Host ""

# Kill any existing Vault dev servers
Get-Process -Name vault -ErrorAction SilentlyContinue | Stop-Process -Force

# Start Vault in dev mode in the background
$env:VAULT_ADDR = "http://127.0.0.1:8200"
$env:VAULT_TOKEN = "root"

Write-Host "Vault Address: $env:VAULT_ADDR" -ForegroundColor Yellow
Write-Host "Vault Token: $env:VAULT_TOKEN" -ForegroundColor Yellow
Write-Host ""

# Start Vault server in background
$vaultJob = Start-Job -ScriptBlock {
    $env:VAULT_ADDR = "http://127.0.0.1:8200"
    vault server -dev -dev-root-token-id=root -dev-listen-address=127.0.0.1:8200
}

# Wait for Vault to start
Write-Host "Waiting for Vault to start..." -ForegroundColor Cyan
Start-Sleep -Seconds 3

# Verify Vault is running
try {
    $status = vault status 2>&1
    if ($LASTEXITCODE -ne 0 -and $status -notmatch "Sealed.*false") {
        throw "Vault failed to start"
    }
    Write-Host "✓ Vault is running" -ForegroundColor Green
}
catch {
    Write-Host "✗ Failed to start Vault" -ForegroundColor Red
    Stop-Job -Job $vaultJob
    Remove-Job -Job $vaultJob
    exit 1
}

Write-Host ""
Write-Host "=== Creating Sample Secrets ===" -ForegroundColor Cyan
Write-Host ""

# Enable KV v2 secrets engine (dev mode already has it at secret/)
# Create nested secrets for testing

# Database credentials
Write-Host "Creating secret: secret/database/production/credentials" -ForegroundColor Yellow
vault kv put secret/database/production/credentials `
    username="admin" `
    password="SuperSecret123!" `
    host="db.example.com" `
    port="5432" `
    database="myapp_prod"

# API keys
Write-Host "Creating secret: secret/api/external/services" -ForegroundColor Yellow
vault kv put secret/api/external/services `
    stripe_key="sk_test_51234567890" `
    sendgrid_key="SG.1234567890abcdef" `
    aws_access_key="AKIAIOSFODNN7EXAMPLE" `
    aws_secret_key="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

# Application configuration
Write-Host "Creating secret: secret/app/config/production" -ForegroundColor Yellow
vault kv put secret/app/config/production `
    environment="production" `
    debug="false" `
    log_level="info" `
    secret_key="a1b2c3d4e5f6g7h8i9j0" `
    encryption_key="0j9i8h7g6f5e4d3c2b1a"

# Team credentials
Write-Host "Creating secret: secret/team/devops/ssh" -ForegroundColor Yellow
vault kv put secret/team/devops/ssh `
    private_key="-----BEGIN RSA PRIVATE KEY-----\n(placeholder)\n-----END RSA PRIVATE KEY-----" `
    public_key="ssh-rsa AAAAB3... devops@example.com" `
    passphrase="MySSHPassphrase123"

# Customer data
Write-Host "Creating secret: secret/customers/acme-corp/credentials" -ForegroundColor Yellow
vault kv put secret/customers/acme-corp/credentials `
    account_id="ACME-12345" `
    api_token="acme_token_abcdef123456" `
    webhook_url="https://acme.example.com/webhooks" `
    contact_email="admin@acme-corp.com"

Write-Host ""
Write-Host "=== Vault Setup Complete ===" -ForegroundColor Green
Write-Host ""
Write-Host "Vault is running at: $env:VAULT_ADDR" -ForegroundColor Cyan
Write-Host "Root Token: $env:VAULT_TOKEN" -ForegroundColor Cyan
Write-Host ""
Write-Host "Sample secrets created:" -ForegroundColor Yellow
Write-Host "  • secret/database/production/credentials" -ForegroundColor White
Write-Host "  • secret/api/external/services" -ForegroundColor White
Write-Host "  • secret/app/config/production" -ForegroundColor White
Write-Host "  • secret/team/devops/ssh" -ForegroundColor White
Write-Host "  • secret/customers/acme-corp/credentials" -ForegroundColor White
Write-Host ""
Write-Host "To view a secret, run:" -ForegroundColor Yellow
Write-Host "  vault kv get secret/database/production/credentials" -ForegroundColor White
Write-Host ""
Write-Host "To mount the filesystem, run:" -ForegroundColor Yellow
Write-Host '  .\safeguard.exe -auth-method token -vault-token root -debug' -ForegroundColor White
Write-Host ""
Write-Host "To stop Vault, run:" -ForegroundColor Yellow
Write-Host "  Get-Process -Name vault | Stop-Process" -ForegroundColor White
Write-Host ""
Write-Host "Environment variables:" -ForegroundColor Yellow
Write-Host "  `$env:VAULT_ADDR = `"http://127.0.0.1:8200`"" -ForegroundColor White
Write-Host "  `$env:VAULT_TOKEN = `"root`"" -ForegroundColor White
Write-Host ""

# Keep the script running to show status
Write-Host "Press Ctrl+C to stop Vault and exit" -ForegroundColor Cyan
Write-Host ""

# Monitor the Vault job
try {
    while ($true) {
        $jobState = (Get-Job -Id $vaultJob.Id).State
        if ($jobState -ne "Running") {
            Write-Host "Vault server stopped unexpectedly" -ForegroundColor Red
            break
        }
        Start-Sleep -Seconds 5
    }
}
finally {
    # Cleanup
    Write-Host ""
    Write-Host "Stopping Vault server..." -ForegroundColor Yellow
    Stop-Job -Job $vaultJob -ErrorAction SilentlyContinue
    Remove-Job -Job $vaultJob -ErrorAction SilentlyContinue
    Get-Process -Name vault -ErrorAction SilentlyContinue | Stop-Process -Force
    Write-Host "Vault stopped" -ForegroundColor Green
}
