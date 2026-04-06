#!/bin/bash
# Stop local Vault dev server

echo "Stopping Vault dev server..."

# Try to kill using saved PID
if [ -f /tmp/vault-dev.pid ]; then
    PID=$(cat /tmp/vault-dev.pid)
    if kill $PID 2>/dev/null; then
        echo "✓ Vault server stopped (PID: $PID)"
        rm /tmp/vault-dev.pid
    else
        echo "Process $PID not found, trying pkill..."
        pkill -f "vault server -dev" && echo "✓ Vault server stopped" || echo "No Vault processes found"
    fi
else
    pkill -f "vault server -dev" && echo "✓ Vault server stopped" || echo "No Vault processes found"
fi
