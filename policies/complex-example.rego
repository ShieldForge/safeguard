# Policy: Complex multi-factor access control
# This demonstrates combining multiple conditions for fine-grained control

package vault

import future.keywords.if
import future.keywords.in

# Default deny
default result = {"allow": false, "reason": ""}

# Define trusted processes
trusted_processes := {
    "powershell.exe",
    "cmd.exe",
    "python.exe",
    "node.exe",
}

# Define admin users
admin_users := {
    "OBSIDIAN\\admin",
    "OBSIDIAN\\root",
}

# Define developer users
developer_users := {
    "OBSIDIAN\\dev1",
    "OBSIDIAN\\dev2",
    "OBSIDIAN\\jonat",
}

# Block suspicious processes
suspicious_processes := {
    "malware.exe",
    "suspicious.exe",
    "untrusted.exe",
    "explorer.exe"
}

result = {"allow": false, "reason": "Suspicious process blocked"} {
    input.process_name in suspicious_processes
}

# Admins can access anything
result = {"allow": true, "reason": "Admin user full access"} {
    input.username in admin_users
    not input.process_name in suspicious_processes
}

# Developers can access dev and test secrets using trusted processes
result = {"allow": true, "reason": "Developer accessing dev secrets"} {
    input.username in developer_users
    input.process_name in trusted_processes
    startswith(input.path, "secret/dev")
}

result = {"allow": true, "reason": "Developer accessing test secrets"} {
    input.username in developer_users
    input.process_name in trusted_processes
    startswith(input.path, "secret/test")
}

# Production secrets require admin AND must be from trusted process
result = {"allow": true, "reason": "Admin reading prod secrets"} {
    input.username in admin_users
    input.process_name in trusted_processes
    startswith(input.path, "secret/prod")
    input.operation == "READ"  # Only allow read, not write
}

# Specific application can access its own secrets
result = {"allow": true, "reason": "MyApp accessing own secrets"} {
    glob.match("C:\\Program Files\\MyApp\\**", [], input.process_path)
    startswith(input.path, "secret/myapp")
}

# Any authenticated user can read documentation
result = {"allow": true, "reason": "Public documentation access"} {
    startswith(input.path, "secret/docs")
    input.operation == "READ"
    input.username != ""  # Must be authenticated
}

# Specific PID for automated tasks
result = {"allow": true, "reason": "Automated task (PID 1234)"} {
    input.pid == 1234
    startswith(input.path, "secret/automation")
}
