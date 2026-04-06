# Modular policy: Application-specific access

package vault

# MyApp can only access its own secrets
result = {"allow": true, "reason": "MyApp accessing its own secrets"} {
    contains(input.process_path, "Program Files\\MyApp")
    startswith(input.path, "secret/myapp")
    not is_denied
}

# Automated tasks with specific PID
result = {"allow": true, "reason": "Automated task (PID 1234) accessing automation secrets"} {
    input.pid == 1234
    startswith(input.path, "secret/automation")
    not is_denied
}
