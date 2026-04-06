# Modular policy: Public/shared access

package vault

# Anyone can read documentation (unless explicitly denied)
result = {"allow": true, "reason": "Public read access to documentation"} {
    startswith(input.path, "secret/docs")
    input.operation == "READ"
    input.username != ""  # Must be authenticated
    not is_denied
}
