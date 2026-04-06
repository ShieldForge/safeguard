# Path Mapping Example

This example demonstrates how to use path mapping to expose local files and directories in the mounted Vault filesystem.

## Setup

### Example 1: Mapping a Single File

1. Create a test file to map:

```bash
# Windows PowerShell
New-Item -Path "test-file.txt" -ItemType File -Value "Hello from local filesystem!"

# Linux/macOS
echo "Hello from local filesystem!" > test-file.txt
```

2. Create a mapping configuration:

```bash
# Windows PowerShell
@"
{
  "mappings": [
    {
      "virtual_path": "/local/test.txt",
      "real_path": "test-file.txt",
      "read_only": true
    }
  ]
}
"@ | Out-File -Encoding utf8 test-mapping.json

# Linux/macOS
cat > test-mapping.json << 'EOF'
{
  "mappings": [
    {
      "virtual_path": "/local/test.txt",
      "real_path": "test-file.txt",
      "read_only": true
    }
  ]
}
EOF
```

### Example 2: Mapping a Directory

1. Create a test directory with files:

```bash
# Windows PowerShell
New-Item -Path "test-data" -ItemType Directory
"Config content" | Out-File -FilePath "test-data\config.txt"
"Secrets content" | Out-File -FilePath "test-data\secrets.txt"
New-Item -Path "test-data\subdir" -ItemType Directory
"Nested content" | Out-File -FilePath "test-data\subdir\nested.txt"

# Linux/macOS
mkdir -p test-data/subdir
echo "Config content" > test-data/config.txt
echo "Secrets content" > test-data/secrets.txt
echo "Nested content" > test-data/subdir/nested.txt
```

2. Create a directory mapping configuration:

```bash
# Windows PowerShell
@"
{
  "mappings": [
    {
      "virtual_path": "/data",
      "real_path": "test-data",
      "read_only": true
    }
  ]
}
"@ | Out-File -Encoding utf8 dir-mapping.json

# Linux/macOS
cat > dir-mapping.json << 'EOF'
{
  "mappings": [
    {
      "virtual_path": "/data",
      "real_path": "test-data",
      "read_only": true
    }
  ]
}
EOF
```

## Running the Examples

1. Start Vault (if not already running):

```bash
# Windows PowerShell
.\scripts\setup-vault.ps1

# Linux/macOS
./scripts/setup-vault.sh
```

2. Mount the filesystem with path mapping:

**For file mapping example:**
```bash
# Windows PowerShell (in another terminal)
.\safeguard.exe -auth-method token -vault-token root -mapping-config test-mapping.json -debug

# Linux/macOS (in another terminal)
./safeguard -auth-method token -vault-token root -mapping-config test-mapping.json -debug
```

**For directory mapping example:**
```bash
# Windows PowerShell (in another terminal)
.\safeguard.exe -auth-method token -vault-token root -mapping-config dir-mapping.json -debug

# Linux/macOS (in another terminal)
./safeguard -auth-method token -vault-token root -mapping-config dir-mapping.json -debug
```

## Test the Mapping

### Test File Mapping

Access both Vault secrets and mapped files:

```bash
# Windows PowerShell (in a third terminal)
cd V:
dir

# You'll see both Vault structure and mapped files
type secret\database\production\credentials.json
type local\test.txt

# Linux/macOS
cd /tmp/vault   # or /mnt/vault
ls

# You'll see both Vault structure and mapped files
cat secret/database/production/credentials.json
cat local/test.txt
```

### Test Directory Mapping

```bash
# Windows PowerShell
cd V:
dir data
type data\config.txt
dir data\subdir

# Linux/macOS
ls /tmp/vault/data
cat /tmp/vault/data/config.txt
ls /tmp/vault/data/subdir
```

## Expected Output

When you read a mapped file, you should see:

```
Hello from local filesystem!
```

When you list a mapped directory:

```
config.txt
secrets.txt
subdir/
```

## Advanced Example: Mixed Mappings

Map multiple files and directories from different locations:

```json
{
  "mappings": [
    {
      "virtual_path": "/config/app.json",
      "real_path": "C:\\config\\app.json",
      "read_only": true,
      "comment": "Single config file"
    },
    {
      "virtual_path": "/certs",
      "real_path": "/etc/ssl/certs",
      "read_only": true,
      "comment": "Entire certificate directory"
    },
    {
      "virtual_path": "/data",
      "real_path": "./local-data",
      "read_only": true,
      "comment": "Application data directory"
    }
  ]
}
```

This allows you to:
- Access a single config file at `/config/app.json`
- Browse all certificates in `/certs/` directory
- Access all files in `/data/` and its subdirectories

## Cleanup

```bash
# Windows PowerShell
Remove-Item test-file.txt, test-mapping.json -ErrorAction SilentlyContinue
Remove-Item -Recurse test-data, dir-mapping.json -ErrorAction SilentlyContinue
.\scripts\stop-vault.ps1

# Linux/macOS
rm -f test-file.txt test-mapping.json
rm -rf test-data dir-mapping.json
./scripts/stop-vault.sh
```

## Notes

- Mapped files and directories are read-only (write operations not supported)
- Files/directories must exist when the configuration is loaded
- Paths are case-insensitive on Windows, case-sensitive on Linux/macOS
- Directory mappings allow access to all subdirectories and files recursively
- You can mix file and directory mappings in the same configuration
```

## Notes

- Mapped files are read-only (write operations not supported)
- Files must exist when the configuration is loaded
- Paths are case-insensitive on Windows, case-sensitive on Linux/macOS
- Mapped files respect the same access control policies as Vault secrets
- All access to mapped files is logged in the audit log (if enabled)
