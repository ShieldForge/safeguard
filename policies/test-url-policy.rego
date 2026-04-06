package vault

# Example policy for testing URL-based policy loading
# This policy allows read access to paths starting with "secret/"

default result = {
    "allow": false,
    "reason": "default deny"
}

result = {
    "allow": true,
    "reason": "allowed to read secret paths"
} {
    input.operation == "READ"
    startswith(input.path, "secret/")
}

result = {
    "allow": true,
    "reason": "allowed to list secret paths"
} {
    input.operation == "READDIR"
    startswith(input.path, "secret/")
}
