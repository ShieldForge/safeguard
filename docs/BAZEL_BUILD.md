# Building with Bazel

This guide explains how to build safeguard using [Bazel](https://bazel.build/), Google's build system.

## Why Bazel?

Bazel offers several advantages:

- **Fast incremental builds** - Only rebuilds what changed
- **Reproducible builds** - Same inputs always produce same outputs
- **Hermetic builds** - Isolated from system dependencies
- **Cross-platform** - Consistent builds across Windows, Linux, and macOS
- **Built-in caching** - Local and remote build caching support
- **Parallel execution** - Maximizes build performance

## Prerequisites

### Install Bazel

**Windows:**
```powershell
# Using Chocolatey
choco install bazel

# Or download from https://github.com/bazelbuild/bazel/releases
```

**macOS:**
```bash
# Using Homebrew
brew install bazel

# Or using Bazelisk (version manager)
brew install bazelisk
```

**Linux:**
```bash
# Ubuntu/Debian
sudo apt install apt-transport-https curl gnupg
curl -fsSL https://bazel.build/bazel-release.pub.gpg | gpg --dearmor > bazel.gpg
sudo mv bazel.gpg /etc/apt/trusted.gpg.d/
echo "deb [arch=amd64] https://storage.googleapis.com/bazel-apt stable jdk1.8" | sudo tee /etc/apt/sources.list.d/bazel.list
sudo apt update && sudo apt install bazel

# Or download from https://github.com/bazelbuild/bazel/releases
```

### Verify Installation

```bash
bazel --version
# Should show: bazel 7.x.x or similar
```

## Building

### Basic Build

Build the main binary:

```bash
# Build the safeguard binary
bazel build //:safeguard

# The binary will be in: bazel-bin/safeguard_/safeguard
# (Windows: bazel-bin/safeguard_/safeguard.exe)
```

### Build Aliases

For convenience, you can use the build alias:

```bash
bazel build //:build
```

### Run Directly

Build and run in one command:

```bash
bazel run //:safeguard -- -help

# With arguments
bazel run //:safeguard -- -auth-method token -vault-token root -debug
```

### Build Configurations

**Fast build (default for development):**
```bash
bazel build //:safeguard --config=fast
```

**Optimized build (for production):**
```bash
bazel build //:safeguard --config=release
```

**Debug build (with debug symbols):**
```bash
bazel build //:safeguard --config=debug
```

**Verbose output (for troubleshooting):**
```bash
bazel build //:safeguard --config=verbose
```

## Testing

### Run All Tests

```bash
bazel test //...
```

### Run Specific Package Tests

```bash
# Test filesystem package
bazel test //pkg/filesystem:filesystem_test

# Test vault client
bazel test //pkg/vault:vault_test

# Test authentication
bazel test //pkg/auth:auth_test
```

### Test with Coverage

```bash
bazel coverage //...

# View coverage report
genhtml bazel-out/_coverage/_coverage_report.dat -o coverage_html
```

### Test Output Options

```bash
# Show all test output
bazel test //... --test_output=all

# Show only errors
bazel test //... --test_output=errors

# Show detailed test information
bazel test //... --test_output=streamed
```

## Building Specific Packages

### Build Individual Libraries

```bash
# Build logger package
bazel build //pkg/logger:logger

# Build vault client
bazel build //pkg/vault:vault

# Build filesystem
bazel build //pkg/filesystem:filesystem

# Build auth
bazel build //pkg/auth:auth
```

## Dependency Management

### Update Dependencies from go.mod

Bazel can automatically generate dependency rules from your `go.mod`:

```bash
# Update external Go dependencies
bazel run //:gazelle-update-repos
```

### Regenerate BUILD Files

If you add new Go files or change package structure:

```bash
# Regenerate BUILD.bazel files
bazel run //:gazelle
```

## Clean Build Artifacts

```bash
# Clean build outputs
bazel clean

# Deep clean (removes all cached artifacts)
bazel clean --expunge
```

## Build Outputs

Bazel creates several symlinks in your workspace:

- `bazel-bin/` - Contains build outputs (binaries, libraries)
- `bazel-out/` - Intermediate build artifacts
- `bazel-testlogs/` - Test results and logs
- `bazel-safeguard/` - Symlink to the actual workspace

The actual binary will be at:
- Linux/macOS: `bazel-bin/safeguard_/safeguard`
- Windows: `bazel-bin/safeguard_/safeguard.exe`

## Cross-Compilation

Bazel supports cross-compilation for different platforms:

```bash
# Build for Linux from any platform
bazel build //:safeguard --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64

# Build for Windows from any platform
bazel build //:safeguard --platforms=@io_bazel_rules_go//go/toolchain:windows_amd64

# Build for macOS from any platform
bazel build //:safeguard --platforms=@io_bazel_rules_go//go/toolchain:darwin_amd64
```

## Bazel Cache

### Local Cache

Bazel automatically caches build artifacts locally. To clear the cache:

```bash
bazel clean --expunge
```

### Remote Cache (Optional)

For teams, you can configure remote caching:

```bash
# Add to .bazelrc.user (not committed)
build --remote_cache=https://your-cache-server.com
build --remote_upload_local_results=true
```

## Troubleshooting

### Build Fails with Missing Dependencies

```bash
# Fetch all dependencies
bazel fetch //...

# Update repositories
bazel sync
```

### Slow First Build

The first build downloads all dependencies and toolchains. Subsequent builds are much faster due to caching.

### CGo Issues on Windows

If you encounter CGo compilation errors on Windows:

1. Install [Build Tools for Visual Studio](https://visualstudio.microsoft.com/downloads/#build-tools-for-visual-studio-2022)
2. Ensure MSVC is in your PATH
3. Set environment variables:
```powershell
$env:BAZEL_VC = "C:\Program Files\Microsoft Visual Studio\2022\BuildTools\VC"
```

### Permission Errors

On Windows, run PowerShell as Administrator if you encounter permission errors.

## IDE Integration

### VS Code

Install the Bazel extension:
```
ext install BazelBuild.vscode-bazel
```

Then use Bazel: Build Target from the command palette.

### IntelliJ/GoLand

Install the Bazel plugin:
1. Go to Settings → Plugins
2. Search for "Bazel"
3. Install and restart

Import the project as a Bazel project.

## Comparing with Go Build

| Feature | Bazel | Go Build |
|---------|-------|----------|
| **First Build** | Slower (downloads toolchain) | Faster |
| **Incremental Build** | Much faster | Fast |
| **Reproducibility** | Guaranteed | Varies |
| **Cross-compilation** | Built-in | Manual flags |
| **Caching** | Advanced (local + remote) | Basic |
| **Complexity** | Higher | Lower |

## Bazel Configuration Files

- `WORKSPACE` - Defines external dependencies and workspace settings
- `BUILD.bazel` - Build rules for each package
- `.bazelrc` - Build configuration options
- `deps.bzl` - External Go dependencies

## Best Practices

1. **Use Gazelle** - Automatically generate BUILD files instead of writing them manually
2. **Keep BUILD files simple** - Let Gazelle handle most of the configuration
3. **Use configs** - Define build configurations in `.bazelrc` for consistency
4. **Cache remote** - Set up remote caching for teams
5. **Run tests frequently** - Bazel makes testing fast with smart caching

## Migration from Go Build

You can use both build systems in parallel:

```bash
# Go build (traditional)
go build -o safeguard ./cmd/cli

# Bazel build (new)
bazel build //:safeguard
```

Both produce functionally identical binaries.

## Performance Tips

1. **Use fast config for development:**
   ```bash
   bazel build //:safeguard --config=fast
   ```

2. **Parallel execution:**
   ```bash
   bazel build //... --jobs=8
   ```

3. **Incremental builds:**
   Only changed files and their dependents are rebuilt automatically

4. **Disk cache:**
   Bazel maintains a disk cache across builds for maximum speed

## Additional Resources

- [Bazel Documentation](https://bazel.build/docs)
- [rules_go Documentation](https://github.com/bazelbuild/rules_go)
- [Gazelle Documentation](https://github.com/bazelbuild/bazel-gazelle)
- [Bazel Go Tutorial](https://bazel.build/tutorials/go)

## Getting Help

If you encounter issues with Bazel builds:

1. Check the error message carefully
2. Try `bazel clean` and rebuild
3. Verify Bazel version: `bazel --version`
4. Check the troubleshooting section above
5. Consult the [Bazel documentation](https://bazel.build/docs)
