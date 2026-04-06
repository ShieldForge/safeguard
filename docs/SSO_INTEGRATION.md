# SSO Integration Guide

## Overview

safeguard now supports SSO authentication for obtaining Vault tokens automatically. This eliminates the need to manually provide tokens and integrates with your organization's identity provider.

> **Note:** The OIDC, LDAP, and token methods described here apply to the **HashiCorp Vault** provider (`-vault-provider hashicorp`, the default). Cloud providers (AWS Secrets Manager, GCP Secret Manager, Azure Key Vault) use SDK-managed authentication and do not need explicit auth configuration — see [Cloud Provider Authentication](#cloud-provider-authentication) at the bottom of this guide.

## Supported Authentication Methods

### 1. OIDC (OpenID Connect) - Recommended for SSO

**Best for**: Organizations using Okta, Auth0, Azure AD, Google, or any OIDC-compliant provider

**How it works**:
1. Application opens browser to your SSO provider
2. User authenticates with their corporate credentials
3. SSO provider redirects back with authorization code
4. Application exchanges code for Vault token
5. Drive mounts with authenticated token

**Usage**:
```powershell
safeguard.exe -mount V: -vault-addr https://vault.company.com -auth-method oidc -auth-role employee
```

**Vault Setup**:
```bash
# Enable OIDC
vault auth enable oidc

# Configure with your provider (example: Okta)
vault write auth/oidc/config \
    oidc_discovery_url="https://your-org.okta.com" \
    oidc_client_id="0oa1234567890" \
    oidc_client_secret="super-secret-key" \
    default_role="default"

# Create role with policies
vault write auth/oidc/role/employee \
    bound_audiences="0oa1234567890" \
    allowed_redirect_uris="http://localhost:8250/oidc/callback" \
    user_claim="email" \
    groups_claim="groups" \
    policies="reader,writer"
```

### 2. LDAP

**Best for**: Organizations with Active Directory or LDAP directories

**Usage**:
```powershell
safeguard.exe -mount V: -vault-addr https://vault.company.com -auth-method ldap -ldap-username john.doe -ldap-password P@ssw0rd
```

**Vault Setup**:
```bash
# Enable LDAP
vault auth enable ldap

# Configure LDAP
vault write auth/ldap/config \
    url="ldaps://ldap.company.com" \
    userdn="ou=users,dc=company,dc=com" \
    groupdn="ou=groups,dc=company,dc=com" \
    binddn="cn=vault,ou=service,dc=company,dc=com" \
    bindpass="service-password"

# Map groups to policies
vault write auth/ldap/groups/engineers policies=reader,writer
```

### 3. Token (Legacy/Development)

**Best for**: Development, testing, or CI/CD pipelines

**Usage**:
```powershell
# Direct token
safeguard.exe -mount V: -vault-addr http://localhost:8200 -auth-method token -vault-token hvs.xxxxx

# Or environment variable
$env:VAULT_TOKEN = "hvs.xxxxx"
safeguard.exe -mount V: -vault-addr http://localhost:8200 -auth-method token
```

### 4. AWS (Coming Soon)

For EC2 instances or Lambda functions using IAM authentication.

### 5. AppRole (Coming Soon)

For service-to-service authentication.

## Authentication Flow

### OIDC Flow
```
1. User runs: safeguard.exe -auth-method oidc -auth-role employee
2. App requests auth URL from Vault
3. App opens browser to SSO provider
4. User authenticates with SSO (username/password/MFA)
5. SSO redirects to http://localhost:8250/oidc/callback?code=xxx&state=yyy
6. App receives callback and exchanges code for Vault token
7. App uses token to mount virtual drive
8. User can now access secrets via V: drive
```

### LDAP Flow
```
1. User runs: safeguard.exe -auth-method ldap -ldap-username john -ldap-password pass
2. App sends credentials to Vault's LDAP auth endpoint
3. Vault validates credentials against LDAP/AD
4. Vault returns token with appropriate policies
5. App uses token to mount virtual drive
```

## Configuration Examples

### Enterprise Setup with Okta

```powershell
# Production configuration
safeguard.exe `
  -mount V: `
  -vault-addr https://vault.company.com `
  -auth-method oidc `
  -auth-role employee `
  -auth-mount oidc
```

### Active Directory Integration

```powershell
# AD authentication
safeguard.exe `
  -mount V: `
  -vault-addr https://vault.company.com `
  -auth-method ldap `
  -ldap-username john.doe@company.com `
  -ldap-password "P@ssw0rd"
```

### Development with Token

```powershell
# Dev mode with root token
$env:VAULT_TOKEN = "hvs.dev-root-token"
safeguard.exe -mount V: -auth-method token -debug
```

## Troubleshooting

### Browser doesn't open
**Problem**: OIDC auth but browser doesn't launch automatically

**Solution**: 
- Copy the URL from console and open manually
- Check firewall isn't blocking localhost:8250
- Ensure browser is installed and accessible

### Authentication timeout
**Problem**: OIDC callback times out after 5 minutes

**Solution**:
- Complete authentication faster
- Check if callback URL is reachable
- Verify Vault's allowed_redirect_uris includes http://localhost:8250/oidc/callback

### LDAP authentication fails
**Problem**: "Authentication failed" with LDAP

**Solution**:
- Verify LDAP credentials are correct
- Check Vault's LDAP configuration
- Ensure user is in allowed groups
- Test with: `vault login -method=ldap username=john.doe`

### Invalid role
**Problem**: "role not found" error

**Solution**:
- Verify role exists: `vault read auth/oidc/role/employee`
- Check spelling of role name
- Ensure role has necessary policies

## Security Best Practices

1. **Use HTTPS**: Always use HTTPS for production Vault instances
2. **Token TTL**: Configure short-lived tokens with renewal
3. **MFA**: Enable MFA in your OIDC provider
4. **Least Privilege**: Grant minimal necessary policies to roles
5. **Audit**: Enable Vault audit logging for authentication events
6. **Network Security**: Restrict Vault access to trusted networks
7. **Callback URL**: Use unique port per application if running multiple

## Advanced Configuration

### Custom Auth Mount Path

If OIDC is mounted at a custom path:
```powershell
safeguard.exe -auth-method oidc -auth-mount oidc-custom -auth-role employee
```

### Multiple OIDC Providers

```bash
# Azure AD
vault write auth/oidc-azure/config ...
safeguard.exe -auth-method oidc -auth-mount oidc-azure -auth-role employee

# Google
vault write auth/oidc-google/config ...
safeguard.exe -auth-method oidc -auth-mount oidc-google -auth-role employee
```

### Debug Mode

Enable verbose logging:
```powershell
safeguard.exe -auth-method oidc -auth-role employee -debug
```

This shows:
- Authentication flow steps
- Vault API requests
- File system operations
- Error details

## Integration with Popular Providers

### Okta
```bash
vault write auth/oidc/config \
    oidc_discovery_url="https://your-org.okta.com" \
    oidc_client_id="YOUR_CLIENT_ID" \
    oidc_client_secret="YOUR_SECRET"
```

### Azure AD
```bash
vault write auth/oidc/config \
    oidc_discovery_url="https://login.microsoftonline.com/YOUR_TENANT_ID/v2.0" \
    oidc_client_id="YOUR_CLIENT_ID" \
    oidc_client_secret="YOUR_SECRET"
```

### Google Workspace
```bash
vault write auth/oidc/config \
    oidc_discovery_url="https://accounts.google.com" \
    oidc_client_id="YOUR_CLIENT_ID" \
    oidc_client_secret="YOUR_SECRET"
```

### Auth0
```bash
vault write auth/oidc/config \
    oidc_discovery_url="https://YOUR_DOMAIN.auth0.com/" \
    oidc_client_id="YOUR_CLIENT_ID" \
    oidc_client_secret="YOUR_SECRET"
```

## Cloud Provider Authentication

When using a cloud secret backend (`-vault-provider aws-secrets-manager`, `gcp-secret-manager`, or `azure-keyvault`), the `-auth-method` flag is ignored. Authentication is handled by the cloud SDK's default credential chain.

### AWS Secrets Manager

```bash
# Uses AWS default credential chain (env vars, ~/.aws/credentials, instance profile, ECS task role)
safeguard -vault-provider aws-secrets-manager -mount V:

# Explicit region
safeguard -vault-provider aws-secrets-manager -mount V: \
  -vault-addr "https://secretsmanager.eu-west-1.amazonaws.com"
```

Configure credentials via any standard AWS mechanism:
- `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` environment variables
- `~/.aws/credentials` profile
- EC2 instance profile or ECS task role
- IAM Roles for Service Accounts (IRSA) on EKS

### GCP Secret Manager

```bash
# Uses Application Default Credentials (ADC)
safeguard -vault-provider gcp-secret-manager -mount /mnt/secrets
```

Configure credentials via:
- `gcloud auth application-default login` (local development)
- `GOOGLE_APPLICATION_CREDENTIALS` environment variable pointing to a service account key
- Workload Identity on GKE
- Attached service account on Compute Engine / Cloud Run

### Azure Key Vault

```bash
# Uses DefaultAzureCredential
safeguard -vault-provider azure-keyvault -mount V:
```

Configure credentials via:
- `az login` (local development)
- `AZURE_CLIENT_ID` / `AZURE_TENANT_ID` / `AZURE_CLIENT_SECRET` environment variables
- Managed Identity on Azure VMs / App Service / AKS
