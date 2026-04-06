# URL-Based Policy Support

safeguard supports loading policies from HTTP/HTTPS URLs, with automatic caching and optional build-time embedding. The builder also supports embedding policies from local files, directories, or zip archives.

## Features

### 1. Runtime URL Policy Loading

Policies can be loaded from URLs at runtime and are cached in memory:

```bash
safeguard -mount V: -policy-path https://example.com/policies/policy.rego
```

**How it works:**
- The policy is downloaded from the URL when the application starts
- The policy content is cached in memory
- No temporary files are created
- The `Reload()` method will re-download the policy if needed

**Use cases:**
- Centralized policy management
- Dynamic policy updates
- Remote policy distribution
- Testing policies without local files

### 2. Build-Time Policy Embedding from URL

When building custom binaries via the build server, you can choose to embed the policy at build time:

**Via Web UI:**
1. Navigate to http://localhost:8082
2. Enter a policy URL in the "Default Policy Path" field (e.g., `https://example.com/policy.rego`)
3. Check the "Embed policy from URL into binary" checkbox
4. Build the binary

**Via API:**
```bash
curl -X POST http://localhost:8082/api/build \
  -H "Content-Type: application/json" \
  -d '{
    "default_policy_path": "https://example.com/policy.rego",
    "embed_policy_from_url": true,
    "version": "1.0.0"
  }'
```

**How it works:**
- During build, the policy is downloaded from the URL
- The policy is saved to a temporary .rego file
- The temporary file path is embedded in the binary
- The binary includes the policy content (not the URL)
- Cleanup happens automatically after build

**Benefits:**
- Self-contained binaries that don't require network access at runtime
- Consistent policy version across deployments
- Works in air-gapped environments

### 3. Build-Time Policy Embedding from Files

The builder can embed policy files directly from a local file, directory, or **zip archive** into the binary. This uses the `embed_policy_files` option.

**Supported sources:**

| Source | Description |
|--------|-------------|
| Single `.rego` file | Embeds one policy file |
| Directory of `.rego` files | Embeds all `.rego` files in the directory |
| `.zip` archive | Extracts and embeds all `.rego` files from the archive |

**Via API (zip archive):**
```bash
curl -X POST http://localhost:8082/api/build \
  -H "Content-Type: application/json" \
  -d '{
    "default_policy_path": "./policies.zip",
    "embed_policy_files": true,
    "version": "1.0.0"
  }'
```

**Via API (directory):**
```bash
curl -X POST http://localhost:8082/api/build \
  -H "Content-Type: application/json" \
  -d '{
    "default_policy_path": "./policies/",
    "embed_policy_files": true,
    "version": "1.0.0"
  }'
```

**How it works:**
- The builder reads `.rego` files from the specified path (file, directory, or zip)
- A Go source file (`_embedded_policies_gen.go`) is generated with an `init()` function that populates `embeddedPolicyFiles`
- The generated file is compiled into the binary
- The temporary generated file is cleaned up after the build
- When `embed_policy_files` is used, the `-policy-path` flag is cleared in the built binary (embedded policies are used instead)

**Preparing a zip archive:**
```bash
# Create a zip of your policy files
cd policies/
zip ../policies.zip *.rego

# Or include a directory structure (filenames are flattened)
zip -r ../policies.zip modular-example/*.rego
```

### 4. URL vs File Embedding vs Zip Embedding Decision Guide

| Scenario | Recommendation |
|----------|---------------|
| Development/Testing | Runtime URL loading (no embedding) |
| Production with policy updates | Runtime URL loading (no embedding) |
| Air-gapped environments | Embed from file/directory/zip |
| Compliance/audit requirements | Embed from file/directory/zip |
| Quick policy changes | Runtime URL loading (no embedding) |
| Immutable deployments | Embed from file/directory/zip |
| Distributing policies as a bundle | Zip archive embedding |
| CI/CD pipelines | Zip archive or directory embedding |

## Examples

### Example 1: Basic URL Policy

Create a policy and host it:

```rego
# policy.rego
package vault

default result = {
    "allow": false,
    "reason": "default deny"
}

result = {
    "allow": true,
    "reason": "allowed operation"
} {
    input.operation == "READ"
    startswith(input.path, "secret/")
}
```

Use it at runtime:
```bash
safeguard -mount V: -policy-path https://myserver.com/policy.rego
```

### Example 2: GitHub Raw Policy URL

You can use GitHub raw URLs for policies:

```bash
safeguard -mount V: \
  -policy-path https://raw.githubusercontent.com/myorg/policies/main/vault-policy.rego
```

### Example 3: Build with Embedded Policy

Build a custom binary with an embedded policy:

```json
{
  "default_vault_addr": "https://vault.example.com",
  "default_auth_method": "oidc",
  "default_policy_path": "https://policies.example.com/prod-policy.rego",
  "embed_policy_from_url": true,
  "version": "2.0.0",
  "build_tag": "production",
  "target_os": "windows",
  "target_arch": "amd64"
}
```

### Example 4: Testing Local Policy as URL

For testing, you can serve policies locally:

```bash
# Terminal 1: Serve policies via HTTP
cd policies
python -m http.server 8000

# Terminal 2: Use the local URL
safeguard -mount V: -policy-path http://localhost:8000/test-url-policy.rego
```

## Security Considerations

### URL Policies (Runtime)
- ✅ Always use HTTPS in production
- ✅ Validate policy server certificates
- ✅ Consider implementing policy signature verification
- ⚠️ Policy is cached in memory but not persisted to disk
- ⚠️ Policy server must be accessible at startup

### Embedded Policies (Build-Time)
- ✅ Policy is self-contained in binary
- ✅ No runtime network access required
- ✅ Policy version is fixed per build
- ⚠️ Policy is embedded as a file path reference
- ⚠️ Binary must include the policy file in deployment

## Troubleshooting

### "Failed to download policy from URL"
- Check that the URL is accessible
- Verify network connectivity
- Ensure the policy server supports HTTP GET requests
- Check firewall/proxy settings

### "Failed to compile policy"
- Verify the policy syntax is valid REGO
- Check that the policy defines `data.vault.result`
- Test policy locally first: `safeguard -policy-path ./local-policy.rego`

### Policy not updating at runtime
- URL policies are cached in memory
- Restart the application to fetch the latest policy
- Or implement a reload mechanism via signal handling

## Implementation Details

### URL Detection
```go
func isURL(path string) bool {
    return strings.HasPrefix(path, "http://") || 
           strings.HasPrefix(path, "https://")
}
```

### Policy Download
- HTTP client with 30-second timeout
- Supports both HTTP and HTTPS
- Returns error on non-200 status codes
- Content is read entirely into memory

### Caching
- URL-based policies are cached in the `PolicyEvaluator` struct
- Cache persists for the lifetime of the evaluator
- `Reload()` method re-downloads from URL

### Build-Time Embedding
- Policy is downloaded during build process
- Saved to temporary file in work directory
- Temporary file path is embedded via ldflags
- Cleanup happens via defer after build completes

## API Reference

### BuildConfig
```go
type BuildConfig struct {
    DefaultPolicyPath  string `json:"default_policy_path"`
    EmbedPolicyFromURL bool   `json:"embed_policy_from_url"`  // Download URL policy and embed at build time
    EmbedPolicyFiles   bool   `json:"embed_policy_files"`     // Embed local policy files/dir/zip into binary
    OutputFilename     string `json:"output_filename"`        // Custom output binary name (optional)
    // ... other fields
}
```

### PolicyEvaluator
```go
type PolicyEvaluator struct {
    policyPath   string
    isURL        bool
    cachedPolicy string // for URL-based policies
    // ... other fields
}
```
