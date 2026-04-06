# Platform-Specific Installation Guide

## Windows

### Prerequisites
1. **Install WinFsp**
   - Download from: https://winfsp.dev/rel/
   - Run the installer (WinFsp-*.msi)
   - Choose "Complete" installation
   - Restart your terminal after installation

2. **Verify Installation**
   ```powershell
   # Check if WinFsp is installed
   Get-ItemProperty HKLM:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\* | Where-Object {$_.DisplayName -like "*WinFsp*"}
   ```

### Build & Run
```powershell
# Build
go build -o safeguard.exe ./cmd/cli

# Run with default mount point (V:)
.\safeguard.exe -vault-addr http://127.0.0.1:8200

# Run with custom drive letter
.\safeguard.exe -mount Z: -vault-addr http://127.0.0.1:8200

# Access mounted drive
cd V:
dir
type secret\myapp\config
```

### Troubleshooting Windows Build

#### `fatal error: fuse_common.h: No such file or directory`

This happens when building with CGO under MSYS2, Git Bash, or MinGW. The `cgofuse` library needs the WinFsp FUSE headers, but the compiler doesn't know where WinFsp is installed.

**Quick fix** — set `CPATH` in your current terminal and rebuild:

```bash
# MSYS2 / Git Bash
export CPATH="C:/Program Files (x86)/WinFsp/inc/fuse"
go build -o safeguard.exe ./cmd/cli
```

```powershell
# PowerShell
$env:CPATH = "C:\Program Files (x86)\WinFsp\inc\fuse"
go build -o safeguard.exe ./cmd/cli
```

```cmd
:: Command Prompt
set CPATH=C:\Program Files (x86)\WinFsp\inc\fuse
go build -o safeguard.exe ./cmd/cli
```

**Permanent fix** — add it to your shell profile so you never hit this again:

```bash
# MSYS2 / Git Bash — append to ~/.bashrc
echo 'export CPATH="C:/Program Files (x86)/WinFsp/inc/fuse"' >> ~/.bashrc
source ~/.bashrc
```

```powershell
# PowerShell — append to $PROFILE
Add-Content $PROFILE 'Set-Item Env:CPATH "C:\Program Files (x86)\WinFsp\inc\fuse"'
```

Or set it as a system-wide environment variable via **Settings → System → Advanced → Environment Variables**.

**Why this happens:** `cgofuse` uses `#cgo windows CFLAGS: -I/usr/local/include/winfsp` which only resolves inside Docker/xgo. On a native Windows toolchain (MSYS2 GCC, MinGW, etc.) the WinFsp headers aren't on the default include path. Setting `CPATH` tells the C compiler where to find them.

**Alternative** — use `CGO_CFLAGS` if you don't want to set `CPATH` globally:

```bash
export CGO_CFLAGS="-DFUSE_USE_VERSION=28 -I\"C:/Program Files (x86)/WinFsp/inc/fuse\""
go build -o safeguard.exe ./cmd/cli
```

> **Note:** If WinFsp is installed in a non-default location, adjust the path accordingly. The headers should be at `<WinFsp>/inc/fuse/fuse_common.h`.

### Unmount
Press `Ctrl+C` in the terminal running safeguard, or:
```powershell
# Force unmount if needed
umount V:
```

---

## macOS

### Prerequisites
1. **Install macFUSE**
   ```bash
   # Using Homebrew (recommended)
   brew install --cask macfuse
   
   # Or download directly from:
   # https://github.com/osxfuse/osxfuse/releases
   ```

2. **Allow Kernel Extension**
   - Open **System Preferences** → **Security & Privacy**
   - Click the lock icon to make changes
   - Click "Allow" next to the message about "System software from developer 'Benjamin Fleischer' was blocked"
   - **Restart your Mac**

3. **Verify Installation**
   ```bash
   # Check if macFUSE is loaded
   kextstat | grep -i fuse
   
   # You should see something like:
   # com.github.osxfuse.filesystems.osxfuse
   ```

### Build & Run
```bash
# Build
go build -o safeguard ./cmd/cli

# Create mount point
mkdir -p /tmp/vault

# Run with default mount point
./safeguard -vault-addr http://127.0.0.1:8200

# Run with custom mount point
./safeguard -mount ~/vault -vault-addr http://127.0.0.1:8200

# Access mounted filesystem
cd /tmp/vault
ls -la
cat secret/myapp/config
```

### Unmount
Press `Ctrl+C` in the terminal running safeguard, or:
```bash
# Force unmount if needed
umount /tmp/vault

# Or on macOS:
diskutil unmount force /tmp/vault
```

### Troubleshooting macOS

**Problem**: "osxfuse is not installed"
```bash
# Reinstall macFUSE
brew reinstall --cask macfuse
# Then restart your Mac
```

**Problem**: "fuse: failed to open /dev/osxfuse0: Permission denied"
```bash
# Check permissions
ls -l /dev/osxfuse*

# Reset permissions
sudo kextunload -b com.github.osxfuse.filesystems.osxfuse
sudo kextload -b com.github.osxfuse.filesystems.osxfuse
```

**Problem**: "operation not permitted"
- Go to System Preferences → Security & Privacy → Privacy
- Select "Full Disk Access" and add Terminal/iTerm2

---

## Linux

### Prerequisites

#### Ubuntu / Debian
```bash
# Install FUSE
sudo apt-get update
sudo apt-get install -y fuse libfuse-dev

# Add your user to the fuse group
sudo usermod -a -G fuse $USER

# Log out and log back in for group changes to take effect
```

#### Fedora / RHEL / CentOS
```bash
# Install FUSE
sudo dnf install -y fuse fuse-devel

# Add your user to the fuse group
sudo usermod -a -G fuse $USER

# Log out and log back in
```

#### Arch Linux
```bash
# Install FUSE
sudo pacman -S fuse2

# Add your user to the fuse group
sudo usermod -a -G fuse $USER
```

#### Verify Installation
```bash
# Check if FUSE is available
which fusermount

# Check kernel module
lsmod | grep fuse

# If not loaded, load it
sudo modprobe fuse

# Check your groups
groups
# Should include 'fuse'
```

### Build & Run
```bash
# Build
go build -o safeguard ./cmd/cli

# Create mount point (choose one)
sudo mkdir -p /mnt/vault
sudo chown $USER:$USER /mnt/vault

# Or use a user directory (no sudo needed)
mkdir -p ~/vault

# Run with default mount point
./safeguard -vault-addr http://127.0.0.1:8200

# Run with custom mount point
./safeguard -mount ~/vault -vault-addr http://127.0.0.1:8200

# Access mounted filesystem (in another terminal)
cd /mnt/vault
ls -la
cat secret/myapp/config
```

### Unmount
Press `Ctrl+C` in the terminal running safeguard, or:
```bash
# Unmount using fusermount
fusermount -u /mnt/vault

# Or using umount
umount /mnt/vault

# Force unmount if stuck
sudo umount -l /mnt/vault  # lazy unmount
```

### Troubleshooting Linux

**Problem**: "fusermount: failed to open /dev/fuse: Permission denied"
```bash
# Add user to fuse group
sudo usermod -a -G fuse $USER
newgrp fuse  # or log out and back in

# Check /dev/fuse permissions
ls -l /dev/fuse
# Should be: crw-rw-rw- 1 root fuse

# Fix permissions if needed
sudo chmod 666 /dev/fuse
```

**Problem**: "fuse: device not found"
```bash
# Load FUSE kernel module
sudo modprobe fuse

# Make it persistent across reboots
echo "fuse" | sudo tee -a /etc/modules

# Check if loaded
lsmod | grep fuse
```

**Problem**: "Transport endpoint is not connected"
```bash
# The mount point is in a bad state
fusermount -uz /mnt/vault  # unmount and detach
# or
sudo umount -l /mnt/vault

# Then try mounting again
```

**Problem**: "allow_other" option not working
```bash
# Edit /etc/fuse.conf
sudo nano /etc/fuse.conf

# Uncomment this line:
# user_allow_other

# Save and try again
```

---

## Cross-Platform Usage Examples

### Quick Start (All Platforms)

```bash
# 1. Start Vault in dev mode (for testing)
vault server -dev

# 2. In another terminal, mount the filesystem
# Windows:
safeguard.exe -auth-method token -vault-token hvs.CAESI... 

# macOS:
./safeguard -auth-method token -vault-token hvs.CAESI...

# Linux:
./safeguard -auth-method token -vault-token hvs.CAESI...

# 3. Access secrets
# Windows: cd V:
# macOS/Linux: cd /tmp/vault  (or /mnt/vault)
```

### Production Usage with OIDC

All platforms use the same command:
```bash
./safeguard \
  -mount <mount-point> \
  -vault-addr https://vault.company.com \
  -auth-method oidc \
  -auth-role employee
```

Where `<mount-point>` is:
- Windows: `V:` or `W:` or any available drive letter
- macOS: `/tmp/vault` or `~/vault`
- Linux: `/mnt/vault` or `~/vault`

---

## Building for Multiple Platforms

### Cross-Compilation

```bash
# Build for Windows from Linux/Mac
GOOS=windows GOARCH=amd64 go build -o safeguard-windows.exe ./cmd/cli

# Build for macOS from Linux/Windows
GOOS=darwin GOARCH=amd64 go build -o safeguard-macos ./cmd/cli

# Build for macOS ARM (M1/M2)
GOOS=darwin GOARCH=arm64 go build -o safeguard-macos-arm64 ./cmd/cli

# Build for Linux from Windows/Mac
GOOS=linux GOARCH=amd64 go build -o safeguard-linux ./cmd/cli

# Build for Linux ARM (Raspberry Pi)
GOOS=linux GOARCH=arm64 go build -o safeguard-linux-arm64 ./cmd/cli
```

### Build Script (All Platforms)

Save as `build.sh` (Linux/macOS) or `build.ps1` (Windows):

```bash
#!/bin/bash
# build.sh

VERSION=${1:-dev}

echo "Building safeguard v${VERSION} for all platforms..."

# Windows
GOOS=windows GOARCH=amd64 go build -o "dist/safeguard-${VERSION}-windows-amd64.exe" ./cmd/cli

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o "dist/safeguard-${VERSION}-darwin-amd64" ./cmd/cli

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o "dist/safeguard-${VERSION}-darwin-arm64" ./cmd/cli

# Linux
GOOS=linux GOARCH=amd64 go build -o "dist/safeguard-${VERSION}-linux-amd64" ./cmd/cli

# Linux ARM
GOOS=linux GOARCH=arm64 go build -o "dist/safeguard-${VERSION}-linux-arm64" ./cmd/cli

echo "Build complete! Binaries in dist/"
ls -lh dist/
```

```powershell
# build.ps1 (Windows PowerShell)

param(
    [string]$Version = "dev"
)

Write-Host "Building safeguard v$Version for all platforms..."

$env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -o "dist\safeguard-$Version-windows-amd64.exe" ./cmd/cli

$env:GOOS = "darwin"; $env:GOARCH = "amd64"
go build -o "dist\safeguard-$Version-darwin-amd64" ./cmd/cli

$env:GOOS = "darwin"; $env:GOARCH = "arm64"
go build -o "dist\safeguard-$Version-darwin-arm64" ./cmd/cli

$env:GOOS = "linux"; $env:GOARCH = "amd64"
go build -o "dist\safeguard-$Version-linux-amd64" ./cmd/cli

$env:GOOS = "linux"; $env:GOARCH = "arm64"
go build -o "dist\safeguard-$Version-linux-arm64" ./cmd/cli

Write-Host "Build complete! Binaries in dist\"
Get-ChildItem dist\
```

---

## Docker Support

### Dockerfile

```dockerfile
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git fuse-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o safeguard ./cmd/cli

FROM alpine:latest

# Install FUSE runtime
RUN apk add --no-cache fuse

COPY --from=builder /app/safeguard /usr/local/bin/

# FUSE requires privileged mode
ENTRYPOINT ["/usr/local/bin/safeguard"]
```

### Docker Run

```bash
# Build
docker build -t safeguard .

# Run (requires --privileged for FUSE)
docker run -it --rm \
  --privileged \
  --cap-add SYS_ADMIN \
  --device /dev/fuse \
  safeguard \
  -mount /mnt/vault \
  -vault-addr http://vault.example.com \
  -auth-method token \
  -vault-token hvs.xxxxx
```

---

## Running at Startup

### Windows — Task Scheduler

The recommended way to run safeguard at login on Windows is via Task Scheduler.

#### Create the task (PowerShell)

```powershell
# Adjust paths and flags to match your environment
$action = New-ScheduledTaskAction `
    -Execute "C:\Program Files\safeguard\safeguard.exe" `
    -Argument "-mount V: -vault-addr https://vault.company.com -auth-method token -vault-token %VAULT_TOKEN%"

$trigger = New-ScheduledTaskTrigger -AtLogOn

$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -ExecutionTimeLimit ([TimeSpan]::Zero) `
    -RestartCount 3 `
    -RestartInterval (New-TimeSpan -Minutes 1)

Register-ScheduledTask `
    -TaskName "Safeguard" `
    -Action $action `
    -Trigger $trigger `
    -Settings $settings `
    -Description "Mount Vault secrets as a virtual drive"
```

#### Create the task (GUI)

1. Open **Task Scheduler** (`taskschd.msc`)
2. Click **Create Task…**
3. **General** tab — Name: `Safeguard`, select *Run only when user is logged on*
4. **Triggers** tab — New → *At log on*
5. **Actions** tab — New → *Start a program*
   - Program: `C:\Program Files\safeguard\safeguard.exe`
   - Arguments: `-mount V: -vault-addr https://vault.company.com -auth-method oidc`
6. **Settings** tab — Check *If the task fails, restart every 1 minute* (up to 3 times)
7. Click **OK**

#### Manage the task

```powershell
# Check status
Get-ScheduledTask -TaskName "Safeguard"

# Run now
Start-ScheduledTask -TaskName "Safeguard"

# Stop
Stop-ScheduledTask -TaskName "Safeguard"

# Remove
Unregister-ScheduledTask -TaskName "Safeguard" -Confirm:$false
```

> **Note:** If your auth method requires interactive login (e.g., OIDC browser flow), you must use *Run only when user is logged on* so the task can open a browser window. For non-interactive methods (token, approle, aws), you can select *Run whether user is logged on or not*.

---

### macOS — launchd

macOS uses **launchd** to manage startup processes. A LaunchAgent runs when the user logs in.

#### Create the plist

Save the following as `~/Library/LaunchAgents/com.safeguard.mount.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.safeguard.mount</string>

    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/safeguard</string>
        <string>-mount</string>
        <string>/tmp/vault</string>
        <string>-vault-addr</string>
        <string>https://vault.company.com</string>
        <string>-auth-method</string>
        <string>token</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>

    <key>StandardOutPath</key>
    <string>/tmp/safeguard.log</string>

    <key>StandardErrorPath</key>
    <string>/tmp/safeguard.err</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>VAULT_TOKEN</key>
        <string>hvs.your-token-here</string>
    </dict>
</dict>
</plist>
```

#### Load and manage

```bash
# Load (starts immediately and on future logins)
launchctl load ~/Library/LaunchAgents/com.safeguard.mount.plist

# Unload (stops and removes from future logins)
launchctl unload ~/Library/LaunchAgents/com.safeguard.mount.plist

# Check status
launchctl list | grep safeguard

# View logs
tail -f /tmp/safeguard.log
```

> **Note:** LaunchAgents run in the user session, so OIDC browser auth will work normally. For system-wide mounts (all users), place the plist in `/Library/LaunchDaemons/` instead — but note that daemons run as root and cannot open a browser.

---

### Linux — systemd

Create `/etc/systemd/system/safeguard.service`:

```ini
[Unit]
Description=Safeguard - Vault FUSE Filesystem
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=vault
Group=vault
ExecStart=/usr/local/bin/safeguard \
  -mount /mnt/vault \
  -vault-addr https://vault.company.com \
  -auth-method approle
ExecStop=/bin/fusermount -u /mnt/vault
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable safeguard
sudo systemctl start safeguard
sudo systemctl status safeguard
```

#### User-level service (no root required)

If you prefer to run as your own user without `sudo`, create a user service instead:

```bash
mkdir -p ~/.config/systemd/user
```

Save as `~/.config/systemd/user/safeguard.service`:

```ini
[Unit]
Description=Safeguard - Vault FUSE Filesystem
After=network-online.target

[Service]
Type=simple
ExecStart=%h/.local/bin/safeguard \
  -mount %h/vault \
  -vault-addr https://vault.company.com \
  -auth-method token
ExecStop=/bin/fusermount -u %h/vault
Restart=on-failure
RestartSec=10
Environment=VAULT_TOKEN=hvs.your-token-here

[Install]
WantedBy=default.target
```

```bash
systemctl --user daemon-reload
systemctl --user enable safeguard
systemctl --user start safeguard
systemctl --user status safeguard

# Allow the user service to run after logout
loginctl enable-linger $USER
```

> **Note:** Non-interactive auth methods (token, approle, aws) are recommended for services. If you need OIDC, authenticate once interactively and use the resulting token, or set up AppRole for automated renewal.
