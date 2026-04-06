# Modular policy: Security deny rules

package vault

# Explicit deny for suspicious processes - takes precedence
result = {"allow": false, "reason": "Suspicious process blocked"} {
    suspicious_processes[input.process_name]
}
