# Dynamic Mount Point Discovery

## Overview

The application now dynamically discovers available secret engine mount points from HashiCorp Vault instead of assuming a hardcoded "secret" mount. This makes the filesystem work with any Vault configuration, including custom mount points and multiple KV engines.

## Implementation Details

### API Endpoint

The application queries Vault's system mounts endpoint:
```
GET /v1/sys/mounts
```

This returns all configured secret engines with metadata including:
- Mount path
- Engine type
- Description
- Configuration options

### Key Components

#### 1. Mount Discovery (`pkg/vault/adapter/hashicorp.go`)

**`ListMounts()` Method**
- Queries `/v1/sys/mounts` endpoint
- Parses response to extract KV secret engines
- Caches mount information for performance
- Returns a map of mount paths to their metadata

```go
func (c *Client) ListMounts() (map[string]MountInfo, error) {
    // Makes HTTP request to /v1/sys/mounts
    // Caches results in c.mounts
    // Returns mount information
}
```

**`GetMountForPath()` Method**
- Determines which mount a given path belongs to
- Parses the first segment of the path
- Returns mount information or error if not found

```go
func (c *Client) GetMountForPath(path string) (*MountInfo, error) {
    // Extracts mount from path like "secret/app1/db"
    // Returns mount info for "secret"
}
```

**`RefreshMounts()` Method**
- Forces a refresh of the mount cache
- Useful after Vault configuration changes
- Called automatically on first use

#### 2. Dynamic Path Construction

**`constructAPIPath()` Method**
- Updated to work with any mount point
- Extracts mount from the path
- Builds the appropriate KV v2 API path
- Handles both data and metadata endpoints

Example:
- Input: `"secret/app1/database"` → Output: `"/v1/secret/data/app1/database"`
- Input: `"custom-kv/myapp/config"` → Output: `"/v1/custom-kv/data/myapp/config"`

#### 3. Root Directory Listing

**`List()` Method**
- When path is empty or root (`""` or `"/"`), returns mount points
- Shows all available secret engines as directories
- Each mount appears as a folder in the filesystem

Example root listing:
```
secret/
kv/
custom-secrets/
```

### Testing

The test infrastructure was updated to support dynamic mounts:

#### Mock Vault Server
- Implements `/v1/sys/mounts` endpoint
- Supports multiple mount configurations
- Dynamically handles requests to any mount path

#### Test Data Format
All test data now includes mount prefixes:
```go
mock.SetSecret("secret/app1/database", data)  // Instead of "app1/database"
mock.SetList("secret/app1", keys)             // Instead of "app1"
```

#### Test Coverage
- Tests verify mount discovery works correctly
- Tests ensure paths are constructed properly for different mounts
- Tests validate root directory shows available mounts

## Benefits

### 1. Flexibility
- Works with any Vault mount configuration
- No code changes needed for custom mounts
- Supports multiple KV engines simultaneously

### 2. User Experience
- Root directory shows what's available
- No guesswork about mount names
- Navigate naturally through the hierarchy

### 3. Production Ready
- Works with enterprise Vault configurations
- Handles custom naming conventions
- Adapts to Vault changes dynamically

## Usage Examples

### Standard Configuration
```bash
# Vault has default "secret/" mount
$ ls /mnt/vault
secret/

$ ls /mnt/vault/secret
app1/  app2/  config
```

### Custom Mounts
```bash
# Vault has multiple custom mounts
$ ls /mnt/vault
production/  staging/  development/

$ cat /mnt/vault/production/database/creds
username: prod_user
password: ***
```

### Multiple Environments
```bash
# Different teams using different mounts
$ ls /mnt/vault
team-a-secrets/  team-b-secrets/  shared/

$ cat /mnt/vault/team-a-secrets/api-keys
key: abc123
```

## Configuration

No additional configuration required! The application automatically:
1. Connects to Vault
2. Authenticates
3. Queries available mounts
4. Displays them in the filesystem

## Performance Considerations

### Mount Caching
- Mount information is cached after first query
- Reduces API calls to Vault
- Cache is stored in memory

### Refresh Strategy
- Call `RefreshMounts()` to update cache
- Automatic refresh on initialization
- Manual refresh available if needed

### Network Overhead
- Single API call to discover mounts
- Minimal impact on performance
- Cached for subsequent operations

## Backward Compatibility

The implementation maintains compatibility with existing Vault configurations:
- Works with standard "secret/" mount
- Handles legacy setups automatically
- No migration required

## Technical Notes

### Path Parsing
The application expects paths in the format:
```
<mount>/<path>/<to>/<secret>
```

For example:
- `secret/app1/database` → Mount: `secret`, Path: `app1/database`
- `kv/prod/api` → Mount: `kv`, Path: `prod/api`

### API Endpoint Format
KV v2 endpoints follow this pattern:
- **Data**: `/v1/<mount>/data/<path>`
- **Metadata**: `/v1/<mount>/metadata/<path>`

The application automatically constructs the correct URL based on the operation.

### Error Handling
- Invalid mount paths return appropriate errors
- Missing mounts are detected early
- Clear error messages for troubleshooting

## Future Enhancements

Possible improvements for future versions:
- Automatic mount refresh on a schedule
- Support for non-KV secret engines
- Mount-specific configuration options
- Filtering of mount types
