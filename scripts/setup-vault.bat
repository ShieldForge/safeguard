@echo off
REM Setup script for local Vault instance with default secrets
REM This script launches Vault in dev mode and populates it with sample secrets

setlocal enabledelayedexpansion

echo.
echo === HashiCorp Vault Local Setup ===
echo.

REM Check if Vault is installed
vault version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Vault is not installed
    echo.
    echo Please install Vault from: https://www.vaultproject.io/downloads
    echo Or use: choco install vault
    exit /b 1
)

echo [OK] Vault is installed
echo.

REM Kill any existing Vault dev servers
taskkill /F /IM vault.exe >nul 2>&1

REM Set environment variables
set VAULT_ADDR=http://127.0.0.1:8200
set VAULT_TOKEN=root

echo Vault Address: %VAULT_ADDR%
echo Vault Token: %VAULT_TOKEN%
echo.
echo Starting Vault in dev mode...
echo.

REM Start Vault server in background
start /B vault server -dev -dev-root-token-id=root -dev-listen-address=127.0.0.1:8200 >nul 2>&1

REM Wait for Vault to start
echo Waiting for Vault to start...
timeout /t 3 /nobreak >nul

REM Verify Vault is running
vault status >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Failed to start Vault
    taskkill /F /IM vault.exe >nul 2>&1
    exit /b 1
)

echo [OK] Vault is running
echo.
echo === Creating Sample Secrets ===
echo.

REM Create nested secrets for testing

REM Database credentials
echo Creating secret: secret/database/production/credentials
vault kv put secret/database/production/credentials username="admin" password="SuperSecret123!" host="db.example.com" port="5432" database="myapp_prod"

REM API keys
echo Creating secret: secret/api/external/services
vault kv put secret/api/external/services stripe_key="sk_test_51234567890" sendgrid_key="SG.1234567890abcdef" aws_access_key="AKIAIOSFODNN7EXAMPLE" aws_secret_key="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY" npm_token="npm_EWRdewfdcc43345FWDEWDF"

REM Application configuration
echo Creating secret: secret/app/config/production
vault kv put secret/app/config/production environment="production" debug="false" log_level="info" secret_key="a1b2c3d4e5f6g7h8i9j0" encryption_key="0j9i8h7g6f5e4d3c2b1a"

REM Team credentials
echo Creating secret: secret/team/devops/ssh
vault kv put secret/team/devops/ssh private_key="-----BEGIN RSA PRIVATE KEY-----\n(placeholder)\n-----END RSA PRIVATE KEY-----" public_key="ssh-rsa AAAAB3... devops@example.com" passphrase="MySSHPassphrase123"

REM Customer data
echo Creating secret: secret/customers/acme-corp/credentials
vault kv put secret/customers/acme-corp/credentials account_id="ACME-12345" api_token="acme_token_abcdef123456" webhook_url="https://acme.example.com/webhooks" contact_email="admin@acme-corp.com"

echo.
echo === Vault Setup Complete ===
echo.
echo Vault is running at: %VAULT_ADDR%
echo Root Token: %VAULT_TOKEN%
echo.
echo Sample secrets created:
echo   - secret/database/production/credentials
echo   - secret/api/external/services
echo   - secret/app/config/production
echo   - secret/team/devops/ssh
echo   - secret/customers/acme-corp/credentials
echo.
echo To view a secret, run:
echo   vault kv get secret/database/production/credentials
echo.
echo To mount the filesystem, run:
echo   safeguard.exe -auth-method token -vault-token root -debug
echo.
echo To stop Vault, run:
echo   scripts\stop-vault.bat
echo   or: taskkill /F /IM vault.exe
echo.
echo Environment variables have been set for this session:
echo   set VAULT_ADDR=http://127.0.0.1:8200
echo   set VAULT_TOKEN=root
echo.
echo Vault is running in the background. Press Ctrl+C to exit this script.
echo The Vault server will continue running.
echo.

pause
