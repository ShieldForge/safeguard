# Policy: Path-based access control
# This policy allows different users/processes access to different secrets

package vault

# Default deny
default result = {"allow": false, "reason": ""}

# Allow PowerShell to read production secrets
result = {"allow": true, "reason": "PowerShell reading prod secrets"} {
    input.process_name == "powershell.exe"
    startswith(input.path, "secret/prod")
    input.operation == "READ"
}

# Allow specific app to access its configuration
result = {"allow": true, "reason": "MyApp accessing own config"} {
    glob.match("C:\\Program Files\\MyApp\\**", [], input.process_path)
    startswith(input.path, "secret/myapp")
}

# Allow admin users to access everything
result = {"allow": true, "reason": "Admin user full access"} {
    input.username == "OBSIDIAN\\admin"
}

# Allow specific user to only read (no write) dev secrets
result = {"allow": true, "reason": "Developer reading dev secrets"} {
    input.username == "OBSIDIAN\\developer"
    startswith(input.path, "secret/dev")
    input.operation != "WRITE"
}

# Allow read-only access to docs for everyone
result = {"allow": true, "reason": "Public docs access"} {
    startswith(input.path, "secret/docs")
    input.operation == "READ"
}
