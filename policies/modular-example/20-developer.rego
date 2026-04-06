# Modular policy: Developer access rules

package vault

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
