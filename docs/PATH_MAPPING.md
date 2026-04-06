# Path Mapping Configuration

This document describes how to configure virtual path mappings to expose real files and directories in the mounted Vault filesystem.

## Overview

Path mapping allows you to expose real files and directories from your local filesystem as if they were Vault secrets. This is useful for:

- Providing configuration files and directories alongside Vault secrets
- Exposing certificates, keys, or entire PKI directories stored locally
- Creating a unified view of secrets and local files
- Testing and development scenarios
- Mounting application configuration directories

## Configuration File Format

The configuration is a JSON file with the following structure:

```json
{
  "mappings": [
    {
      "virtual_path": "/path/in/mounted/filesystem",
      "real_path": "/actual/path/on/disk",
      "read_only": true
    }
  ]
}
```

### Fields

- **`virtual_path`** (required): The path where the file/directory will appear in the mounted filesystem
  - Must start with `/` or be a relative path
  - Case sensitivity depends on the OS (case-insensitive on Windows)
  - Example: `/config/app.json`, `/certs`, `/data/myapp`

- **`real_path`** (required): The actual file or directory path on the local filesystem
  - Can be absolute or relative
  - Must exist (can be a regular file or directory)
  - If it's a directory, all contents will be accessible recursively
  - Example: `/etc/myapp/config.json`, `C:\certs`, `/var/data/myapp`

- **`read_only`** (optional): Whether the file/directory is read-only
  - Default: `true`
  - Currently, only read-only mode is supported

- **`is_directory`** (optional): Auto-detected, indicates if this is a directory mapping
  - You don't need to specify this field; it's detected automatically

## Example Configuration

See [mapping-config.example.json](mapping-config.example.json) for a complete example.

```json
{
  "mappings": [
    {
      "virtual_path": "/config/database.json",
      "real_path": "/etc/myapp/database.json",
      "read_only": true
    },
    {
      "virtual_path": "/certs",
      "real_path": "/opt/ssl",
      "read_only": true
    },
    {
      "virtual_path": "/data/myapp",
      "real_path": "C:\\ProgramData\\MyApp",
      "read_only": true
    }
  ]
}
```

### Mapping Types

The configuration supports two types of mappings:

1. **File Mapping**: Maps a single file to a virtual path
   ```json
   {
     "virtual_path": "/config/app.json",
     "real_path": "/etc/myapp/config.json"
   }
   ```

2. **Directory Mapping**: Maps an entire directory (including all subdirectories and files)
   ```json
   {
     "virtual_path": "/certs",
     "real_path": "/opt/ssl"
   }
   ```
   
   With directory mapping, all files and subdirectories are accessible:
   - `/certs/server.crt` → `/opt/ssl/server.crt`
   - `/certs/private/server.key` → `/opt/ssl/private/server.key`
   - `/certs/ca/root.crt` → `/opt/ssl/ca/root.crt`

## Usage

### Command Line

Use the `-mapping-config` flag to specify the configuration file:

```bash
# Linux/macOS
./safeguard \
  -auth-method token \
  -vault-token root \
  -mapping-config ./mapping-config.json

# Windows
.\safeguard.exe -auth-method token -vault-token root -mapping-config .\mapping-config.json
```

### Accessing Mapped Files and Directories

Once mounted, mapped files and directories appear just like Vault secrets:

```bash
# Linux/macOS - accessing individual files
cat /tmp/vault/config/database.json

# Linux/macOS - accessing directory contents
ls /tmp/vault/certs
cat /tmp/vault/certs/server.crt
cat /tmp/vault/certs/private/server.key

# Windows - accessing individual files
type V:\config\database.json

# Windows - accessing directory contents
dir V:\certs
type V:\certs\server.crt
type V:\certs\private\server.key
```

## Path Resolution

### Virtual Paths

Virtual paths are normalized automatically:
- Leading/trailing slashes are removed
- Multiple consecutive slashes are collapsed
- On Windows, paths are case-insensitive

These all refer to the same virtual file:
- `/config/app.json`
- `config/app.json`
- `/config//app.json/`

### Real Paths

Real paths are resolved to absolute paths:
- Relative paths are resolved from the current working directory
- Symlinks are followed
- The file or directory must exist when the configuration is loaded

### Directory Mapping Resolution

When a directory is mapped, all paths under that virtual path are automatically resolved:

```json
{
  "virtual_path": "/data",
  "real_path": "/opt/myapp/data"
}
```

Path resolution:
- `/data` → `/opt/myapp/data` (the directory itself)
- `/data/file.txt` → `/opt/myapp/data/file.txt`
- `/data/subdir/file.txt` → `/opt/myapp/data/subdir/file.txt`

## Limitations

```json
{
  "mappings": [
    {
      "virtual_path": "/dev/local-secrets.env",
      "real_path": "./local-secrets.env",
      "read_only": true
    },
    {
      "virtual_path": "/dev/test-data.json",
      "real_path": "./test-data.json",
      "read_only": true
    }
  ]
}
```

## Troubleshooting

### Configuration Not Loading

Check the logs for errors:
```
WARN: Failed to load path mapping config
```

Common issues:
- Configuration file doesn't exist
- Invalid JSON syntax
- Real file doesn't exist
- Permission denied reading the file

### File Not Appearing

If a mapped file doesn't appear in the mounted filesystem:

1. Check the virtual path is correct (case-sensitivity on Linux/macOS)
2. Verify the real file exists: `ls -l /path/to/real/file`
3. Check file permissions
4. Enable debug logging: `-debug` flag

### Permission Denied

If you get "permission denied" when accessing a mapped file:

1. Check the real file permissions
2. Verify the user running safeguard can read the file
3. Check access control policies (if enabled)
4. Review audit logs for denied access attempts

## Combining with Vault Secrets

Mapped files and Vault secrets coexist in the same filesystem:

```bash
# Vault secret
cat /tmp/vault/secret/database/credentials.json

# Mapped file
cat /tmp/vault/config/app.json
```

Both are accessed the same way, but:
- Vault secrets come from the Vault server
- Mapped files come from the local filesystem
- Access control applies to both
- Both are logged in the audit log

## Best Practices

1. **Use absolute paths** for real files in production
2. **Document mappings** in your deployment documentation
3. **Apply access control** using REGO policies
4. **Enable audit logging** to track access
5. **Keep mappings minimal** - prefer storing secrets in Vault when possible
6. **Test mappings** before deploying to production
7. **Use descriptive virtual paths** that indicate the file's purpose

## See Also

- [Policy Quick Start](POLICY_QUICKSTART.md)
- [Process Monitoring](PROCESS_MONITORING.md)
- [Audit Logging](../README.md#process-monitoring--access-control)
