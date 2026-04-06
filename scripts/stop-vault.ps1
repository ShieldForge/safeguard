#!/usr/bin/env pwsh
# Stop local Vault dev server

Write-Host "Stopping Vault dev server..." -ForegroundColor Yellow

$vaultProcesses = Get-Process -Name vault -ErrorAction SilentlyContinue

if ($vaultProcesses) {
    $vaultProcesses | Stop-Process -Force
    Write-Host "✓ Vault server stopped" -ForegroundColor Green
} else {
    Write-Host "No Vault processes found" -ForegroundColor Yellow
}
