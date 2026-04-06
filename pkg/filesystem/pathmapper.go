// Package filesystem provides FUSE filesystem implementation for mounting HashiCorp Vault
// as a virtual filesystem, along with path mapping functionality to expose local files
// and directories within the virtual filesystem.
//
// The package includes:
//   - VaultFS: A FUSE filesystem that proxies to Vault's HTTP API
//   - PathMapper: Virtual path to real file/directory mapping with support for virtual directories
//   - PolicyEvaluator: OPA/REGO-based access control policies
//   - Process monitoring and audit logging capabilities
package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PathMapping represents a mapping from a virtual path in the mounted filesystem
// to a real file or directory on the local disk.
//
// Virtual paths can be deeply nested (e.g., /app/config/dev/settings.txt) and the
// PathMapper will automatically create intermediate virtual directories
// (e.g., /app, /app/config, /app/config/dev).
type PathMapping struct {
	VirtualPath string `json:"virtual_path"` // Path in the mounted filesystem
	RealPath    string `json:"real_path"`    // Actual file/directory path on disk
	ReadOnly    bool   `json:"read_only"`    // Whether the file is read-only (default: true)
	IsDirectory bool   `json:"is_directory"` // Whether this is a directory mapping (auto-detected)
}

// PathMapperConfig represents the JSON configuration file structure for path mappings.
// This is typically loaded from a file like mapping-config.json.
type PathMapperConfig struct {
	Mappings []PathMapping `json:"mappings"`
}

// PathMapper manages virtual path to real file/directory mappings, providing
// symlink-like functionality within the FUSE filesystem.
//
// It supports:
//   - Mapping individual files to virtual paths
//   - Mapping entire directories to virtual paths
//   - Automatic creation of virtual inte instance.
//
// The caseSensitive parameter determines whether path comparisons are case-sensitive.
// On Windows, you typically want case-insensitive matching (false), while on
// Linux/macOS, case-sensitive matching (true) is more common.rmediate directories
//   - Case-sensitive or case-insensitive path matching
//
// Example:
//
//	If you map "/app/config/dev/db.txt" to "C:\\configs\\dev.txt",
//	the PathMapper will make "/app", "/app/config", and "/app/config/dev"
//	appear as virtual directories, and allow access to "db.txt" through the
//	virtual path.
type PathMapper struct {
	mappings      map[string]*PathMapping // Key: normalized virtual path
	reverseMap    map[string]string       // Key: real path, Value: virtual path
	caseSensitive bool
}

// NewPathMapper creates a new PathMapper instance.
//
// The caseSensitive parameter determines whether path comparisons are case-sensitive.
// On Windows, you typically want case-insensitive matching (false), while on
// Linux/macOS, case-sensitive matching (true) is more common.
func NewPathMapper(caseSensitive bool) *PathMapper {
	return &PathMapper{
		mappings:      make(map[string]*PathMapping),
		reverseMap:    make(map[string]string),
		caseSensitive: caseSensitive,
	}
}

// LoadFromFile loads path mappings from a JSON configuration file.
//
// The configuration file should have the following format:
//
//	{
//	  "mappings": [
//	    {
//	      "virtual_path": "/config/app.txt",
//	      "real_path": "C:\\temp\\app.txt",
//	      "read_only": true
//	    }
//	  ]
//	}
//
// Returns an error if the file cannot be read, parsed, or if any mapping is invalid.
func (pm *PathMapper) LoadFromFile(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config PathMapperConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return pm.LoadMappings(config.Mappings)
}

// LoadMappings loads and validates path mappings from a slice.
//
// This method:
//   - Validates that virtual_path and real_path are non-empty
//   - Checks that real_path exists on disk
//   - Auto-detects if the real path is a file or directory
//   - Normalizes paths for consistent internal storage
//
// Returns an error if any mapping is invalid or if a real path doesn't exist.

func (pm *PathMapper) LoadMappings(mappings []PathMapping) error {
	for i, mapping := range mappings {
		// Validate the mapping
		if mapping.VirtualPath == "" {
			return fmt.Errorf("mapping %d: virtual_path cannot be empty", i)
		}
		if mapping.RealPath == "" {
			return fmt.Errorf("mapping %d: real_path cannot be empty", i)
		}

		// Check if real file/directory exists
		info, err := os.Stat(mapping.RealPath)
		if err != nil {
			return fmt.Errorf("mapping %d: real_path '%s' does not exist: %w", i, mapping.RealPath, err)
		}

		// Detect if this is a directory mapping
		isDirectory := info.IsDir()

		// Normalize paths
		virtualPath := pm.normalizePath(mapping.VirtualPath)
		realPath, err := filepath.Abs(mapping.RealPath)
		if err != nil {
			return fmt.Errorf("mapping %d: failed to resolve real_path: %w", i, err)
		}

		// Store the mapping
		mappingCopy := mapping
		mappingCopy.VirtualPath = virtualPath
		mappingCopy.RealPath = realPath

		pm.mappings[virtualPath] = &mappingCopy
		mappingCopy.IsDirectory = isDirectory
		pm.reverseMap[realPath] = virtualPath
	}

	return nil
}

// normalizePath normalizes a path for consistent lookups
func (pm *PathMapper) normalizePath(path string) string {
	// Convert to forward slashes
	path = filepath.ToSlash(path)

	// Replace multiple consecutive slashes with a single slash
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	// Remove leading and trailing slashes
	path = strings.Trim(path, "/")

	// Case normalization for case-insensitive systems
	if !pm.caseSensitive {
		path = strings.ToLower(path)
	}

	return path
}

// GetMapping returns the PathMapping for a given virtual path.
// Returns nil if the path is not directly mapped.
//
// Note: This only checks for exact matches. Use IsMappedOrUnder to check if
// a path is under a mapped directory, or use IsVirtualDirectory to check if
// a path is an intermediate virtual directory.
func (pm *PathMapper) GetMapping(virtualPath string) *PathMapping {
	normalizedPath := pm.normalizePath(virtualPath)
	return pm.mappings[normalizedPath]
}

// IsMapped checks if a virtual path is directly mapped to a real file or directory.
// Returns false for intermediate virtual directories and paths under mapped directories.
//
// Use IsMappedOrUnder for a more comprehensive check that includes paths under
// mapped directories and intermediate virtual directories.
func (pm *PathMapper) IsMapped(virtualPath string) bool {
	return pm.GetMapping(virtualPath) != nil
}

// GetRealPath returns the real filesystem path for a directly mapped virtual path.
// Returns an empty string if the virtual path is not directly mapped.
//
// This only works for exact matches. For paths under mapped directories, use
// ResolveMappedPath instead.
func (pm *PathMapper) GetRealPath(virtualPath string) string {
	mapping := pm.GetMapping(virtualPath)
	if mapping == nil {
		return ""
	}
	return mapping.RealPath
}

// GetAllMappings returns a copy of all configured path mappings.
// Useful for inspecting or iterating over all mappings.
func (pm *PathMapper) GetAllMappings() []PathMapping {
	result := make([]PathMapping, 0, len(pm.mappings))
	for _, mapping := range pm.mappings {
		result = append(result, *mapping)
	}
	return result
}

// GetRootEntries returns the top-level entries that should appear when listing
// the root (/) directory of the virtual filesystem.
//
// The returned map has entry names as keys and boolean values indicating whether
// each entry is a directory (true) or file (false).
//
// For example, if you have mappings:
//   - /app/config/dev/db.txt
//   - /app/logs/app.log
//   - /standalone.txt
//
// GetRootEntries() will return:
//
//	map[string]bool{
//	  "app": true,           // directory (has sub-paths)
//	  "standalone.txt": false // file
//	}
func (pm *PathMapper) GetRootEntries() map[string]bool {
	entries := make(map[string]bool)

	for virtualPath, mapping := range pm.mappings {
		// The virtualPath is already normalized (leading/trailing slashes removed)
		// Empty string means it's mapped at root level "/"
		if virtualPath == "" {
			// This is a root-level mapping (virtual_path was "/")
			// We can't list this as an entry since it IS the root
			// Instead, the mapping itself should be checked via IsMappedOrUnder
			continue
		}

		// Split the virtual path to get the first component
		parts := strings.Split(virtualPath, "/")
		if len(parts) > 0 && parts[0] != "" {
			// This is the root-level entry
			rootEntry := parts[0]
			// Mark if it's a directory (either explicitly or has sub-paths)
			isDir := mapping.IsDirectory || len(parts) > 1
			entries[rootEntry] = isDir
		}
	}

	return entries
}

// Count returns the total number of configured path mappings.
func (pm *PathMapper) Count() int {
	return len(pm.mappings)
}

// ReadMappedFile reads and returns the complete content of a directly mapped file.
//
// Returns an error if:
//   - The virtual path is not mapped
//   - The path points to a directory
//   - The file cannot be read
//
// For files under mapped directories, use ReadMappedPath instead.
func (pm *PathMapper) ReadMappedFile(virtualPath string) ([]byte, error) {
	mapping := pm.GetMapping(virtualPath)
	if mapping == nil {
		return nil, fmt.Errorf("path not mapped: %s", virtualPath)
	}

	data, err := os.ReadFile(mapping.RealPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapped file: %w", err)
	}

	return data, nil
}

// GetMappedFileInfo returns os.FileInfo for a directly mapped file or directory.
//
// Returns an error if the virtual path is not mapped or if the file cannot be accessed.
func (pm *PathMapper) GetMappedFileInfo(virtualPath string) (os.FileInfo, error) {
	mapping := pm.GetMapping(virtualPath)
	if mapping == nil {
		return nil, fmt.Errorf("path not mapped: %s", virtualPath)
	}

	info, err := os.Stat(mapping.RealPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat mapped file: %w", err)
	}

	return info, nil
}

// ResolveMappedPath resolves a virtual path that may be under a mapped directory,
// returning both the root mapping and the resolved real filesystem path.
//
// This method handles:
//   - Exact matches: /config/app.txt mapped directly
//   - Paths under mapped directories: /data/myapp/src/main.go when /data/myapp is mapped
//
// Returns:
//   - The PathMapping for the root of the mapping
//   - The resolved real filesystem path
//   - (nil, "") if the path is not mapped
//
// Example:
//
//	If "/data/source" is mapped to "C:\\source",
//	ResolveMappedPath("/data/source/main.go") returns
//	(mapping for "/data/source", "C:\\source\\main.go")
func (pm *PathMapper) ResolveMappedPath(virtualPath string) (*PathMapping, string) {
	normalizedPath := pm.normalizePath(virtualPath)

	// Check for exact match first
	if mapping := pm.mappings[normalizedPath]; mapping != nil {
		return mapping, mapping.RealPath
	}

	// Check if path is under a mapped directory
	for virtualPrefix, mapping := range pm.mappings {
		if !mapping.IsDirectory {
			continue
		}

		// Check if the normalized path is under this directory
		if normalizedPath == virtualPrefix || strings.HasPrefix(normalizedPath, virtualPrefix+"/") {
			// Calculate relative path
			relPath := strings.TrimPrefix(normalizedPath, virtualPrefix)
			relPath = strings.TrimPrefix(relPath, "/")

			// Join with real path
			realPath := filepath.Join(mapping.RealPath, relPath)
			return mapping, realPath
		}
	}

	return nil, ""
}

// IsMappedOrUnder checks if a virtual path is accessible through path mappings.
//
// Returns true if:
//   - The path is directly mapped
//   - The path is under a mapped directory (e.g., /data/source/main.go when /data/source is mapped)
//   - The path is an intermediate virtual directory (e.g., /app when /app/config/dev is mapped)
//
// This is the most comprehensive check for whether a path exists in the mapped filesystem.
func (pm *PathMapper) IsMappedOrUnder(virtualPath string) bool {
	mapping, _ := pm.ResolveMappedPath(virtualPath)
	if mapping != nil {
		return true
	}
	// Also check if this is an intermediate virtual directory
	return pm.IsVirtualDirectory(virtualPath)
}

// IsVirtualDirectory checks if a path is an intermediate directory that was automatically
// created to support nested path mappings.
//
// Virtual directories don't correspond to real directories on disk - they exist only
// in the virtual filesystem to provide structure.
//
// Example:
//
//	If "/app/config/dev/settings.txt" is mapped,
//	then "/app", "/app/config", and "/app/config/dev" are all virtual directories.
//
// Returns false for paths that are directly mapped or don't lead to any mappings.
func (pm *PathMapper) IsVirtualDirectory(virtualPath string) bool {
	normalizedPath := pm.normalizePath(virtualPath)

	// Check if any mapping starts with this path as a prefix
	for mappedPath := range pm.mappings {
		if mappedPath == normalizedPath {
			continue // exact match, not a parent directory
		}
		// Check if this is a parent directory of the mapped path
		if strings.HasPrefix(mappedPath, normalizedPath+"/") || (normalizedPath == "" && mappedPath != "") {
			return true
		}
	}
	return false
}

// ListVirtualDirectory lists the child entries under a virtual directory path.
//
// This method examines all mappings and returns the appropriate first-level
// children that should appear under the given virtual directory.
//
// Example:
//
//	If you have mappings:
//	  - /app/config/dev/db.txt
//	  - /app/config/prod/db.txt
//	  - /app/logs/app.log
//
//	ListVirtualDirectory("/app") returns: ["config/", "logs/"]
//	ListVirtualDirectory("/app/config") returns: ["dev/", "prod/"]
//
// Returns an error if the path is not a virtual directory.
func (pm *PathMapper) ListVirtualDirectory(virtualPath string) ([]string, error) {
	normalizedPath := pm.normalizePath(virtualPath)

	entries := make(map[string]bool) // map to track unique entries, value indicates if it's a directory

	// Find all mappings that are under this path
	for mappedPath, mapping := range pm.mappings {
		var relativePath string

		if normalizedPath == "" {
			// Listing root
			relativePath = mappedPath
		} else if mappedPath == normalizedPath {
			// Exact match - this is an actual mapped entry
			relativePath = ""
		} else if strings.HasPrefix(mappedPath, normalizedPath+"/") {
			// This mapping is under the requested path
			relativePath = strings.TrimPrefix(mappedPath, normalizedPath+"/")
		} else {
			continue
		}

		if relativePath == "" {
			continue
		}

		// Get the first component of the relative path
		parts := strings.Split(relativePath, "/")
		if len(parts) > 0 && parts[0] != "" {
			firstComponent := parts[0]
			// It's a directory if there are more parts, or if the mapping itself is a directory
			isDir := len(parts) > 1 || mapping.IsDirectory
			entries[firstComponent] = isDir
		}
	}

	// Convert to string slice
	result := make([]string, 0, len(entries))
	for entry, isDir := range entries {
		if isDir {
			result = append(result, entry+"/")
		} else {
			result = append(result, entry)
		}
	}

	return result, nil
}

// ListMappedDirectory lists the contents of a mapped directory
func (pm *PathMapper) ListMappedDirectory(virtualPath string) ([]string, error) {
	mapping, realPath := pm.ResolveMappedPath(virtualPath)

	// If this is a virtual directory (intermediate path), list virtual entries
	if mapping == nil && pm.IsVirtualDirectory(virtualPath) {
		return pm.ListVirtualDirectory(virtualPath)
	}

	if mapping == nil {
		return nil, fmt.Errorf("path not mapped: %s", virtualPath)
	}

	// Check if the resolved real path is a directory
	info, err := os.Stat(realPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", virtualPath)
	}

	// Read directory contents
	entries, err := os.ReadDir(realPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Convert to string slice
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		result = append(result, name)
	}

	return result, nil
}

// virtualDirInfo implements os.FileInfo for virtual directories
type virtualDirInfo struct {
	name string
}

func (v *virtualDirInfo) Name() string       { return v.name }
func (v *virtualDirInfo) Size() int64        { return 0 }
func (v *virtualDirInfo) Mode() os.FileMode  { return os.ModeDir | 0755 }
func (v *virtualDirInfo) ModTime() time.Time { return time.Now() }
func (v *virtualDirInfo) IsDir() bool        { return true }
func (v *virtualDirInfo) Sys() interface{}   { return nil }

// GetMappedPathInfo returns file information for a path (including paths under mapped directories)
func (pm *PathMapper) GetMappedPathInfo(virtualPath string) (os.FileInfo, error) {
	_, realPath := pm.ResolveMappedPath(virtualPath)

	// Check if this is a virtual directory
	if realPath == "" && pm.IsVirtualDirectory(virtualPath) {
		normalizedPath := pm.normalizePath(virtualPath)
		name := normalizedPath
		if idx := strings.LastIndex(normalizedPath, "/"); idx >= 0 {
			name = normalizedPath[idx+1:]
		}
		if name == "" {
			name = "/"
		}
		return &virtualDirInfo{name: name}, nil
	}

	if realPath == "" {
		return nil, fmt.Errorf("path not mapped: %s", virtualPath)
	}

	info, err := os.Stat(realPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat: %w", err)
	}

	return info, nil
}

// ReadMappedPath reads the content of a mapped file (including files under mapped directories)
func (pm *PathMapper) ReadMappedPath(virtualPath string) ([]byte, error) {
	mapping, realPath := pm.ResolveMappedPath(virtualPath)
	if mapping == nil {
		return nil, fmt.Errorf("path not mapped: %s", virtualPath)
	}

	// Check if it's a directory
	info, err := os.Stat(realPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("cannot read directory as file: %s", virtualPath)
	}

	data, err := os.ReadFile(realPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}
