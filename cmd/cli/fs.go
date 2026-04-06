package main

import (
	"runtime"
	"safeguard/pkg/filesystem"
	"safeguard/pkg/logger"
	"safeguard/pkg/vault"

	"github.com/winfsp/cgofuse/fuse"
)

func buildFilesystem(log *logger.Logger, f *appFlags, vaultClient vault.ClientInterface, allowedPIDs []int, allowedUIDs []uint32) *filesystem.VaultFS {
	fsOpts := filesystem.VaultFSOptions{
		Debug:             *f.debug,
		Monitor:           *f.monitor,
		AuditLogPath:      *f.auditLog,
		AllowedPIDs:       allowedPIDs,
		AllowedUIDs:       allowedUIDs,
		AccessControl:     *f.accessControl,
		PolicyPath:        *f.policyPath,
		MappingConfigPath: *f.mappingConfig,
		Logger:            log,
	}
	return filesystem.NewVaultFSWithOptions(vaultClient, fsOpts)
}

func mountFilesystem(log *logger.Logger, fs *filesystem.VaultFS, mountPoint string) *fuse.FileSystemHost {
	log.Info("Mounting virtual drive", map[string]interface{}{
		"mount_point": mountPoint,
	})
	host, err := filesystem.Mount(fs, mountPoint)
	if err != nil {
		log.Fatal("Failed to mount filesystem", map[string]interface{}{
			"mount_point": mountPoint,
			"error":       err.Error(),
		})
	}
	log.Info("Virtual drive mounted successfully", map[string]interface{}{
		"mount_point": mountPoint,
	})
	return host
}

// getDefaultMountPoint returns the default mount point for the current OS
func getDefaultMountPoint() string {
	switch runtime.GOOS {
	case "windows":
		return "V:"
	case "darwin":
		return "/tmp/vault"
	case "linux":
		return "/mnt/vault"
	default:
		return "/tmp/vault"
	}
}
