# Policy: Allow specific users
# This policy allows access only to specified users

package vault

# Default deny
default result = {"allow": false, "reason": ""}

# Allow specific usernames
result = {"allow": true, "reason": "User jonat"} {
    input.username == "OBSIDIAN\\jonat"
}

result = {"allow": true, "reason": "Admin user"} {
    input.username == "OBSIDIAN\\admin"
}

# Allow any user from a specific domain
result = {"allow": true, "reason": "OBSIDIAN domain user"} {
    startswith(input.username, "OBSIDIAN\\")
}
