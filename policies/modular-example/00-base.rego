# Modular policy: Base configuration and defaults

package vault

# Default result
default result = {"allow": false, "reason": ""}

# Define sets of trusted entities for reuse across policies
trusted_processes := {
    "powershell.exe",
    "cmd.exe",
    "python.exe",
    "more.com",
    "safeguard.exe",  # Allow the filesystem process itself
}

admin_users := {
    "OBSIDIAN\\admin",
    "OBSIDIAN\\root",
}

developer_users := {
    "OBSIDIAN\\dev1",
    "OBSIDIAN\\dev2",
    "OBSIDIAN\\jonat",
}

# Define suspicious processes (checked by deny rules)
suspicious_processes := {
    "malware.exe",
    "suspicious.exe",
    "untrusted.exe",
    "explorer.exe",
    "notepad.exe"
}

# Allow the filesystem process itself to access all secrets (required for internal operations)
result = {"allow": true, "reason": "Filesystem process internal access"} {
    input.process_name == "safeguard.exe"
    not is_denied
}

# Helper to check if access should be denied
is_denied {
    suspicious_processes[input.process_name]
}

# Developers can list directories
result = {"allow": true, "reason": "Developer listing directories"} {
    developer_users[input.username]
    trusted_processes[input.process_name]
    input.operation == "READ"
    not is_denied
}

# Developers can access dev secrets using trusted processes
result = {"allow": true, "reason": "Developer accessing dev secrets with trusted process"} {
    developer_users[input.username]
    trusted_processes[input.process_name]
    not is_denied
}
