# Policy: Allow specific processes by name or path
# This policy allows access only to specified processes

package vault

# Default deny
default result = {"allow": false, "reason": ""}

# Allow PowerShell
result = {"allow": true, "reason": "PowerShell process"} {
    input.process_name == "powershell.exe"
}

# Allow Windows Command Prompt
result = {"allow": true, "reason": "Command prompt process"} {
    input.process_name == "cmd.exe"
}

# Allow specific application by path
result = {"allow": true, "reason": "MyApp application"} {
    glob.match("C:\\Program Files\\MyApp\\**", [], input.process_path)
}
