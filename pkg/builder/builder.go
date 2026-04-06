package builder

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BuildConfig holds the configuration for building a custom binary
type BuildConfig struct {
	// Build variables to embed
	DefaultVaultAddr     string `json:"default_vault_addr"`
	DefaultAuthMethod    string `json:"default_auth_method"`
	DefaultVaultProvider string `json:"default_vault_provider"`
	DefaultMountPoint    string `json:"default_mount_point"`
	DefaultAuthRole      string `json:"default_auth_role"`
	DefaultAuthMount     string `json:"default_auth_mount"`
	DefaultPolicyPath    string `json:"default_policy_path"`
	DefaultMappingPath   string `json:"default_mapping_config"`
	DefaultAuditLog      string `json:"default_audit_log"`
	DefaultAllowedPIDs   string `json:"default_allowed_pids"`
	DefaultAllowedUIDs   string `json:"default_allowed_uids"`
	DefaultDebug         bool   `json:"default_debug"`
	DefaultMonitor       bool   `json:"default_monitor"`
	DefaultAccessControl bool   `json:"default_access_control"`
	DisableCliFlags      bool   `json:"disable_cli_flags"`

	// Logging configuration
	DefaultLogFile       string `json:"default_log_file"`
	DefaultLogMaxSize    string `json:"default_log_max_size"`
	DefaultLogMaxBackups string `json:"default_log_max_backups"`
	DefaultLogMaxAge     string `json:"default_log_max_age"`
	DefaultLogCompress   string `json:"default_log_compress"`

	// Cache configuration
	DefaultCacheEnabled string `json:"default_cache_enabled"`
	DefaultCacheTTL     string `json:"default_cache_ttl"`

	// Policy embedding options
	EmbedPolicyFromURL bool `json:"embed_policy_from_url"` // If true and policy path is URL, download and embed
	EmbedPolicyFiles   bool `json:"embed_policy_files"`    // If true, embed policy files into binary as resources

	// Authentication credentials (optional - for testing only)
	DefaultLdapUsername string `json:"default_ldap_username"`
	DefaultLdapPassword string `json:"default_ldap_password"`
	DefaultVaultToken   string `json:"default_vault_token"`

	// Build metadata
	Version  string `json:"version"`
	BuildTag string `json:"build_tag"`

	// Target platform
	TargetOS   string `json:"target_os"`   // linux, windows, darwin
	TargetArch string `json:"target_arch"` // amd64, arm64, 386

	// Directory overrides (optional — server defaults used if empty)
	SourceDir string `json:"source_dir"`
	WorkDir   string `json:"work_dir"`
	OutputDir string `json:"output_dir"`

	// Output filename override (optional — auto-generated if empty)
	OutputFilename string `json:"output_filename"`
}

// BuildResult contains information about the completed build
type BuildResult struct {
	BinaryPath string      `json:"binary_path"`
	Checksum   string      `json:"checksum"`
	Size       int64       `json:"size"`
	BuildTime  time.Time   `json:"build_time"`
	Config     BuildConfig `json:"config"`
}

// Builder handles the compilation of custom binaries
type Builder struct {
	workDir   string
	sourceDir string
	outputDir string
}

// NewBuilder creates a new Builder instance
func NewBuilder(sourceDir, workDir, outputDir string) (*Builder, error) {
	// Create work and output directories if they don't exist
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &Builder{
		workDir:   workDir,
		sourceDir: sourceDir,
		outputDir: outputDir,
	}, nil
}

// Build compiles a custom binary with the specified configuration
func (b *Builder) Build(config BuildConfig) (*BuildResult, error) {
	return b.BuildWithLog(config, nil)
}

// BuildWithLog compiles a custom binary, streaming progress lines to logWriter if non-nil.
func (b *Builder) BuildWithLog(config BuildConfig, logWriter io.Writer) (*BuildResult, error) {
	logf := func(format string, args ...interface{}) {
		if logWriter != nil {
			fmt.Fprintf(logWriter, format+"\n", args...)
		}
	}

	startTime := time.Now()
	logf("Build started at %s", startTime.Format(time.RFC3339))

	// Set defaults
	if config.TargetOS == "" {
		config.TargetOS = runtime.GOOS
	}
	if config.TargetArch == "" {
		config.TargetArch = runtime.GOARCH
	}
	if config.Version == "" {
		config.Version = "custom"
	}

	logf("Target: %s/%s, Version: %s", config.TargetOS, config.TargetArch, config.Version)

	// Handle policy URL embedding
	var policyCleanup func()
	if config.EmbedPolicyFromURL && config.DefaultPolicyPath != "" && isURL(config.DefaultPolicyPath) {
		logf("Downloading policy from URL: %s", config.DefaultPolicyPath)
		// Download policy and create temporary file
		tempPolicyPath, cleanup, err := b.downloadAndSavePolicy(config.DefaultPolicyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to download policy from URL: %w", err)
		}
		policyCleanup = cleanup
		defer policyCleanup()

		// Update config to use the temporary file path
		config.DefaultPolicyPath = tempPolicyPath
	}

	// Embed policy files into the binary as Go source
	var genFileCleanup func()
	if config.EmbedPolicyFiles && config.DefaultPolicyPath != "" && !isURL(config.DefaultPolicyPath) {
		logf("Embedding policy files from: %s", config.DefaultPolicyPath)
		cleanup, count, err := b.generateEmbeddedPolicies(config.DefaultPolicyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to embed policy files: %w", err)
		}
		genFileCleanup = cleanup
		defer genFileCleanup()
		logf("Embedded %d policy file(s)", count)
		// Clear the policy path ldflags — the binary will use embedded policies instead
		config.DefaultPolicyPath = ""
	}

	// Generate output filename
	var binaryName string
	if config.OutputFilename != "" {
		binaryName = config.OutputFilename
		if config.TargetOS == "windows" && !strings.HasSuffix(binaryName, ".exe") {
			binaryName += ".exe"
		}
	} else {
		prefix := "safeguard"
		binaryName = fmt.Sprintf("%s-%s-%s-%s", prefix, config.Version, config.TargetOS, config.TargetArch)
		if config.BuildTag != "" {
			binaryName += "-" + config.BuildTag
		}
		if config.TargetOS == "windows" {
			binaryName += ".exe"
		}
	}

	outputPath := filepath.Join(b.outputDir, binaryName)

	logf("Output binary: %s", binaryName)

	// Build ldflags for embedding variables
	ldflags := b.buildLdflags(config)

	logf("Preparing go build command…")
	logf("Source directory: %s", b.sourceDir)
	logf("GOOS=%s GOARCH=%s CGO_ENABLED=1", config.TargetOS, config.TargetArch)

	// Determine the build target.
	//
	// When sourceDir is the project root (contains go.mod) we use the full
	// package path; otherwise assume we're already inside the package.
	buildTarget := "."
	cgoEnabled := "1"
	if _, err := os.Stat(filepath.Join(b.sourceDir, "go.mod")); err == nil {
		buildTarget = "./cmd/cli"
	}

	logf("CGO_ENABLED=%s", cgoEnabled)

	// Prepare go build command
	cmd := exec.Command("go", "build",
		"-o", outputPath,
		"-ldflags", ldflags,
		buildTarget,
	)

	// Set environment for cross-compilation
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOOS=%s", config.TargetOS),
		fmt.Sprintf("GOARCH=%s", config.TargetArch),
		"CGO_ENABLED="+cgoEnabled,
	)

	// Set working directory to source
	cmd.Dir = b.sourceDir

	// Capture output — also stream stderr to logWriter for real-time feedback
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	if logWriter != nil {
		cmd.Stderr = io.MultiWriter(&stderr, logWriter)
	} else {
		cmd.Stderr = &stderr
	}

	logf("Compiling… this may take a minute…")

	// Run build
	if err := cmd.Run(); err != nil {
		logf("Build FAILED: %v", err)
		return nil, fmt.Errorf("build failed: %w\nstdout: %s\nstderr: %s",
			err, stdout.String(), stderr.String())
	}

	elapsed := time.Since(startTime).Round(time.Millisecond)
	logf("Compilation complete in %s", elapsed)

	// Calculate checksum
	logf("Calculating SHA256 checksum…")
	checksum, err := calculateChecksum(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Get file size
	stat, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat binary: %w", err)
	}

	logf("Build successful — %s (%.2f MB)", binaryName, float64(stat.Size())/1024/1024)

	return &BuildResult{
		BinaryPath: outputPath,
		Checksum:   checksum,
		Size:       stat.Size(),
		BuildTime:  startTime,
		Config:     config,
	}, nil
}

// buildLdflags constructs the ldflags string with embedded variables
func (b *Builder) buildLdflags(config BuildConfig) string {
	var flags []string

	// Add build-time variable overrides
	// These will replace the default values in the flag definitions
	if config.DefaultVaultAddr != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultVaultAddr=%s'", config.DefaultVaultAddr))
	}
	if config.DefaultAuthMethod != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultAuthMethod=%s'", config.DefaultAuthMethod))
	}
	if config.DefaultVaultProvider != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultVaultProvider=%s'", config.DefaultVaultProvider))
	}
	if config.DefaultMountPoint != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultMountPoint=%s'", config.DefaultMountPoint))
	}
	if config.DefaultAuthRole != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultAuthRole=%s'", config.DefaultAuthRole))
	}
	if config.DefaultAuthMount != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultAuthMount=%s'", config.DefaultAuthMount))
	}
	if config.DefaultPolicyPath != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultPolicyPath=%s'", config.DefaultPolicyPath))
	}
	if config.DefaultMappingPath != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultMappingPath=%s'", config.DefaultMappingPath))
	}
	if config.DefaultAuditLog != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultAuditLog=%s'", config.DefaultAuditLog))
	}
	if config.DefaultAllowedPIDs != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultAllowedPIDs=%s'", config.DefaultAllowedPIDs))
	}
	if config.DefaultAllowedUIDs != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultAllowedUIDs=%s'", config.DefaultAllowedUIDs))
	}
	if config.DefaultDebug {
		flags = append(flags, "-X 'main.defaultDebug=true'")
	}
	if config.DefaultMonitor {
		flags = append(flags, "-X 'main.defaultMonitor=true'")
	}
	if config.DefaultAccessControl {
		flags = append(flags, "-X 'main.defaultAccessControl=true'")
	}
	// disableCliFlags is a CLI-only variable.
	if config.DisableCliFlags {
		flags = append(flags, "-X 'main.disableCliFlags=true'")
	}

	if config.Version != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.version=%s'", config.Version))
	}
	if config.BuildTag != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.buildTag=%s'", config.BuildTag))
	}

	// Add credential overrides (optional - use with caution)
	if config.DefaultLdapUsername != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultLdapUsername=%s'", config.DefaultLdapUsername))
	}
	if config.DefaultLdapPassword != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultLdapPassword=%s'", config.DefaultLdapPassword))
	}
	if config.DefaultVaultToken != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultVaultToken=%s'", config.DefaultVaultToken))
	}

	// Log configuration overrides
	if config.DefaultLogFile != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultLogFile=%s'", config.DefaultLogFile))
	}
	if config.DefaultLogMaxSize != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultLogMaxSize=%s'", config.DefaultLogMaxSize))
	}
	if config.DefaultLogMaxBackups != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultLogMaxBackups=%s'", config.DefaultLogMaxBackups))
	}
	if config.DefaultLogMaxAge != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultLogMaxAge=%s'", config.DefaultLogMaxAge))
	}
	if config.DefaultLogCompress != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultLogCompress=%s'", config.DefaultLogCompress))
	}
	if config.DefaultCacheEnabled != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultCacheEnabled=%s'", config.DefaultCacheEnabled))
	}
	if config.DefaultCacheTTL != "" {
		flags = append(flags, fmt.Sprintf("-X 'main.defaultCacheTTL=%s'", config.DefaultCacheTTL))
	}

	// Add standard build flags for smaller binaries
	flags = append(flags, "-s", "-w") // Strip debug info

	return strings.Join(flags, " ")
}

// CreateArchive creates a tar.gz or zip archive of the binary
func (b *Builder) CreateArchive(result *BuildResult) (string, error) {
	archiveName := strings.TrimSuffix(filepath.Base(result.BinaryPath), filepath.Ext(result.BinaryPath))

	var archivePath string
	var err error

	if result.Config.TargetOS == "windows" {
		archivePath = filepath.Join(b.outputDir, archiveName+".zip")
		err = createZipArchive(result.BinaryPath, archivePath)
	} else {
		archivePath = filepath.Join(b.outputDir, archiveName+".tar.gz")
		err = createTarGzArchive(result.BinaryPath, archivePath)
	}

	if err != nil {
		return "", fmt.Errorf("failed to create archive: %w", err)
	}

	return archivePath, nil
}

// calculateChecksum calculates SHA256 checksum of a file
func calculateChecksum(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// createTarGzArchive creates a tar.gz archive
func createTarGzArchive(srcPath, dstPath string) error {
	outFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    filepath.Base(srcPath),
		Size:    stat.Size(),
		Mode:    int64(stat.Mode()),
		ModTime: stat.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

// createZipArchive creates a zip archive
func createZipArchive(srcPath, dstPath string) error {
	outFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	zw := zip.NewWriter(outFile)
	defer zw.Close()

	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(stat)
	if err != nil {
		return err
	}
	header.Name = filepath.Base(srcPath)
	header.Method = zip.Deflate

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// isURL checks if a path is an HTTP or HTTPS URL
func isURL(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// downloadAndSavePolicy downloads a policy from URL and saves it to a temporary file
func (b *Builder) downloadAndSavePolicy(url string) (string, func(), error) {
	// Download the policy
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch policy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read policy content: %w", err)
	}

	// Create temporary directory for policy
	tempDir := filepath.Join(b.workDir, "policies")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create temp policy dir: %w", err)
	}

	// Save policy to temporary file
	tempFile := filepath.Join(tempDir, "embedded-policy.rego")
	if err := os.WriteFile(tempFile, body, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write policy file: %w", err)
	}

	// Return cleanup function
	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempFile, cleanup, nil
}

// generateEmbeddedPolicies reads .rego files from a path (file, directory, or zip archive)
// and generates a Go source file in the source directory that populates embeddedPolicyFiles
// via init(). Returns a cleanup function (to remove the generated file) and the number
// of policy files embedded.
func (b *Builder) generateEmbeddedPolicies(policyPath string) (func(), int, error) {
	policies := map[string]string{}

	info, err := os.Stat(policyPath)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot stat policy path %s: %w", policyPath, err)
	}

	switch {
	case !info.IsDir() && isZipFile(policyPath):
		// Extract .rego files from zip archive
		zr, err := zip.OpenReader(policyPath)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot open zip archive %s: %w", policyPath, err)
		}
		defer zr.Close()
		for _, f := range zr.File {
			if f.FileInfo().IsDir() || !strings.HasSuffix(f.Name, ".rego") {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return nil, 0, fmt.Errorf("cannot read %s from zip: %w", f.Name, err)
			}
			content, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, 0, fmt.Errorf("cannot read %s from zip: %w", f.Name, err)
			}
			// Use only the base filename to flatten any directory structure in the zip
			policies[filepath.Base(f.Name)] = string(content)
		}

	case info.IsDir():
		entries, err := os.ReadDir(policyPath)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot read policy directory: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".rego") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(policyPath, entry.Name()))
			if err != nil {
				return nil, 0, fmt.Errorf("cannot read policy file %s: %w", entry.Name(), err)
			}
			policies[entry.Name()] = string(content)
		}

	default:
		content, err := os.ReadFile(policyPath)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot read policy file: %w", err)
		}
		policies[filepath.Base(policyPath)] = string(content)
	}

	if len(policies) == 0 {
		return func() {}, 0, nil
	}

	// Generate Go source file
	var buf bytes.Buffer
	buf.WriteString("// Code generated by safeguard builder; DO NOT EDIT.\n")
	buf.WriteString("package main\n\nfunc init() {\n")
	for name, content := range policies {
		buf.WriteString(fmt.Sprintf("\tembeddedPolicyFiles[%q] = %s\n", name, backtickQuote(content)))
	}
	buf.WriteString("}\n")

	genPath := filepath.Join(b.sourceDir, "_embedded_policies_gen.go")
	if err := os.WriteFile(genPath, buf.Bytes(), 0644); err != nil {
		return nil, 0, fmt.Errorf("failed to write generated file: %w", err)
	}

	cleanup := func() {
		os.Remove(genPath)
	}

	return cleanup, len(policies), nil
}

// backtickQuote wraps content in backticks for a Go raw string literal.
// If the content contains backticks, it falls back to a double-quoted (interpreted) literal.
func backtickQuote(s string) string {
	if !strings.Contains(s, "`") {
		return "`" + s + "`"
	}
	return fmt.Sprintf("%q", s)
}

// isZipFile returns true if the path has a .zip extension.
func isZipFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".zip")
}
