package main

import (
	"fmt"
	"os"
	"path/filepath"
	"safeguard/pkg/filesystem"
	"safeguard/pkg/logger"
	"strconv"
	"strings"
)

func parseAccessLists(log *logger.Logger, f *appFlags) ([]int, []uint32) {
	var allowedPIDs []int
	var allowedUIDs []uint32

	if *f.allowedPIDsStr != "" {
		for _, pidStr := range strings.Split(*f.allowedPIDsStr, ",") {
			pidStr = strings.TrimSpace(pidStr)
			if pidStr == "" {
				continue
			}
			pid, err := strconv.Atoi(pidStr)
			if err != nil {
				log.Fatal("Invalid PID", map[string]interface{}{
					"pid":   pidStr,
					"error": err.Error(),
				})
			}
			allowedPIDs = append(allowedPIDs, pid)
		}
		if len(allowedPIDs) > 0 {
			log.Info("Access control configured", map[string]interface{}{
				"allowed_pids": allowedPIDs,
			})
		}
	}

	if *f.allowedUIDsStr != "" {
		for _, uidStr := range strings.Split(*f.allowedUIDsStr, ",") {
			uidStr = strings.TrimSpace(uidStr)
			if uidStr == "" {
				continue
			}
			uid, err := strconv.ParseUint(uidStr, 10, 32)
			if err != nil {
				log.Fatal("Invalid UID", map[string]interface{}{
					"uid":   uidStr,
					"error": err.Error(),
				})
			}
			allowedUIDs = append(allowedUIDs, uint32(uid))
		}
		if len(allowedUIDs) > 0 {
			log.Info("Access control configured", map[string]interface{}{
				"allowed_uids": allowedUIDs,
			})
		}
	}

	return allowedPIDs, allowedUIDs
}

func validatePolicies(log *logger.Logger, f *appFlags) {
	if *f.monitor {
		log.Info("Process monitoring enabled", nil)
	}
	if *f.auditLog != "" {
		log.Info("Audit logging enabled", map[string]interface{}{
			"audit_log": *f.auditLog,
		})
	}
	if *f.policyPath != "" {
		if err := filesystem.ValidatePolicy(*f.policyPath); err != nil {
			log.Fatal("Invalid policy file", map[string]interface{}{
				"policy_path": *f.policyPath,
				"error":       err.Error(),
			})
		}
		log.Info("Policy-based access control enabled", map[string]interface{}{
			"policy_path": *f.policyPath,
		})
	}
}

// extractEmbeddedPolicies writes the build-time embedded policy files to a
// temporary directory and returns its path. The caller is responsible for
// cleaning up the directory when it is no longer needed.
func extractEmbeddedPolicies() (string, error) {
	dir, err := os.MkdirTemp("", "safeguard-policies-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	for name, content := range embeddedPolicyFiles {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			os.RemoveAll(dir)
			return "", fmt.Errorf("failed to write policy %s: %w", name, err)
		}
	}
	return dir, nil
}
