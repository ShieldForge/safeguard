# Modular policy: Admin access rules

package vault

# Admins can access everything (unless explicitly denied)
result = {"allow": true, "reason": "Admin user granted full access"} {
    admin_users[input.username]
    not is_denied
}
