# Example PowerShell script for building custom safeguard binaries
# This demonstrates how to automate the build process

# Configuration
$BuildServerUrl = "http://localhost:8080"

# Function to build a custom binary
function Build-CustomSafeguard {
    param(
        [string]$VaultAddr,
        [string]$AuthMethod = "oidc",
        [string]$MountPoint = "",
        [string]$Version = "1.0.0",
        [string]$BuildTag = "",
        [string]$TargetOS = "windows",
        [string]$TargetArch = "amd64"
    )
    
    Write-Host "Building custom safeguard binary..." -ForegroundColor Cyan
    
    # Build configuration
    $config = @{
        default_vault_addr  = $VaultAddr
        default_auth_method = $AuthMethod
        default_mount_point = $MountPoint
        version             = $Version
        build_tag           = $BuildTag
        target_os           = $TargetOS
        target_arch         = $TargetArch
    }
    
    # Remove empty values
    $config = $config.GetEnumerator() | Where-Object { $_.Value -ne "" } | ForEach-Object -Begin { $h = @{} } -Process { $h[$_.Key] = $_.Value } -End { $h }
    
    $json = $config | ConvertTo-Json
    
    Write-Host "Configuration:" -ForegroundColor Yellow
    Write-Host $json
    
    try {
        $response = Invoke-RestMethod -Uri "$BuildServerUrl/api/build" `
            -Method Post `
            -ContentType "application/json" `
            -Body $json `
            -ErrorAction Stop
        
        Write-Host "`n✅ Build Successful!" -ForegroundColor Green
        Write-Host "Binary: $($response.binary_path)"
        Write-Host "Size: $([math]::Round($response.size / 1MB, 2)) MB"
        Write-Host "Checksum: $($response.checksum)"
        
        return $response
    }
    catch {
        Write-Host "`n❌ Build Failed!" -ForegroundColor Red
        Write-Host "Error: $($_.Exception.Message)"
        
        if ($_.ErrorDetails.Message) {
            $errorDetails = $_.ErrorDetails.Message | ConvertFrom-Json
            Write-Host "Details: $($errorDetails.error)"
        }
        
        return $null
    }
}

# Function to download a built binary
function Download-Binary {
    param(
        [string]$FileName,
        [string]$OutputPath
    )
    
    Write-Host "Downloading $FileName..." -ForegroundColor Cyan
    
    try {
        Invoke-WebRequest -Uri "$BuildServerUrl/api/download/$FileName" `
            -OutFile $OutputPath `
            -ErrorAction Stop
        
        Write-Host "✅ Downloaded to: $OutputPath" -ForegroundColor Green
        
        # Verify file exists and show size
        $file = Get-Item $OutputPath
        Write-Host "File size: $([math]::Round($file.Length / 1MB, 2)) MB"
    }
    catch {
        Write-Host "❌ Download Failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# Function to check build environment
function Test-BuildEnvironment {
    Write-Host "Checking build environment..." -ForegroundColor Cyan
    
    try {
        $result = Invoke-RestMethod -Uri "$BuildServerUrl/api/validate" -ErrorAction Stop
        
        Write-Host "`nStatus: $($result.status)" -ForegroundColor $(if ($result.status -eq "ready") { "Green" } else { "Yellow" })
        
        if ($result.issues -and $result.issues.Count -gt 0) {
            Write-Host "`n⚠️  Issues:" -ForegroundColor Yellow
            $result.issues | ForEach-Object { Write-Host "  - $_" -ForegroundColor Red }
        }
        
        if ($result.info -and $result.info.Count -gt 0) {
            Write-Host "`nℹ️  Environment Info:" -ForegroundColor Cyan
            $result.info | ForEach-Object { Write-Host "  - $_" }
        }
        
        Write-Host "`nPaths:" -ForegroundColor Cyan
        Write-Host "  Source: $($result.source)"
        Write-Host "  Output: $($result.output)"
        
        return $result.status -eq "ready"
    }
    catch {
        Write-Host "❌ Failed to check environment: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

# Example 1: Build for a specific customer with LDAP auto-prompting
function Example-CustomerBuild {
    Write-Host "`n=== Example 1: Customer Build with LDAP Auto-Prompting ===" -ForegroundColor Magenta
    Write-Host "This build sets LDAP as the default auth method." -ForegroundColor Yellow
    Write-Host "When users run this binary, they'll be automatically prompted for LDAP credentials." -ForegroundColor Yellow
    Write-Host ""
    
    $result = Build-CustomSafeguard `
        -VaultAddr "https://vault.acme-corp.com" `
        -AuthMethod "ldap" `
        -MountPoint "V:" `
        -Version "1.0.0" `
        -BuildTag "acme-corp" `
        -TargetOS "windows" `
        -TargetArch "amd64"
    
    if ($result) {
        $fileName = Split-Path $result.binary_path -Leaf
        Download-Binary -FileName $fileName -OutputPath ".\acme-corp-safeguard.exe"
        
        Write-Host "`n💡 Usage Example:" -ForegroundColor Cyan
        Write-Host "  .\acme-corp-safeguard.exe" -ForegroundColor White
        Write-Host "  # Will prompt: 'LDAP Username: ' and 'LDAP Password: '" -ForegroundColor Gray
        Write-Host ""
        Write-Host "  .\acme-corp-safeguard.exe -ldap-username john -ldap-password secret" -ForegroundColor White
        Write-Host "  # No prompting when credentials are provided" -ForegroundColor Gray
    }
}

# Example 2: Build for multiple platforms
function Example-MultiPlatformBuild {
    Write-Host "`n=== Example 2: Multi-Platform Build ===" -ForegroundColor Magenta
    
    $platforms = @(
        @{ OS = "windows"; Arch = "amd64"; Ext = ".exe" }
        @{ OS = "linux"; Arch = "amd64"; Ext = "" }
        @{ OS = "darwin"; Arch = "arm64"; Ext = "" }
    )
    
    foreach ($platform in $platforms) {
        Write-Host "`nBuilding for $($platform.OS)/$($platform.Arch)..." -ForegroundColor Yellow
        
        $result = Build-CustomSafeguard `
            -VaultAddr "https://vault.company.com" `
            -AuthMethod "oidc" `
            -Version "1.0.0" `
            -BuildTag "multiplatform" `
            -TargetOS $platform.OS `
            -TargetArch $platform.Arch
        
        if ($result) {
            $fileName = Split-Path $result.binary_path -Leaf
            $outputName = "safeguard-$($platform.OS)-$($platform.Arch)$($platform.Ext)"
            Download-Binary -FileName $fileName -OutputPath ".\$outputName"
        }
        
        Start-Sleep -Seconds 2
    }
}

# Example 3: Build with full configuration
function Example-FullConfigBuild {
    Write-Host "`n=== Example 3: Full Configuration Build ===" -ForegroundColor Magenta
    
    $result = Build-CustomSafeguard `
        -VaultAddr "https://vault.enterprise.com" `
        -AuthMethod "approle" `
        -MountPoint "Z:" `
        -Version "2.0.0" `
        -BuildTag "enterprise-locked" `
        -TargetOS "windows" `
        -TargetArch "amd64"
    
    if ($result) {
        Write-Host "`n📋 Build Summary:" -ForegroundColor Cyan
        Write-Host "  Version: $($result.config.version)"
        Write-Host "  Tag: $($result.config.build_tag)"
        Write-Host "  Platform: $($result.config.target_os)/$($result.config.target_arch)"
        Write-Host "  Vault: $($result.config.default_vault_addr)"
        Write-Host "  Auth: $($result.config.default_auth_method)"
    }
}

# Main menu
function Show-Menu {
    Write-Host "`n" -NoNewline
    Write-Host "╔════════════════════════════════════════╗" -ForegroundColor Cyan
    Write-Host "║  safeguard Build Automation Script   ║" -ForegroundColor Cyan
    Write-Host "╚════════════════════════════════════════╝" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "1. Check build environment"
    Write-Host "2. Run Example 1: LDAP build with auto-prompting"
    Write-Host "3. Run Example 2: Multi-platform build"
    Write-Host "4. Run Example 3: Full configuration build"
    Write-Host "5. Custom build (interactive)"
    Write-Host "Q. Quit"
    Write-Host ""
}

# Interactive custom build
function Start-InteractiveBuild {
    Write-Host "`n=== Interactive Custom Build ===" -ForegroundColor Magenta
    
    $vaultAddr = Read-Host "Vault Address (e.g., https://vault.company.com)"
    $authMethod = Read-Host "Auth Method (oidc/ldap/token/aws/approle) [oidc]"
    if ([string]::IsNullOrWhiteSpace($authMethod)) { $authMethod = "oidc" }
    
    $mountPoint = Read-Host "Mount Point (e.g., V: or /mnt/vault) [auto]"
    $version = Read-Host "Version [1.0.0]"
    if ([string]::IsNullOrWhiteSpace($version)) { $version = "1.0.0" }
    
    $buildTag = Read-Host "Build Tag (e.g., customer-name)"
    $targetOS = Read-Host "Target OS (windows/linux/darwin) [windows]"
    if ([string]::IsNullOrWhiteSpace($targetOS)) { $targetOS = "windows" }
    
    $targetArch = Read-Host "Target Arch (amd64/arm64/386) [amd64]"
    if ([string]::IsNullOrWhiteSpace($targetArch)) { $targetArch = "amd64" }
    
    $result = Build-CustomSafeguard `
        -VaultAddr $vaultAddr `
        -AuthMethod $authMethod `
        -MountPoint $mountPoint `
        -Version $version `
        -BuildTag $buildTag `
        -TargetOS $targetOS `
        -TargetArch $targetArch
    
    if ($result) {
        $download = Read-Host "`nDownload binary? (Y/n)"
        if ($download -ne "n") {
            $fileName = Split-Path $result.binary_path -Leaf
            $outputPath = Read-Host "Output path [$fileName]"
            if ([string]::IsNullOrWhiteSpace($outputPath)) { $outputPath = $fileName }
            Download-Binary -FileName $fileName -OutputPath $outputPath
        }
    }
}

# Main script
Write-Host "safeguard Build Server URL: $BuildServerUrl" -ForegroundColor Cyan

# Check if server is running
try {
    Invoke-RestMethod -Uri "$BuildServerUrl/api/health" -TimeoutSec 2 | Out-Null
    Write-Host "✅ Build server is running" -ForegroundColor Green
}
catch {
    Write-Host "❌ Build server is not accessible at $BuildServerUrl" -ForegroundColor Red
    Write-Host "Please start the build server first:" -ForegroundColor Yellow
    Write-Host "  cd cmd\builder" -ForegroundColor Yellow
    Write-Host "  go run main.go -source ../.. -output ./builds" -ForegroundColor Yellow
    exit 1
}

# Interactive menu loop
do {
    Show-Menu
    $choice = Read-Host "Select an option"
    
    switch ($choice) {
        "1" { Test-BuildEnvironment }
        "2" { Example-CustomerBuild }
        "3" { Example-MultiPlatformBuild }
        "4" { Example-FullConfigBuild }
        "5" { Start-InteractiveBuild }
        "Q" { Write-Host "Goodbye!" -ForegroundColor Cyan; break }
        default { Write-Host "Invalid option" -ForegroundColor Red }
    }
    
    if ($choice -ne "Q") {
        Write-Host "`nPress Enter to continue..." -NoNewline
        Read-Host
    }
} while ($choice -ne "Q")
