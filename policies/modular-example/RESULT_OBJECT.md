# Policy Result Object

## Overview

All policies now return a single `result` object containing both the access decision (`allow`) and an explanation (`reason`). This provides better performance by requiring only a single policy evaluation instead of two separate queries.

## Result Object Format

The result object is a JSON structure with two fields:

```rego
result = {"allow": true, "reason": "Explanation text"}
```

- **allow** (boolean): Whether access should be granted
- **reason** (string): Human-readable explanation for the decision

## How It Works

In REGO, you define a `result` rule that returns the combined object:

```rego
# Default: deny with no reason
default result = {"allow": false, "reason": ""}

# Allow rule with reason
result = {"allow": true, "reason": "Admin accessing production database"} {
    input.username == "admin"
    input.path == "secret/prod/database"
}
```

## Single Query Evaluation

The PolicyEvaluator queries `data.vault.result` once and extracts both values:
1. Evaluates the policy against the input
2. Returns the result object containing both `allow` and `reason`
3. No need for a second query

This is more efficient than the previous two-query approach (separate `allow` and `reason` queries).

## Deny Rules and Precedence

When multiple rules could match, explicit deny rules take precedence. All allow rules should check `not is_denied` to ensure suspicious or blocked processes are denied:

```rego
# In base config, define what should be denied
is_denied {
    suspicious_processes[input.process_name]
}

# Allow rules check the deny condition
result = {"allow": true, "reason": "Admin access"} {
    admin_users[input.username]
    not is_denied  # Ensures suspicious processes are blocked even for admins
}

# Explicit deny rule returns false with reason
result = {"allow": false, "reason": "Suspicious process blocked"} {
    suspicious_processes[input.process_name]
}
```

This pattern ensures security rules (denies) take precedence over access rules (allows).

## Example Results in modular-example

### 00-base.rego
- **{"allow": true, "reason": "Filesystem process internal access"}** - When safeguard.exe accesses secrets for internal operations

### 10-admin.rego
- **{"allow": true, "reason": "Admin user granted full access"}** - When admin users access any path

### 20-developer.rego
- **{"allow": true, "reason": "Developer accessing dev secrets with trusted process"}** - Dev users with trusted processes on dev paths
- **{"allow": true, "reason": "Developer accessing test secrets with trusted process"}** - Dev users with trusted processes on test paths

### 30-applications.rego
- **{"allow": true, "reason": "MyApp accessing its own secrets"}** - MyApp accessing its designated secret path
- **{"allow": true, "reason": "Automated task (PID 1234) accessing automation secrets"}** - Specific PID accessing automation secrets

### 40-public.rego
- **{"allow": true, "reason": "Public read access to documentation"}** - Any user reading public documentation

### 99-deny.rego
- **{"allow": false, "reason": "Suspicious process blocked"}** - When a process in the suspicious list attempts access

## Benefits

1. **Audit Logging**: See exactly why access was granted in logs
2. **Debugging**: Understand which rule matched for a given request
3. **Compliance**: Provide justification for access decisions
4. **Observability**: Track policy behavior in production

## Usage in Logs

When running with `-debug`, you'll see logs like:

```
Access granted to user 'OBSIDIAN\dev1' for path 'secret/dev/config' (reason: Developer accessing dev secrets with trusted process)
```

This makes it immediately clear which policy rule was applied and why.

## Best Practices

1. **Always Return Both Fields**: Each result rule should return both `allow` and `reason`
2. **Set Default**: Use `default result = {"allow": false, "reason": ""}` to handle cases where no rules match
3. **Be Specific**: Reasons should clearly explain the access decision
4. **Include Context**: Mention the user type, path pattern, or other relevant context
5. **Keep It Concise**: Reasons should be one-line summaries, not full explanations
6. **Check Denies**: All allow rules should include `not is_denied` to respect security blocks
7. **Single Evaluation**: The result object allows a single query for both decision and reason

## Default Result

If no rules match, the system uses the default: `{"allow": false, "reason": ""}`, which results in denial with reason "policy decision".
