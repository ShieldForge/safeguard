# REGO Policy Examples for safeguard

This directory contains example REGO policies for fine-grained access control in safeguard.

## Using Policies

You can use either:
- **Single policy file**: `-policy-path policies/allow-specific-processes.rego`
- **Policy directory**: `-policy-path policies/modular-example`
- **Zip archive**: `-policy-path policies.zip` (at build time with `embed_policy_files`)

When using a directory, all `.rego` files are loaded and combined into a single policy.

## Policy Structure

All policies must:
1. Declare `package vault`
2. Define a `result` rule that returns `{"allow": bool, "reason": string}`
3. Use `default result = {"allow": false, "reason": ""}` for a deny-by-default approach

See [modular-example/RESULT_OBJECT.md](modular-example/RESULT_OBJECT.md) for full details on the result object format.

## Input Data

The policy receives the following input data:

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

### Input Fields

- **path**: The secret path being accessed (e.g., `secret/myapp/config`)
- **operation**: The FUSE operation (`GETATTR`, `READDIR`, `OPEN`, `READ`)
- **pid**: Process ID of the accessing process
- **uid**: User ID (may not be reliable on Windows)
- **gid**: Group ID (may not be reliable on Windows)
- **process_name**: Name of the executable (e.g., `powershell.exe`)
- **process_path**: Full path to the executable
- **username**: Resolved username (e.g., `OBSIDIAN\\jonat` on Windows)
- **metadata**: Optional metadata map (empty by default)

## Example Policies

> **Note**: The safeguard process itself is always allowed access regardless of policies. This is necessary for mounting and internal filesystem operations. Additionally, system/kernel operations (PID -1) are automatically allowed. Policies only restrict access from other processes accessing the mounted filesystem.

### 1. allow-specific-processes.rego

Allows access only from specific processes by name or path pattern.

**Use case**: Restrict access to only trusted applications.

```bash
safeguard.exe -policy-path policies/allow-specific-processes.rego -access-control
```

### 2. allow-specific-users.rego

Allows access only from specific users or domains.

**Use case**: Restrict access to specific users or organizational units.

```bash
safeguard.exe -policy-path policies/allow-specific-users.rego -access-control
```

### 3. path-based-access.rego

Different users and processes get access to different secret paths.

**Use case**: Segregate access to production, development, and test secrets.

```bash
safeguard.exe -policy-path policies/path-based-access.rego -access-control
```

### 4. time-based-access.rego

Restricts access based on time of day and day of week.

**Use case**: Enforce business hours access, allow emergency access for admins.

```bash
safeguard.exe -policy-path policies/time-based-access.rego -access-control
```

### 5. complex-example.rego

Demonstrates advanced policy with multiple factors and deny rules.

**Use case**: Production environments with strict security requirements.

```bash
safeguard.exe -policy-path policies/complex-example.rego -access-control
```

### 6. modular-example/ (Directory)

Demonstrates organizing policies into multiple files for better maintainability.

**Use case**: Large teams with complex access requirements.

```bash
safeguard.exe -policy-path policies/modular-example -access-control
```

See [modular-example/README.md](modular-example/README.md) for details on organizing policies into multiple files.

## Creating Your Own Policy

1. Start with the template:

```rego
package vault

# Default deny
default result = {"allow": false, "reason": ""}

# Add your result rules here
result = {"allow": true, "reason": "Explanation"} {
    # Your conditions
}
```

2. Test your policy syntax:

```bash
safeguard.exe -policy-path your-policy.rego -access-control
```

3. The application will validate the policy at startup and report any errors.

## Policy Development Tips

### Testing Conditions

Use multiple `result` rules - the first matching rule determines the outcome:

```rego
# Rule 1
result = {"allow": true, "reason": "Admin user"} {
    input.username == "admin"
}

# Rule 2
result = {"allow": true, "reason": "Trusted process"} {
    input.process_name == "trusted.exe"
}
```

### Using Glob Patterns

```rego
result = {"allow": true, "reason": "MyApp access"} {
    glob.match("C:\\Program Files\\MyApp\\**", [], input.process_path)
}
```

### String Operations

```rego
# Check prefix
result = {"allow": true, "reason": "Dev access"} {
    startswith(input.path, "secret/dev")
}
```

### Sets for Multiple Values

```rego
allowed_users := {
    "user1",
    "user2",
    "user3",
}

result = {"allow": true, "reason": "Authorized user"} {
    input.username in allowed_users
}
```

### Combining Conditions

```rego
result = {"allow": true, "reason": "Admin prod read"} {
    input.username == "admin"
    input.operation == "READ"
    startswith(input.path, "secret/prod")
}
```

All conditions within a `result` block must be true (AND logic).

### Deny Rules

Implement explicit deny rules that override allow rules:

```rego
is_denied {
    input.process_name == "malware.exe"
}

result = {"allow": false, "reason": "Suspicious process blocked"} {
    is_denied
}

result = {"allow": true, "reason": "Allowed"} {
    not is_denied
    # Your conditions
}
```

## Migration from PID/UID Lists

If you're currently using `-allowed-pids` or `-allowed-uids`, you can replicate that with a policy:

**Old:**
```bash
safeguard.exe -access-control -allowed-pids 1234,5678
```

**New:**
```rego
package vault

default result = {"allow": false, "reason": ""}

result = {"allow": true, "reason": "Allowed PID"} {
    input.pid in {1234, 5678}
}
```

```bash
safeguard.exe -policy-path my-policy.rego -access-control
```

The policy-based approach is more powerful as you can combine multiple conditions.

## Troubleshooting

### Policy Not Loading

- Check file path is correct
- Verify policy file is readable
- Check for syntax errors in the REGO code

### Access Denied Unexpectedly

- Enable debug mode to see policy decisions: `-debug`
- Check the audit log for decision reasons: `-audit-log vault-audit.log`
- Verify your policy evaluates to `true` for the given input

### Policy Syntax Errors

The application will report syntax errors at startup:
```
Failed to load policy file policies/my-policy.rego: <error details>
```

## References

- [OPA Documentation](https://www.openpolicyagent.org/docs/latest/)
- [REGO Language Reference](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [REGO Playground](https://play.openpolicyagent.org/) - Test policies online
