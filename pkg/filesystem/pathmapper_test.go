package filesystem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPathMapper_LoadMappings(t *testing.T) {
	// Create a temporary test file
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("test content")
	tmpFile.Close()

	pm := NewPathMapper(true)

	mappings := []PathMapping{
		{
			VirtualPath: "/secrets/config.txt",
			RealPath:    tmpFile.Name(),
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	if pm.Count() != 1 {
		t.Errorf("Expected 1 mapping, got %d", pm.Count())
	}

	if !pm.IsMapped("/secrets/config.txt") {
		t.Error("Path should be mapped")
	}

	realPath := pm.GetRealPath("/secrets/config.txt")
	expectedPath, _ := filepath.Abs(tmpFile.Name())
	if realPath != expectedPath {
		t.Errorf("Expected real path %s, got %s", expectedPath, realPath)
	}
}

func TestPathMapper_NormalizePath(t *testing.T) {
	pm := NewPathMapper(true)

	tests := []struct {
		input    string
		expected string
	}{
		{"/secrets/config.txt", "secrets/config.txt"},
		{"secrets/config.txt", "secrets/config.txt"},
		{"/secrets/config.txt/", "secrets/config.txt"},
		{"//secrets//config.txt//", "secrets/config.txt"},
	}

	for _, tt := range tests {
		result := pm.normalizePath(tt.input)
		if result != tt.expected {
			t.Errorf("normalizePath(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestPathMapper_CaseInsensitive(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	pm := NewPathMapper(false) // case-insensitive

	mappings := []PathMapping{
		{
			VirtualPath: "/Secrets/Config.TXT",
			RealPath:    tmpFile.Name(),
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	// All these should match in case-insensitive mode
	testPaths := []string{
		"/secrets/config.txt",
		"/SECRETS/CONFIG.TXT",
		"/Secrets/Config.Txt",
	}

	for _, path := range testPaths {
		if !pm.IsMapped(path) {
			t.Errorf("Path %q should be mapped (case-insensitive)", path)
		}
	}
}

func TestPathMapper_LoadFromFile(t *testing.T) {
	// Create a temporary test file to map
	tmpFile, err := os.CreateTemp("", "mapped-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("mapped content")
	tmpFile.Close()

	// Create config file
	config := PathMapperConfig{
		Mappings: []PathMapping{
			{
				VirtualPath: "/virtual/file.txt",
				RealPath:    tmpFile.Name(),
				ReadOnly:    true,
			},
		},
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	configFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}
	defer os.Remove(configFile.Name())
	configFile.Write(configData)
	configFile.Close()

	// Load from file
	pm := NewPathMapper(true)
	err = pm.LoadFromFile(configFile.Name())
	if err != nil {
		t.Fatalf("Failed to load from file: %v", err)
	}

	if pm.Count() != 1 {
		t.Errorf("Expected 1 mapping, got %d", pm.Count())
	}

	if !pm.IsMapped("/virtual/file.txt") {
		t.Error("Path should be mapped")
	}
}

func TestPathMapper_ReadMappedFile(t *testing.T) {
	expectedContent := "test file content"
	tmpFile, err := os.CreateTemp("", "read-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(expectedContent)
	tmpFile.Close()

	pm := NewPathMapper(true)
	mappings := []PathMapping{
		{
			VirtualPath: "/data/test.txt",
			RealPath:    tmpFile.Name(),
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	content, err := pm.ReadMappedFile("/data/test.txt")
	if err != nil {
		t.Fatalf("Failed to read mapped file: %v", err)
	}

	if string(content) != expectedContent {
		t.Errorf("Expected content %q, got %q", expectedContent, string(content))
	}
}

func TestPathMapper_InvalidMappings(t *testing.T) {
	pm := NewPathMapper(true)

	tests := []struct {
		name     string
		mappings []PathMapping
		wantErr  bool
	}{
		{
			name: "empty virtual path",
			mappings: []PathMapping{
				{VirtualPath: "", RealPath: "/some/file.txt"},
			},
			wantErr: true,
		},
		{
			name: "empty real path",
			mappings: []PathMapping{
				{VirtualPath: "/virtual/path", RealPath: ""},
			},
			wantErr: true,
		},
		{
			name: "non-existent file",
			mappings: []PathMapping{
				{VirtualPath: "/virtual/path", RealPath: "/non/existent/file.txt"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pm.LoadMappings(tt.mappings)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadMappings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPathMapper_GetMappedFileInfo(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "info-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("test")
	tmpFile.Close()

	pm := NewPathMapper(true)
	mappings := []PathMapping{
		{
			VirtualPath: "/info/test.txt",
			RealPath:    tmpFile.Name(),
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	info, err := pm.GetMappedFileInfo("/info/test.txt")
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	if info.Size() != 4 {
		t.Errorf("Expected size 4, got %d", info.Size())
	}

	if info.IsDir() {
		t.Error("Expected file, got directory")
	}
}

func TestPathMapper_DirectoryMapping(t *testing.T) {
	// Create a temporary directory with some files
	tmpDir, err := os.MkdirTemp("", "dir-mapping-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files in the directory
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "file3.txt"), []byte("content3"), 0644)

	pm := NewPathMapper(true)
	mappings := []PathMapping{
		{
			VirtualPath: "/data",
			RealPath:    tmpDir,
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	// Test that the directory itself is mapped
	if !pm.IsMappedOrUnder("/data") {
		t.Error("Directory should be mapped")
	}

	// Test that files under the directory are accessible
	if !pm.IsMappedOrUnder("/data/file1.txt") {
		t.Error("File under mapped directory should be accessible")
	}

	// Test resolving paths under the directory
	mapping, realPath := pm.ResolveMappedPath("/data/file1.txt")
	if mapping == nil {
		t.Fatal("Failed to resolve path under mapped directory")
	}

	expectedPath := filepath.Join(tmpDir, "file1.txt")
	if realPath != expectedPath {
		t.Errorf("Expected real path %s, got %s", expectedPath, realPath)
	}

	// Test reading a file under the mapped directory
	content, err := pm.ReadMappedPath("/data/file1.txt")
	if err != nil {
		t.Fatalf("Failed to read file under mapped directory: %v", err)
	}

	if string(content) != "content1" {
		t.Errorf("Expected content 'content1', got %q", string(content))
	}

	// Test listing the mapped directory
	entries, err := pm.ListMappedDirectory("/data")
	if err != nil {
		t.Fatalf("Failed to list mapped directory: %v", err)
	}

	// Should have 2 files and 1 subdirectory
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	// Test subdirectory access
	if !pm.IsMappedOrUnder("/data/subdir") {
		t.Error("Subdirectory should be accessible")
	}

	if !pm.IsMappedOrUnder("/data/subdir/file3.txt") {
		t.Error("File under subdirectory should be accessible")
	}

	// Test reading file in subdirectory
	content, err = pm.ReadMappedPath("/data/subdir/file3.txt")
	if err != nil {
		t.Fatalf("Failed to read file in subdirectory: %v", err)
	}

	if string(content) != "content3" {
		t.Errorf("Expected content 'content3', got %q", string(content))
	}

	// Test GetMappedPathInfo for directory
	info, err := pm.GetMappedPathInfo("/data")
	if err != nil {
		t.Fatalf("Failed to get directory info: %v", err)
	}

	if !info.IsDir() {
		t.Error("Expected directory, got file")
	}

	// Test GetMappedPathInfo for file
	info, err = pm.GetMappedPathInfo("/data/file1.txt")
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	if info.IsDir() {
		t.Error("Expected file, got directory")
	}
}

func TestPathMapper_MixedMappings(t *testing.T) {
	// Create a temporary directory and a temporary file
	tmpDir, err := os.MkdirTemp("", "mixed-mapping-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile, err := os.CreateTemp("", "mixed-file-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("standalone file")
	tmpFile.Close()

	// Create a file in the directory
	os.WriteFile(filepath.Join(tmpDir, "dirfile.txt"), []byte("dir content"), 0644)

	pm := NewPathMapper(true)
	mappings := []PathMapping{
		{
			VirtualPath: "/folder",
			RealPath:    tmpDir,
			ReadOnly:    true,
		},
		{
			VirtualPath: "/standalone.txt",
			RealPath:    tmpFile.Name(),
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	// Test directory mapping
	if !pm.IsMappedOrUnder("/folder/dirfile.txt") {
		t.Error("File under directory mapping should be accessible")
	}

	// Test file mapping
	if !pm.IsMappedOrUnder("/standalone.txt") {
		t.Error("Standalone file should be mapped")
	}

	// Test that unrelated paths are not mapped
	if pm.IsMappedOrUnder("/other/path.txt") {
		t.Error("Unmapped path should not be accessible")
	}
}

func TestPathMapper_NestedDirectories(t *testing.T) {
	// Create nested directory structure
	tmpDir, err := os.MkdirTemp("", "nested-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested structure: tmpDir/level1/level2/file.txt
	level1 := filepath.Join(tmpDir, "level1")
	level2 := filepath.Join(level1, "level2")
	os.MkdirAll(level2, 0755)
	os.WriteFile(filepath.Join(level2, "deep.txt"), []byte("deep content"), 0644)

	pm := NewPathMapper(true)
	mappings := []PathMapping{
		{
			VirtualPath: "/root",
			RealPath:    tmpDir,
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	// Test accessing deeply nested file
	deepPath := "/root/level1/level2/deep.txt"
	if !pm.IsMappedOrUnder(deepPath) {
		t.Error("Deeply nested file should be accessible")
	}

	content, err := pm.ReadMappedPath(deepPath)
	if err != nil {
		t.Fatalf("Failed to read deeply nested file: %v", err)
	}

	if string(content) != "deep content" {
		t.Errorf("Expected 'deep content', got %q", string(content))
	}

	// Test listing nested directories
	entries, err := pm.ListMappedDirectory("/root/level1")
	if err != nil {
		t.Fatalf("Failed to list nested directory: %v", err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry (level2/), got %d", len(entries))
	}
}

func TestPathMapper_GetRootEntries(t *testing.T) {
	// Create temporary test files and directories
	tmpFile1, err := os.CreateTemp("", "test1-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file 1: %v", err)
	}
	defer os.Remove(tmpFile1.Name())
	tmpFile1.Close()

	tmpFile2, err := os.CreateTemp("", "test2-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file 2: %v", err)
	}
	defer os.Remove(tmpFile2.Name())
	tmpFile2.Close()

	tmpDir, err := os.MkdirTemp("", "testdir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	pm := NewPathMapper(true)

	mappings := []PathMapping{
		{
			VirtualPath: "/config/app.txt",
			RealPath:    tmpFile1.Name(),
			ReadOnly:    true,
		},
		{
			VirtualPath: "/config/db.txt",
			RealPath:    tmpFile2.Name(),
			ReadOnly:    true,
		},
		{
			VirtualPath: "/data",
			RealPath:    tmpDir,
			ReadOnly:    true,
		},
		{
			VirtualPath: "/standalone.txt",
			RealPath:    tmpFile1.Name(),
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	// Get root entries
	rootEntries := pm.GetRootEntries()

	// Should have 3 root entries: config (dir), data (dir), standalone.txt (file)
	expectedEntries := map[string]bool{
		"config":         true,  // directory (has sub-paths)
		"data":           true,  // directory (explicitly marked)
		"standalone.txt": false, // file
	}

	if len(rootEntries) != len(expectedEntries) {
		t.Errorf("Expected %d root entries, got %d", len(expectedEntries), len(rootEntries))
	}

	for entry, expectedIsDir := range expectedEntries {
		isDir, exists := rootEntries[entry]
		if !exists {
			t.Errorf("Expected root entry %q to exist", entry)
		}
		if isDir != expectedIsDir {
			t.Errorf("Entry %q: expected isDir=%v, got %v", entry, expectedIsDir, isDir)
		}
	}
}

func TestPathMapper_GetRootEntries_WithRootMapping(t *testing.T) {
	// Create temporary directory for root-level mapping
	tmpDir, err := os.MkdirTemp("", "rootmap-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some files in the temp directory
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)

	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	pm := NewPathMapper(true)

	mappings := []PathMapping{
		{
			VirtualPath: "/",
			RealPath:    tmpDir,
			ReadOnly:    true,
		},
		{
			VirtualPath: "/config/app.txt",
			RealPath:    tmpFile.Name(),
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	// Get root entries
	rootEntries := pm.GetRootEntries()

	// Should have only 1 entry: config (dir)
	// The "/" mapping should not appear in the root entries list
	// because it IS the root itself
	expectedEntries := map[string]bool{
		"config": true, // directory (has sub-paths)
	}

	if len(rootEntries) != len(expectedEntries) {
		t.Errorf("Expected %d root entries, got %d: %v", len(expectedEntries), len(rootEntries), rootEntries)
	}

	for entry, expectedIsDir := range expectedEntries {
		isDir, exists := rootEntries[entry]
		if !exists {
			t.Errorf("Expected root entry %q to exist", entry)
		}
		if isDir != expectedIsDir {
			t.Errorf("Entry %q: expected isDir=%v, got %v", entry, expectedIsDir, isDir)
		}
	}

	// Verify that the root mapping is accessible
	if !pm.IsMappedOrUnder("/") {
		t.Error("Root path should be mapped")
	}

	if !pm.IsMappedOrUnder("") {
		t.Error("Empty path (normalized root) should be mapped")
	}
}

func TestPathMapper_VirtualDirectories(t *testing.T) {
	// Create temporary test file
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	pm := NewPathMapper(true)

	mappings := []PathMapping{
		{
			VirtualPath: "/app/config/dev/settings.txt",
			RealPath:    tmpFile.Name(),
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	// Test that intermediate directories are recognized as virtual directories
	testCases := []struct {
		path        string
		shouldExist bool
		description string
	}{
		{"/app", true, "first level virtual directory"},
		{"/app/config", true, "second level virtual directory"},
		{"/app/config/dev", true, "third level virtual directory"},
		{"/app/config/dev/settings.txt", true, "actual mapped file"},
		{"/app/other", false, "non-existent path"},
		{"/other", false, "different root path"},
	}

	for _, tc := range testCases {
		result := pm.IsMappedOrUnder(tc.path)
		if result != tc.shouldExist {
			t.Errorf("IsMappedOrUnder(%q) = %v, expected %v (%s)", tc.path, result, tc.shouldExist, tc.description)
		}

		isVirtual := pm.IsVirtualDirectory(tc.path)
		if tc.path == "/app" || tc.path == "/app/config" || tc.path == "/app/config/dev" {
			if !isVirtual {
				t.Errorf("IsVirtualDirectory(%q) should be true", tc.path)
			}
		}
	}

	// Test listing virtual directories
	entries, err := pm.ListMappedDirectory("/app")
	if err != nil {
		t.Fatalf("Failed to list /app: %v", err)
	}
	if len(entries) != 1 || entries[0] != "config/" {
		t.Errorf("ListMappedDirectory('/app') = %v, expected ['config/']", entries)
	}

	entries, err = pm.ListMappedDirectory("/app/config")
	if err != nil {
		t.Fatalf("Failed to list /app/config: %v", err)
	}
	if len(entries) != 1 || entries[0] != "dev/" {
		t.Errorf("ListMappedDirectory('/app/config') = %v, expected ['dev/']", entries)
	}

	entries, err = pm.ListMappedDirectory("/app/config/dev")
	if err != nil {
		t.Fatalf("Failed to list /app/config/dev: %v", err)
	}
	if len(entries) != 1 || entries[0] != "settings.txt" {
		t.Errorf("ListMappedDirectory('/app/config/dev') = %v, expected ['settings.txt']", entries)
	}

	// Test GetMappedPathInfo for virtual directories
	info, err := pm.GetMappedPathInfo("/app")
	if err != nil {
		t.Fatalf("Failed to get info for /app: %v", err)
	}
	if !info.IsDir() {
		t.Error("Virtual directory /app should be reported as a directory")
	}
}

func TestPathMapper_MultipleVirtualPaths(t *testing.T) {
	// Create temporary test files
	tmpFile1, err := os.CreateTemp("", "test1-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file 1: %v", err)
	}
	defer os.Remove(tmpFile1.Name())
	tmpFile1.Close()

	tmpFile2, err := os.CreateTemp("", "test2-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file 2: %v", err)
	}
	defer os.Remove(tmpFile2.Name())
	tmpFile2.Close()

	pm := NewPathMapper(true)

	mappings := []PathMapping{
		{
			VirtualPath: "/app/config/dev/db.txt",
			RealPath:    tmpFile1.Name(),
			ReadOnly:    true,
		},
		{
			VirtualPath: "/app/config/prod/db.txt",
			RealPath:    tmpFile2.Name(),
			ReadOnly:    true,
		},
	}

	err = pm.LoadMappings(mappings)
	if err != nil {
		t.Fatalf("Failed to load mappings: %v", err)
	}

	// Test listing /app/config should show both dev/ and prod/
	entries, err := pm.ListMappedDirectory("/app/config")
	if err != nil {
		t.Fatalf("Failed to list /app/config: %v", err)
	}

	expectedEntries := map[string]bool{
		"dev/":  true,
		"prod/": true,
	}

	if len(entries) != len(expectedEntries) {
		t.Errorf("ListMappedDirectory('/app/config') returned %d entries, expected %d: %v", len(entries), len(expectedEntries), entries)
	}

	for _, entry := range entries {
		if !expectedEntries[entry] {
			t.Errorf("Unexpected entry: %q", entry)
		}
	}
}
