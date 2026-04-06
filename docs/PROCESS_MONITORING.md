# Process Monitoring and Access Control

## Overview

safeguard now includes comprehensive process monitoring and access control features that allow you to:

1. **Monitor which processes access secrets** - See PID, UID, and GID for every operation
2. **Control access by process** - Restrict access to specific PIDs or UIDs
3. **Audit all access attempts** - Log all operations to a file for compliance

## Features

### Automatic Process and User Resolution

The monitoring system automatically resolves:
- **Process ID (PID)** → **Process Name** and **Executable Path**
- **User ID (UID)** → **Username**

This means you see human-readable information instead of just numeric IDs:
- Instead of `PID: 12345`, you see `PID: 12345 (powershell.exe) - C:\Windows\System32\...`
- Instead of `UID: 1000`, you see `UID: 1000 (johnsmith)`

**Platform Support:**
- **Windows**: Uses Windows API to query process information
- **Linux**: Reads from `/proc/[pid]/` filesystem
- **macOS**: Uses `/proc/[pid]/` when available

### 1. Process Monitoring

Enable process monitoring to log which process is performing each filesystem operation:

```bash
# Windows
.\safeguard.exe -mount V: -auth-method token -vault-token hvs.xxx -monitor

# Linux/macOS
./safeguard -mount /mnt/vault -auth-method token -vault-token hvs.xxx -monitor
```

**Output Example:**
```
OPEN: secret/myapp/database [PID: 12345 (powershell.exe) - C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe, UID: 1000 (johnsmith), GID: 1000]
READ: secret/myapp/database [PID: 12345 (powershell.exe) - C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe, UID: 1000 (johnsmith), GID: 1000]
READDIR: secret/myapp [PID: 12346 (cmd.exe) - C:\Windows\System32\cmd.exe, UID: 1000 (johnsmith), GID: 1000]
```

### 2. Audit Logging

Log all access attempts (successful and denied) to a file:

```bash
# Windows
.\safeguard.exe -mount V: -auth-method token -vault-token hvs.xxx -audit-log vault-audit.log

# Linux/macOS
./safeguard -mount /mnt/vault -auth-method token -vault-token hvs.xxx -audit-log /var/log/vault-audit.log
```

**Audit Log Format:**
```
2026-01-13T10:30:45Z | SUCCESS | READ | secret/myapp/database | PID:12345(powershell.exe)[C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe] | UID:1000(johnsmith) | GID:1000
2026-01-13T10:30:46Z | DENIED | READ | secret/restricted | PID:12346(cmd.exe)[C:\Windows\System32\cmd.exe] | UID:1001(janesmith) | GID:1001
2026-01-13T10:30:47Z | SUCCESS | READDIR | secret/myapp | PID:12345(powershell.exe)[C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe] | UID:1000(johnsmith) | GID:1000
```

### 3. Access Control

safeguard supports two methods of access control:

1. **Policy-based (REGO)** - Recommended for complex rules
2. **Legacy PID/UID lists** - Simple allow lists

#### Policy-Based Access Control (REGO)

Use [Open Policy Agent](https://www.openpolicyagent.org/) REGO policies for fine-grained, flexible access control:

```bash
# Windows - Use a REGO policy file
.\safeguard.exe -mount V: -auth-method token -vault-token hvs.xxx \
  -access-control -policy-path policies/path-based-access.rego

# Linux/macOS
./safeguard -mount /mnt/vault -auth-method token -vault-token hvs.xxx \
  -access-control -policy-path policies/allow-specific-users.rego
```

**Example Policy** (Allow specific processes):
```rego
package vault

default allow = false

# Allow PowerShell
allow {
    input.process_name == "powershell.exe"
}

# Allow specific application
allow {
    glob.match("C:\\Program Files\\MyApp\\**", [], input.process_path)
}

# Allow admin users for production secrets
allow {
    input.username == "OBSIDIAN\\admin"
    startswith(input.path, "secret/prod")
}
```

**See [policies/README.md](../policies/README.md) for complete policy documentation and examples.**

#### Legacy PID/UID-Based Access Control

For simple use cases, use allow lists:

**By Process ID (PID):**

```bash
# Windows - Allow only PowerShell process
.\safeguard.exe -mount V: -auth-method token -vault-token hvs.xxx \
  -access-control -allowed-pids "12345,12346"

# Linux/macOS - Allow only specific application
./safeguard -mount /mnt/vault -auth-method token -vault-token hvs.xxx \
  -access-control -allowed-pids "1234,5678"
```

**By User ID (UID):**

```bash
# Linux/macOS - Allow only user ID 1000 and 1001
./safeguard -mount /mnt/vault -auth-method token -vault-token hvs.xxx \
  -access-control -allowed-uids "1000,1001"

# Windows - UIDs work differently, typically based on SID mapping
.\safeguard.exe -mount V: -auth-method token -vault-token hvs.xxx \
  -access-control -allowed-uids "1000"
```

#### Combined Restrictions

Use both PID and UID restrictions:

```bash
# Allow specific PIDs OR specific UIDs
./safeguard -mount /mnt/vault -auth-method token -vault-token hvs.xxx \
  -access-control \
  -allowed-pids "1234,5678" \
  -allowed-uids "1000,1001"
```

## Usage Examples

### Example 1: Security Audit Mode

Monitor and log all access for security compliance:

```bash
# Enable monitoring and audit logging
.\safeguard.exe -mount V: \
  -auth-method oidc -auth-role reader \
  -monitor \
  -audit-log C:\logs\vault-access.log \
  -debug
```

**What you get:**
- Console output showing all operations with process info
- Audit log file with timestamped access records
- Debug information for troubleshooting

### Example 2: Restricted Application Access

Only allow a specific application to access secrets:

```bash
# Step 1: Find your application's PID
# Windows PowerShell:
Get-Process notepad | Select-Object Id
# Output: Id: 12345

# Step 2: Run with access control
.\safeguard.exe -mount V: \
  -auth-method token -vault-token hvs.xxx \
  -access-control \
  -allowed-pids "12345" \
  -monitor \
  -audit-log vault-audit.log
```

**Result:**
- Only PID 12345 (notepad) can access secrets
- All other processes get "Access Denied"
- All attempts are logged to vault-audit.log

### Example 3: Multi-User Environment

Allow specific users to access secrets:

```bash
# Linux - Allow users with UID 1000 and 1002
./safeguard -mount /mnt/vault \
  -auth-method ldap \
  -ldap-username admin \
  -ldap-password secret \
  -access-control \
  -allowed-uids "1000,1002" \
  -audit-log /var/log/vault-audit.log
```

### Example 4: Full Security Stack

Combine all features for maximum security:

```bash
# Windows - Complete monitoring and access control
.\safeguard.exe -mount V: \
  -auth-method oidc -auth-role security-team \
  -monitor \
  -debug \
  -audit-log C:\vault\logs\audit-$(Get-Date -Format "yyyy-MM-dd").log \
  -access-control \
  -allowed-uids "1000,1001"
```

## Finding Process Information

### Windows

**Find Process ID:**
```powershell
# By process name
Get-Process | Where-Object {$_.Name -like "*powershell*"} | Select-Object Id, Name

# Currently running PowerShell
$PID

# All processes accessing a file (requires Handle.exe from Sysinternals)
handle.exe "V:\secret"
```

**Find User ID:**
```powershell
# Current user's SID/UID mapping
whoami /user
```

### Linux/macOS

**Find Process ID:**
```bash
# By process name
ps aux | grep myapp

# Processes accessing the mount
lsof /mnt/vault

# Current shell PID
echo $$
```

**Find User ID:**
```bash
# Current user
id -u

# Specific user
id -u username

# All users
cat /etc/passwd | cut -d: -f1,3
```

## Access Control Behavior

### When Access Control is Disabled (default)
- All processes can access secrets
- No PID/UID restrictions
- Monitoring still works if enabled

### When Access Control is Enabled
- **With allowed PIDs set**: Only those PIDs can access
- **With allowed UIDs set**: Only those UIDs can access
- **With both set**: Process must match either PID OR UID list
- **With neither set**: All access is allowed (same as disabled)

### Access Denied Behavior
When a process is denied access:
- Returns `EACCES` (Permission Denied) error
- Logs denial in console (if debug/monitor enabled)
- Records denial in audit log (if configured)

## Security Considerations

### PID-Based Access Control
**Pros:**
- Very specific - targets exact process
- Useful for testing and debugging
- Works cross-platform

**Cons:**
- PIDs change when process restarts
- Requires updating configuration
- Not suitable for long-running services

**Best for:** Development, testing, single-session use

### UID-Based Access Control
**Pros:**
- Persistent across process restarts
- Standard Unix security model
- Works well for multi-user systems

**Cons:**
- Windows UID mapping can be complex
- Less granular than PID control

**Best for:** Production, multi-user environments, long-running services

## Troubleshooting

### Problem: Access Denied for My Process

**Solution:**
1. Enable monitoring to see the PID/UID:
   ```bash
   -monitor -debug
   ```

2. Check the logged PID/UID in console output

3. Add to allowed list:
   ```bash
   -access-control -allowed-pids "YOUR_PID"
   ```

### Problem: Audit Log Not Writing

**Check:**
1. File path is writable
2. Directory exists
3. Sufficient disk space
4. File permissions (Linux/macOS)

**Solution:**
```bash
# Linux/macOS
mkdir -p /var/log/vault
chmod 755 /var/log/vault

# Windows
New-Item -ItemType Directory -Path C:\vault\logs -Force
```

### Problem: Can't Find Process ID

**Windows:**
```powershell
# Find all PowerShell processes
Get-Process powershell | Select-Object Id, Name, Path

# Find process by window title
Get-Process | Where-Object {$_.MainWindowTitle -like "*MyApp*"}
```

**Linux/macOS:**
```bash
# Find by name
pgrep myapp

# Find with details
ps aux | grep myapp

# Find what's accessing the mount
lsof | grep /mnt/vault
```

## Performance Impact

- **Monitoring**: Minimal (<1% overhead)
- **Audit Logging**: Low (<2% overhead for disk I/O)
- **Access Control**: Negligible (simple map lookup)

All features are designed to have minimal impact on filesystem performance.

## Integration Examples

### With Docker Containers

```bash
# Get container's main process PID
CONTAINER_PID=$(docker inspect -f '{{.State.Pid}}' mycontainer)

# Allow container access
./safeguard -mount /mnt/vault \
  -auth-method token -vault-token hvs.xxx \
  -access-control \
  -allowed-pids "$CONTAINER_PID"
```

### With Systemd Services

```bash
# Find service PID
systemctl show --property MainPID myservice

# Allow service access
./safeguard -mount /mnt/vault \
  -access-control \
  -allowed-pids "$(systemctl show --property MainPID myservice | cut -d= -f2)"
```

### With Scheduled Tasks

For tasks that run repeatedly, use UID-based access:

```bash
# Find task user ID
id -u taskuser

# Configure access
./safeguard -mount /mnt/vault \
  -access-control \
  -allowed-uids "$(id -u taskuser)"
```

## Compliance and Auditing

The audit log format is designed for compliance requirements and includes resolved names:

- **Timestamp**: ISO 8601 format with timezone
- **Status**: SUCCESS or DENIED
- **Operation**: Type of filesystem operation
- **Path**: Secret path accessed
- **Process Info**: PID with process name and executable path
- **User Info**: UID with username, plus GID

This enhanced format provides:
1. **Immediate readability** - No need to look up PIDs or UIDs
2. **Full audit trail** - Know exactly which program accessed what
3. **Forensic value** - Executable path helps identify malicious actors
4. **Compliance ready** - Human-readable logs for auditors

This format can be easily parsed by log aggregation tools (Splunk, ELK, etc.) or compliance monitoring systems.

### Example Log Analysis

```bash
# Count successful reads
grep "SUCCESS | READ" vault-audit.log | wc -l

# Find denied access attempts
grep "DENIED" vault-audit.log

# Operations by specific user
grep "UID:1000(johnsmith)" vault-audit.log

# Operations by specific process
grep "powershell.exe" vault-audit.log

# Access to specific secret
grep "secret/production/database" vault-audit.log

# Find all processes that accessed secrets
grep "SUCCESS" vault-audit.log | grep -oP 'PID:\d+\([^)]+\)' | sort -u
```
