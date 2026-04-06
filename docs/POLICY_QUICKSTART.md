# Quick Start: REGO Policy-Based Access Control

This guide shows you how to quickly get started with REGO policy-based access control in safeguard.

## Important: How Policies Work

> **Note**: Policies only apply to processes **accessing** the mounted filesystem. The safeguard process itself and system/kernel operations (PID -1) are always allowed access - this is required for mounting and internal operations. Your policies control which external processes (PowerShell, Python, etc.) can read the mounted secrets.

## What is REGO?

REGO is the policy language used by [Open Policy Agent (OPA)](https://www.openpolicyagent.org/). It allows you to write flexible, fine-grained access control rules that can combine multiple conditions like process name, username, secret path, time of day, and more.

## Why Use REGO Policies?

**Instead of:**
```bash
# Limited to simple PID/UID lists
safeguard.exe -access-control -allowed-pids "12345,67890"
```

**You can:**
```bash
# Use flexible, composable rules
safeguard.exe -access-control -policy-path my-policy.rego
```

With policies you can:
- ✅ Allow PowerShell but only for dev secrets
- ✅ Allow admin users to access production
- ✅ Allow specific apps to access only their secrets
- ✅ Combine multiple conditions (user AND process AND path)
- ✅ Time-based restrictions (business hours only)
- ✅ Deny rules for suspicious processes

## Quick Example

### Option 1: Single Policy File

#### 1. Create a Policy File

Create `my-policy.rego`:

```rego
package vault

# Default: deny everything
default result = {"allow": false, "reason": ""}

# Allow PowerShell to read dev secrets
result = {"allow": true, "reason": "PowerShell dev access"} {
    input.process_name == "powershell.exe"
    startswith(input.path, "secret/dev")
}

# Allow admin users to access everything
result = {"allow": true, "reason": "Admin user"} {
    input.username == "OBSIDIAN\\admin"
}
```

> **Note**: Policies must return a `result` object with `allow` (boolean) and `reason` (string) fields. The engine queries `data.vault.result` — bare `allow` rules will not work. See [policies/modular-example/RESULT_OBJECT.md](../policies/modular-example/RESULT_OBJECT.md) for details.

#### 2. Run with Policy

```bash
safeguard.exe -mount V: -auth-method token -vault-token hvs.xxx \
  -access-control -policy-path my-policy.rego
```

### Option 2: Policy Directory (Modular Approach)

For larger projects, organize policies into multiple files:

#### 1. Create a Policy Directory

```bash
mkdir policies
```

**policies/base.rego**:
```rego
package vault

default result = {"allow": false, "reason": ""}

admin_users := {"OBSIDIAN\\admin"}
```

**policies/dev-access.rego**:
```rego
package vault

result = {"allow": true, "reason": "PowerShell dev access"} {
    input.process_name == "powershell.exe"
    startswith(input.path, "secret/dev")
}
```

**policies/admin-access.rego**:
```rego
package vault

result = {"allow": true, "reason": "Admin user"} {
    input.username in admin_users
}
```

#### 2. Run with Policy Directory

```bash
safeguard.exe -mount V: -auth-method token -vault-token hvs.xxx \
  -access-control -policy-path policies
```

All `.rego` files in the directory are loaded automatically!

### Why Use a Directory?

**Benefits of modular policies:**
- ✅ **Organization**: Separate concerns (admin rules, dev rules, etc.)
- ✅ **Maintainability**: Update one file without touching others
- ✅ **Collaboration**: Team members work on different policy files
- ✅ **Reusability**: Share definitions (like user sets) across files
- ✅ **Readability**: Smaller files are easier to understand

**Example: Adding a new team**

Just create a new file in the directory - no need to modify existing policies!

```bash
# Add new policy file
echo 'package vault

qa_users := {"OBSIDIAN\\qa1", "OBSIDIAN\\qa2"}

result = {"allow": true, "reason": "QA team test access"} {
    input.username in qa_users
    startswith(input.path, "secret/test")
}' > policies/qa-team.rego

# Reload (or restart) - new rules are automatically included!
```

### 3. Test It

```powershell
# PowerShell can read dev secrets ✅
cd V:\secret\dev
type config

# PowerShell CANNOT read prod secrets ❌
cd V:\secret\prod
type database
# Access Denied

# Admin user can read anything ✅
# (When running as admin user)
cd V:\secret\prod
type database
# Works!
```

## Common Patterns

### Allow Specific Processes

```rego
package vault

default result = {"allow": false, "reason": ""}

# List of allowed processes
result = {"allow": true, "reason": "Trusted process"} {
    input.process_name in {
        "powershell.exe",
        "cmd.exe",
        "python.exe"
    }
}
```

### Allow Specific Users

```rego
package vault

default result = {"allow": false, "reason": ""}

# Allow specific users
result = {"allow": true, "reason": "Authorized user"} {
    input.username in {
        "OBSIDIAN\\admin",
        "OBSIDIAN\\developer"
    }
}
```

### Path-Based Access

```rego
package vault

default result = {"allow": false, "reason": ""}

# Developers can access dev secrets
result = {"allow": true, "reason": "Developer dev access"} {
    startswith(input.username, "OBSIDIAN\\dev_")
    startswith(input.path, "secret/dev")
}

# Admins can access everything
result = {"allow": true, "reason": "Admin user"} {
    input.username == "OBSIDIAN\\admin"
}

# Anyone can read documentation
result = {"allow": true, "reason": "Public documentation"} {
    startswith(input.path, "secret/docs")
    input.operation == "READ"
}
```

### Application-Specific Access

```rego
package vault

default result = {"allow": false, "reason": ""}

# MyApp can only access its own secrets
result = {"allow": true, "reason": "MyApp accessing own secrets"} {
    glob.match("C:\\Program Files\\MyApp\\**", [], input.process_path)
    startswith(input.path, "secret/myapp")
}

# Allow specific PID (for scheduled tasks)
result = {"allow": true, "reason": "Scheduled task"} {
    input.pid == 12345
    startswith(input.path, "secret/automation")
}
```

### Combined Conditions

```rego
package vault

default result = {"allow": false, "reason": ""}

# Production requires trusted process AND admin user AND read-only
result = {"allow": true, "reason": "Authorized production read"} {
    input.process_name == "powershell.exe"
    input.username == "OBSIDIAN\\admin"
    startswith(input.path, "secret/prod")
    input.operation == "READ"
}
```

## Input Data Reference

Your policy receives this information about each access attempt:

```json
{
    "path": "secret/myapp/config",
    "operation": "READ",
    "pid": 12345,
    "uid": 1000,
    "gid": 1000,
    "process_name": "powershell.exe",
    "process_path": "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
    "username": "OBSIDIAN\\jonat",
    "metadata": {}
}
```

### Operations

- `GETATTR` - Getting file attributes (ls, dir, stat)
- `READDIR` - Listing directory contents
- `OPEN` - Opening a file
- `READ` - Reading file contents

## Testing Your Policy

### 1. Check for Syntax Errors

The application validates your policy at startup:

```bash
safeguard.exe -access-control -policy-path my-policy.rego
# If invalid: "Invalid policy file: <error details>"
# If valid: "Policy-based access control enabled: my-policy.rego"
```

### 2. Use Debug Mode

See policy decisions in real-time:

```bash
safeguard.exe -access-control -policy-path my-policy.rego -debug
```

Output shows when access is denied:
```
Access denied by policy: policy decision
```

### 3. Use Audit Logging

Track all access attempts:

```bash
safeguard.exe -access-control -policy-path my-policy.rego -audit-log vault-audit.log
```

Check the audit log:
```
2026-01-13T10:30:46Z | DENIED | READ | secret/prod/database | PID:12345(powershell.exe) | UID:1000(johnsmith)
```

## Example Policies

We provide several example policies in the `policies/` directory:

1. **allow-specific-processes.rego** - Only allow trusted processes
2. **allow-specific-users.rego** - Only allow specific users/domains
3. **path-based-access.rego** - Different users get different paths
4. **time-based-access.rego** - Restrict by time/day
5. **complex-example.rego** - Multi-factor security

Try them:

```bash
safeguard.exe -access-control -policy-path policies/path-based-access.rego
```

## Debugging Tips

### Access Denied Unexpectedly?

1. Enable debug mode: `-debug`
2. Check audit log: `-audit-log vault-audit.log`
3. Verify your input values match your policy

Example: Check your username format
```powershell
# Windows username format is usually "DOMAIN\\username"
whoami
# Output: obsidian\jonat

# In policy, use:
input.username == "OBSIDIAN\\jonat"  # Note: uppercase and double backslash
```

### Policy Not Loading?

- Check file path is correct (absolute or relative)
- Verify file is readable
- Check for REGO syntax errors
- Make sure `package vault` is declared
- Ensure you have a rule named `result` (not bare `allow`)

### Still Not Working?

1. Start with a simple policy that allows everything:
```rego
package vault

default result = {"allow": false, "reason": ""}

# Temporary: allow everything for testing
result = {"allow": true, "reason": "test allow all"} {
    true
}
```

2. Gradually add restrictions
3. Test each change

## Migration from PID/UID Lists

**Before:**
```bash
safeguard.exe -access-control -allowed-pids "1234,5678" -allowed-uids "1000"
```

**After:**
```rego
package vault

default result = {"allow": false, "reason": ""}

# Same logic as PID list
result = {"allow": true, "reason": "Allowed PID"} {
    input.pid in {1234, 5678}
}

# OR same logic as UID list
result = {"allow": true, "reason": "Allowed UID"} {
    input.uid == 1000
}
```

**Better:**
```rego
package vault

default result = {"allow": false, "reason": ""}

# More maintainable
result = {"allow": true, "reason": "Trusted application"} {
    # Instead of PIDs, use process names
    input.process_name == "myapp.exe"
}

result = {"allow": true, "reason": "Admin user"} {
    # Instead of UIDs, use usernames
    input.username == "OBSIDIAN\\admin"
}
```

## Learn More

- See [policies/README.md](../policies/README.md) for complete policy documentation
- See [PROCESS_MONITORING.md](PROCESS_MONITORING.md) for security features
- Try [OPA Playground](https://play.openpolicyagent.org/) to test REGO online
- Read [OPA Documentation](https://www.openpolicyagent.org/docs/latest/)

## Summary

1. Create a `.rego` file with `result` rules (returning `{"allow": bool, "reason": string}`)
2. Run: `safeguard.exe -access-control -policy-path your-policy.rego`
3. Test with `-debug` and `-audit-log` flags
4. Start simple, add complexity as needed

Policies are much more powerful and maintainable than simple PID/UID lists!
