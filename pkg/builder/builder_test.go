package builder

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewBuilder(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	workDir := filepath.Join(tmpDir, "work")
	outputDir := filepath.Join(tmpDir, "output")

	os.MkdirAll(srcDir, 0755)

	b, err := NewBuilder(srcDir, workDir, outputDir)
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}
	if b == nil {
		t.Fatal("NewBuilder() returned nil")
	}
	if b.sourceDir != srcDir {
		t.Errorf("sourceDir = %v, want %v", b.sourceDir, srcDir)
	}
	if b.workDir != workDir {
		t.Errorf("workDir = %v, want %v", b.workDir, workDir)
	}
	if b.outputDir != outputDir {
		t.Errorf("outputDir = %v, want %v", b.outputDir, outputDir)
	}

	// Verify directories were created
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		t.Error("NewBuilder() did not create work directory")
	}
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("NewBuilder() did not create output directory")
	}
}

func TestNewBuilder_InvalidWorkDir(t *testing.T) {
	// Use a path that can't be created (file as parent)
	tmpDir := t.TempDir()
	blocker := filepath.Join(tmpDir, "blocker")
	os.WriteFile(blocker, []byte("x"), 0644)

	_, err := NewBuilder(tmpDir, filepath.Join(blocker, "sub"), tmpDir)
	if err == nil {
		t.Error("NewBuilder() expected error for invalid work directory")
	}
}

func TestBuildLdflags(t *testing.T) {
	b := &Builder{}

	tests := []struct {
		name     string
		config   BuildConfig
		contains []string
		excludes []string
	}{
		{
			name:     "empty config only has strip flags",
			config:   BuildConfig{},
			contains: []string{"-s", "-w"},
			excludes: []string{"-X"},
		},
		{
			name:     "vault addr",
			config:   BuildConfig{DefaultVaultAddr: "http://vault:8200"},
			contains: []string{"-X 'main.defaultVaultAddr=http://vault:8200'"},
		},
		{
			name:     "auth method",
			config:   BuildConfig{DefaultAuthMethod: "ldap"},
			contains: []string{"-X 'main.defaultAuthMethod=ldap'"},
		},
		{
			name:   "version and tag",
			config: BuildConfig{Version: "1.0.0", BuildTag: "prod"},
			contains: []string{
				"-X 'main.version=1.0.0'",
				"-X 'main.buildTag=prod'",
			},
		},
		{
			name: "boolean flags",
			config: BuildConfig{
				DefaultDebug:         true,
				DefaultMonitor:       true,
				DefaultAccessControl: true,
				DisableCliFlags:      true,
			},
			contains: []string{
				"-X 'main.defaultDebug=true'",
				"-X 'main.defaultMonitor=true'",
				"-X 'main.defaultAccessControl=true'",
				"-X 'main.disableCliFlags=true'",
			},
		},
		{
			name: "credential overrides",
			config: BuildConfig{
				DefaultLdapUsername: "admin",
				DefaultLdapPassword: "secret",
				DefaultVaultToken:   "hvs.xxx",
			},
			contains: []string{
				"-X 'main.defaultLdapUsername=admin'",
				"-X 'main.defaultLdapPassword=secret'",
				"-X 'main.defaultVaultToken=hvs.xxx'",
			},
		},
		{
			name: "log configuration",
			config: BuildConfig{
				DefaultLogFile:       "/var/log/app.log",
				DefaultLogMaxSize:    "50",
				DefaultLogMaxBackups: "3",
				DefaultLogMaxAge:     "7",
				DefaultLogCompress:   "true",
			},
			contains: []string{
				"-X 'main.defaultLogFile=/var/log/app.log'",
				"-X 'main.defaultLogMaxSize=50'",
				"-X 'main.defaultLogMaxBackups=3'",
				"-X 'main.defaultLogMaxAge=7'",
				"-X 'main.defaultLogCompress=true'",
			},
		},
		{
			name: "cache configuration",
			config: BuildConfig{
				DefaultCacheEnabled: "true",
				DefaultCacheTTL:     "120",
			},
			contains: []string{
				"-X 'main.defaultCacheEnabled=true'",
				"-X 'main.defaultCacheTTL=120'",
			},
		},
		{
			name: "all path-related fields",
			config: BuildConfig{
				DefaultVaultProvider: "hashicorp",
				DefaultMountPoint:    "V:",
				DefaultAuthRole:      "my-role",
				DefaultAuthMount:     "auth/custom",
				DefaultPolicyPath:    "/etc/policies",
				DefaultMappingPath:   "/etc/mapping.json",
				DefaultAuditLog:      "/var/log/audit.log",
				DefaultAllowedPIDs:   "1234,5678",
				DefaultAllowedUIDs:   "1000",
			},
			contains: []string{
				"-X 'main.defaultVaultProvider=hashicorp'",
				"-X 'main.defaultMountPoint=V:'",
				"-X 'main.defaultAuthRole=my-role'",
				"-X 'main.defaultAuthMount=auth/custom'",
				"-X 'main.defaultPolicyPath=/etc/policies'",
				"-X 'main.defaultMappingPath=/etc/mapping.json'",
				"-X 'main.defaultAuditLog=/var/log/audit.log'",
				"-X 'main.defaultAllowedPIDs=1234,5678'",
				"-X 'main.defaultAllowedUIDs=1000'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := b.buildLdflags(tt.config)
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("buildLdflags() missing %q in result: %s", want, result)
				}
			}
			for _, notWant := range tt.excludes {
				if strings.Contains(result, notWant) {
					t.Errorf("buildLdflags() should not contain %q in result: %s", notWant, result)
				}
			}
		})
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"http://example.com/policy.rego", true},
		{"https://example.com/policy.rego", true},
		{"/path/to/policy.rego", false},
		{"policy.rego", false},
		{"ftp://example.com/policy.rego", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isURL(tt.path); got != tt.want {
				t.Errorf("isURL(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsZipFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"policies.zip", true},
		{"policies.ZIP", true},
		{"policies.Zip", true},
		{"policies.tar.gz", false},
		{"policies.rego", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isZipFile(tt.path); got != tt.want {
				t.Errorf("isZipFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestBacktickQuote(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple string",
			input: "package policy\ndefault allow = true",
			want:  "`package policy\ndefault allow = true`",
		},
		{
			name:  "string with backtick falls back to double-quote",
			input: "contains ` backtick",
			want:  `"contains ` + "`" + ` backtick"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := backtickQuote(tt.input)
			if got != tt.want {
				t.Errorf("backtickQuote() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateChecksum(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.bin")
	os.WriteFile(tmpFile, []byte("hello world"), 0644)

	checksum, err := calculateChecksum(tmpFile)
	if err != nil {
		t.Fatalf("calculateChecksum() error = %v", err)
	}
	if len(checksum) != 64 { // SHA256 hex = 64 chars
		t.Errorf("checksum length = %d, want 64", len(checksum))
	}

	// Same content should produce same checksum
	tmpFile2 := filepath.Join(t.TempDir(), "test2.bin")
	os.WriteFile(tmpFile2, []byte("hello world"), 0644)
	checksum2, _ := calculateChecksum(tmpFile2)
	if checksum != checksum2 {
		t.Error("Same content produced different checksums")
	}

	// Different content should produce different checksum
	tmpFile3 := filepath.Join(t.TempDir(), "test3.bin")
	os.WriteFile(tmpFile3, []byte("different"), 0644)
	checksum3, _ := calculateChecksum(tmpFile3)
	if checksum == checksum3 {
		t.Error("Different content produced same checksum")
	}
}

func TestCalculateChecksum_NotFound(t *testing.T) {
	_, err := calculateChecksum("/nonexistent/file")
	if err == nil {
		t.Error("calculateChecksum() expected error for missing file")
	}
}

func TestCreateTarGzArchive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcFile := filepath.Join(tmpDir, "test-binary")
	os.WriteFile(srcFile, []byte("binary content"), 0755)

	// Create archive
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	err := createTarGzArchive(srcFile, archivePath)
	if err != nil {
		t.Fatalf("createTarGzArchive() error = %v", err)
	}

	// Verify archive exists and can be read
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	header, err := tr.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}

	if header.Name != "test-binary" {
		t.Errorf("Archive entry name = %v, want test-binary", header.Name)
	}
	if header.Size != 14 {
		t.Errorf("Archive entry size = %v, want 14", header.Size)
	}
}

func TestCreateZipArchive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcFile := filepath.Join(tmpDir, "test-binary.exe")
	os.WriteFile(srcFile, []byte("binary content"), 0755)

	// Create archive
	archivePath := filepath.Join(tmpDir, "test.zip")
	err := createZipArchive(srcFile, archivePath)
	if err != nil {
		t.Fatalf("createZipArchive() error = %v", err)
	}

	// Verify archive contents
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("Failed to open zip: %v", err)
	}
	defer zr.Close()

	if len(zr.File) != 1 {
		t.Fatalf("Zip has %d entries, want 1", len(zr.File))
	}

	entry := zr.File[0]
	if entry.Name != "test-binary.exe" {
		t.Errorf("Zip entry name = %v, want test-binary.exe", entry.Name)
	}

	rc, _ := entry.Open()
	content, _ := io.ReadAll(rc)
	rc.Close()
	if string(content) != "binary content" {
		t.Errorf("Zip content = %v, want 'binary content'", string(content))
	}
}

func TestCreateArchive(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	workDir := filepath.Join(tmpDir, "work")
	outputDir := filepath.Join(tmpDir, "output")
	os.MkdirAll(srcDir, 0755)

	b, err := NewBuilder(srcDir, workDir, outputDir)
	if err != nil {
		t.Fatalf("NewBuilder() error = %v", err)
	}

	// Create a fake binary
	binaryPath := filepath.Join(outputDir, "safeguard-1.0.0-windows-amd64.exe")
	os.WriteFile(binaryPath, []byte("fake binary"), 0755)

	t.Run("windows produces zip", func(t *testing.T) {
		result := &BuildResult{
			BinaryPath: binaryPath,
			Config:     BuildConfig{TargetOS: "windows"},
		}
		archivePath, err := b.CreateArchive(result)
		if err != nil {
			t.Fatalf("CreateArchive() error = %v", err)
		}
		if !strings.HasSuffix(archivePath, ".zip") {
			t.Errorf("Windows archive should be .zip, got %v", archivePath)
		}
	})

	t.Run("linux produces tar.gz", func(t *testing.T) {
		binaryPathLinux := filepath.Join(outputDir, "safeguard-1.0.0-linux-amd64")
		os.WriteFile(binaryPathLinux, []byte("fake binary"), 0755)
		result := &BuildResult{
			BinaryPath: binaryPathLinux,
			Config:     BuildConfig{TargetOS: "linux"},
		}
		archivePath, err := b.CreateArchive(result)
		if err != nil {
			t.Fatalf("CreateArchive() error = %v", err)
		}
		if !strings.HasSuffix(archivePath, ".tar.gz") {
			t.Errorf("Linux archive should be .tar.gz, got %v", archivePath)
		}
	})
}

func TestGenerateEmbeddedPolicies_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)

	b := &Builder{sourceDir: srcDir}

	// Create a single policy file
	policyFile := filepath.Join(tmpDir, "test.rego")
	os.WriteFile(policyFile, []byte("package test\ndefault allow = true"), 0644)

	cleanup, count, err := b.generateEmbeddedPolicies(policyFile)
	if err != nil {
		t.Fatalf("generateEmbeddedPolicies() error = %v", err)
	}
	defer cleanup()

	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	// Verify generated file exists
	genPath := filepath.Join(srcDir, "_embedded_policies_gen.go")
	content, err := os.ReadFile(genPath)
	if err != nil {
		t.Fatalf("Generated file not found: %v", err)
	}
	if !strings.Contains(string(content), "test.rego") {
		t.Error("Generated file should reference test.rego")
	}
	if !strings.Contains(string(content), "package main") {
		t.Error("Generated file should be in package main")
	}
}

func TestGenerateEmbeddedPolicies_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)

	b := &Builder{sourceDir: srcDir}

	// Create a directory with multiple policy files
	policyDir := filepath.Join(tmpDir, "policies")
	os.MkdirAll(policyDir, 0755)
	os.WriteFile(filepath.Join(policyDir, "a.rego"), []byte("package a"), 0644)
	os.WriteFile(filepath.Join(policyDir, "b.rego"), []byte("package b"), 0644)
	os.WriteFile(filepath.Join(policyDir, "readme.md"), []byte("not a policy"), 0644) // should be skipped

	cleanup, count, err := b.generateEmbeddedPolicies(policyDir)
	if err != nil {
		t.Fatalf("generateEmbeddedPolicies() error = %v", err)
	}
	defer cleanup()

	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestGenerateEmbeddedPolicies_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)

	b := &Builder{sourceDir: srcDir}

	emptyDir := filepath.Join(tmpDir, "empty")
	os.MkdirAll(emptyDir, 0755)

	cleanup, count, err := b.generateEmbeddedPolicies(emptyDir)
	if err != nil {
		t.Fatalf("generateEmbeddedPolicies() error = %v", err)
	}
	defer cleanup()

	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestGenerateEmbeddedPolicies_ZipArchive(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)

	b := &Builder{sourceDir: srcDir}

	// Create a zip with .rego files
	zipPath := filepath.Join(tmpDir, "policies.zip")
	zf, _ := os.Create(zipPath)
	zw := zip.NewWriter(zf)

	w, _ := zw.Create("policy1.rego")
	w.Write([]byte("package policy1"))
	w, _ = zw.Create("policy2.rego")
	w.Write([]byte("package policy2"))
	w, _ = zw.Create("readme.txt")
	w.Write([]byte("skip me"))

	zw.Close()
	zf.Close()

	cleanup, count, err := b.generateEmbeddedPolicies(zipPath)
	if err != nil {
		t.Fatalf("generateEmbeddedPolicies() error = %v", err)
	}
	defer cleanup()

	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestGenerateEmbeddedPolicies_NonexistentPath(t *testing.T) {
	b := &Builder{sourceDir: t.TempDir()}
	_, _, err := b.generateEmbeddedPolicies("/nonexistent/path")
	if err == nil {
		t.Error("generateEmbeddedPolicies() expected error for nonexistent path")
	}
}

func TestGenerateEmbeddedPolicies_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	os.MkdirAll(srcDir, 0755)

	b := &Builder{sourceDir: srcDir}

	policyFile := filepath.Join(tmpDir, "test.rego")
	os.WriteFile(policyFile, []byte("package test"), 0644)

	cleanup, _, err := b.generateEmbeddedPolicies(policyFile)
	if err != nil {
		t.Fatalf("generateEmbeddedPolicies() error = %v", err)
	}

	genPath := filepath.Join(srcDir, "_embedded_policies_gen.go")
	if _, err := os.Stat(genPath); os.IsNotExist(err) {
		t.Fatal("Generated file should exist before cleanup")
	}

	cleanup()

	if _, err := os.Stat(genPath); !os.IsNotExist(err) {
		t.Error("Generated file should be removed after cleanup")
	}
}
