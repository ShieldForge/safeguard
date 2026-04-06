# Modular Policy Example

This directory demonstrates how to organize policies into multiple files for better maintainability.

> **Important**: These policies only apply to external processes accessing the mounted filesystem. The safeguard process itself is automatically allowed - this is necessary for mounting and filesystem operations.

## Structure

- **00-base.rego** - Defines shared data (sets of users, processes) and default deny
- **10-admin.rego** - Admin access rules
- **20-developer.rego** - Developer access rules
- **30-applications.rego** - Application-specific access rules
- **40-public.rego** - Public/shared resource access
- **99-deny.rego** - Security deny rules (checked last)

## Benefits of Modular Policies

1. **Organization**: Each file handles a specific aspect of access control
2. **Maintainability**: Easy to update one area without affecting others
3. **Readability**: Smaller files are easier to understand
4. **Collaboration**: Team members can work on different policy files
5. **Reusability**: Shared definitions (like user sets) defined once

## Usage

Point to the directory instead of a single file:

```bash
safeguard.exe -mount V: -auth-method token -vault-token hvs.xxx \
  -access-control -policy-path policies/modular-example
```

## How It Works

All `.rego` files in the directory are loaded and combined into a single policy. They all share the same `package vault` namespace, so:

- Sets defined in `00-base.rego` (like `admin_users`) are available to all other files
- Multiple `allow` rules across files are OR'd together (if any is true, access is granted)
- The `deny` rules in `99-deny.rego` can override any allow rules

## Naming Convention

Files are numbered to make the load order clear:
- `00-*` - Base configuration
- `10-99` - Specific access rules
- `99-*` - Deny rules (checked last)

While OPA doesn't require files to be loaded in order, this makes it easier for humans to understand the policy structure.

## Example: Adding a New Team

To add access for a new team, create a new file:

**25-qa-team.rego**:
```rego
package vault

qa_users := {
    "OBSIDIAN\\qa1",
    "OBSIDIAN\\qa2",
}

# QA can access test and staging secrets
allow {
    input.username in qa_users
    startswith(input.path, "secret/test")
}

allow {
    input.username in qa_users
    startswith(input.path, "secret/staging")
}
```

Just add the file to the directory - no need to modify existing policies!

## Testing

Validate all policies in the directory:

```bash
# The application will validate all .rego files at startup
safeguard.exe -access-control -policy-path policies/modular-example

# Output: "Policy-based access control enabled: policies/modular-example"
```

## Comparison: Single File vs Directory

**Single File Approach:**
```bash
-policy-path policies/complex-example.rego
```
- One large file with all rules
- Harder to navigate
- Harder to collaborate on

**Directory Approach:**
```bash
-policy-path policies/modular-example
```
- Multiple focused files
- Easy to understand and maintain
- Better for teams

Both approaches are fully supported!
