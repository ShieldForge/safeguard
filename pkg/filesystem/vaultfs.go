package filesystem

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"safeguard/pkg/logger"
	"safeguard/pkg/vault"

	"github.com/rs/zerolog"
	"github.com/winfsp/cgofuse/fuse"
)

// VaultFS implements a FUSE filesystem that provides transparent access to HashiCorp Vault
// secrets as if they were files and directories in a mounted filesystem.
//
// Features:
//   - Mount Vault secret engines as directories
//   - Access secrets as files through normal filesystem operations
//   - Optional path mapping to include local files in the virtual filesystem
//   - Process monitoring and audit logging
//   - Policy-based access control using OPA/REGO
//   - Cross-platform support (Windows, macOS, Linux)
//
// The filesystem is read-only by default for security. All operations are proxied
// to Vault's HTTP API, and the filesystem automatically discovers available
// secret engines through Vault's mount listing.
type VaultFS struct {
	fuse.FileSystemBase
	client          vault.ClientInterface
	debug           bool
	monitor         bool
	auditLogger     *zerolog.Logger
	auditFile       *os.File
	allowedPIDs     map[int]bool
	allowedUIDs     map[uint32]bool
	accessControl   bool
	policyEvaluator *PolicyEvaluator
	pathMapper      *PathMapper
	logger          *logger.Logger
}

// ProcessInfo holds information about the process that initiated a filesystem operation.
// This information is extracted from the FUSE context and used for access control,
// monitoring, and audit logging.
type ProcessInfo struct {
	UID uint32
	GID uint32
	PID int
}

// getProcessInfo safely gets process information from FUSE context
// Returns nil if not in a FUSE context (e.g., during tests)
// Note: PID can be -1 for system/kernel operations or when context is unavailable
func getProcessInfo() *ProcessInfo {
	defer func() {
		// Recover from panic if Getcontext fails (e.g., in tests)
		if r := recover(); r != nil {
			// Silently ignore - we're not in a FUSE context
		}
	}()

	uid, gid, pid := fuse.Getcontext()
	if pid == 0 {
		return nil // Not in FUSE context
	}

	// PID can be -1 for system/kernel operations or during mounting
	return &ProcessInfo{UID: uid, GID: gid, PID: pid}
}

// VaultFSOptions contains configuration options for creating a VaultFS instance.
//
// These options control various aspects of the filesystem's behavior, including
// debugging, monitoring, access control, and path mapping.
type VaultFSOptions struct {
	Debug             bool
	Monitor           bool
	AuditLogPath      string
	AllowedPIDs       []int
	AllowedUIDs       []uint32
	AccessControl     bool
	PolicyPath        string
	MappingConfigPath string
	Logger            *logger.Logger
}

// NewVaultFS creates a new VaultFS instance with default options.
//
// This is a convenience function that creates a VaultFS with minimal configuration.
// For more control over filesystem behavior, use NewVaultFSWithOptions instead.
//
// Parameters:
//   - client: A Vault client interface for accessing the Vault API
//   - debug: Whether to enable debug logging
func NewVaultFS(client vault.ClientInterface, debug bool) *VaultFS {
	return NewVaultFSWithOptions(client, VaultFSOptions{Debug: debug})
}

// NewVaultFSWithOptions creates a new VaultFS instance with advanced configuration options.
//
// This function provides full control over the filesystem's behavior, including:
//   - Process monitoring and audit logging
//   - Access control via PID/UID allow lists or REGO policies
//   - Path mapping for including local files in the virtual filesystem
//
// Options:
//   - Debug: Enable verbose debug logging
//   - Monitor: Enable process monitoring and detailed access logging
//   - AuditLogPath: Path to audit log file (if empty, no audit logging)
//   - AllowedPIDs: List of process IDs allowed to access the filesystem
//   - AllowedUIDs: List of user IDs allowed to access the filesystem
//   - AccessControl: Enable access control checks (requires PolicyPath or allow lists)
//   - PolicyPath: Path to REGO policy file/directory for fine-grained access control
//   - MappingConfigPath: Path to JSON configuration file for path mappings
//
// Returns a configured VaultFS ready to be mounted.
func NewVaultFSWithOptions(client vault.ClientInterface, opts VaultFSOptions) *VaultFS {
	// Use provided logger or create a default one
	log := opts.Logger
	if log == nil {
		log = logger.New(os.Stdout, opts.Debug)
	}

	fs := &VaultFS{
		client:        client,
		debug:         opts.Debug,
		monitor:       opts.Monitor,
		accessControl: opts.AccessControl,
		allowedPIDs:   make(map[int]bool),
		allowedUIDs:   make(map[uint32]bool),
		logger:        log,
	}

	// Set up allowed PIDs
	for _, pid := range opts.AllowedPIDs {
		fs.allowedPIDs[pid] = true
	}

	// Set up allowed UIDs
	for _, uid := range opts.AllowedUIDs {
		fs.allowedUIDs[uid] = true
	}

	// Initialize policy evaluator if policy path is specified
	if opts.PolicyPath != "" {
		policyEval, err := NewPolicyEvaluator(opts.PolicyPath)
		if err != nil {
			log.Warn("Failed to load policy file", map[string]interface{}{
				"policy_path": opts.PolicyPath,
				"error":       err.Error(),
			})
			log.Warn("Falling back to PID/UID-based access control", nil)
		} else {
			fs.policyEvaluator = policyEval
			log.Info("Policy-based access control enabled", map[string]interface{}{
				"policy_path": opts.PolicyPath,
			})
		}
	}

	// Open audit log if specified
	if opts.AuditLogPath != "" {
		logFile, err := os.OpenFile(opts.AuditLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Warn("Failed to open audit log", map[string]interface{}{
				"audit_log": opts.AuditLogPath,
				"error":     err.Error(),
			})
		} else {
			fs.auditFile = logFile
			// Create a dedicated zerolog logger for audit logs
			auditLogger := zerolog.New(logFile).With().Timestamp().Logger()
			fs.auditLogger = &auditLogger
			log.Info("Audit logging enabled", map[string]interface{}{
				"audit_log": opts.AuditLogPath,
			})
		}
	}

	// Initialize path mapper if mapping config is specified
	if opts.MappingConfigPath != "" {
		caseSensitive := runtime.GOOS != "windows"
		pathMapper := NewPathMapper(caseSensitive)

		err := pathMapper.LoadFromFile(opts.MappingConfigPath)
		if err != nil {
			log.Warn("Failed to load path mapping config", map[string]interface{}{
				"config_path": opts.MappingConfigPath,
				"error":       err.Error(),
			})
		} else {
			fs.pathMapper = pathMapper
			log.Info("Path mapping enabled", map[string]interface{}{
				"config_path": opts.MappingConfigPath,
				"mappings":    pathMapper.Count(),
			})
		}
	}

	return fs
}

// Mount mounts the filesystem at the specified mount point.
// It starts the FUSE event loop in a background goroutine and waits briefly
// for the mount to initialise. For callers that need reliable success/failure
// feedback (e.g. Windows services), use MountAsync instead.
func Mount(fs *VaultFS, mountPoint string) (*fuse.FileSystemHost, error) {
	host, errCh := MountAsync(fs, mountPoint)

	// Wait up to 5 seconds for the mount to either succeed or fail quickly.
	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
		// Mount goroutine returned nil (success then exit) – shouldn't happen
		// this fast, but treat as OK.
		return host, nil
	case <-time.After(5 * time.Second):
		// Still running → mount is alive and serving requests.
		return host, nil
	}
}

// MountAsync starts the FUSE event loop in a background goroutine and returns
// immediately. The returned channel receives nil if host.Mount() eventually
// exits cleanly, or an error if it fails. Callers should read from errCh to
// detect a mount failure that occurs after the function returns.
func MountAsync(fs *VaultFS, mountPoint string) (*fuse.FileSystemHost, <-chan error) {
	host := fuse.NewFileSystemHost(fs)
	options := getMountOptions(fs.debug)

	errCh := make(chan error, 1)
	go func() {
		if !host.Mount(mountPoint, options) {
			errCh <- fmt.Errorf("host.Mount returned false for %s", mountPoint)
		} else {
			errCh <- nil
		}
	}()

	return host, errCh
}

// getMountOptions returns platform-specific mount options
func getMountOptions(debug bool) []string {
	var options []string

	switch runtime.GOOS {
	case "windows":
		// Windows-specific options for WinFsp
		options = []string{
			"-o", "volname=VaultFS",
			"-o", "FileSystemName=VaultFS",
		}

	case "darwin":
		// macOS-specific options for OSXFUSE/macFUSE
		options = []string{
			"-o", "volname=VaultFS",
			"-o", "local",
			"-o", "allow_other",
		}

	case "linux":
		// Linux-specific options for FUSE
		options = []string{
			"-o", "fsname=VaultFS",
			"-o", "allow_other",
		}

	default:
		// Generic options for other Unix-like systems
		options = []string{
			"-o", "fsname=VaultFS",
		}
	}

	if debug {
		options = append(options, "-d")
	}

	return options
}

// Getattr gets file attributes
func (fs *VaultFS) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
	procInfo := getProcessInfo()

	// Log process information
	if fs.debug || fs.monitor {
		fs.logAccess("GETATTR", path, procInfo)
	}

	// Access control check
	if fs.accessControl && !fs.checkAccess(procInfo, path, "GETATTR") {
		if fs.debug {
			if procInfo != nil {
				fs.logger.Warn("Access denied", map[string]interface{}{
					"operation": "GETATTR",
					"path":      path,
					"pid":       procInfo.PID,
					"uid":       procInfo.UID,
				})
			} else {
				fs.logger.Warn("Access denied", map[string]interface{}{
					"operation": "GETATTR",
					"path":      path,
					"context":   "none",
				})
			}
		}
		fs.auditAccess("GETATTR", path, procInfo, false)
		return -fuse.EACCES
	}

	// Check if this path is mapped to a real file/directory
	if fs.pathMapper != nil && fs.pathMapper.IsMappedOrUnder(path) {
		info, err := fs.pathMapper.GetMappedPathInfo(path)
		if err != nil {
			if fs.debug {
				fs.logger.Error("Error getting mapped path info", map[string]interface{}{
					"path":  path,
					"error": err.Error(),
				})
			}
			return -fuse.ENOENT
		}

		// Fill in stat structure for the real file/directory
		if info.IsDir() {
			stat.Mode = fuse.S_IFDIR | 0755
			stat.Nlink = 2
		} else {
			stat.Mode = fuse.S_IFREG | 0444 // Regular file, read-only
			stat.Size = info.Size()
			stat.Nlink = 1
		}
		fs.auditAccess("GETATTR", path, procInfo, true)
		return 0
	}

	path = normalizePath(path)

	// Root directory
	if path == "" || path == "/" {
		stat.Mode = fuse.S_IFDIR | 0755
		stat.Nlink = 2
		fs.auditAccess("GETATTR", path, procInfo, true)
		return 0
	}

	// Check if it's a secret in Vault
	exists, isDir, err := fs.client.PathExists(context.Background(), path)
	if err != nil {
		if fs.debug {
			fs.logger.Error("Error checking path", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
		}
		return -fuse.ENOENT
	}

	if !exists {
		return -fuse.ENOENT
	}

	if isDir {
		stat.Mode = fuse.S_IFDIR | 0755
		stat.Nlink = 2
	} else {
		// It's a file (secret)
		stat.Mode = fuse.S_IFREG | 0644
		stat.Nlink = 1
		stat.Size = 4096 // Placeholder size
	}

	fs.auditAccess("GETATTR", path, procInfo, true)
	return 0
}

// Readdir reads directory contents
func (fs *VaultFS) Readdir(path string, fill func(name string, stat *fuse.Stat_t, ofst int64) bool, ofst int64, fh uint64) int {
	procInfo := getProcessInfo()

	// Log process information
	if fs.debug || fs.monitor {
		fs.logAccess("READDIR", path, procInfo)
	}

	// Access control check
	if fs.accessControl && !fs.checkAccess(procInfo, path, "READDIR") {
		if fs.debug {
			if procInfo != nil {
				fs.logger.Warn("Access denied", map[string]interface{}{
					"operation": "READDIR",
					"path":      path,
					"pid":       procInfo.PID,
					"uid":       procInfo.UID,
				})
			} else {
				fs.logger.Warn("Access denied", map[string]interface{}{
					"operation": "READDIR",
					"path":      path,
					"context":   "none",
				})
			}
		}
		fs.auditAccess("READDIR", path, procInfo, false)
		return -fuse.EACCES
	}

	fs.auditAccess("READDIR", path, procInfo, true)

	path = normalizePath(path)

	// Handle root directory listing specially - merge mapped and Vault entries
	if path == "" || path == "/" {
		// Add default entries
		fill(".", nil, 0)
		fill("..", nil, 0)

		// Track all entries to avoid duplicates
		addedEntries := make(map[string]bool)

		// If root is directly mapped to a directory, list its contents
		if fs.pathMapper != nil && fs.pathMapper.IsMappedOrUnder("") {
			entries, err := fs.pathMapper.ListMappedDirectory("")
			if err == nil {
				// Add mapped directory entries
				for _, entry := range entries {
					cleanEntry := strings.TrimSuffix(entry, "/")
					stat := &fuse.Stat_t{}
					if strings.HasSuffix(entry, "/") {
						stat.Mode = fuse.S_IFDIR | 0755
					} else {
						stat.Mode = fuse.S_IFREG | 0444
					}
					fill(cleanEntry, stat, 0)
					addedEntries[cleanEntry] = true
				}
			}
		}

		// Add root-level entries from mappings (first component of each mapped path)
		if fs.pathMapper != nil {
			rootEntries := fs.pathMapper.GetRootEntries()
			for entry, isDir := range rootEntries {
				// Skip if already added from root directory mapping
				if addedEntries[entry] {
					continue
				}
				stat := &fuse.Stat_t{}
				if isDir {
					stat.Mode = fuse.S_IFDIR | 0755
				} else {
					stat.Mode = fuse.S_IFREG | 0444
				}
				fill(entry, stat, 0)
				addedEntries[entry] = true
			}
		}

		// Add secrets from Vault at the root level
		entries, err := fs.client.List(context.Background(), path)
		if err == nil {
			for _, entry := range entries {
				// Remove trailing slash for comparison
				cleanEntry := strings.TrimSuffix(entry, "/")

				// Skip if already added from mappings
				if addedEntries[cleanEntry] {
					continue
				}

				stat := &fuse.Stat_t{}
				if strings.HasSuffix(entry, "/") {
					stat.Mode = fuse.S_IFDIR | 0755
					entry = cleanEntry
				} else {
					stat.Mode = fuse.S_IFREG | 0644
				}
				fill(entry, stat, 0)
			}
		} else if fs.debug {
			fs.logger.Error("Error listing Vault root", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
		}

		return 0
	}

	// Check if this path is a mapped directory (non-root)
	if fs.pathMapper != nil && fs.pathMapper.IsMappedOrUnder(path) {
		entries, err := fs.pathMapper.ListMappedDirectory(path)
		if err != nil {
			if fs.debug {
				fs.logger.Error("Error listing mapped directory", map[string]interface{}{
					"path":  path,
					"error": err.Error(),
				})
			}
			return -fuse.ENOENT
		}

		// Add default entries
		fill(".", nil, 0)
		fill("..", nil, 0)

		// Add mapped directory entries
		for _, entry := range entries {
			stat := &fuse.Stat_t{}
			if strings.HasSuffix(entry, "/") {
				stat.Mode = fuse.S_IFDIR | 0755
				entry = strings.TrimSuffix(entry, "/")
			} else {
				stat.Mode = fuse.S_IFREG | 0444
			}
			fill(entry, stat, 0)
		}

		return 0
	}

	// List secrets from Vault for non-root, non-mapped paths
	entries, err := fs.client.List(context.Background(), path)
	if err != nil {
		fs.logger.Error("Error listing path", map[string]interface{}{
			"path":  path,
			"error": err.Error(),
		})
		return -fuse.ENOENT
	}

	// Add default entries
	fill(".", nil, 0)
	fill("..", nil, 0)

	// Add actual entries
	for _, entry := range entries {
		stat := &fuse.Stat_t{}
		if strings.HasSuffix(entry, "/") {
			stat.Mode = fuse.S_IFDIR | 0755
			entry = strings.TrimSuffix(entry, "/")
		} else {
			stat.Mode = fuse.S_IFREG | 0644
		}
		fill(entry, stat, 0)
	}

	return 0
}

// Open opens a file
func (fs *VaultFS) Open(path string, flags int) (int, uint64) {
	procInfo := getProcessInfo()

	// Log process information
	if fs.debug || fs.monitor {
		fs.logAccess("OPEN", path, procInfo)
	}

	// Access control check
	if fs.accessControl && !fs.checkAccess(procInfo, path, "OPEN") {
		if fs.debug {
			if procInfo != nil {
				fs.logger.Warn("Access denied", map[string]interface{}{
					"operation": "OPEN",
					"path":      path,
					"pid":       procInfo.PID,
					"uid":       procInfo.UID,
				})
			} else {
				fs.logger.Warn("Access denied", map[string]interface{}{
					"operation": "OPEN",
					"path":      path,
					"context":   "none",
				})
			}
		}
		fs.auditAccess("OPEN", path, procInfo, false)
		return -fuse.EACCES, ^uint64(0)
	}

	// Check if this path is mapped to a real file/directory
	if fs.pathMapper != nil && fs.pathMapper.IsMappedOrUnder(path) {
		// Mapped files/directories exist, so return success
		fs.auditAccess("OPEN", path, procInfo, true)
		return 0, 0
	}

	path = normalizePath(path)

	// Check if secret exists
	exists, isDir, err := fs.client.PathExists(context.Background(), path)
	if err != nil || !exists || isDir {
		return -fuse.ENOENT, ^uint64(0)
	}

	fs.auditAccess("OPEN", path, procInfo, true)
	return 0, 0
}

// Read reads data from a file
func (fs *VaultFS) Read(path string, buff []byte, ofst int64, fh uint64) int {
	procInfo := getProcessInfo()

	// Log process information
	if fs.debug || fs.monitor {
		fs.logAccess("READ", path, procInfo)
	}

	// Access control check
	if fs.accessControl && !fs.checkAccess(procInfo, path, "READ") {
		if fs.debug {
			if procInfo != nil {
				fs.logger.Warn("Access denied", map[string]interface{}{
					"operation": "READ",
					"path":      path,
					"pid":       procInfo.PID,
					"uid":       procInfo.UID,
				})
			} else {
				fs.logger.Warn("Access denied", map[string]interface{}{
					"operation": "READ",
					"path":      path,
					"context":   "none",
				})
			}
		}
		fs.auditAccess("READ", path, procInfo, false)
		return -fuse.EACCES
	}

	// Check if this path is mapped to a real file
	if fs.pathMapper != nil && fs.pathMapper.IsMappedOrUnder(path) {
		content, err := fs.pathMapper.ReadMappedPath(path)
		if err != nil {
			if fs.debug {
				fs.logger.Error("Error reading mapped file", map[string]interface{}{
					"path":  path,
					"error": err.Error(),
				})
			}
			return -fuse.EIO
		}

		// Audit successful access
		fs.auditAccess("READ", path, procInfo, true)

		// Handle offset
		if ofst >= int64(len(content)) {
			return 0
		}

		// Copy data to buffer
		n := copy(buff, content[ofst:])
		return n
	}

	path = normalizePath(path)

	// Read secret from Vault
	data, err := fs.client.Read(context.Background(), path)
	if err != nil {
		if fs.debug {
			fs.logger.Error("Error reading secret", map[string]interface{}{
				"path":  path,
				"error": err.Error(),
			})
		}
		return -fuse.EIO
	}

	// Audit successful secret access
	fs.auditAccess("READ", path, procInfo, true)

	// Convert to string representation
	content := formatSecretData(data)

	// Handle offset
	if ofst >= int64(len(content)) {
		return 0
	}

	n := copy(buff, content[ofst:])
	return n
}

// logAccess logs process access information
func (fs *VaultFS) logAccess(operation, path string, procInfo *ProcessInfo) {
	if procInfo == nil {
		fs.logger.Info("Access operation", map[string]interface{}{
			"operation": operation,
			"path":      path,
			"context":   "none",
		})
		return
	}

	// Resolve process details for richer logging
	details := resolveProcessDetails(procInfo)
	fs.logger.Info("Access operation", map[string]interface{}{
		"operation":    operation,
		"path":         path,
		"pid":          procInfo.PID,
		"uid":          procInfo.UID,
		"gid":          procInfo.GID,
		"process_name": details.ProcessName,
		"process_path": details.ExecutablePath,
		"username":     details.Username,
	})
}

// checkAccess checks if the process has access based on policy or PID/UID restrictions
func (fs *VaultFS) checkAccess(procInfo *ProcessInfo, path, operation string) bool {
	if procInfo == nil {
		if fs.policyEvaluator != nil {
			fs.logger.Warn("Unable to determine process info", map[string]interface{}{
				"path":    path,
				operation: operation,
			})
		}
		return false
	}

	// Always allow system/kernel operations (PID -1) and the safeguard process itself
	// PID -1 indicates kernel/system context or unavailable context (common during mounting)
	// This is necessary for mounting and internal filesystem operations
	currentPID := os.Getpid()

	if currentPID == -1 {
		// try again to get current PID if it was unavailable before
		currentPID = os.Getpid()
	}

	if procInfo.PID == -1 || procInfo.PID == currentPID {
		if fs.debug {
			if procInfo.PID == -1 {
				fs.logger.Info("Access granted: System/kernel operation", map[string]interface{}{
					"path":    path,
					operation: operation,
					"pid":     -1,
				})
			} else {
				fs.logger.Info("Access granted: Current process", map[string]interface{}{
					"path":    path,
					operation: operation,
					"pid":     currentPID,
				})
			}
		}
		return true
	}

	// If policy evaluator is configured, use it
	if fs.policyEvaluator != nil {
		return fs.checkAccessWithPolicy(procInfo, path, operation)
	}

	// Fall back to legacy PID/UID-based access control
	// If no restrictions are set, allow all access
	if len(fs.allowedPIDs) == 0 && len(fs.allowedUIDs) == 0 {
		return true
	}

	// Check if PID is allowed
	if len(fs.allowedPIDs) > 0 {
		if fs.allowedPIDs[procInfo.PID] {
			return true
		}
	}

	// Check if UID is allowed
	if len(fs.allowedUIDs) > 0 {
		if fs.allowedUIDs[procInfo.UID] {
			return true
		}
	}

	return false
}

// checkAccessWithPolicy evaluates access using the policy evaluator
func (fs *VaultFS) checkAccessWithPolicy(procInfo *ProcessInfo, path, operation string) bool {
	ctx := context.Background()
	request := BuildAccessRequest(procInfo, path, operation)

	allowed, reason, err := fs.policyEvaluator.Evaluate(ctx, request)
	if err != nil {
		fs.logger.Error("Policy evaluation error", map[string]interface{}{
			"operation": operation,
			"path":      path,
			"error":     err.Error(),
		})
		return false
	}

	fs.logger.Info("Policy evaluation result", map[string]interface{}{
		"process_name": request.ProcessName,
		"process_path": request.ProcessPath,
		"username":     request.Username,
		"operation":    request.Operation,
		"path":         request.Path,
		"allowed":      allowed,
		"reason":       reason,
	})

	return allowed
}

// auditAccess writes access information to the audit log using zerolog
func (fs *VaultFS) auditAccess(operation, path string, procInfo *ProcessInfo, success bool) {
	if fs.auditLogger == nil {
		return
	}

	status := "SUCCESS"
	if !success {
		status = "DENIED"
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "failed to get hostname, error: " + err.Error()
	}

	// Build the audit log event
	event := fs.auditLogger.Info().
		Str("status", status).
		Str("operation", operation).
		Str("path", path).
		Str("hostname", hostname)

	if procInfo != nil {
		event = event.
			Int("pid", procInfo.PID).
			Uint32("uid", procInfo.UID).
			Uint32("gid", procInfo.GID)

		// Resolve process details
		details := resolveProcessDetails(procInfo)
		if details != nil {
			if details.ProcessName != "" {
				event = event.Str("process_name", details.ProcessName)
			}
			if details.ExecutablePath != "" {
				event = event.Str("process_path", details.ExecutablePath)
			}
			if details.Username != "" {
				event = event.Str("username", details.Username)
			}
		}
	}

	event.Msg("audit")
}

// Close closes the audit log file
func (fs *VaultFS) Close() error {
	if fs.auditFile != nil {
		return fs.auditFile.Close()
	}
	return nil
}

// normalizePath normalizes a path for Vault
func normalizePath(path string) string {
	// Ensure Windows-style separators are normalized even when running on Unix.
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	return path
}

// formatSecretData formats secret data as a readable string
func formatSecretData(data map[string]interface{}) []byte {
	var result strings.Builder

	for key, value := range data {
		result.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}

	return []byte(result.String())
}
